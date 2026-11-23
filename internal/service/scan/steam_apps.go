// internal/service/scan/steam_apps.go
//
// 「Steam 游戏」的两种本地扫描（2026-11-21 新增 Steam 元数据源）：
//
//  1. ScanSteamApps —— 按 **appid** 列出本机已安装的 Steam 游戏。
//     数据来源是每个库 `steamapps\appmanifest_<appid>.acf`（Steam 卸载/校验都靠它），
//     里面有 appid / 名称 / 安装目录 / 占用大小。以前「扫描添加」只能拿到
//     `common\<文件夹名>`，还得拿去 DLsite 搜；有了 appid 就能直接问 Steam 商店。
//
//  2. SteamCoverImage —— 在 `<Steam>\appcache\librarycache` 下找封面。
//     用户要求：librarycache 里按游戏 id 找文件夹 → 进去不限深度递归找文件名含
//     `600x900` 或 `capsule` 的图片；**没有对应文件夹就没有封面**（不联网兜底）。
//     Steam 客户端的 2:3 竖版封面正是 `<appid>\library_600x900.jpg`（中文界面还会
//     优先用 `library_600x900_schinese.jpg`）。
//
// 这里只负责「在磁盘上找到东西」，不联网、不写库：
// 商店详情在 internal/service/getMeta/getFromSteam.go，入库编排在 library_service.go。
package scan

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// SteamApp 是一个已安装的 Steam 应用（对应一份 appmanifest_<appid>.acf）。
type SteamApp struct {
	AppID      string `json:"appId"`      // 商店 appid（纯数字，如 "1229490"）
	Name       string `json:"name"`       // appmanifest 里的显示名称
	InstallDir string `json:"installDir"` // 安装目录名（common 下的文件夹名）
	// FolderPath 是游戏安装目录的绝对路径（<库>\steamapps\common\<InstallDir>），
	// 入库时用它挑主程序（PickGameExecutable）。
	FolderPath  string `json:"folderPath"`
	LibraryRoot string `json:"libraryRoot"` // 这个游戏所在的库根（<盘>\SteamLibrary）
	SteamRoot   string `json:"steamRoot"`   // Steam 客户端安装目录（appcache 在它下面）
	SizeOnDisk  int64  `json:"sizeOnDisk"`  // 字节；读不到时为 0
}

// steamStateFullyInstalled 是 appmanifest `StateFlags` 里「已完整安装」的位。
// 实测 R.E.P.O. 的 StateFlags 是 6（= 4 已安装 | 2 待更新），所以只能用按位判断，
// 不能写 `StateFlags == 4`。
const steamStateFullyInstalled = 4

// ScanSteamApps 列出本机所有库里已完整安装的 Steam 应用（按名称排序、按 appid 去重）。
//
// 跨库去重是必须的：同一个游戏可能先在 C 盘装过又迁到 D 盘，两份 manifest 都还在。
// 某个库读不动（盘没插/权限）只跳过；一个 Steam 都找不到时返回 ErrSteamNotFound。
func (s *ScanService) ScanSteamApps(ctx context.Context) ([]SteamApp, error) {
	roots := steamInstallRoots()
	if len(roots) == 0 {
		return nil, ErrSteamNotFound
	}
	return scanSteamAppsFromRoots(ctx, roots)
}

