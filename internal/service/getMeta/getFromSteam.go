// internal/service/getMeta/getFromSteam.go
//
// Steam 元数据源（2026-11-21 新增）。
//
// 只有一件事：拿 appid 去 Steam 商店的 appdetails 接口换一份完整数据，
// 再**归一化成游戏库已经在用的作品 map**（id/name/maker/workType/…）。
// 这样：
//   - library_works 不用加列（来源只体现在 source='steam' 上）；
//   - /works 现在那套组件不改也能显示（用户明确说作品信息界面以后才改，这次不动 UI）；
//   - 完整 appdetails 原样塞进 work["steam"]，以后要展示什么字段都不必重新抓。
//
// 封面不在这里：Steam 的 2:3 竖版封面在本机 appcache\librarycache 里（见 scan.SteamCoverImage），
// 由 library_service.go 在入库时导入 covers/。
package getmeta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// steamStoreBaseURL 与两个接口：详情 + 商店页（sourceUrl 指回这里）。
	// 其余可扩展接口的地址统一收在 steam_endpoints.go（2026-11-22 实测清单）。
	steamStoreBaseURL  = "https://store.steampowered.com"
	steamAppDetailsFmt = steamAppDetailsAPIURL + "?appids=%s&l=schinese&cc=CN"
	steamAppPageFmt    = steamStoreBaseURL + "/app/%s/"

	// appdetails 里 about_the_game / detailed_description 很长，给 16MB 余量（与封面同口径）。
	maxSteamResponseBytes = 16 << 20

	// steamTagLimit 归一化标签的上限（与 DLsite 的 defaultTagLimit 保持一致）。
	steamTagLimit = 64
)

// FetchSteamApp 抓一个 Steam 游戏，返回游戏库使用的作品 map。
//
// 语言固定 schinese / 地区 CN：与游戏库的中文界面一致，简介与类型名直接是中文。
func (s *MetaService) FetchSteamApp(ctx context.Context, rawAppID string) (map[string]any, error) {
	appID := NormalizeSteamAppID(rawAppID)
	if appID == "" {
		return nil, errors.New("Steam 游戏编号无效（应为纯数字 appid）")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(steamAppDetailsFmt, appID), nil)
	if err != nil {
		return nil, fmt.Errorf("构造 Steam 请求失败: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.6")

	resp, err := s.ensureClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Steam 商店失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Steam 商店接口返回 HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSteamResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取 Steam 返回失败: %w", err)
	}
	if len(body) > maxSteamResponseBytes {
		return nil, fmt.Errorf("Steam 返回的数据超过 %d 字节上限，已放弃", maxSteamResponseBytes)
	}
	work, err := parseSteamAppDetails(appID, body)
	if err != nil {
		return nil, err
	}
	// 标签与评价汇总改由 GetItems 提供（appdetails 的 genres/categories 是商店分类，
	// 不是玩家标签）。这一步失败不算错误：详情已经拿到了，只是标签退回 appdetails 那套。
	s.enrichSteamWork(ctx, appID, work)
	return work, nil
}

// NormalizeSteamAppID 校验 appid：**必须整串都是数字**（允许首尾空白）。
// 不截取数字片段 —— 否则 "RJ1234" 会被当成 appid 1234 发去商店接口。
func NormalizeSteamAppID(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	for i := 0; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return ""
		}
	}
	return text
}

// ensureClient 兜底：MetaService 零值（单测/直接字面量构造）也要能发请求。
func (s *MetaService) ensureClient() *http.Client {
	if s == nil || s.client == nil {
		return &http.Client{Timeout: defaultTimeout}
	}
	return s.client
}

// FetchSteamWork 是 FetchSteamApp 的 Wails 包装：LibraryService 的 source='steam'
// 分支通过它抓详情（与 DLsite 的 FetchWork 同一个返回形状：作品 map）。
func (s *MetaApiService) FetchSteamWork(ctx context.Context, appID string) (map[string]any, error) {
	return s.ensure().FetchSteamApp(ctx, appID)
}

