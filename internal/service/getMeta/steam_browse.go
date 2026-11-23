// internal/service/getMeta/steam_browse.go
//
// Steam 的「第二套接口」（2026-11-22 接入，接口清单见 doc/steam-api.md）：
//
//   - IStoreBrowseService/GetItems：一次拿一批 appid 的**用户标签**（tagid + 权重）与
//     **评价汇总**。appdetails 的 genres/categories 只是商店分类，不是玩家打的标签，
//     所以 Steam 的 tags 改由这里出（用户要求「tag 的获取尝试使用别的接口」）。
//   - IStoreService/GetMostPopularTags：tagid → 中文名（GetItems 只给 id）。进程内缓存 24h。
//   - store.steampowered.com/api/storesearch：按名字搜 appid。两条用途：
//     ① 游戏库「手动搜索」可以切到 Steam 来源；
//     ② 扫描「按文件夹名」时，先拿文件夹名来猜 appid（Steam 源原本只能靠 appmanifest，
//     即"必须已安装"才能识别）。
//   - ISteamNews / ISteamUserStats：/works 的「Steam 动态」（公告 + 在线人数）。
//
// 这几个接口都有"会变"的风险（GetItems 的 key 文档写必填、匿名实际可用），
// 所以每个调用都允许单独失败：拿不到就退回原来的行为，绝不因此让整条抓取链失败。
package getmeta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	// steamTagCacheTTL 标签字典的有效期：内容是"最热门标签"的名字表，一天足够了。
	steamTagCacheTTL = 24 * time.Hour
	// steamBrowseBatch GetItems 批量上限。实测 100 个 id 能一次拿完，按 100 切批。
	steamBrowseBatch = 100
	// steamSearchLimit storesearch 单次最多返回 10 条（服务端硬限制，没有分页参数）。
	steamSearchLimit = 10
	// steamNewsLimit /works 显示几条公告。
	steamNewsLimit = 3
	// steamMatchThreshold 文件夹名与商店名字的相似度下限（1 = 完全一致；否则是编辑距离相似度）。
	steamMatchThreshold = 0.75
	// steamNameFloor 低于这个相似度就直接算"不像"，连候选都不考虑（避免「Game」这类
	// 通用名把所有搜索结果都当成匹配）。
	steamNameFloor = 0.6
)

// ===== GetItems =====

// steamBrowseItem 只声明用得到的字段（GetItems 返回字段极多，整份留在 raw 里）。
type steamBrowseItem struct {
	AppID   int    `json:"appid"`
	Name    string `json:"name"`
	Visible bool   `json:"visible"`
	// tags / tagids 是玩家标签（带权重），中文名靠 GetMostPopularTags 换。
	TagIDs []int `json:"tagids"`
	Tags   []struct {
		TagID  int   `json:"tagid"`
		Weight int64 `json:"weight"`
	} `json:"tags"`
	Reviews struct {
		SummaryFiltered struct {
			ReviewCount      int64  `json:"review_count"`
			PercentPositive  int    `json:"percent_positive"`
			ReviewScore      int    `json:"review_score"`
			ReviewScoreLabel string `json:"review_score_label"`
		} `json:"summary_filtered"`
	} `json:"reviews"`
	Assets struct {
		AssetURLFormat string `json:"asset_url_format"`
		LibraryCapsule string `json:"library_capsule"`
	} `json:"assets"`
}

