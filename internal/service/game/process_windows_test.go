//go:build windows

// internal/service/game/process_windows_test.go
//
// 真实起一个进程来验证「按完整路径认出运行中的游戏」—— 这是运行状态功能的根，
// 只靠 mock 文件系统验证不到；用系统自带的 cmd.exe 起一个几秒的 ping，查完就杀。
package game

import (
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestFindRunningProcess_FindsSpawnedProcess(t *testing.T) {
	executable, err := exec.LookPath("cmd.exe")
	if err != nil {
		t.Skipf("本机找不到 cmd.exe: %v", err)
	}

	// ping 只用来让进程活几秒；窗口隐藏，测试期间不闪黑框
	cmd := exec.Command(executable, "/c", "ping", "-n", "6", "127.0.0.1")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动测试进程失败: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	target := filepath.Clean(executable)
	pid := uint32(cmd.Process.Pid)

	// 进程刚起来时映像查询可能瞬时失败，给一小段重试窗口
	deadline := time.Now().Add(2 * time.Second)
	for {
		if processAlive(pid, target) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("按完整路径认不出刚启动的进程: %s (pid=%d)", target, pid)
		}
		time.Sleep(50 * time.Millisecond)
	}

	if found, ok := findRunningProcess(target); !ok {
		t.Fatalf("扫描进程应该能找到 %s", target)
	} else if found == 0 {
		t.Fatal("扫到的 PID 不该是 0")
	}

	// 负例：随便一个不存在的 exe 不应被认成正在运行
	if _, ok := findRunningProcess(filepath.Join(t.TempDir(), "not-running.exe")); ok {
		t.Fatal("不存在的程序不该被判定为运行中")
	}
}
