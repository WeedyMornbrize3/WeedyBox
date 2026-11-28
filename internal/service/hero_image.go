// internal/service/hero_image.go
//
// 详情页 hero 大图的本地缓存。
//
// 背景（2026-11-27 用户要求）：
//
//	「hero图抖动 → hero图改用缓存，不再使用网络，网络只是兜底。默认不通过联网获取；
//	  如果本地不存在缓存（hero 和 hero 模糊图都保存在 covers 文件夹中）才通过网络获取，
//	  并且网络获取直接下载到本地 covers 当作以后缓存。将文件夹绝对路径或相对路径存入数据库中。」
//
// 抖动是怎么来的：以前前端先渲染远程预览图，Steam 本机 hero 读出来后再把 <img> 的 src
// 换掉 —— 两次加载、两张不同的图，看起来就是"抖一下"。现在解析与缓存全在后端：
//
//  1. 库里记的 hero_abs_path 有文件 → 直接读，**不碰网络**；
//  2. 盘上按 hero_<来源>_<编号>.<ext> 找得到 → 直接读（未入库的作品也走这条）；
//  3. 真没有缓存才取图：Steam 先读客户端 librarycache 的本地 hero（仍然不联网），
//     其余按前端算好的首个候选 / 元数据里的画廊图下载；
//  4. 取到的图原样写进 covers/，并把目录、相对路径、绝对路径写进 library_works 的
//     hero_* 列 —— 下次进页面就是纯本地读取，页面也只渲染这一张（不会再换 src）。
//
// 文件名只保留一张：换图/换格式时写新文件并把同作品的旧 hero 删掉。
package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
	getmeta "WeedyBox/internal/service/getMeta"
	"WeedyBox/internal/service/scan"
)

// HeroImage 返回详情页 hero 大图的 data URL（只从本地缓存/本地 Steam 缓存取，
// 网络仅在没有缓存时兜底一次并落盘）。
//
// fallbackURL 是前端按元数据算出来的首选远程图（DLsite 画廊第二张 / Steam 商店截图）；
// 本地已经有 hero 时**完全忽略**它 —— 这是"默认不联网"的落点。
// 没有 hero 可给时返回空串 + nil（页面退成渐变底，不算错误）。
func (s *LibraryService) HeroImage(ctx context.Context, workID, source, fallbackURL string) (string, error) {
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return "", nil
	}
	data, ext, err := s.heroImageBytes(ctx, source, workID, fallbackURL)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	if ext == "" {
		ext = ".jpg"
	}
	return "data:" + mimeBySuffix(ext) + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// heroImageBytes 取得一个作品的 hero 原图字节与后缀。
// 就地（幂等地）把新取到的图写进 covers/ 并记库；缓存命中时不联网。
func (s *LibraryService) heroImageBytes(ctx context.Context, source, workID, fallbackURL string) ([]byte, string, error) {
	workID = strings.ToUpper(strings.TrimSpace(workID))
	if workID == "" {
		return nil, "", nil
	}
	src := normalizeWorkSource(source)

	dir := s.ensureMeta().CoversDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", fmt.Errorf("创建封面目录失败: %w", err)
	}

	row := s.findWorkRow(ctx, src, workID)

	// ① 数据库指针：入库过的作品优先读这一格（不再关心 URL 有没有变，缓存为准）
	if row != nil && strings.TrimSpace(row.HeroAbsPath) != "" {
		if data, err := os.ReadFile(row.HeroAbsPath); err == nil && len(data) > 0 {
			return data, suffixOrSniff(row.HeroSuffix, data), nil
		}
		// 文件被手动删了：清掉指针，继续往下找/取
		_ = s.DB.ClearWorkHeroImage(ctx, src, workID)
	}

	// ② 盘上命名规则（未入库的作品、或指针丢了的旧数据）
	if path, ok := findHeroImageFile(dir, src, workID); ok {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			ext := suffixOrSniff(filepath.Ext(path), data)
			s.recordHeroImage(ctx, src, workID, path, data, ext, "")
			return data, ext, nil
		}
	}

	// ③ Steam 本机 hero：客户端 appcache\librarycache 里的本地图，不联网
	if src == database.SourceSteam {
		if appID := getmeta.NormalizeSteamAppID(workID); appID != "" {
			if local, err := scan.SteamHeroImage(ctx, appID); err == nil && strings.TrimSpace(local) != "" {
				if data, rerr := os.ReadFile(local); rerr == nil && len(data) > 0 {
					ext := suffixOrSniff(filepath.Ext(local), data)
					path := s.writeHeroImage(dir, src, workID, data, ext)
					s.recordHeroImage(ctx, src, workID, path, data, ext, "steam-local:"+filepath.ToSlash(local))
					return data, ext, nil
				}
			}
		}
	}

	// ④ 网络兜底：前端首选 URL → 元数据里的候选（Steam 截图 / 画廊预览图 / 封面）
	for _, url := range heroDownloadCandidates(src, row, fallbackURL) {
		data, err := s.downloadHeroBytes(ctx, url)
		if err != nil || len(data) == 0 {
			continue
		}
		ext := getmeta.ImageExt(data)
		path := s.writeHeroImage(dir, src, workID, data, ext)
		s.recordHeroImage(ctx, src, workID, path, data, ext, url)
		return data, ext, nil
	}

	return nil, "", nil
}

