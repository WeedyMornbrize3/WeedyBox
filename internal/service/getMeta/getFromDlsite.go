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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

// ===== 常量 =====

const (
	dlsiteBaseURL      = "https://www.dlsite.com"
	dlsiteManiaxWork   = dlsiteBaseURL + "/maniax/work/=/product_id/%s.html"
	dlsiteProWork      = dlsiteBaseURL + "/pro/work/=/product_id/%s.html"
	// 搜索页模板：per_page 固定 30 —— DLsite 对 per_page 有硬上限（实测传 60/120 也只回 30 条），
	// 想要更多结果只能翻页，因此后端按 page 逐页取（见 SearchWorks / parseSearchPageCount）。
	dlsiteSearchManiax = dlsiteBaseURL + "/maniax/fsr/=/language/jp/keyword/%s/work_type_category/game/per_page/%d/page/%d/"

	// 默认超时：页面约 120~160KB，正常网络 1s 内可回；留 20s 给慢网络。
	defaultTimeout = 20 * time.Second
	// DLsite 搜索每页条数（写死 30：服务端对 per_page 上限就是 30）
	searchPerPage = 30
	// 搜索候选返回上限
	defaultCandidateLimit = 5
	// 标签数量上限（0 表示不限制）。
	// 后端放宽到 64：具体显示多少个由前端设置项决定（默认 24），这里只保证"取得到"。
	defaultTagLimit = 64
	// 响应体读取上限，防止异常页面把内存吃满
	maxBodyBytes = 8 << 20
	// 默认 UA 里带上项目名，便于对方识别
	defaultUserAgent = "WeedyBox/0.1 (+https://github.com/WeedyMornbrize3/WeedyBox)"
)

// RJ/RE/VJ + 至少 4 位数字；新编号是 8 位数字（含前导零）
var dlsiteIDRegex = regexp.MustCompile(`(?i)\b(RJ|RE|VJ)\d{4,}\b`)

// 系列编号只出现在内嵌 JSON 里（"series_id":"SRI0000043774"）
var seriesIDRegex = regexp.MustCompile(`"series_id"\s*:\s*"(SRI\d+)"`)

// 现役页面（2026-11 起）的系列链接：/maniax/fsr/=/title_id/SRI0000026263/order/...
var seriesIDInHrefRegex = regexp.MustCompile(`(SRI\d+)`)

// 参与「标签」的信息表字段（中日文候选）。
// 用户要求标签要包含：声优 / 分类 / 插画 / 剧情 / 作者 / 社团名。
// 注意：分类与声优是**多值字段**，不走这里 —— 它们在页面里由多个元素组成，
// 取整段 dd 文本会把相邻值粘成一串（中文页面里元素之间没有空格）。
var tagOutlineLabels = [][]string{
	{"サークル名", "社团名", "社團名"},
	{"作者"},
	{"シナリオ", "剧情"},
	{"イラスト", "插画"},
}

// 社团编号：/circle/profile/=/maker_id/RG24495.html 或 JSON-LD brand.url
var makerIDRegex = regexp.MustCompile(`(RG\d+)`)

// 搜索分页链接里的页码：.../per_page/30/page/2
var searchPageRegex = regexp.MustCompile(`/page/(\d+)`)

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

// SearchWorkByName 按关键词搜索，返回**第一页**候选（不抓详情）。
func (s *MetaService) SearchWorkByName(ctx context.Context, keyword string) ([]DLsiteSearchItem, error) {
	items, _, _, err := s.SearchResultPage(ctx, keyword, 1)
	return items, err
}

// SearchResultPage 取搜索结果里的某一页：返回该页候选 + 总页数 + 结果总数。
//
// DLsite 每页固定 30 条（per_page 改大无效），所以「不限制结果数」只能靠逐页取；
// 总数来自页面上的 `data-search-result-count` 属性（如 1710），
// 总页数优先由总数算出，取不到时退回分页块 `table.global_pagination .page_no` 的最大页码。
func (s *MetaService) SearchResultPage(ctx context.Context, keyword string, page int) ([]DLsiteSearchItem, int, int, error) {
	text := strings.TrimSpace(keyword)
	if text == "" {
		return nil, 0, 0, errors.New("搜索关键词不能为空")
	}
	if page < 1 {
		page = 1
	}
	doc, status, err := s.fetchDocument(ctx, buildSearchURL(text, page))
	if err != nil {
		return nil, 0, 0, err
	}
	if status != http.StatusOK {
		return nil, 0, 0, fmt.Errorf("DLsite 搜索返回状态码 %d", status)
	}

	items := make([]DLsiteSearchItem, 0, searchPerPage)
	doc.Find(".search_result_img_box_inner").Each(func(_ int, sel *goquery.Selection) {
		item, ok := parseSearchItem(sel)
		if !ok {
			return
		}
		items = append(items, item)
	})

	total := parseSearchResultCount(doc)
	pageCount := 0
	if total > 0 {
		pageCount = (total + searchPerPage - 1) / searchPerPage
	}
	if pageCount <= 0 {
		pageCount = parseSearchPageCount(doc)
	}
	return items, pageCount, total, nil
}