// parseSteamAppDetails 解析 appdetails 的外层结构（{"<appid>": {"success":…, "data":…}}），
// 再交给 buildSteamWork 归一化。拆开是为了能离线测（用真实返回的 JSON 片段当 fixture）。
func parseSteamAppDetails(appID string, body []byte) (map[string]any, error) {
	var payload map[string]struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析 Steam 返回失败: %w", err)
	}
	entry, ok := payload[appID]
	if !ok {
		return nil, fmt.Errorf("Steam 商店没有返回 appid %s 的数据", appID)
	}
	if !entry.Success {
		return nil, fmt.Errorf("Steam 商店查不到 appid %s（可能已下架，或不是商店上架的应用）", appID)
	}
	if len(entry.Data) == 0 || string(entry.Data) == "[]" {
		return nil, fmt.Errorf("Steam 没有返回 appid %s 的详情（可能不是游戏）", appID)
	}
	return buildSteamWork(appID, entry.Data)
}

// ===== appdetails 的字段（只声明用得到的；原始数据另存一份，不怕漏字段）=====

type steamAppDetails struct {
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	RequiredAge      any      `json:"required_age"`
	IsFree           bool     `json:"is_free"`
	ShortDescription string   `json:"short_description"`
	HeaderImage      string   `json:"header_image"`
	CapsuleImage     string   `json:"capsule_image"`
	CapsuleImageV5   string   `json:"capsule_imagev5"`
	Website          string   `json:"website"`
	Developers       []string `json:"developers"`
	Publishers       []string `json:"publishers"`
	// genres / categories 的 id 类型不统一（genres.id 是字符串、categories.id 是数字），
	// 我们只用 description，索性不声明 id —— 一个字段的类型差异不该让整份数据解析失败。
	Genres []struct {
		Description string `json:"description"`
	} `json:"genres"`
	Categories []struct {
		Description string `json:"description"`
	} `json:"categories"`
	ReleaseDate struct {
		ComingSoon bool   `json:"coming_soon"`
		Date       string `json:"date"`
	} `json:"release_date"`
	Platforms struct {
		Windows bool `json:"windows"`
		Mac     bool `json:"mac"`
		Linux   bool `json:"linux"`
	} `json:"platforms"`
	// 这两个字段在部分条目上是 null 或 []，用 RawMessage 自己解析，避免整份数据解析失败
	PriceOverview   json.RawMessage `json:"price_overview"`
	Recommendations *struct {
		Total int64 `json:"total"`
	} `json:"recommendations"`
	Achievements *struct {
		Total int `json:"total"`
	} `json:"achievements"`
	Metacritic *struct {
		Score int    `json:"score"`
		URL   string `json:"url"`
	} `json:"metacritic"`
}

// buildSteamWork 把 appdetails 的 data 映射成游戏库的作品 map。
//
// 字段口径与 DLsiteWork 对齐（前端 /works 与 /testing 直接复用同一套组件）：
// id=appid，maker=开发商，workType=类型，releaseDate=发售日，summary=简介，
// tags=类型/分类/开发商/发行商，outline=开发商…等明细，steam=完整原始数据。
func buildSteamWork(appID string, raw json.RawMessage) (map[string]any, error) {
	var data steamAppDetails
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("解析 Steam 详情失败: %w", err)
	}
	name := strings.TrimSpace(data.Name)
	if name == "" {
		return nil, fmt.Errorf("Steam 返回的数据缺少游戏名称（appid %s）", appID)
	}

	// 原始数据整份留着：作品信息界面以后要改，到时直接读这里，不用重新抓
	rawMap := map[string]any{}
	_ = json.Unmarshal(raw, &rawMap)

	developers := trimList(data.Developers)
	publishers := trimList(data.Publishers)
	workType := steamTypeLabel(data.Type)
	coverURL := firstNonEmpty(data.HeaderImage, data.CapsuleImageV5, data.CapsuleImage)
	price, currency := parseSteamPrice(data.PriceOverview, data.IsFree)

	outline := make([]map[string]string, 0, 8)
	addOutline := func(label, value string) {
		if strings.TrimSpace(value) != "" {
			outline = append(outline, map[string]string{"label": label, "value": value})
		}
	}
	addOutline("开发商", strings.Join(developers, " / "))
	addOutline("发行商", strings.Join(publishers, " / "))
	addOutline("平台", strings.Join(steamPlatforms(data.Platforms.Windows, data.Platforms.Mac, data.Platforms.Linux), " / "))
	addOutline("类型", workType)
	if data.Recommendations != nil && data.Recommendations.Total > 0 {
		addOutline("Steam 评测数", strconv.FormatInt(data.Recommendations.Total, 10))
	}
	if data.Achievements != nil && data.Achievements.Total > 0 {
		addOutline("成就", strconv.Itoa(data.Achievements.Total))
	}
	if data.Metacritic != nil && data.Metacritic.Score > 0 {
		addOutline("Metacritic", fmt.Sprintf("%d / 100", data.Metacritic.Score))
	}
	if data.Website != "" {
		addOutline("官网", data.Website)
	}

	return map[string]any{
		"id":          appID,
		"name":        name,
		"maker":       strings.Join(developers, " / "),
		"makerId":     "",
		"series":      "",
		"seriesId":    "",
		"workType":    workType,
		"ageRating":   steamAgeRating(data.RequiredAge),
		"releaseDate": strings.TrimSpace(data.ReleaseDate.Date),
		"price":       price,
		"currency":    currency,
		"coverUrl":    coverURL,
		"summary":     strings.TrimSpace(data.ShortDescription),
		"tags":        collectSteamTags(data),
		"outline":     outline,
		"sourceUrl":   fmt.Sprintf(steamAppPageFmt, appID),
		"fetchedAt":   time.Now().Format(time.RFC3339),
		// 前端据此跳过非游戏条目（Steamworks 共享组件、工具、DLC 等）
		"steamType": strings.TrimSpace(data.Type),
		// 完整原始数据（/testing 的原始 JSON 与以后重做的作品信息界面都靠它）
		"steam": rawMap,
	}, nil
}

