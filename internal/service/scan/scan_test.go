// internal/service/scan/scan_test.go
//
// 用 t.TempDir() 搭真实目录结构测，不 mock 文件系统：这两个方法的全部价值
// 就在「只读一层、不递归」的边界上，mock 出来的树证明不了这一点。
package scan

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("创建目录 %s 失败: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mkdirAll(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写文件 %s 失败: %v", path, err)
	}
}

// TestScanExecutablesInSubfolders 覆盖「只读一层」的全部边界：
// 根目录的 exe、更深层的 exe、没有 exe 的文件夹、名字像 exe 的目录都不能出错。
func TestScanExecutablesInSubfolders(t *testing.T) {
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "root.exe"), "根目录的散装 exe，不算游戏文件夹")
	writeFile(t, filepath.Join(root, "readme.txt"), "普通文件，不是目录")

	writeFile(t, filepath.Join(root, "Alpha", "game.exe"), "alpha")
	writeFile(t, filepath.Join(root, "Alpha", "Launcher.EXE"), "launcher")
	writeFile(t, filepath.Join(root, "Alpha", "notes.txt"), "不是 exe")
	writeFile(t, filepath.Join(root, "Alpha", "nested", "deep.exe"), "二层，不递归")
	mkdirAll(t, filepath.Join(root, "Alpha", "fake.exe")) // 名字像 exe 的目录

	writeFile(t, filepath.Join(root, "Beta", "tool.exe"), "tool")

	mkdirAll(t, filepath.Join(root, "Gamma")) // 没有 exe 的文件夹

	result, err := (&ScanService{}).ScanExecutablesInSubfolders(context.Background(), root)
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	if result.Root != filepath.Clean(root) {
		t.Errorf("Root = %q，期望 %q", result.Root, filepath.Clean(root))
	}
	if len(result.Skipped) != 0 {
		t.Errorf("不该有跳过的文件夹，实际 %+v", result.Skipped)
	}
	if len(result.Folders) != 2 {
		t.Fatalf("文件夹数 = %d，期望 2（Alpha/Beta）：%+v", len(result.Folders), result.Folders)
	}

	alpha := result.Folders[0]
	if alpha.FolderName != "Alpha" {
		t.Fatalf("第一个文件夹 = %q，期望 Alpha（ReadDir 按名称排序）", alpha.FolderName)
	}
	if alpha.FolderPath != filepath.Join(result.Root, "Alpha") {
		t.Errorf("Alpha.FolderPath = %q，期望 %q", alpha.FolderPath, filepath.Join(result.Root, "Alpha"))
	}

	// os.ReadDir 按文件名排序：'L'(76) < 'g'(103)，所以 Launcher.EXE 在前。
	wantAlphaNames := []string{"Launcher.EXE", "game.exe"}
	if len(alpha.Executables) != len(wantAlphaNames) {
		t.Fatalf("Alpha 的 exe 数 = %d，期望 %d：%+v", len(alpha.Executables), len(wantAlphaNames), alpha.Executables)
	}
	for i, want := range wantAlphaNames {
		got := alpha.Executables[i]
		if got.FileName != want {
			t.Errorf("Alpha.Executables[%d].FileName = %q，期望 %q", i, got.FileName, want)
		}
		if got.Path != filepath.Join(result.Root, "Alpha", want) {
			t.Errorf("Alpha.Executables[%d].Path = %q，期望 %q", i, got.Path, filepath.Join(result.Root, "Alpha", want))
		}
	}
	// Name 去扩展名，大小写原样保留（.EXE → Launcher）
	if alpha.Executables[0].Name != "Launcher" {
		t.Errorf("Name = %q，期望 Launcher（去掉 .EXE 扩展名）", alpha.Executables[0].Name)
	}
	if alpha.Executables[1].Name != "game" {
		t.Errorf("Name = %q，期望 game", alpha.Executables[1].Name)
	}
	if alpha.Executables[1].Size != int64(len("alpha")) {
		t.Errorf("game.exe Size = %d，期望 %d", alpha.Executables[1].Size, len("alpha"))
	}

	beta := result.Folders[1]
	if beta.FolderName != "Beta" || len(beta.Executables) != 1 || beta.Executables[0].FileName != "tool.exe" {
		t.Errorf("Beta 结果不对: %+v", beta)
	}
}

