// Package getmeta 提供外部站点元数据抓取。
//
// 当前实现：DLsite（音声 / 游戏）。
//
// 设计要点（都是实测踩出来的，改之前先看）：
//  1. 必须走 HTTPS。http:// 会被重定向，且部分网络下直接超时。
//  2. 12 位新编号（RJ01721474）**前导零不能丢**，丢了页面返回 404「該当作品がありません」。
//  3. 必须发成人确认 Cookie（adultchecked=1），否则 R18 作品页拿不到完整内容。
//  4. 标题要从 <h1 id="work_name"> 的**文本**取。早期页面是
//     <h1 id="work_name"><a>标题</a></h1>，现在的页面里没有 <a>，
//     只找 a 会得到空串（go.felesatra.moe/dlsite 就栽在这里，故本项目弃用它自写）。
//  5. 曲目表选择器 ol.work_tracklist_list 在当前页面已不存在，音声曲目改由 Vue 组件渲染，
//     故本实现不解析曲目，只取静态 HTML 里稳定的基础字段。
package getmeta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ===== 常量 =====

const (
	dlsiteBaseURL      = "https://www.dlsite.com"
	dlsiteManiaxWork   = dlsiteBaseURL + "/maniax/work/=/product_id/%s.html"
	dlsiteProWork      = dlsiteBaseURL + "/pro/work/=/product_id/%s.html"
	dlsiteSearchManiax = dlsiteBaseURL + "/maniax/fsr/=/language/jp/keyword/%s/work_type_category/game/"

	// 默认超时：页面约 120~160KB，正常网络 1s 内可回；留 20s 给慢网络。
	defaultTimeout = 20 * time.Second
	// 搜索候选返回上限
	defaultCandidateLimit = 5
	// 标签数量上限（0 表示不限制）
	defaultTagLimit = 24
	// 响应体读取上限，防止异常页面把内存吃满
	maxBodyBytes = 8 << 20
	// 默认 UA 里带上项目名，便于对方识别
	defaultUserAgent = "WeedyBox/0.1 (+https://github.com/WeedyMornbrize3/WeedyBox)"
)

// RJ/RE/VJ + 至少 4 位数字；新编号是 8 位数字（含前导零）
var dlsiteIDRegex = regexp.MustCompile(`(?i)\b(RJ|RE|VJ)\d{4,}\b`)

// 系列编号只出现在页面内嵌 JSON 里（"series_id":"SRI0000043774"）
var seriesIDRegex = regexp.MustCompile(`"series_id"\s*:\s*"(SRI\d+)"`)

// 封面 URL 里显式的尺寸后缀，例如 _img_main_240x240.jpg → 240
var coverSizeSuffixRegex = regexp.MustCompile(`_(\d+)x(\d+)\.[a-z0-9]+$`)

// ===== 数据结构 =====

// DLsiteWork 是一次抓取的结果。字段都是页面静态 HTML 里稳定提供的；
// 某个作品没有的字段留零值（例如单作没有「系列名」时 Series 为空）。
//
// 价格与评分在页面里是 <meta itemprop="…" content="…">，**值在 content 属性上**，
// 取 .Text() 只会得到空串——这是实测踩过的坑。
type DLsiteWork struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	AlternateID string    `json:"alternateId,omitempty"` // itemprop="alternateName" 的拼音 slug
	Maker       string    `json:"maker"`
	MakerID     string    `json:"makerId,omitempty"`
	Series      string    `json:"series,omitempty"`
	SeriesID    string    `json:"seriesId,omitempty"`    // 页面内嵌 JSON 的 series_id，如 SRI0000043774
	WorkType    string    `json:"workType,omitempty"`    // 作品形式，如「ボイス・ASMR」
	AgeRating   string    `json:"ageRating,omitempty"`   // 年龄指定，如 R18 / 全年齢
	FileFormats []string  `json:"fileFormats,omitempty"` // 文件格式，如 WAV / MP3
	FileSize    string    `json:"fileSize,omitempty"`    // 文件容量，如 16.83GB
	ReleaseDate string    `json:"releaseDate,omitempty"` // 归一化为 2006-01-02
	Price       string    `json:"price,omitempty"`       // 现金价格，如 1540
	Currency    string    `json:"currency,omitempty"`    // 币种，如 JPY
	PriceUntil  string    `json:"priceUntil,omitempty"`  // 价格有效期（ISO8601）
	Rating      string    `json:"rating,omitempty"`      // 平均分，如 5
	RatingCount string    `json:"ratingCount,omitempty"`
	CoverURL    string    `json:"coverUrl,omitempty"`
	Summary     string    `json:"summary,omitempty"` // 作品内容（简介）
	Tags        []string  `json:"tags,omitempty"`
	Outline     []Outline `json:"outline,omitempty"` // #work_outline 的原始 kv，未识别字段也不丢
	SourceURL   string    `json:"sourceUrl"`
	FetchedAt   time.Time `json:"fetchedAt"`
}

