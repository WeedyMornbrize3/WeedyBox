//go:build !windows

// internal/service/game/launch_other.go
//
// 非 Windows 的兜底实现。本项目只构建 Windows（Taskfile GOOS 固定），
// 这里只是为了让 `GOOS=darwin go build ./...` 之类的检查不炸，
// 与 scan/steam_other.go 同一套思路。
package game

import (
	"errors"
	"os/exec"
	"path/filepath"
)

func startDetached(path string) (uint32, error) {
	cmd := exec.Command(path)
	cmd.Dir = filepath.Dir(path)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := uint32(cmd.Process.Pid)
	_ = cmd.Process.Release()
	return pid, nil
}

// startURL 非 Windows 兜底：只是为了 GOOS=darwin go build ./... 能过，
// 本项目只构建 Windows，实际不会走到这里。
func startURL(url string) error {
	return errors.New("当前平台不支持通过系统协议启动: " + url)
}
