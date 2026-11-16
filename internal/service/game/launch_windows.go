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