// Outline 是作品信息表（#work_outline）里的一行，保留日文原始标签，
// 这样页面新增字段时前端也能拿到，不必等后端加字段。
type Outline struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// DLsiteSearchItem 是搜索结果里的一项（只含列表页能拿到的信息）。
type DLsiteSearchItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Cover string `json:"cover,omitempty"`
}

// ===== 服务 =====

// MetaService 负责从外部站点抓取元数据。
type MetaService struct {
	client   *http.Client
	tagLimit int
}

// MetaOption 用于调整默认行为。
type MetaOption func(*MetaService)

// WithHTTPClient 替换 HTTP 客户端（测试时可注入自定义 Transport / 超时）。
func WithHTTPClient(c *http.Client) MetaOption {
	return func(s *MetaService) { s.client = c }
}

// WithTagLimit 设置标签数量上限，0 表示不限制。
func WithTagLimit(n int) MetaOption {
	return func(s *MetaService) { s.tagLimit = n }
}

// NewMetaService 创建一个抓取服务；不传选项时用默认客户端与标签上限。
func NewMetaService(options ...MetaOption) *MetaService {
	s := &MetaService{
		client:   &http.Client{Timeout: defaultTimeout},
		tagLimit: defaultTagLimit,
	}
	for _, opt := range options {
		opt(s)
	}
	if s.client == nil {
		s.client = &http.Client{Timeout: defaultTimeout}
	}
	return s
}

// ===== 对外方法（Wails 会把这些导出成前端绑定）=====

// GetWorkByID 按 RJ/RE/VJ 编号抓取作品信息。
func (s *MetaService) GetWorkByID(ctx context.Context, id string) (*DLsiteWork, error) {
	normalized, ok := NormalizeDLsiteID(id)
	if !ok {
		return nil, fmt.Errorf("无效的 DLsite 编号: %q（应形如 RJ01721474 / RE123456 / VJ123456）", id)
	}

	doc, status, err := s.fetchDocument(ctx, buildWorkURL(normalized))
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("DLsite 返回状态码 %d（编号 %s 不存在或页面结构变化）", status, normalized)
	}
	return s.parseWork(doc, normalized)
}

// SearchWorkByName 按关键词搜索，返回候选列表（不抓详情）。
func (s *MetaService) SearchWorkByName(ctx context.Context, keyword string) ([]DLsiteSearchItem, error) {
	text := strings.TrimSpace(keyword)
	if text == "" {
		return nil, errors.New("搜索关键词不能为空")
	}
	doc, status, err := s.fetchDocument(ctx, buildSearchURL(text))
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("DLsite 搜索返回状态码 %d", status)
	}

	items := make([]DLsiteSearchItem, 0, 16)
	doc.Find(".search_result_img_box_inner").Each(func(_ int, sel *goquery.Selection) {
		item, ok := parseSearchItem(sel)
		if !ok {
			return
		}
		items = append(items, item)
	})
	return items, nil
}

// GetWorkByName 搜索后取第一个候选并抓取详情。
func (s *MetaService) GetWorkByName(ctx context.Context, keyword string) (*DLsiteWork, error) {
	items, err := s.SearchWorkByName(ctx, keyword)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("没有搜索到结果")
	}
	if len(items) > defaultCandidateLimit {
		items = items[:defaultCandidateLimit]
	}
	return s.GetWorkByID(ctx, items[0].ID)
}