// findWorkRow 读库记录；数据库未就绪/查不到都返回 nil（缓存功能降级，不影响页面）。
func (s *LibraryService) findWorkRow(ctx context.Context, source, workID string) *model.LibraryWork {
	if s.DB == nil {
		return nil
	}
	row, err := s.DB.FindLibraryWork(ctx, source, workID)
	if err != nil {
		return nil
	}
	return row
}

// downloadHeroBytes 下载 hero 原图，**优先探测源站同名 .webp 档位**。
//
// 用户 2026-11-27：「所有图片尽量改为 webp 格式」。DLsite 的图片通常有同名 webp
// （体积小一个数量级），所以走和封面同一套探测逻辑：webp 拿不到就静默回落原图。
// data URL 本身已经是本地图，不再探测。
func (s *LibraryService) downloadHeroBytes(ctx context.Context, rawURL string) ([]byte, error) {
	rawURL = strings.TrimSpace(rawURL)
	if strings.HasPrefix(rawURL, "data:") {
		return s.loadHeroImageBytes(ctx, rawURL)
	}
	data, _, err := s.ensureMeta().FetchImagePreferringWebP(ctx, rawURL)
	return data, err
}

// loadHeroImageBytes 取得 hero 原图字节。
// data URL（本地图 / 测试）走本地 base64 解码；http(s) 交给 getMeta 的下载函数
// （同一套 UA/Referer/重试）。
func (s *LibraryService) loadHeroImageBytes(ctx context.Context, rawURL string) ([]byte, error) {
	rawURL = strings.TrimSpace(rawURL)
	if strings.HasPrefix(rawURL, "data:") {
		comma := strings.Index(rawURL, ",")
		if comma < 0 || comma == len(rawURL)-1 {
			return nil, fmt.Errorf("hero data URL 格式不正确")
		}
		payload := rawURL[comma+1:]
		if !strings.Contains(rawURL[:comma], ";base64") {
			return nil, fmt.Errorf("hero data URL 不是 base64 编码")
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("hero data URL 解码失败: %w", err)
		}
		return data, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return s.ensureMeta().FetchImageBytes(ctx, rawURL)
}

// heroDownloadCandidates 列出"还能去哪找 hero"的 URL（按优先级）。
//
// 前端传的 fallbackURL 已经按展示口径排好序（DLsite 画廊第二张、Steam 商店截图），
// 所以它排第一；后面是元数据里的候选，只在前面的 URL 下载失败时才会用到。
func heroDownloadCandidates(source string, row *model.LibraryWork, fallbackURL string) []string {
	out := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	add := func(raw string) {
		url := strings.TrimSpace(raw)
		if url == "" {
			return
		}
		// data URL 只在兼容旧调用方时出现（本地 Steam hero 的 base64）；
		// 新前端不再把图片内容塞进参数，这里允许它作为最后兜底。
		if _, ok := seen[url]; ok {
			return
		}
		seen[url] = struct{}{}
		out = append(out, url)
	}

	add(fallbackURL)
	if row == nil {
		return out
	}
	work, ok := workFromRawJSON(row)
	if !ok {
		return out
	}

	if source == database.SourceSteam {
		if steam, ok := work["steam"].(map[string]any); ok {
			if shots, ok := steam["screenshots"].([]any); ok {
				for _, item := range shots {
					shot, ok := item.(map[string]any)
					if !ok {
						continue
					}
					add(strOf(shot["path_full"]))
				}
			}
			add(strOf(steam["background_raw"]))
			add(strOf(steam["background"]))
		}
	}

	// 画廊大图：第一张是封面，第二张才是 hero（与前端 backgroundUrl 的口径一致）
	previews := stringListOf(work["previewImages"])
	if len(previews) > 1 {
		add(previews[1])
	}
	if len(previews) > 0 {
		add(previews[0])
	}
	coverURL := strOf(work["coverUrl"])
	if derived := deriveDlsiteHeroURL(coverURL); derived != "" {
		add(derived)
	}
	add(coverURL)
	return out
}

// deriveDlsiteHeroURL 由封面直链推一张预览大图（`_img_main.jpg` → `_img_smp1.jpg`）。
// 老入库记录没有 previewImages，只有这条推导路可用；推出来的图不一定存在，
// 下载失败就继续试下一个候选。
func deriveDlsiteHeroURL(coverURL string) string {
	coverURL = strings.TrimSpace(coverURL)
	if coverURL == "" {
		return ""
	}
	// 只处理 DLsite 的命名（其它来源原样返回，交给下载失败兜底）
	lower := strings.ToLower(coverURL)
	for _, suffix := range []string{".jpg", ".jpeg", ".png", ".webp"} {
		if strings.HasSuffix(lower, suffix) {
			base := coverURL[:len(coverURL)-len(suffix)]
			if idx := strings.LastIndex(base, "_img_main"); idx >= 0 {
				return base[:idx] + "_img_smp1.jpg"
			}
		}
	}
	return ""
}

// stringListOf 把 raw_json 里的字符串数组取出来（JSON 反序列化后是 []any）。
func stringListOf(v any) []string {
	items, ok := v.([]any)
	if !ok {
		if direct, ok := v.([]string); ok {
			return direct
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

// writeHeroImage 把 hero 图写进 covers/，返回绝对路径。
// 同一作品只留一张：换扩展名（jpg ↔ webp）时把旧文件一起清掉。
func (s *LibraryService) writeHeroImage(dir, source, workID string, data []byte, ext string) string {
	if ext == "" || ext == "." {
		ext = ".jpg"
	}
	name := fmt.Sprintf("hero_%s_%s%s", normalizeBlurSource(source), safeFileToken(workID), ext)
	path := filepath.Join(dir, name)

	// 先写临时文件再改名：避免读到半张图
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		fmt.Printf("写入 hero 图缓存失败: %v\n", err)
		return ""
	}
	pruneHeroImageFiles(dir, source, workID, path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		_ = os.Rename(tmp, path)
	}
	return path
}

// recordHeroImage 把 hero 图的落地位置写进库（只对已入库的作品生效）。
func (s *LibraryService) recordHeroImage(ctx context.Context, source, workID, absPath string, data []byte, ext, fromURL string) {
	if s.DB == nil || strings.TrimSpace(absPath) == "" {
		return
	}
	dir := filepath.Dir(absPath)
	dirName := filepath.Base(dir)
	rec := &model.HeroImageFile{
		Dir:     dirName,
		Path:    filepath.ToSlash(filepath.Join(dirName, filepath.Base(absPath))),
		AbsPath: absPath,
		Bytes:   int64(len(data)),
		Suffix:  ext,
		URL:     fromURL,
	}
	if err := s.DB.SetWorkHeroImage(ctx, source, workID, rec); err != nil {
		fmt.Printf("记录 hero 图位置失败（文件已保存，本次仍算成功）: %v\n", err)
	}
}

// findHeroImageFile 在 covers/ 里按命名规则找这个作品已缓存的 hero 图。
func findHeroImageFile(dir, source, workID string) (string, bool) {
	pattern := filepath.Join(dir, fmt.Sprintf("hero_%s_%s.*", normalizeBlurSource(source), safeFileToken(workID)))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", false
	}
	for _, path := range matches {
		if strings.HasSuffix(strings.ToLower(path), ".tmp") {
			continue
		}
		return path, true
	}
	return "", false
}

// pruneHeroImageFiles 删掉同作品其它 hero 文件，keepPath 为空 = 全删。
// 正在写的 .tmp 一律跳过（writeHeroImage 先写 tmp 再改名，进度中的文件不能当旧缓存删）。
func pruneHeroImageFiles(dir, source, workID, keepPath string) {
	pattern := filepath.Join(dir, fmt.Sprintf("hero_%s_%s.*", normalizeBlurSource(source), safeFileToken(workID)))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, path := range matches {
		if strings.HasSuffix(strings.ToLower(path), ".tmp") {
			continue
		}
		if keepPath != "" && filepath.Clean(path) == filepath.Clean(keepPath) {
			continue
		}
		_ = os.Remove(path)
	}
}

// removeHeroCache 删除一个作品的 hero 原图与模糊图（作品被移除时调用）。
// 删不掉只忽略：它们都是可再生成的缓存。
func (s *LibraryService) removeHeroCache(workID, source string) {
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return
	}
	dir := s.ensureMeta().CoversDir()
	pruneHeroImageFiles(dir, source, workID, "")
	s.pruneHeroBlurCache(workID, source, "")
}

// suffixOrSniff 决定缓存图的后缀：指针里记了就用它，否则按真实内容判断。
func suffixOrSniff(suffix string, data []byte) string {
	suffix = strings.ToLower(strings.TrimSpace(suffix))
	switch suffix {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".avif":
		return suffix
	}
	return getmeta.ImageExt(data)
}