// scanSteamAppsFromRoots 从若干 Steam 安装目录出发汇总已安装应用（注册表在单测里
// 没法伪造，所以把「找到 Steam 之后」的这一半单独抽出来，测试直接喂临时目录）。
func scanSteamAppsFromRoots(ctx context.Context, steamRoots []string) ([]SteamApp, error) {
	apps := make([]SteamApp, 0, 64)
	seen := make(map[string]struct{}, 64) // key = appid

	for _, steamRoot := range steamRoots {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("扫描被取消: %w", err)
		}
		for _, libraryRoot := range steamLibraryRoots(steamRoot) {
			steamapps := filepath.Join(libraryRoot, "steamapps")
			entries, err := os.ReadDir(steamapps)
			if err != nil {
				// 库不存在（外接盘没插/被删）是正常状态，不是扫描失败
				continue
			}
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return nil, fmt.Errorf("扫描被取消: %w", err)
				}
				if entry.IsDir() || !isAppManifestName(entry.Name()) {
					continue
				}
				app, ok := readSteamAppManifest(filepath.Join(steamapps, entry.Name()), steamRoot, libraryRoot)
				if !ok {
					continue
				}
				if _, dup := seen[app.AppID]; dup {
					continue
				}
				seen[app.AppID] = struct{}{}
				apps = append(apps, app)
			}
		}
	}

	// 按名称排序，大小写不敏感（与 Steam 库/资源管理器一致）
	sort.Slice(apps, func(i, j int) bool {
		left, right := strings.ToLower(apps[i].Name), strings.ToLower(apps[j].Name)
		if left != right {
			return left < right
		}
		return apps[i].AppID < apps[j].AppID
	})
	return apps, nil
}

// isAppManifestName 判断是不是 appmanifest_<appid>.acf（大小写不敏感）。
func isAppManifestName(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "appmanifest_") && strings.HasSuffix(lower, ".acf")
}

// appmanifest 是 Valve 的 KeyValues 文本，只需要四五个键；用行级正则足够，
// 与 libraryfolders.vdf 的处理方式保持一致（不为 VDF 引第三方依赖）。
var (
	appManifestAppID      = regexp.MustCompile(`(?m)^\s*"appid"\s+"([^"]*)"`)
	appManifestName       = regexp.MustCompile(`(?m)^\s*"name"\s+"((?:[^"\\]|\\.)*)"`)
	appManifestInstallDir = regexp.MustCompile(`(?m)^\s*"installdir"\s+"((?:[^"\\]|\\.)*)"`)
	appManifestStateFlags = regexp.MustCompile(`(?m)^\s*"StateFlags"\s+"(\d+)"`)
	appManifestSizeOnDisk = regexp.MustCompile(`(?m)^\s*"SizeOnDisk"\s+"(\d+)"`)
	appManifestAppIDPath  = regexp.MustCompile(`(?i)^appmanifest_(\d+)\.acf$`)
)

// readSteamAppManifest 读一份 appmanifest；不完整/未安装完成/读不动都返回 ok=false。
func readSteamAppManifest(manifestPath, steamRoot, libraryRoot string) (SteamApp, bool) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return SteamApp{}, false
	}
	text := string(data)

	appID := firstMatch(appManifestAppID, text)
	if appID == "" {
		// 文件内容坏了就退回文件名里的编号（manifest 的文件名永远是 appmanifest_<id>.acf）
		if match := appManifestAppIDPath.FindStringSubmatch(filepath.Base(manifestPath)); len(match) == 2 {
			appID = match[1]
		}
	}
	appID = strings.TrimSpace(appID)
	if appID == "" || !isDigits(appID) {
		return SteamApp{}, false
	}

	// StateFlags 的第 4 位才是「已安装」；位不存在（旧格式/读不到）时不拦，
	// 免得因为一个可选字段把整台机器的游戏都漏掉。
	if flagsText := firstMatch(appManifestStateFlags, text); flagsText != "" {
		if flags, perr := strconv.Atoi(flagsText); perr == nil && flags&steamStateFullyInstalled == 0 {
			return SteamApp{}, false
		}
	}

	installDir := unescapeVDF(firstMatch(appManifestInstallDir, text))
	if installDir == "" {
		return SteamApp{}, false
	}
	name := unescapeVDF(firstMatch(appManifestName, text))
	if name == "" {
		name = installDir
	}

	size := int64(0)
	if sizeText := firstMatch(appManifestSizeOnDisk, text); sizeText != "" {
		if parsed, perr := strconv.ParseInt(sizeText, 10, 64); perr == nil {
			size = parsed
		}
	}

	return SteamApp{
		AppID:       appID,
		Name:        name,
		InstallDir:  installDir,
		FolderPath:  filepath.Join(libraryRoot, "steamapps", "common", installDir),
		LibraryRoot: filepath.Clean(libraryRoot),
		SteamRoot:   filepath.Clean(steamRoot),
		SizeOnDisk:  size,
	}, true
}