// ===== HTTP =====

// buildWorkURL VJ 走 pro 站点，其余（RJ/RE）走 maniax。
func buildWorkURL(id string) string {
	if strings.HasPrefix(id, "VJ") {
		return fmt.Sprintf(dlsiteProWork, id)
	}
	return fmt.Sprintf(dlsiteManiaxWork, id)
}

func buildSearchURL(keyword string) string {
	escaped := strings.ReplaceAll(url.PathEscape(keyword), "%20", "+")
	return fmt.Sprintf(dlsiteSearchManiax, escaped)
}

// fetchDocument 发请求并解析成 goquery 文档。
// 返回值里的 status 让调用方自己决定怎么报错（404 与网络错误要区分开）。
func (s *MetaService) fetchDocument(ctx context.Context, reqURL string) (*goquery.Document, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja,en;q=0.8")
	// 成人内容确认：不带它时 R18 页面会退化，简介与类型标签会缺
	req.Header.Set("Cookie", "adultchecked=1; locale=zh-cn") //请求头调整语言 locale

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求 DLsite 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("解析 HTML 失败: %w", err)
	}
	return doc, resp.StatusCode, nil
}

// ===== 解析 =====

func (s *MetaService) parseWork(doc *goquery.Document, id string) (*DLsiteWork, error) {
	// 只取 #work_name 的文本，不要再往里找 <a>——现在的页面没有 <a>。
	name := cleanText(doc.Find("#work_name").First().Text())
	if name == "" {
		return nil, fmt.Errorf("页面里没有找到作品名（编号 %s，可能页面结构已变化）", id)
	}

	outline := parseOutline(doc)
	makerLink := doc.Find("#work_maker span.maker_name a").First()

	work := &DLsiteWork{
		ID:          id,
		Name:        name,
		AlternateID: metaContent(doc, "alternateName"),
		Maker:       cleanText(makerLink.Text()),
		MakerID:     lastPathSegment(makerLink.AttrOr("href", ""), "maker_id/"),
		Series:      outlineValue(outline, "シリーズ名"),
		SeriesID:    extractSeriesID(doc),
		WorkType:    outlineValue(outline, "作品形式"),
		AgeRating:   outlineValue(outline, "年齢指定"),
		FileSize:    outlineValue(outline, "ファイル容量"),
		ReleaseDate: normalizeJapaneseDate(outlineValue(outline, "販売日")),
		// 价格与评分都是 <meta itemprop="…" content="…">，值在 content 属性上
		Price:       metaContent(doc, "price"),
		Currency:    metaContent(doc, "priceCurrency"),
		PriceUntil:  metaContent(doc, "priceValidUntil"),
		Rating:      metaContent(doc, "ratingValue"),
		RatingCount: metaContent(doc, "ratingCount"),
		CoverURL:    extractCoverURL(doc),
		Summary:     extractSummary(doc),
		Tags:        s.extractTags(doc, outline),
		Outline:     outline,
		SourceURL:   buildWorkURL(id),
		FetchedAt:   time.Now(),
	}
	work.FileFormats = splitSlashList(outlineValue(outline, "ファイル形式"))
	return work, nil
}

// parseOutline 把 #work_outline 的每一行「th → td」抽出来。
// 用「标签→值」的通用形式而不是硬编码日文标签，页面加字段时不用改代码。
func parseOutline(doc *goquery.Document) []Outline {
	out := make([]Outline, 0, 16)
	seen := map[string]bool{}
	doc.Find("#work_outline tr").Each(func(_ int, tr *goquery.Selection) {
		label := cleanText(tr.Find("th").First().Text())
		if label == "" || seen[label] {
			return
		}
		td := tr.Find("td").First().Clone()
		td.Find("span.add_follow, template, script").Remove()

		value := cleanText(td.Text())
		if value == "" {
			// 「作品形式」这类行的文本被图标吃掉时，退回 title 属性
			if title, ok := td.Find("[title]").First().Attr("title"); ok {
				value = cleanText(title)
			}
		}
		seen[label] = true
		out = append(out, Outline{Label: label, Value: value})
	})
	return out
}

