// internal/service/game/launch_windows.go
//
// Windows 下的进程启动：用独立进程（DETACHED_PROCESS）把游戏拉起来，
// 然后立刻 Release 句柄 —— 游戏的生命周期与 WeedyBox 无关，关掉客户端
// 不应该把游戏带走，客户端也不该等游戏退出。
package game

import (
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// startDetached 启动一个可执行文件并立刻与它脱钩，返回进程号。
//
// 关键点：
//   - 工作目录设为 exe 所在目录：很多游戏按相对路径找 pak/存档/配置，
//     继承 WeedyBox 的工作目录（可执行文件旁）会直接黑屏或报缺资源；
//   - CreationFlags 用 DETACHED_PROCESS：不继承控制台、不随父进程退出结束，
//     也不会在 WeedyBox 窗口后面闪一个黑框；
//   - Process.Release 释放句柄：之后游戏退出也不会变成僵尸进程。
//     返回的 PID 会存进运行缓存，状态轮询先用它快速判断，不必每次全量扫进程。
func startDetached(path string) (uint32, error) {
	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS,
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := uint32(cmd.Process.Pid)
	// Release 的错误不影响"已经启动"这个事实；句柄释放失败也只影响资源回收
	_ = cmd.Process.Release()
	return pid, nil
}

// startURL 用系统默认的协议处理器打开一个 URL（这里只有 steam://rungameid/<appid>）。
//
// 走 ShellExecuteW 的 "open" 动作，而不是自己拼 cmd /c start：
//   - 由 shell 解析 steam: 协议并交给 Steam 客户端，不经过控制台窗口（不会闪黑框）；
//   - 不产生子进程 PID —— Steam 客户端是常驻进程，游戏由它拉起，
//     运行状态靠 exe 检测（Status）而不是这个返回值；
//   - 没有脚本注入 / 引号转义问题（URL 整个作为一个参数传出去）。
func startURL(url string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(url)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL)
}
