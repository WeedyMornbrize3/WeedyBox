//go:build !windows

// internal/service/game/process_other.go
//
// 非 Windows 兜底：不检测进程，状态永远显示"未运行"。
// 本项目只构建 Windows，这里只是保证其他 GOOS 的 build 检查不炸
// （与 launch_other.go / scan/steam_other.go 同一套思路）。
package game

func findRunningProcess(path string) (uint32, bool) { return 0, false }

func processAlive(pid uint32, path string) bool { return false }