// ===== 封面：<Steam>\appcache\librarycache\<appid>\ =====

// SteamCoverImage 在 Steam 的 appcache\librarycache 里找某个 appid 的封面图。
//
// 用户要求（2026-11-21）：「查找封面图片文件夹目录在 ...\Steam\appcache\librarycache 内，
// 目录下含有游戏 id 名称的文件夹，进入查找相关图像，没有文件夹则不适用图像」。
// 所以：先按 id 找目录（精确同名优先），再**不限深度**递归找文件名含 `600x900` 或
// `capsule` 的图片；一个候选都没有就返回空串（不是错误，也不联网兜底）。
func SteamCoverImage(ctx context.Context, appID string) (string, error) {
	return steamCoverImageInRoots(ctx, steamInstallRoots(), appID)
}

// steamCoverImageInRoots 是 SteamCoverImage 的可测试版本（注册表不好伪造）。
func steamCoverImageInRoots(ctx context.Context, steamRoots []string, appID string) (string, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return "", nil
	}
	for _, steamRoot := range steamRoots {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("查找封面被取消: %w", err)
		}
		cacheRoot := filepath.Join(steamRoot, "appcache", "librarycache")
		folder := findSteamCacheFolder(cacheRoot, appID)
		if folder == "" {
			continue
		}
		image, err := bestSteamCoverIn(ctx, folder)
		if err != nil {
			return "", err
		}
		if image != "" {
			return image, nil
		}
	}
	return "", nil
}

// findSteamCacheFolder 在 librarycache 下找名字含 appid 的文件夹：精确同名优先，
// 否则取第一个包含它的（Steam 不同版本偶尔会加后缀）。没有就返回空串。
func findSteamCacheFolder(cacheRoot, appID string) string {
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return ""
	}
	target := strings.ToLower(appID)
	fallback := ""
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if name == target {
			return filepath.Join(cacheRoot, entry.Name())
		}
		if fallback == "" && strings.Contains(name, target) {
			fallback = filepath.Join(cacheRoot, entry.Name())
		}
	}
	return fallback
}

// bestSteamCoverIn 递归找该目录下最合适的封面图（按名字打分；越浅越优先）。
func bestSteamCoverIn(ctx context.Context, folder string) (string, error) {
	best := ""
	bestScore := -1
	walkErr := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil // 单个文件/子目录读不动就跳过，不影响整只游戏的封面
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if !isSteamCoverImageName(name) {
			return nil
		}
		score := steamCoverScore(name)
		if rel, relErr := filepath.Rel(folder, path); relErr == nil {
			score -= strings.Count(rel, string(filepath.Separator)) // 越深越差
		}
		if score > bestScore {
			best, bestScore = path, score
		}
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) {
			return "", fmt.Errorf("查找封面被取消: %w", walkErr)
		}
		// 其余遍历错误（目录恰好被删）当作"没找到"，封面不是关键路径
	}
	return best, nil
}

// isSteamCoverImageName 只认图片，且文件名必须带 `600x900` 或 `capsule`（用户要求）。
func isSteamCoverImageName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		return false
	}
	return strings.Contains(name, "600x900") || strings.Contains(name, "capsule")
}

// steamCoverScore 给候选封面打分：
//
//	600x900（Steam 库的 2:3 竖版封面）> capsule > 其它；
//	同档里中文版优先（游戏库界面是中文，与 Steam 中文界面显示同一张图）。
func steamCoverScore(name string) int {
	score := 0
	if strings.Contains(name, "600x900") {
		score += 100
	}
	if strings.Contains(name, "capsule") {
		score += 40
	}
	if strings.Contains(name, "schinese") {
		score += 20
	}
	return score
}