// parseSearchResultCount 取结果总数：页面根节点带 data-search-result-count（如 1710）。
// 取不到返回 0，由调用方退回"按分页块最大页码估算"。
func parseSearchResultCount(doc *goquery.Document) int {
	raw := strings.TrimSpace(doc.Find("[data-search-result-count]").First().AttrOr("data-search-result-count", ""))
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(strings.ReplaceAll(raw, ",", ""))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// parseSearchPageCount 从分页块里取最大页码（即总页数）；没有分页块就是 1 页。
func parseSearchPageCount(doc *goquery.Document) int {
	count := 1
	doc.Find("table.global_pagination .page_no a").Each(func(_ int, sel *goquery.Selection) {
		match := searchPageRegex.FindStringSubmatch(sel.AttrOr("href", ""))
		if len(match) < 2 {
			return
		}
		if n, err := strconv.Atoi(match[1]); err == nil && n > count {
			count = n
		}
	})
	return count
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

func buildSearchURL(keyword string, page int) string {
	escaped := strings.ReplaceAll(url.PathEscape(keyword), "%20", "+")
	if page < 1 {
		page = 1
	}
	return fmt.Sprintf(dlsiteSearchManiax, escaped, searchPerPage, page)
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

	// 页面有两代模板，两代都解析、谁有值用谁：
	//   旧（fixture 里的页面）：#work_outline 的 th→td 表格 + #work_maker + <meta itemprop>
	//   现役（2026-11 起）：div.c-productInfo 的 dt→dd 列表 + <script ld+json> 的 Product
	// 只认旧模板的后果是整页解析成"只有名字+封面"的空壳 —— 扫描入库那批就是这么坏的。
	outline := parseOutline(doc)
	if len(outline) == 0 {
		outline = parseProductInfo(doc)
	}
	product := parseProductJSONLD(doc)

	makerLink := doc.Find("#work_maker span.maker_name a").First()
	maker := cleanText(makerLink.Text())
	makerID := lastPathSegment(makerLink.AttrOr("href", ""), "maker_id/")
	if maker == "" {
		maker = outlineValueAny(outline, "サークル名", "社团名", "社團名")
	}
	if makerID == "" {
		if m := makerIDRegex.FindStringSubmatch(productInfoLinkAny(doc, "サークル名", "社团名", "社團名")); len(m) == 2 {
			makerID = m[1]
		}
	}
	if makerID == "" {
		if m := makerIDRegex.FindStringSubmatch(jsonLDValue(product, "brand", "url")); len(m) == 2 {
			makerID = m[1]
		}
	}

	seriesID := extractSeriesID(doc)
	if seriesID == "" {
		seriesID = seriesIDFromPageLinks(doc)
	}
	if seriesID == "" {
		if m := seriesIDInHrefRegex.FindStringSubmatch(productInfoLinkAny(doc, "シリーズ名", "系列名")); len(m) == 2 {
			seriesID = m[1]
		}
	}

	work := &DLsiteWork{
		ID:          id,
		Name:        name,
		AlternateID: firstNonEmpty(metaContent(doc, "alternateName"), jsonLDValue(product, "alternateName")),
		Maker:       firstNonEmpty(maker, jsonLDValue(product, "brand", "name")),
		MakerID:     makerID,
		Series:      outlineValueAny(outline, "シリーズ名", "系列名"),
		SeriesID:    seriesID,
		WorkType:    outlineValueAny(outline, "作品形式"),
		AgeRating:   outlineValueAny(outline, "年齢指定", "年龄指定", "年齡指定"),
		FileSize:    outlineValueAny(outline, "ファイル容量", "文件容量", "檔案容量"),
		ReleaseDate: normalizeJapaneseDate(outlineValueAny(outline, "販売日", "发售日", "發售日")),
		// 价格/评分：旧页面在 <meta itemprop> 的 content 上，现役页面在 JSON-LD Product 里
		Price:       firstNonEmpty(jsonLDValue(product, "offers", "price"), metaContent(doc, "price")),
		Currency:    firstNonEmpty(jsonLDValue(product, "offers", "priceCurrency"), metaContent(doc, "priceCurrency")),
		PriceUntil:  metaContent(doc, "priceValidUntil"),
		Rating:      firstNonEmpty(jsonLDValue(product, "aggregateRating", "ratingValue"), metaContent(doc, "ratingValue")),
		RatingCount: firstNonEmpty(jsonLDValue(product, "aggregateRating", "ratingCount"), metaContent(doc, "ratingCount")),
		CoverURL:    extractCoverURL(doc),
		Summary:     extractSummary(doc),
		Tags:        s.extractTags(doc, outline),
		Outline:     outline,
		SourceURL:   buildWorkURL(id),
		FetchedAt:   time.Now(),
	}
	// 文件格式：旧页面是「WAV / mp3」；现役页面把每个格式放在 dd 里独立的 span/a 上
	if formats := productInfoPartsAny(doc, "ファイル形式", "文件形式", "檔案形式"); len(formats) > 0 {
		work.FileFormats = formats
	} else {
		work.FileFormats = splitSlashList(outlineValueAny(outline, "ファイル形式", "文件形式", "檔案形式"))
	}
	return work, nil
}

// parseProductInfo 解析现役页面的「dt 标签 → dd 值」商品信息列表。
// 同样是「标签→值」，所以直接复用 Outline 结构（作品库详情页的字段表原样能显示）。
func parseProductInfo(doc *goquery.Document) []Outline {
	out := make([]Outline, 0, 12)
	seen := map[string]bool{}
	doc.Find("div.c-productInfo__item").Each(func(_ int, item *goquery.Selection) {
		label := cleanText(item.Find("dt.c-productInfo__key").First().Text())
		if label == "" || seen[label] {
			return
		}
		dd := item.Find("dd.c-productInfo__value").First()
		if dd.Length() == 0 {
			return
		}
		value := productInfoText(dd)
		if value == "" {
			return
		}
		seen[label] = true
		out = append(out, Outline{Label: label, Value: value})
	})
	return out
}

// productInfoText 取 dd 的纯文本（丢掉 script/template，避免把内嵌 JSON 拼进来）。
func productInfoText(sel *goquery.Selection) string {
	trimmed := sel.Clone()
	trimmed.Find("script, template").Remove()
	return cleanText(trimmed.Text())
}

// productInfoLink 取「标签」对应 dd 里第一个链接的 href（社团主页 / 系列页）。
func productInfoLink(doc *goquery.Document, label string) string {
	href := ""
	doc.Find("div.c-productInfo__item").EachWithBreak(func(_ int, item *goquery.Selection) bool {
		if cleanText(item.Find("dt.c-productInfo__key").First().Text()) != label {
			return true
		}
		href = item.Find("dd a").First().AttrOr("href", "")
		return false
	})
	return href
}

// productInfoParts 把「标签」对应 dd 里的每个链接/标签单独取出来（文件格式、ジャンル 这类多值字段）。
func productInfoParts(doc *goquery.Document, label string) []string {
	parts := make([]string, 0, 8)
	doc.Find("div.c-productInfo__item").Each(func(_ int, item *goquery.Selection) {
		if cleanText(item.Find("dt.c-productInfo__key").First().Text()) != label {
			return
		}
		item.Find("dd a, dd span").Each(func(_ int, tag *goquery.Selection) {
			if value := cleanText(tag.Text()); value != "" {
				parts = append(parts, value)
			}
		})
	})
	return parts
}

// productInfoLinkAny 按多个候选标签取 dd 里的链接（标签随 locale 变）。
func productInfoLinkAny(doc *goquery.Document, labels ...string) string {
	for _, label := range labels {
		if href := productInfoLink(doc, label); href != "" {
			return href
		}
	}
	return ""
}

// productInfoPartsAny 按多个候选标签取 dd 里的多值字段（标签随 locale 变）。
func productInfoPartsAny(doc *goquery.Document, labels ...string) []string {
	for _, label := range labels {
		if parts := productInfoParts(doc, label); len(parts) > 0 {
			return parts
		}
	}
	return nil
}

// parseProductJSONLD 找页面里 @type=Product 的 JSON-LD。
// 现役页面把社团、价格、评分、拼音 slug、简介都放在这里（原来的 <meta itemprop> 已消失），
// 这是比 CSS class 更稳的来源 —— schema.org 的字段名不会跟着站点改版变。
func parseProductJSONLD(doc *goquery.Document) map[string]any {
	var product map[string]any
	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, script *goquery.Selection) bool {
		var data map[string]any
		if err := json.Unmarshal([]byte(script.Text()), &data); err != nil {
			return true
		}
		if kind, _ := data["@type"].(string); kind == "Product" {
			product = data
			return false
		}
		return true
	})
	return product
}