// fetchSteamBrowse 拿一批 appid 的标签 + 评价（一次最多 steamBrowseBatch 个）。
func (s *MetaService) fetchSteamBrowse(ctx context.Context, appIDs []string) ([]steamBrowseItem, error) {
	if len(appIDs) == 0 {
		return nil, errors.New("没有要查询的 appid")
	}
	if len(appIDs) > steamBrowseBatch {
		appIDs = appIDs[:steamBrowseBatch]
	}
	ids := make([]map[string]any, 0, len(appIDs))
	for _, raw := range appIDs {
		appID := NormalizeSteamAppID(raw)
		if appID == "" {
			continue
		}
		n, err := strconv.Atoi(appID)
		if err != nil {
			continue
		}
		ids = append(ids, map[string]any{"appid": n})
	}
	if len(ids) == 0 {
		return nil, errors.New("没有合法的 appid")
	}

	payload := map[string]any{
		"ids":     ids,
		"context": map[string]any{"language": "schinese", "country_code": "CN"},
		"data_request": map[string]any{
			// 只要标签与评价：字段越少越小、越不容易被改。
			"include_tag_count": 20,
			"include_reviews":   true,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("拼装 GetItems 请求失败: %w", err)
	}

	respBody, err := s.steamGetJSON(ctx, steamBrowseItemsAPIURL+"?input_json="+url.QueryEscape(string(body)))
	if err != nil {
		return nil, err
	}
	return parseSteamBrowseItems(respBody)
}

// parseSteamBrowseItems 解析 GetItems 的外层结构（{"response":{"store_items":[…]}}）。
func parseSteamBrowseItems(body []byte) ([]steamBrowseItem, error) {
	var resp struct {
		Response struct {
			StoreItems []steamBrowseItem `json:"store_items"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析 Steam GetItems 返回失败: %w", err)
	}
	return resp.Response.StoreItems, nil
}

// ===== 标签字典（tagid → 中文名）=====

// steamTagCache 是进程级缓存：标签字典与 appid 无关、也不变（24h 内），
// 每抓一个游戏都重新拉一次 15KB 没有意义。
var steamTagCache = struct {
	sync.Mutex
	names map[int]string
	at    time.Time
}{}

// steamTagNames 取 tagid → 中文名；失败返回 nil（调用方退化成"没有用户标签"）。
func (s *MetaService) steamTagNames(ctx context.Context) map[int]string {
	steamTagCache.Lock()
	if len(steamTagCache.names) > 0 && time.Since(steamTagCache.at) < steamTagCacheTTL {
		names := steamTagCache.names
		steamTagCache.Unlock()
		return names
	}
	steamTagCache.Unlock()

	body, err := s.steamGetJSON(ctx, steamPopularTagsAPIURL+"?language=schinese")
	if err != nil {
		return nil
	}
	names, err := parseSteamTagNames(body)
	if err != nil || len(names) == 0 {
		return nil
	}
	steamTagCache.Lock()
	steamTagCache.names = names
	steamTagCache.at = time.Now()
	steamTagCache.Unlock()
	return names
}

// parseSteamTagNames 解析 GetMostPopularTags（{"response":{"tags":[{tagid,name}]}}）。
func parseSteamTagNames(body []byte) (map[int]string, error) {
	var resp struct {
		Response struct {
			Tags []struct {
				TagID int    `json:"tagid"`
				Name  string `json:"name"`
			} `json:"tags"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析 Steam 标签字典失败: %w", err)
	}
	names := make(map[int]string, len(resp.Response.Tags))
	for _, tag := range resp.Response.Tags {
		if tag.TagID > 0 && strings.TrimSpace(tag.Name) != "" {
			names[tag.TagID] = strings.TrimSpace(tag.Name)
		}
	}
	return names, nil
}

// steamUserTags 把 GetItems 的标签（按权重降序）翻成中文名；字典里没有的 id 直接丢。
func steamUserTags(item steamBrowseItem, names map[int]string) []string {
	if len(names) == 0 {
		return nil
	}
	type weighted struct {
		id     int
		weight int64
		order  int
	}
	list := make([]weighted, 0, len(item.Tags))
	for i, tag := range item.Tags {
		list = append(list, weighted{id: tag.TagID, weight: tag.Weight, order: i})
	}
	if len(list) == 0 {
		for i, id := range item.TagIDs {
			list = append(list, weighted{id: id, weight: 0, order: i})
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].weight != list[j].weight {
			return list[i].weight > list[j].weight
		}
		return list[i].order < list[j].order
	})

	seen := make(map[string]struct{}, len(list))
	tags := make([]string, 0, len(list))
	for _, tag := range list {
		name := strings.TrimSpace(names[tag.id])
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tags = append(tags, name)
	}
	return tags
}

// mergeSteamTags 把用户标签放在前面、原有分类标签接在后面，去重并截断。
func mergeSteamTags(primary, base []string) []string {
	seen := make(map[string]struct{}, len(primary)+len(base))
	out := make([]string, 0, len(primary)+len(base))
	add := func(raw string) {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			return
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, tag)
	}
	for _, tag := range primary {
		add(tag)
	}
	for _, tag := range base {
		add(tag)
	}
	if len(out) > steamTagLimit {
		out = out[:steamTagLimit]
	}
	return out
}