// steamTypeLabel 把 appdetails 的 type 翻成中文（作品形式一栏显示它）。
func steamTypeLabel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "game":
		return "游戏"
	case "dlc":
		return "DLC"
	case "demo":
		return "试玩版"
	case "tool":
		return "工具"
	case "music":
		return "原声音乐"
	case "video":
		return "视频"
	case "mod":
		return "模组"
	case "hardware":
		return "硬件"
	case "series":
		return "系列"
	case "episode":
		return "章节"
	case "advertising":
		return "广告"
	default:
		return strings.TrimSpace(raw)
	}
}

// steamAgeRating 把 required_age（接口里可能是数字也可能是字符串）转成 "18+" 形式。
func steamAgeRating(value any) string {
	age := 0
	switch v := value.(type) {
	case float64:
		age = int(v)
	case int:
		age = v
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			age = parsed
		}
	}
	if age <= 0 {
		return ""
	}
	return fmt.Sprintf("%d+", age)
}

// parseSteamPrice 从 price_overview 里取展示价格（免费游戏返回 "免费"）。
func parseSteamPrice(raw json.RawMessage, isFree bool) (string, string) {
	if len(raw) > 0 && string(raw) != "null" && string(raw) != "[]" {
		var overview struct {
			Currency       string `json:"currency"`
			Final          int64  `json:"final"`
			FinalFormatted string `json:"final_formatted"`
		}
		if err := json.Unmarshal(raw, &overview); err == nil {
			price := strings.TrimSpace(overview.FinalFormatted)
			if price == "" && overview.Final > 0 {
				price = strconv.FormatInt(overview.Final, 10)
			}
			if price != "" {
				// FinalFormatted 本身带货币符号（¥ 68.00），不再重复显示币种
				return price, ""
			}
		}
	}
	if isFree {
		return "免费", ""
	}
	return "", ""
}

// collectSteamTags 汇总标签：类型 → 分类 → 开发商 → 发行商，去重后截断。
// 与 DLsite 的 tags 同义（分类/社团/作者这类"能拿来筛"的词）。
func collectSteamTags(data steamAppDetails) []string {
	tags := make([]string, 0, len(data.Genres)+len(data.Categories)+len(data.Developers)+len(data.Publishers))
	seen := make(map[string]struct{}, cap(tags))
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
		tags = append(tags, tag)
	}
	for _, genre := range data.Genres {
		add(genre.Description)
	}
	for _, category := range data.Categories {
		add(category.Description)
	}
	for _, developer := range data.Developers {
		add(developer)
	}
	for _, publisher := range data.Publishers {
		add(publisher)
	}
	if len(tags) > steamTagLimit {
		tags = tags[:steamTagLimit]
	}
	return tags
}

func steamPlatforms(windows, mac, linux bool) []string {
	platforms := make([]string, 0, 3)
	if windows {
		platforms = append(platforms, "Windows")
	}
	if mac {
		platforms = append(platforms, "macOS")
	}
	if linux {
		platforms = append(platforms, "Linux")
	}
	return platforms
}

func trimList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