func outlineValue(outline []Outline, label string) string {
	for _, o := range outline {
		if o.Label == label {
			return o.Value
		}
	}
	return ""
}

// extractSummary 取「作品内容」。优先限定在 #main_inner 内，
// 避免匹配到页面其它 itemprop="description"（例如导航栏的站点简介）。
func extractSummary(doc *goquery.Document) string {
	sel := doc.Find(`#main_inner div[itemprop="description"]`).First()
	if sel.Length() == 0 {
		sel = doc.Find(`div[itemprop="description"]`).First()
	}
	return cleanText(sel.Text())
}

// extractCoverURL 找封面图。
//
// 优先级刻意写成「越大越优先」，而不是页面里谁先出现：
// `#work_thumb img` 排在很前面，但它指向 _img_main_240x240.jpg（只有 46KB），
// 于是旧实现总是拿到最小那张。og:image / itemprop="image" / slider 才是大图。
func extractCoverURL(doc *goquery.Document) string {
	if u := normalizeSourceURL(doc.Find(`meta[property="og:image"]`).First().AttrOr("content", "")); u != "" {
		return u
	}
	if u := normalizeSourceURL(doc.Find(`meta[itemprop="image"]`).First().AttrOr("content", "")); u != "" {
		return u
	}
	return extractCoverURLWidest(doc.Find("img, source"))
}

// extractCoverURLWidest 扫描页面图片，在同一个作品的多档位里挑最宽的一张。
// 档位宽度来自 URL 后缀：_smp1（1470×1000）> 无后缀 _img_main（560×420）
// > _img_smp（不确定）> _img_main_240x240（240×240）。
func extractCoverURLWidest(sel *goquery.Selection) string {
	best, bestScore := "", 0
	sel.Each(func(_ int, s *goquery.Selection) {
		for _, raw := range []string{
			s.AttrOr("data-src", ""),
			firstSrcSetURL(s.AttrOr("srcset", "")),
			s.AttrOr("src", ""),
		} {
			u := normalizeSourceURL(raw)
			if u == "" || !strings.Contains(u, "_img_") {
				continue
			}
			if score := coverWidthScore(u); score > bestScore {
				best, bestScore = u, score
			}
		}
	})
	return best
}

// coverWidthScore 给 DLsite 的封面 URL 估一个「宽度档位」分数。
func coverWidthScore(raw string) int {
	u := strings.ToLower(raw)
	if m := coverSizeSuffixRegex.FindStringSubmatch(u); len(m) == 3 {
		w, _ := strconv.Atoi(m[1])
		return w
	}
	switch {
	case strings.Contains(u, "_smp"):
		return 1470 // 预览大图 _smp1
	case strings.Contains(u, "_img_main"):
		return 560 // 标准封面（og:image 就是这一档）
	default:
		return 1
	}
}

// extractTags 收集类型标签：.main_genre 里的链接（ジャンル）+ 作品形式 / 年齢指定 / 販売形式。
func (s *MetaService) extractTags(doc *goquery.Document, outline []Outline) []string {
	names := make([]string, 0, 24)
	doc.Find(".main_genre a").Each(func(_ int, sel *goquery.Selection) {
		names = append(names, sel.Text())
	})
	for _, label := range []string{"作品形式", "年齢指定", "販売形式"} {
		if v := outlineValue(outline, label); v != "" {
			names = append(names, v)
		}
	}

	// 去重（不区分大小写），保持页面顺序
	seen := map[string]bool{}
	out := make([]string, 0, len(names))
	for _, raw := range names {
		name := cleanText(raw)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, name)
		if s.tagLimit > 0 && len(out) >= s.tagLimit {
			break
		}
	}
	return out
}