// ===== hero 背景图：<Steam>\appcache\librarycache\<appid>\ 里找 library_hero* =====

// SteamHeroImage 在 Steam 的 appcache\librarycache 里找某个 appid 的 hero 背景图。
//
// 用户要求（2026-11-23）：「背景图如果是 steam 源使用本地扫描函数扫描含 hero 字样图片」。
// Steam 客户端库页顶部那张大图就是 `<appid>\<hash>\library_hero.jpg`（中文界面还会
// 下 `library_hero_schinese.jpg`）。找法与封面同构：先按 appid 找目录，再**不限深度**
// 递归找文件名含 hero 的图片；`*_blur*` 只有几 KB 的模糊占位图，`*logo*` 是徽标，
// 都不能当背景，一并排除。**没有对应目录 / 没有候选图就返回空串**（不是错误）：
// 调用方会退回商店接口给的预览图（用户要求「没有就使用 api 获取的预览图填充」）。
func SteamHeroImage(ctx context.Context, appID string) (string, error) {
	return steamHeroImageInRoots(ctx, steamInstallRoots(), appID)
}

// steamHeroImageInRoots 是 SteamHeroImage 的可测试版本（注册表不好伪造）。
func steamHeroImageInRoots(ctx context.Context, steamRoots []string, appID string) (string, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return "", nil
	}
	for _, steamRoot := range steamRoots {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("查找 hero 背景图被取消: %w", err)
		}
		cacheRoot := filepath.Join(steamRoot, "appcache", "librarycache")
		folder := findSteamCacheFolder(cacheRoot, appID)
		if folder == "" {
			continue
		}
		image, err := bestSteamHeroIn(ctx, folder)
		if err != nil {
			return "", err
		}
		if image != "" {
			return image, nil
		}
	}
	return "", nil
}

// bestSteamHeroIn 递归找该目录下最合适的 hero 图（按名字打分；越浅越优先）。
func bestSteamHeroIn(ctx context.Context, folder string) (string, error) {
	best := ""
	bestScore := -1
	walkErr := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil // 单个文件/子目录读不动就跳过，不影响整张背景图
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if !isSteamHeroImageName(name) {
			return nil
		}
		score := steamHeroScore(name)
		if rel, relErr := filepath.Rel(folder, path); relErr == nil {
			score -= strings.Count(rel, string(filepath.Separator)) // 越深越差
		}
		if score > bestScore {
			best, bestScore = path, score
		}
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) {
			return "", fmt.Errorf("查找 hero 背景图被取消: %w", walkErr)
		}
		// 其余遍历错误（目录恰好被删）当作"没找到"，背景图不是关键路径
	}
	return best, nil
}

// isSteamHeroImageName 只认图片，且文件名必须带 hero（用户要求）；
// blur（模糊占位）与 logo（徽标）不是背景图，排除。
func isSteamHeroImageName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		return false
	}
	lower := strings.ToLower(name)
	if !strings.Contains(lower, "hero") {
		return false
	}
	return !strings.Contains(lower, "blur") && !strings.Contains(lower, "logo")
}

// steamHeroScore 给候选 hero 图打分：
//
//	正式 hero（library_hero.jpg）> 其它含 hero 的图；
//	同档里中文版优先（与封面口径一致，也和游戏库中文界面显示同一张图）。
func steamHeroScore(name string) int {
	score := 0
	if strings.HasPrefix(name, "library_hero") {
		score += 100
	}
	if strings.Contains(name, "schinese") {
		score += 20
	}
	return score
}

// ===== 小工具 =====

func firstMatch(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

// unescapeVDF 还原 VDF 里的转义（\\ → \、\" → "）。
func unescapeVDF(text string) string {
	return strings.NewReplacer(`\\`, `\`, `\"`, `"`).Replace(strings.TrimSpace(text))
}

// isDigits 判断字符串是不是纯数字（appid 必须是数字，防止把坏 manifest 当游戏）。
func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