// TestScanExecutablesInSubfoldersTrailingSeparator 尾巴上多一个分隔符也要能扫。
func TestScanExecutablesInSubfoldersTrailingSeparator(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Alpha", "game.exe"), "alpha")

	result, err := (&ScanService{}).ScanExecutablesInSubfolders(context.Background(), root+string(os.PathSeparator))
	if err != nil {
		t.Fatalf("带尾部分隔符的扫描失败: %v", err)
	}
	if result.Root != filepath.Clean(root) {
		t.Errorf("Root = %q，期望 %q", result.Root, filepath.Clean(root))
	}
	if len(result.Folders) != 1 {
		t.Errorf("文件夹数 = %d，期望 1", len(result.Folders))
	}
}

// TestScanSubfolderNames 只认名字：没有 exe 的空文件夹也要出现，
// 而且要包含根目录下所有一级文件夹、不含文件。
func TestScanSubfolderNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Alpha", "game.exe"), "alpha")
	mkdirAll(t, filepath.Join(root, "Beta"))
	mkdirAll(t, filepath.Join(root, "Gamma", "deep")) // 深层目录，只返回第一层
	writeFile(t, filepath.Join(root, "not-a-folder.txt"), "文件不是文件夹")

	names, err := (&ScanService{}).ScanSubfolderNames(context.Background(), root)
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	want := []string{"Alpha", "Beta", "Gamma"}
	if len(names) != len(want) {
		t.Fatalf("名称数 = %d，期望 %d：%v", len(names), len(want), names)
	}
	for i, name := range want {
		if names[i] != name {
			t.Errorf("names[%d] = %q，期望 %q", i, names[i], name)
		}
	}
}

// TestScanSubfolderNamesEmptyRoot 空目录返回空名单而不是 nil/错误。
func TestScanSubfolderNamesEmptyRoot(t *testing.T) {
	names, err := (&ScanService{}).ScanSubfolderNames(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("扫描空目录失败: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("空目录应返回空名单，实际 %v", names)
	}
}

// TestScanRejectsBadRoot 坏输入要在进入 I/O 前就被拒掉，并且报错说清是哪一种坏。
func TestScanRejectsBadRoot(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "plain.txt")
	writeFile(t, filePath, "不是目录")

	cases := []struct {
		name     string
		root     string
		contains string
	}{
		{"空目录", "   ", "扫描目录为空"},
		{"不存在的目录", filepath.Join(root, "no-such-dir"), "不存在"},
		{"路径是文件", filePath, "不是文件夹"},
	}

	service := &ScanService{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.ScanExecutablesInSubfolders(context.Background(), tc.root); err == nil {
				t.Error("ScanExecutablesInSubfolders 未报错")
			} else if !strings.Contains(err.Error(), tc.contains) {
				t.Errorf("ScanExecutablesInSubfolders 报错 %q，期望包含 %q", err.Error(), tc.contains)
			}

			if _, err := service.ScanSubfolderNames(context.Background(), tc.root); err == nil {
				t.Error("ScanSubfolderNames 未报错")
			} else if !strings.Contains(err.Error(), tc.contains) {
				t.Errorf("ScanSubfolderNames 报错 %q，期望包含 %q", err.Error(), tc.contains)
			}
		})
	}
}

// TestScanCancelledContext 已取消的上下文必须立刻报「取消」，不能把死目录当空目录扫完。
func TestScanCancelledContext(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Alpha", "game.exe"), "alpha")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	service := &ScanService{}
	if _, err := service.ScanExecutablesInSubfolders(ctx, root); err == nil || !strings.Contains(err.Error(), "取消") {
		t.Errorf("ScanExecutablesInSubfolders 应报取消，实际 %v", err)
	}
	if _, err := service.ScanSubfolderNames(ctx, root); err == nil || !strings.Contains(err.Error(), "取消") {
		t.Errorf("ScanSubfolderNames 应报取消，实际 %v", err)
	}
}