func parseSearchItem(sel *goquery.Selection) (DLsiteSearchItem, bool) {
	id, _ := NormalizeDLsiteID(sel.AttrOr("data-list_item_product_id", ""))
	if id == "" {
		id, _ = NormalizeDLsiteID(sel.Find("a.work_thumb_inner").First().AttrOr("href", ""))
	}
	if id == "" {
		return DLsiteSearchItem{}, false
	}

	nameSel := sel.Find(".work_name a").First()
	name := cleanText(nameSel.AttrOr("title", ""))
	if name == "" {
		name = cleanText(nameSel.Text())
	}
	if name == "" {
		return DLsiteSearchItem{}, false
	}

	cover := ""
	if img := sel.Find("img").First(); img.Length() > 0 {
		cover = normalizeSourceURL(firstNonEmpty(
			img.AttrOr("data-src", ""),
			firstSrcSetURL(img.AttrOr("srcset", "")),
			img.AttrOr("src", ""),
		))
	}
	return DLsiteSearchItem{ID: id, Name: name, Cover: cover}, true
}

// ===== 小工具 =====

// NormalizeDLsiteID 从任意字符串里抽出 RJ/RE/VJ 编号并转大写。
// 注意：**保留前导零**（RJ01721474 不能变成 RJ1721474，后者页面是 404）。
func NormalizeDLsiteID(raw string) (string, bool) {
	match := dlsiteIDRegex.FindString(strings.TrimSpace(raw))
	if match == "" {
		return "", false
	}
	return strings.ToUpper(match), true
}

// cleanText 压掉所有空白（含换行），页面里简介会带大量缩进空白。
func cleanText(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}

// metaContent 取 <meta itemprop="name" content="值"> 的 content。
// 价格、评分、币种在页面里都只有 content 属性，没有文本节点。
func metaContent(doc *goquery.Document, itemprop string) string {
	return cleanText(doc.Find(fmt.Sprintf(`meta[itemprop="%s"]`, itemprop)).First().AttrOr("content", ""))
}

// extractSeriesID 从页面内嵌 JSON 里取系列编号。
// 注意 FindStringSubmatch 在没匹配时返回 nil，**必须**判长度，
// 否则单作（没有系列）会直接 panic。
func extractSeriesID(doc *goquery.Document) string {
	m := seriesIDRegex.FindStringSubmatch(doc.Text())
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// normalizeJapaneseDate 把「2026年09月26日」/「2026年10月07日 0時」归一化成 2006-01-02。
// 解析不出来时原样返回，不丢信息。
func normalizeJapaneseDate(raw string) string {
	text := cleanText(raw)
	if text == "" {
		return ""
	}
	replaced := strings.NewReplacer(
		"年", "-", "月", "-", "日", "",
		".", "-", "/", "-",
	).Replace(text)

	parts := strings.Split(replaced, "-")
	if len(parts) >= 3 {
		year, yErr := strconv.Atoi(strings.TrimSpace(parts[0]))
		month, mErr := strconv.Atoi(strings.TrimSpace(parts[1]))
		// 「07日 0時」会把第三段变成「07 0時」，取前两个字符即可
		dayText := strings.TrimSpace(parts[2])
		if len(dayText) > 2 {
			dayText = dayText[:2]
		}
		day, dErr := strconv.Atoi(dayText)
		if yErr == nil && mErr == nil && dErr == nil &&
			month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	}
	return text
}

// splitSlashList 把「WAV / mp3 / mp4同梱」拆成 ["WAV","mp3","mp4同梱"]。
func splitSlashList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := cleanText(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func firstSrcSetURL(srcset string) string {
	first := strings.TrimSpace(strings.Split(srcset, ",")[0])
	if first == "" {
		return ""
	}
	return strings.Fields(first)[0]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// normalizeSourceURL 把相对 / 协议相对地址补成绝对地址。
func normalizeSourceURL(raw string) string {
	value := strings.TrimSpace(raw)
	switch {
	case value == "":
		return ""
	case strings.HasPrefix(value, "//"):
		return "https:" + value
	case strings.HasPrefix(value, "http://"), strings.HasPrefix(value, "https://"):
		return value
	case strings.HasPrefix(value, "/"):
		return dlsiteBaseURL + value
	default:
		return value
	}
}

// lastPathSegment 从 ".../maker_id/RG51931.html" 里取出 "RG51931"。
func lastPathSegment(raw, marker string) string {
	i := strings.Index(raw, marker)
	if i < 0 {
		return ""
	}
	return strings.TrimSuffix(raw[i+len(marker):], ".html")
}
