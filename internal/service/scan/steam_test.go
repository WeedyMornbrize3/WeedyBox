// internal/service/scan/steam_test.go
//
// 注册表在单元测试里没法伪造（steamInstallRoots 直连 Windows 注册表），所以
// 这里覆盖的是「找到 Steam 之后」的那一半：vdf 解析 + 多库汇总去重 + 容错。
// 用真实的临时目录当 Steam 根，不 mock 文件系统。
package scan

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// modernVDF 与 Steam 现在写的 libraryfolders.vdf 结构一致（含 apps 段与 contentstatsid）。
const modernVDF = `"libraryfolders"
{
	"0"
	{
		"path"		"C:\\Program Files (x86)\\Steam"
		"label"		""
		"contentstatsid"		"-1234567890"
		"apps"
		{
			"228980"		"1234567890"
		}
	}
	"1"
	{
		"path"		"D:\\SteamLibrary"
		"label"		""
	}
}`

// legacyVDF 是旧版客户端的格式：键是数字，没有 path 行。
const legacyVDF = `"LibraryFolders"
{
	"TimeNextStatsReport"		"1234567890"
	"ContentStatsID"		"-1234567890"
	"1"		"D:\\SteamLibrary"
	"2"		"E:\\Games\\Steam"
}`

func TestParseSteamLibraryFoldersTextModern(t *testing.T) {
	roots := parseSteamLibraryFoldersText(modernVDF)
	want := []string{`C:\Program Files (x86)\Steam`, `D:\SteamLibrary`}

	if len(roots) != len(want) {
		t.Fatalf("库数 = %d，期望 %d：%v", len(roots), len(want), roots)
	}
	for i, value := range want {
		if roots[i] != value {
			t.Errorf("roots[%d] = %q，期望 %q", i, roots[i], value)
		}
	}
}

func TestParseSteamLibraryFoldersTextLegacy(t *testing.T) {
	roots := parseSteamLibraryFoldersText(legacyVDF)
	want := []string{`D:\SteamLibrary`, `E:\Games\Steam`}

	if len(roots) != len(want) {
		t.Fatalf("库数 = %d，期望 %d：%v", len(roots), len(want), roots)
	}
	for i, value := range want {
		if roots[i] != value {
			t.Errorf("roots[%d] = %q，期望 %q", i, roots[i], value)
		}
	}
}

// TestParseSteamLibraryFoldersTextNoLibraries 空文件/没有库记录时返回空，不 panic。
func TestParseSteamLibraryFoldersTextNoLibraries(t *testing.T) {
	for _, text := range []string{"", `"libraryfolders" { }`, "随便一段不是 vdf 的内容"} {
		if roots := parseSteamLibraryFoldersText(text); len(roots) != 0 {
			t.Errorf("文本 %q 解析出 %v，期望空", text, roots)
		}
	}
}

// writeLibraryFoldersVDF 按现役格式写一份 libraryfolders.vdf（路径里的 \ 要转义成 \\）。
func writeLibraryFoldersVDF(t *testing.T, vdfPath string, libraryRoots ...string) {
	t.Helper()

	var builder strings.Builder
	builder.WriteString("\"libraryfolders\"\n{\n")
	for i, libraryRoot := range libraryRoots {
		builder.WriteString(fmt.Sprintf("\t\"%d\"\n\t{\n", i))
		builder.WriteString(fmt.Sprintf("\t\t\"path\"\t\t\"%s\"\n", strings.ReplaceAll(libraryRoot, `\`, `\\`)))
		builder.WriteString("\t}\n")
	}
	builder.WriteString("}\n")

	writeFile(t, vdfPath, builder.String())
}

// TestScanSteamRoots 覆盖多库汇总：主库 + vdf 里的附加库、跨库重名去重（大小写不同）、
// 不存在的库（盘没插）跳过、common 里的普通文件不算。
func TestScanSteamRoots(t *testing.T) {
	steamRoot := t.TempDir()
	extraLibrary := t.TempDir()

	mkdirAll(t, filepath.Join(steamRoot, "steamapps", "common", "GameA"))
	mkdirAll(t, filepath.Join(steamRoot, "steamapps", "common", "GameB"))
	writeFile(t, filepath.Join(steamRoot, "steamapps", "common", "readme.txt"), "文件不算")

	mkdirAll(t, filepath.Join(extraLibrary, "steamapps", "common", "gamea")) // 与 GameA 同名字不同大小写
	mkdirAll(t, filepath.Join(extraLibrary, "steamapps", "common", "GameC"))

	missingLibrary := filepath.Join(steamRoot, "unplugged-drive")
	writeLibraryFoldersVDF(t,
		filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"),
		extraLibrary, missingLibrary,
	)

	names, err := (&ScanService{}).scanSteamRoots(context.Background(), []string{steamRoot})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	want := []string{"GameA", "GameB", "GameC"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("游戏名 = %v，期望 %v", names, want)
	}
}

// TestScanSteamRootsReadsLegacyVDFPath 旧客户端把 vdf 放在 config\ 下，也要能读到。
func TestScanSteamRootsReadsLegacyVDFPath(t *testing.T) {
	steamRoot := t.TempDir()
	extraLibrary := t.TempDir()

	mkdirAll(t, filepath.Join(steamRoot, "steamapps", "common", "MainGame"))
	mkdirAll(t, filepath.Join(extraLibrary, "steamapps", "common", "ExtraGame"))
	writeLibraryFoldersVDF(t, filepath.Join(steamRoot, "config", "libraryfolders.vdf"), extraLibrary)

	names, err := (&ScanService{}).scanSteamRoots(context.Background(), []string{steamRoot})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	want := []string{"ExtraGame", "MainGame"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("游戏名 = %v，期望 %v", names, want)
	}
}

// TestSteamCommonDirsDedupesMainLibrary vdf 里通常也会列出主库自己，别重复返回。
func TestSteamCommonDirsDedupesMainLibrary(t *testing.T) {
	steamRoot := t.TempDir()
	writeLibraryFoldersVDF(t, filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"), steamRoot)

	dirs := steamCommonDirs(steamRoot)
	if len(dirs) != 1 {
		t.Fatalf("common 目录数 = %d，期望 1：%v", len(dirs), dirs)
	}
	if dirs[0] != filepath.Join(steamRoot, "steamapps", "common") {
		t.Errorf("common 目录 = %q，期望 %q", dirs[0], filepath.Join(steamRoot, "steamapps", "common"))
	}
}

// TestScanSteamRootsWithoutCommon 目录存在但没有 steamapps\common：空名单、不报错
// （库盘没插、Steam 刚装还没下游戏，都是正常状态）。
func TestScanSteamRootsWithoutCommon(t *testing.T) {
	names, err := (&ScanService{}).scanSteamRoots(context.Background(), []string{t.TempDir()})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("游戏名 = %v，期望空", names)
	}
}

func TestScanSteamRootsCancelled(t *testing.T) {
	steamRoot := t.TempDir()
	mkdirAll(t, filepath.Join(steamRoot, "steamapps", "common", "Game"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := (&ScanService{}).scanSteamRoots(ctx, []string{steamRoot}); err == nil || !strings.Contains(err.Error(), "取消") {
		t.Errorf("应报取消，实际 %v", err)
	}
}