// jsonLDValue 按 key 路径取 JSON-LD 里的标量值（价格是数字、评分是字符串，统一成文本）。
func jsonLDValue(product map[string]any, keys ...string) string {
	var current any = product
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[key]
	}
	switch value := current.(type) {
	case string:
		return cleanText(value)
	case float64:
		if value == math.Trunc(value) {
			return strconv.FormatInt(int64(value), 10)
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	return ""
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

// outlineValueAny 按多个候选标签取值。
// DLsite 的标签会跟着 locale（Cookie locale=ja / zh-cn）变化，
// 例如 販売日 / 发售日、ファイル容量 / 文件容量 —— 只认日文标签会在中文页上全部取不到。
func outlineValueAny(outline []Outline, labels ...string) string {
	for _, label := range labels {
		if value := outlineValue(outline, label); value != "" {
			return value
		}
	}
	return ""
}

// extractSummary 取「作品内容」。优先限定在 #main_inner 内，
// 避免匹配到页面其它 itemprop="description"（例如导航栏的站点简介）。
// 现役页面没有 itemprop，简介在 section.c-productContent 的正文块里。
func extractSummary(doc *goquery.Document) string {
	sel := doc.Find(`#main_inner div[itemprop="description"]`).First()
	if sel.Length() == 0 {
		sel = doc.Find(`div[itemprop="description"]`).First()
	}
	if sel.Length() == 0 {
		sel = doc.Find(`.c-productContent__text`).First()
	}
	if sel.Length() == 0 {
		sel = doc.Find(`section.c-productContent`).First()
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

// extractTags 收集标签，来源比旧实现广：
//   - 旧页面 .main_genre 的链接、现役页面分类（href*="/genre/"）的每个链接；
//   - 作品形式（按页面上的 a/span 逐个取，避免中文页把多个值拼成一串）；
//   - 信息表里的 社团名 / 作者 / 剧情 / 插画 / 声优 / 分类 / 年龄指定 / 販売形式。
//
// 最后统一切分：用户要求按「空白字符」和 "/" 分割（DLsite 多值字段常写成
// 「双声道立体声/人头麦」「みたかりん 野上菜月」），再去重并按 tagLimit 截断。
func (s *MetaService) extractTags(doc *goquery.Document, outline []Outline) []string {
	raw := make([]string, 0, 32)

	// 旧页面：.main_genre 里的链接
	doc.Find(".main_genre a").Each(func(_ int, sel *goquery.Selection) {
		raw = append(raw, cleanText(sel.Text()))
	})

	// 现役页面：分类标签是 dd 里指向 /genre/ 的一组 <a>。
	// 用 href 识别而不是标题文字 —— 中文页面里这一栏叫「分类」、日文叫「ジャンル」。
	genreCount := 0
	doc.Find("div.c-productInfo__item").Each(func(_ int, item *goquery.Selection) {
		if item.Find(`dd a[href*="/genre/"]`).Length() == 0 {
			return
		}
		item.Find(`dd a[href*="/genre/"]`).Each(func(_ int, tag *goquery.Selection) {
			raw = append(raw, cleanText(tag.Text()))
			genreCount++
		})
	})
	// 一栏分类链接都没有（极旧/极简页面）才退回整段文本
	if genreCount == 0 {
		if value := outlineValueAny(outline, "ジャンル", "分类", "分類"); value != "" {
			raw = append(raw, value)
		}
	}

	// 作品形式：现役页面每个值在独立的 a/span 上，按元素取；
	// 中文页面里文本节点之间没有空格，直接取 dd 文本会变成「模拟有配音有音乐有动画」。
	if parts := productInfoPartsAny(doc, "作品形式"); len(parts) > 0 {
		raw = append(raw, parts...)
	} else if value := outlineValueAny(outline, "作品形式"); value != "" {
		raw = append(raw, value)
	}

	// 声优：一个声优一个元素（中文页面里整段文本没有空格，必须按元素取）
	if parts := productInfoPartsAny(doc, "声優", "声优", "聲優"); len(parts) > 0 {
		raw = append(raw, parts...)
	} else if value := outlineValueAny(outline, "声優", "声优", "聲優"); value != "" {
		raw = append(raw, value)
	}

	// 信息表里的其它字段（用户指定要进标签的那几项）
	for _, labels := range tagOutlineLabels {
		if value := outlineValueAny(outline, labels...); value != "" {
			raw = append(raw, value)
		}
	}
	if value := outlineValueAny(outline, "年齢指定", "年龄指定", "年齡指定"); value != "" {
		raw = append(raw, value)
	}
	if value := outlineValueAny(outline, "販売形式", "发售形式"); value != "" {
		raw = append(raw, value)
	}

	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, piece := range raw {
		for _, tag := range splitTagText(piece) {
			key := strings.ToLower(tag)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, tag)
			if s.tagLimit > 0 && len(out) >= s.tagLimit {
				return out
			}
		}
	}
	return out
}

// splitTagText 按空白字符与 "/" 切分标签文本（用户要求的切分口径）。
func splitTagText(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '/' || unicode.IsSpace(r)
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if value := cleanText(field); value != "" {
			out = append(out, value)
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

	cover := extractSearchItemCover(sel)
	return DLsiteSearchItem{ID: id, Name: name, Cover: cover}, true
}

// extractSearchItemCover 取搜索结果里的缩略图地址。
//
// 现代搜索页的缩略图是个 Vue 组件：
//
//	<thumb-with-ng-filter-block :thumb-candidates="['//img.dlsite.jp/...webp','//img.dlsite.jp/...jpg']">
//
// 它里面唯一的 <img> 是**弹层占位图**（src 是内联 gif data URL），
// 早期版本的解析把它当封面，于是列表里什么都不显示、只有点详情才有图
// （详情页的提取走 extractCoverURL，和这里不是一条路）。所以先读
// `:thumb-candidates` 的第一个候选，退路才看传统图片属性，且跳过 data: 占位。
func extractSearchItemCover(sel *goquery.Selection) string {
	if raw := sel.Find("thumb-with-ng-filter-block").First().AttrOr(":thumb-candidates", ""); raw != "" {
		if url := firstThumbCandidate(raw); url != "" {
			return normalizeSourceURL(url)
		}
	}

	img := sel.Find("img").First()
	if img.Length() == 0 {
		return ""
	}
	for _, candidate := range []string{
		img.AttrOr("data-src", ""),
		img.AttrOr("data-lazy-src", ""),
		firstSrcSetURL(img.AttrOr("srcset", "")),
		img.AttrOr("src", ""),
	} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || strings.HasPrefix(candidate, "data:") {
			continue // 内联占位图不是封面
		}
		return normalizeSourceURL(candidate)
	}
	return ""
}

// firstThumbCandidate 从 Vue 的数组字面量属性里取第一项：
// "['//img/a.webp','//img/a.jpg']" → "//img/a.webp"。
func firstThumbCandidate(raw string) string {
	for _, quote := range []string{"'", `"`} {
		start := strings.Index(raw, quote)
		if start < 0 {
			continue
		}
		rest := raw[start+1:]
		if end := strings.Index(rest, quote); end >= 0 {
			if value := strings.TrimSpace(rest[:end]); value != "" {
				return value
			}
		}
	}
	return ""
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

// seriesIDFromPageLinks 从「系列页链接」里取系列编号（不依赖日文标签，
// 中文页面里那一行叫「系列名」）：/maniax/fsr/=/title_id/SRI0000026263/...
func seriesIDFromPageLinks(doc *goquery.Document) string {
	href := ""
	doc.Find(`a[href*="title_id/"]`).EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		href = sel.AttrOr("href", "")
		return href == ""
	})
	if m := seriesIDInHrefRegex.FindStringSubmatch(href); len(m) == 2 {
		return m[1]
	}
	return ""
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