// enrichSteamWork 用 GetItems 补两样东西：真实的玩家标签、评价汇总。
// **任何一步失败都只是"少一点信息"**，不返回错误 —— 详情已经抓到了，不该因此报错。
func (s *MetaService) enrichSteamWork(ctx context.Context, appID string, work map[string]any) {
	items, err := s.fetchSteamBrowse(ctx, []string{appID})
	if err != nil || len(items) == 0 {
		fmt.Printf("Steam 标签/评价补全失败（保留原有标签）: %v\n", err)
		return
	}
	item := items[0]

	// ① 标签：玩家标签在前，原来的「分类/类型/开发商/发行商」在后
	if userTags := steamUserTags(item, s.steamTagNames(ctx)); len(userTags) > 0 {
		base := stringSlice(work["tags"])
		work["tags"] = mergeSteamTags(userTags, base)
	}

	// ② 评价汇总：存进作品 map（离线也能显示），同时补一行 outline
	reviews := item.Reviews.SummaryFiltered
	if reviews.ReviewCount > 0 {
		work["steamReviews"] = map[string]any{
			"count":           reviews.ReviewCount,
			"percentPositive": reviews.PercentPositive,
			"score":           reviews.ReviewScore,
			"label":           strings.TrimSpace(reviews.ReviewScoreLabel),
		}
		if outline, ok := work["outline"].([]map[string]string); ok {
			text := fmt.Sprintf("%s %d%%", strings.TrimSpace(reviews.ReviewScoreLabel), reviews.PercentPositive)
			if strings.TrimSpace(reviews.ReviewScoreLabel) == "" {
				text = fmt.Sprintf("%d%% 好评", reviews.PercentPositive)
			}
			if reviews.ReviewCount > 0 {
				text = fmt.Sprintf("%s（%d 篇）", text, reviews.ReviewCount)
			}
			work["outline"] = append(outline, map[string]string{"label": "Steam 评价", "value": text})
		}
	}

	// ③ 原始 GetItems 数据留一份：作品界面以后要展示截图/价格等，不用重抓
	rawMap := map[string]any{}
	if blob, err := json.Marshal(item); err == nil {
		_ = json.Unmarshal(blob, &rawMap)
	}
	if len(rawMap) > 0 {
		work["steamBrowse"] = rawMap
	}
}

// ===== 按名字搜索（storesearch）=====

// steamSearchItem 是店铺搜索的一条候选（前端候选列表直接用）。
type steamSearchItem struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Cover   string `json:"cover,omitempty"`
	Price   string `json:"price,omitempty"`
	Type    string `json:"type,omitempty"`
	Matched bool   `json:"matched,omitempty"`
}

// steamSearch 用名字搜 Steam 商店（最多 10 条）。
func (s *MetaService) steamSearch(ctx context.Context, term string) ([]steamSearchItem, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, errors.New("搜索关键词为空")
	}
	body, err := s.steamGetJSON(
		ctx,
		fmt.Sprintf("%s?term=%s&l=schinese&cc=CN", steamStoreSearchAPIURL, url.QueryEscape(term)),
	)
	if err != nil {
		return nil, err
	}
	return parseSteamSearchItems(body)
}

