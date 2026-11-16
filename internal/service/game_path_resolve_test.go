// internal/service/game_path_resolve_test.go
//
// 入库时决定「记录哪个 exe」的接线测试：前端带到 Store 的三种形态
// （直接给 exe / 给文件夹 + 文件夹名 / 什么都不给）分别应该落到什么路径。
// 挑程序的细节规则在 scan 包的 TestPickGameExecutable 里覆盖，这里只验接线。
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStoreGamePath(t *testing.T) {
	ctx := context.Background()
	svc := &LibraryService{}

	t.Run("直接给 exe 时原样使用", func(t *testing.T) {
		exe := `D:\Games\RJ01234567\game.exe`
		if got := svc.resolveStoreGamePath(ctx, StoreRequest{GamePath: exe}); got != exe {
			t.Fatalf("期望 %q，实际 %q", exe, got)
		}
	})

	t.Run("给文件夹时进文件夹挑主程序", func(t *testing.T) {
		root := t.TempDir()
		folder := filepath.Join(root, "MyGame")
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatalf("建目录失败: %v", err)
		}
		for _, name := range []string{"UnityCrashHandler64.exe", "MyGame.exe", "setup.exe"} {
			if err := os.WriteFile(filepath.Join(folder, name), []byte("x"), 0o644); err != nil {
				t.Fatalf("写文件失败: %v", err)
			}
		}

		got := svc.resolveStoreGamePath(ctx, StoreRequest{GameFolder: folder, GameFolderName: "MyGame"})
		want := filepath.Join(folder, "MyGame.exe")
		if got != want {
			t.Fatalf("期望挑中 %q，实际 %q", want, got)
		}
	})

	t.Run("文件夹里没有可用的 exe 时留空", func(t *testing.T) {
		folder := filepath.Join(t.TempDir(), "Empty")
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatalf("建目录失败: %v", err)
		}
		if got := svc.resolveStoreGamePath(ctx, StoreRequest{GameFolder: folder, GameFolderName: "Empty"}); got != "" {
			t.Fatalf("没有 exe 时应留空，实际 %q", got)
		}
	})

	t.Run("什么都不给时留空（启动时再问）", func(t *testing.T) {
		if got := svc.resolveStoreGamePath(ctx, StoreRequest{}); got != "" {
			t.Fatalf("应留空，实际 %q", got)
		}
	})
}
