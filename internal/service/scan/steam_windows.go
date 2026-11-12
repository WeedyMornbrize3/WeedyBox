//go:build windows

// internal/service/scan/steam_windows.go
//
// 「Steam 装在哪」在 Windows 上只有一个可靠来源：注册表。Steam 客户端自己写
// HKCU\Software\Valve\Steam\SteamPath，安装器写 HKLM\SOFTWARE\Valve\Steam\InstallPath。
// 不扫盘猜路径：Steam 可以装在任意盘/任意目录，候选路径穷举必然漏。
package scan

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// steamInstallRoots 返回本机 Steam 安装目录（去重、只保留真实存在的目录）。
//
// 顺序：HKCU 的 SteamPath（最准，Steam 每次启动自己维护）→ HKLM 的 InstallPath →
// 默认安装路径兜底（重装/绿色版可能没有注册表项）。注册表残留指向已卸载的目录时，
// 后面的 os.Stat 会把它筛掉。
func steamInstallRoots() []string {
	candidates := make([]string, 0, 4)

	// ① HKCU\Software\Valve\Steam\SteamPath —— 值是正斜杠（C:/Program Files (x86)/Steam）
	if steamPath := readRegistryString(registry.CURRENT_USER, `Software\Valve\Steam`, "SteamPath", registry.QUERY_VALUE); steamPath != "" {
		candidates = append(candidates, steamPath)
	}

	// ② HKLM\SOFTWARE\Valve\Steam\InstallPath —— 64 位系统上 32 位安装器写入的是
	// WOW6432Node；从 64 位进程看要显式带 WOW64_32KEY 才能读到，所以两个视图都试。
	for _, view := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
		if installPath := readRegistryString(registry.LOCAL_MACHINE, `SOFTWARE\Valve\Steam`, "InstallPath", registry.QUERY_VALUE|view); installPath != "" {
			candidates = append(candidates, installPath)
		}
	}

	// ③ 兜底：默认安装位置
	if programFilesX86 := os.Getenv("ProgramFiles(x86)"); programFilesX86 != "" {
		candidates = append(candidates, filepath.Join(programFilesX86, "Steam"))
	}
	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		candidates = append(candidates, filepath.Join(programFiles, "Steam"))
	}

	roots := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		candidate = filepath.Clean(filepath.FromSlash(candidate))

		key := strings.ToLower(candidate)
		if _, ok := seen[key]; ok {
			continue
		}

		info, err := os.Stat(candidate)
		if err != nil || !info.IsDir() {
			continue // 注册表残留 / 目录已被删
		}
		seen[key] = struct{}{}
		roots = append(roots, candidate)
	}
	return roots
}

// readRegistryString 读一个字符串值；键或值不存在都返回空串（这不是错误，是常态）。
func readRegistryString(root registry.Key, path, name string, access uint32) string {
	key, err := registry.OpenKey(root, path, access)
	if err != nil {
		return ""
	}
	defer key.Close()

	value, _, err := key.GetStringValue(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}