// parseSteamSearchItems 解析 storesearch 的返回（{"total":10,"items":[…]}}）。
func parseSteamSearchItems(body []byte) ([]steamSearchItem, error) {
	var resp struct {
		Total int `json:"total"`
		Items []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			ID    int    `json:"id"`
			Price *struct {
				Currency string `json:"currency"`
				Final    int64  `json:"final"`
			} `json:"price"`
			TinyImage string `json:"tiny_image"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析 Steam 搜索返回失败: %w", err)
	}
	out := make([]steamSearchItem, 0, len(resp.Items))
	for _, item := range resp.Items {
		name := strings.TrimSpace(item.Name)
		if item.ID <= 0 || name == "" {
			continue
		}
		entry := steamSearchItem{
			ID:    strconv.Itoa(item.ID),
			Name:  name,
			Cover: strings.TrimSpace(item.TinyImage),
			Type:  strings.ToLower(strings.TrimSpace(item.Type)),
		}
		if item.Price != nil && item.Price.Final > 0 {
			entry.Price = formatSteamPrice(item.Price.Currency, item.Price.Final)
		}
		out = append(out, entry)
	}
	if len(out) > steamSearchLimit {
		out = out[:steamSearchLimit]
	}
	return out, nil
}

func formatSteamPrice(currency string, cents int64) string {
	symbol := map[string]string{"CNY": "¥", "USD": "$", "EUR": "€", "JPY": "¥"}[strings.ToUpper(currency)]
	text := fmt.Sprintf("%.2f", float64(cents)/100)
	if symbol == "" {
		return strings.TrimSpace(currency + " " + text)
	}
	return symbol + text
}

// MatchSteamApp 用**文件夹名**去 Steam 商店猜一个 appid（扫描「按文件夹名」用）。
//
// 为什么要猜：Steam 源原本只能靠 appmanifest（= 必须已安装）识别；用户拿来的
// 游戏文件夹往往叫「黑神话悟空」「ULTRAKILL v1.0」，和商店名不完全一样。
//
// 做法：清洗文件夹名 → 商店搜索 → 按"归一化后完全一致 > 编辑距离相似度"打分。
// 搜索词太长时逐级截短重试（Steam 的搜索是按词的："黑神话悟空" 0 条、"黑神话" 9 条）。
func (s *MetaService) MatchSteamApp(ctx context.Context, folderName string) (*steamSearchItem, error) {
	cleaned := steamFolderQuery(folderName)
	if cleaned == "" {
		return nil, nil
	}
	folderNorm := normalizeSteamName(folderName)

	var candidates []steamSearchItem
	for _, query := range steamQueryLadder(cleaned) {
		items, err := s.steamSearch(ctx, query)
		if err != nil {
			continue
		}
		if len(items) > 0 {
			candidates = items
			break
		}
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	best, score := pickSteamCandidate(folderNorm, candidates)
	if best == nil || score < steamMatchThreshold {
		return nil, nil
	}
	best.Matched = true
	return best, nil
}

// steamFolderQuery 把文件夹名清洗成适合当搜索词的形态：
// 去掉【】/[]/() 里的注释、去掉结尾的点分版本号（v1.0 / 2.1.3），下划线换成空格。
func steamFolderQuery(name string) string {
	s := name
	for _, pair := range [][2]string{{"【", "】"}, {"[", "]"}, {"(", ")"}, {"（", "）"}} {
		for {
			start := strings.Index(s, pair[0])
			if start < 0 {
				break
			}
			end := strings.Index(s[start+len(pair[0]):], pair[1])
			if end < 0 {
				break
			}
			s = s[:start] + " " + s[start+len(pair[0])+end+len(pair[1]):]
		}
	}
	// 结尾的版本号：必须带小数点，避免把 "Hades 2" 这种真正的名字里的数字削掉
	for {
		trimmed := strings.TrimSpace(s)
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			return ""
		}
		last := fields[len(fields)-1]
		core := strings.TrimPrefix(strings.TrimPrefix(last, "v"), "V")
		if strings.Contains(core, ".") && isDigitsAndDots(core) {
			s = strings.Join(fields[:len(fields)-1], " ")
			continue
		}
		break
	}
	s = strings.NewReplacer("_", " ", "\u3000", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func isDigitsAndDots(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) && r != '.' {
			return false
		}
	}
	return true
}

// steamQueryLadder 给出搜索词的回退阶梯（长词逐级截短）。
// Steam 的搜索按词匹配：把「黑神话悟」这种半个词发过去是 0 条，截到词尾才有结果。
func steamQueryLadder(cleaned string) []string {
	out := []string{cleaned}
	runes := []rune(cleaned)
	n := len(runes)
	if n < 4 {
		return out
	}
	for _, frac := range []float64{0.75, 0.5, 0.35} {
		length := int(float64(n) * frac)
		if length < 2 || length >= n {
			continue
		}
		query := strings.TrimSpace(string(runes[:length]))
		if query == "" {
			continue
		}
		duplicate := false
		for _, existing := range out {
			if existing == query {
				duplicate = true
				break
			}
		}
		if !duplicate {
			out = append(out, query)
		}
	}
	return out
}

// normalizeSteamName 只保留字母与数字（含中文），用来做"长得像不像"的比较。
func normalizeSteamName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// pickSteamCandidate 在候选里挑最像文件夹名的那个，返回它与得分。
// 分数：完全一致 = 2；否则 = 编辑距离相似度（<1）。
func pickSteamCandidate(folderNorm string, items []steamSearchItem) (*steamSearchItem, float64) {
	var best *steamSearchItem
	bestScore := 0.0
	bestIsNoise := true
	for i := range items {
		candidateNorm := normalizeSteamName(items[i].Name)
		score := steamNameScore(folderNorm, candidateNorm)
		if score <= 0 {
			continue
		}
		noise := steamNoiseName(items[i].Name)
		// 同一分数时优先选"正片"（不是原声/试玩/升级包），再选名字更短的
		if score > bestScore ||
			(score == bestScore && bestIsNoise && !noise) ||
			(score == bestScore && bestIsNoise == noise && len([]rune(items[i].Name)) < len([]rune(best.Name))) {
			best = &items[i]
			bestScore = score
			bestIsNoise = noise
		}
	}
	return best, bestScore
}

// steamNameScore 完全一致给 2 分；否则按**编辑距离相似度**给 0~1。
// 相似度 = 1 - 编辑距离 / 较长串长度；低于 steamNameFloor 的算"不像"，返回 0。
//
// 为什么不只比公共前缀（2026-11-23 改）：文件夹名和商店名常常差一两个字母 ——
// 「demon root」vs「Demons Roots」、「黑神话悟」vs「黑神话：悟空」这类明明就是同一个游戏，
// 按前缀占比却会判成 0，于是扫描「按文件夹名」时永远猜不到 Steam 源。
func steamNameScore(folderNorm, candidateNorm string) float64 {
	if folderNorm == "" || candidateNorm == "" {
		return 0
	}
	if folderNorm == candidateNorm {
		return 2
	}
	folder := []rune(folderNorm)
	candidate := []rune(candidateNorm)
	longest := max(len(folder), len(candidate))
	score := 1 - float64(editDistance(folder, candidate))/float64(longest)
	if score < steamNameFloor {
		return 0
	}
	return score
}

// editDistance 是两串的 Levenshtein 距离（按 rune 算：中文/日文名不能按字节切）。
// 只用来比较短游戏名，O(n*m) 的滚动数组足够。Go 1.21+ 的 min 内建只支持有序类型。
func editDistance(left, right []rune) int {
	prev := make([]int, len(right)+1)
	curr := make([]int, len(right)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(left); i++ {
		curr[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 1
			if left[i-1] == right[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[len(right)]
}

// steamNoiseName 判断候选是不是"原声/试玩/工具/升级包"这类陪跑条目。
func steamNoiseName(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range []string{
		"soundtrack", "ost", "original soundtrack", "demo", "试玩", "原声", "音乐精选",
		"tools", "benchmark", "工具", "升级包", "expansion pass", "season pass", "artbook", "画集",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// ===== /works 的「Steam 动态」=====

// steamNewsItem 是一条公告（date 是 Unix 秒，前端自己格式化）。
type steamNewsItem struct {
	GID    string `json:"gid"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Author string `json:"author,omitempty"`
	Date   int64  `json:"date"`
	Feed   string `json:"feedlabel,omitempty"`
}

// FetchSteamStats 取一个 Steam 游戏的动态：评价汇总 + 在线人数 + 最近公告。
// 三项各自独立，全部失败才报错（有一项能用就返回那一项）。
func (s *MetaService) FetchSteamStats(ctx context.Context, rawAppID string) (map[string]any, error) {
	appID := NormalizeSteamAppID(rawAppID)
	if appID == "" {
		return nil, errors.New("Steam 游戏编号无效（应为纯数字 appid）")
	}

	out := map[string]any{
		"id":        appID,
		"fetchedAt": time.Now().Format(time.RFC3339),
	}
	failed := 0

	if items, err := s.fetchSteamBrowse(ctx, []string{appID}); err == nil && len(items) > 0 {
		reviews := items[0].Reviews.SummaryFiltered
		if reviews.ReviewCount > 0 {
			out["reviews"] = map[string]any{
				"count":           reviews.ReviewCount,
				"percentPositive": reviews.PercentPositive,
				"score":           reviews.ReviewScore,
				"label":           strings.TrimSpace(reviews.ReviewScoreLabel),
			}
		}
	} else {
		failed++
	}

	if players, err := s.fetchSteamCurrentPlayers(ctx, appID); err == nil {
		out["players"] = players
	} else {
		failed++
	}

	if news, err := s.fetchSteamNews(ctx, appID); err == nil {
		out["news"] = news
	} else {
		failed++
	}

	if failed >= 3 {
		return nil, errors.New("拿不到 Steam 动态（评价 / 在线人数 / 公告都请求失败了）")
	}
	return out, nil
}

// fetchSteamCurrentPlayers 当前在线人数（"response.result" 为 1 才算成功）。
func (s *MetaService) fetchSteamCurrentPlayers(ctx context.Context, appID string) (int, error) {
	body, err := s.steamGetJSON(ctx, steamCurrentPlayersAPIURL+"?appid="+url.QueryEscape(appID))
	if err != nil {
		return 0, err
	}
	var resp struct {
		Response struct {
			PlayerCount int `json:"player_count"`
			Result      int `json:"result"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("解析 Steam 在线人数失败: %w", err)
	}
	if resp.Response.Result != 1 {
		return 0, errors.New("Steam 没有返回在线人数")
	}
	return resp.Response.PlayerCount, nil
}

// fetchSteamNews 最近公告（标题/链接/作者/时间）。
func (s *MetaService) fetchSteamNews(ctx context.Context, appID string) ([]steamNewsItem, error) {
	u := fmt.Sprintf("%s?appid=%s&count=%d&maxlength=200", steamNewsAPIURL, url.QueryEscape(appID), steamNewsLimit)
	body, err := s.steamGetJSON(ctx, u)
	if err != nil {
		return nil, err
	}
	var resp struct {
		AppNews struct {
			NewsItems []steamNewsItem `json:"newsitems"`
		} `json:"appnews"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("解析 Steam 公告失败: %w", err)
	}
	items := make([]steamNewsItem, 0, len(resp.AppNews.NewsItems))
	for _, item := range resp.AppNews.NewsItems {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.URL) == "" {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// ===== 通用 HTTP 小工具 =====

// steamGetJSON 发一个 GET 并把响应体读回来（限长、非 200 报错）。
// 所有 Steam 接口的请求头/超时都走这里，避免每个接口各写一遍。
func (s *MetaService) steamGetJSON(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构造 Steam 请求失败: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.6")

	resp, err := s.ensureClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Steam 接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Steam 接口返回 HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSteamResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取 Steam 返回失败: %w", err)
	}
	if len(body) > maxSteamResponseBytes {
		return nil, fmt.Errorf("Steam 返回的数据超过 %d 字节上限，已放弃", maxSteamResponseBytes)
	}
	return body, nil
}

// stringSlice 把 work 里的 []string / []any 统一成 []string（json 往返后类型会变）。
func stringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

// ===== 暴露给前端的包装（MetaApiService）=====

// SearchSteamWorks 按名字搜索 Steam 商店，返回最多 10 个候选。
// 游戏库「手动搜索」切到 Steam 来源时用（原来只能按 appid 取详情）。
func (s *MetaApiService) SearchSteamWorks(ctx context.Context, term string) (map[string]any, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, errors.New("请输入要搜索的游戏名")
	}
	items, err := s.ensure().steamSearch(ctx, term)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"query": term,
		"total": len(items),
		"items": items,
	}, nil
}

// MatchSteamAppByFolder 用文件夹名猜 appid（扫描「按文件夹名」时用）。
// matched=false 表示"没猜出来"，前端据此退回 DLsite 名字搜索 —— 不是错误。
func (s *MetaApiService) MatchSteamAppByFolder(ctx context.Context, folderName string) (map[string]any, error) {
	folderName = strings.TrimSpace(folderName)
	if folderName == "" {
		return map[string]any{"matched": false}, nil
	}
	item, err := s.ensure().MatchSteamApp(ctx, folderName)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return map[string]any{"matched": false}, nil
	}
	return map[string]any{
		"matched": true,
		"appId":   item.ID,
		"name":    item.Name,
		"cover":   item.Cover,
	}, nil
}

// FetchSteamStats 取 /works 的「Steam 动态」：评价汇总 + 在线人数 + 最近公告。
func (s *MetaApiService) FetchSteamStats(ctx context.Context, appID string) (map[string]any, error) {
	return s.ensure().FetchSteamStats(ctx, appID)
}
