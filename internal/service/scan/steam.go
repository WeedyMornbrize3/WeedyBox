// internal/service/scan/steam.go
//
// Steam 游戏目录扫描：本机装了 Steam 后，游戏都在 <Steam>\steamapps\common\<游戏名>。
// 这里只做两件事：
//  1. 找出 Steam 安装目录（Windows 走注册表，见 steam_windows.go）；
//  2. 顺着 libraryfolders.vdf 把**所有附加库**也找出来 —— 游戏常被装到别的盘，
//     只看安装目录会漏掉一半。
//
// 拿到 common 目录后**直接复用 ScanSubfolderNames**（用户要求的复用点）：
// 本文件只回答「common 在哪」，「怎么读这一层」不重写第二遍。
//
// 与 ScanSubfolderNames 的关系：那个方法是"给定一个目录，读它的一级子文件夹名"，
// 通用；这里只负责把 Steam 的若干 common 目录喂给它，再做跨库去重。
package scan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ErrSteamNotFound 本机没有可用的 Steam 安装目录（没装 Steam，或注册表项被清掉）。
//
// 单独给一个哨兵错误，是为了让调用方能区分「没装 Steam」和「装了但库里一个游戏都没有」——
// 后者返回空名单 + nil，前端可以安静地显示空列表；前者该提示用户去装/选择目录。
var ErrSteamNotFound = errors.New("未找到 Steam 安装目录")

// ScanSteamGameFolders 返回本机所有 Steam 库 `steamapps\common` 下的一级子文件夹名。
//
// 流程：找 Steam 安装目录 → 读它的 libraryfolders.vdf 拿到全部库根 →
// 对每个 `<库根>\steamapps\common` 复用 ScanSubfolderNames。
//
// 结果去重（Windows 路径大小写不敏感，同一游戏两个库都有时只留第一个）并按名称排序。
// 容错：某个库不在线（外接盘没插）/ 目录被删 / 读不动，只跳过它，不影响其他库；
// 只有「压根找不到 Steam」才报 ErrSteamNotFound。
func (s *ScanService) ScanSteamGameFolders(ctx context.Context) ([]string, error) {
	roots := steamInstallRoots()
	if len(roots) == 0 {
		return nil, ErrSteamNotFound
	}
	return s.scanSteamRoots(ctx, roots)
}

// scanSteamRoots 从若干 Steam 安装目录出发，汇总各库 common 的一级子文件夹名。
//
// 单独抽出来是为了测试：注册表在单元测试里没法伪造，但「多个库 → 汇总去重」
// 这段逻辑必须有覆盖，所以测试直接喂临时目录当 Steam 根。
func (s *ScanService) scanSteamRoots(ctx context.Context, steamRoots []string) ([]string, error) {
	names := make([]string, 0, 64)
	seen := make(map[string]struct{}, 64) // key = ToLower(名称)，Windows 下同名不同大小写算重复

	for _, steamRoot := range steamRoots {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("扫描被取消: %w", err)
		}

		for _, commonDir := range steamCommonDirs(steamRoot) {
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("扫描被取消: %w", err)
			}

			// 先判存在：库盘没插时常见，ScanSubfolderNames 对不存在的目录会报错，
			// 但这里"库不在"是正常状态，不是扫描失败。
			info, statErr := os.Stat(commonDir)
			if statErr != nil || !info.IsDir() {
				continue
			}

			folders, err := s.ScanSubfolderNames(ctx, commonDir)
			if err != nil {
				// 取消要中止整次扫描；其余（权限/坏盘）跳过这个库。
				if ctxErr := ctx.Err(); ctxErr != nil {
					return nil, fmt.Errorf("扫描被取消: %w", ctxErr)
				}
				continue
			}

			for _, folder := range folders {
				key := strings.ToLower(folder)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				names = append(names, folder)
			}
		}
	}

	// 按名称排序，大小写不敏感（跟资源管理器一致：nmrih 落在 N 段，
	// 而不是被字节序甩到 Z 后面）；完全同名时才比原始大小写，保证顺序确定。
	sort.Slice(names, func(i, j int) bool {
		left, right := strings.ToLower(names[i]), strings.ToLower(names[j])
		if left != right {
			return left < right
		}
		return names[i] < names[j]
	})
	return names, nil
}

// steamCommonDirs 返回一个 Steam 安装目录下所有「库的 common 目录」。
//
// 主库固定是 <Steam>\steamapps\common；附加库来自 libraryfolders.vdf 里每条 "path"
// （每条 path 是**库根**，common 仍在它自己的 steamapps 下）。
// 同一个目录只返回一次：vdf 里也会列出主库自己。
func steamCommonDirs(steamRoot string) []string {
	steamRoot = filepath.Clean(steamRoot)

	dirs := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	add := func(dir string) {
		dir = filepath.Clean(dir)
		key := strings.ToLower(dir)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		dirs = append(dirs, dir)
	}

	// 主库
	add(filepath.Join(steamRoot, "steamapps", "common"))

	// 附加库：现役位置在 steamapps\，旧版客户端放在 config\，两处都看（缺一个很正常）。
	vdfPaths := []string{
		filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"),
		filepath.Join(steamRoot, "config", "libraryfolders.vdf"),
	}
	for _, vdfPath := range vdfPaths {
		for _, libraryRoot := range parseSteamLibraryFolders(vdfPath) {
			add(filepath.Join(libraryRoot, "steamapps", "common"))
		}
	}

	return dirs
}

// parseSteamLibraryFolders 读一个 libraryfolders.vdf，返回其中的库根路径。
// 文件不存在 / 读不动都返回 nil：只有主库的机器本来就没有这个文件。
func parseSteamLibraryFolders(vdfPath string) []string {
	data, err := os.ReadFile(vdfPath)
	if err != nil {
		return nil
	}
	return parseSteamLibraryFoldersText(string(data))
}

var (
	// 现役格式：块内一行 "path"		"D:\\SteamLibrary"
	steamLibraryPathPattern = regexp.MustCompile(`(?m)^\s*"path"\s+"((?:[^"\\]|\\.)*)"`)
	// 旧版格式：直接是 "1"		"D:\\SteamLibrary"（键是数字）。
	// 只在一条 "path" 都没找到时启用，避免误吃现役文件里 apps 段的 "228980" "123456"。
	steamLibraryLegacyPattern = regexp.MustCompile(`(?m)^\s*"\d+"\s+"((?:[^"\\]|\\.)*)"`)
)

// parseSteamLibraryFoldersText 解析 .vdf 文本（纯函数，方便用真实样本做测试）。
//
// VDF 是 Valve 的 KeyValues 格式，没必要为它引第三方依赖：库路径只可能出现在
// "path" 行（或旧版数字键行），用两条行级正则就够，比写完整解析器可靠。
func parseSteamLibraryFoldersText(text string) []string {
	matches := steamLibraryPathPattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		matches = steamLibraryLegacyPattern.FindAllStringSubmatch(text, -1)
	}

	// VDF 里反斜杠转义成 \\、引号转义成 \"，反过来还原一次。
	unescape := strings.NewReplacer(`\\`, `\`, `\"`, `"`).Replace

	roots := make([]string, 0, len(matches))
	for _, match := range matches {
		root := strings.TrimSpace(unescape(match[1]))
		if root == "" {
			continue
		}
		roots = append(roots, root)
	}
	return roots
}
