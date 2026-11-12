//go:build !windows

// internal/service/scan/steam_other.go
//
// 非 Windows 平台没有 Steam 注册表可查（macOS 的 Steam 目录规则也完全不同），
// 返回空让上层报 ErrSteamNotFound。本项目只构建 Windows（Taskfile 的 GOOS 固定
// windows），这个文件存在的意义只是让 `GOOS=darwin go build ./...` 不因为
// windows/registry 导入而整体编译失败。
package scan

func steamInstallRoots() []string {
	return nil
}
