// internal/service/scan/steam_apps_test.go
//
// Steam 源的两项本地能力：
//   - 按 appmanifest 列出已安装游戏（appid / 名称 / 安装目录 / 库根）；
//   - 在 appcache\librarycache\<appid>\ 里递归找 600x900 / capsule 封面。
//
// 注册表在单测里没法伪造，所以测的是「找到 Steam 之后」的那一半，用真实临时目录，
// 不 mock 文件系统（与 steam_test.go 同一套路）。
package scan

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// writeAppManifest 按 Steam 的真实格式写一份 appmanifest_<appid>.acf。
func writeAppManifest(t *testing.T, steamappsDir, appID, name, installDir, stateFlags, sizeOnDisk string) {
	t.Helper()

	content := strings.Join([]string{
		`"AppState"`,
		`{`,
		`	"appid"		"` + appID + `"`,
		`	"universe"		"1"`,
		`	"name"		"` + name + `"`,
		`	"StateFlags"		"` + stateFlags + `"`,
		`	"installdir"		"` + installDir + `"`,
		`	"SizeOnDisk"		"` + sizeOnDisk + `"`,
		`}`,
	}, "\n") + "\n"
	writeFile(t, filepath.Join(steamappsDir, "appmanifest_"+appID+".acf"), content)
}

// TestReadSteamAppManifest 锁住 appmanifest 的解析口径：StateFlags 按位判「已安装」
// （6 也是已安装，实测 R.E.P.O. 就是 6），安装目录为空/未安装完成的要拒掉。
func TestReadSteamAppManifest(t *testing.T) {
	steamRoot := t.TempDir()
	libraryRoot := t.TempDir()
	steamapps := filepath.Join(libraryRoot, "steamapps")

	writeAppManifest(t, steamapps, "3241660", "R.E.P.O.", "REPO", "6", "2262209542")
	app, ok := readSteamAppManifest(
		filepath.Join(steamapps, "appmanifest_3241660.acf"), steamRoot, libraryRoot)
	if !ok {
		t.Fatal("StateFlags=6（已安装+待更新）应算已安装")
	}
	if app.AppID != "3241660" || app.Name != "R.E.P.O." || app.InstallDir != "REPO" {
		t.Errorf("appid/名称/安装目录解析不对: %+v", app)
	}
	if app.SizeOnDisk != 2262209542 {
		t.Errorf("SizeOnDisk = %d，期望 2262209542", app.SizeOnDisk)
	}
	wantFolder := filepath.Join(libraryRoot, "steamapps", "common", "REPO")
	if app.FolderPath != wantFolder {
		t.Errorf("FolderPath = %q，期望 %q", app.FolderPath, wantFolder)
	}
	if app.SteamRoot != filepath.Clean(steamRoot) || app.LibraryRoot != filepath.Clean(libraryRoot) {
		t.Errorf("steamRoot/libraryRoot 没带对: %+v", app)
	}

	// 未安装完成（StateFlags=2 只有"待更新"位）→ 不算
	writeAppManifest(t, steamapps, "111", "NotInstalled", "NotInstalled", "2", "0")
	if _, ok := readSteamAppManifest(
		filepath.Join(steamapps, "appmanifest_111.acf"), steamRoot, libraryRoot); ok {
		t.Error("StateFlags=2 不该算已安装")
	}

	// 没有 installdir 的 manifest 无法定位文件夹 → 跳过
	writeFile(t, filepath.Join(steamapps, "appmanifest_222.acf"),
		"\"AppState\"\n{\n\t\"appid\"\t\t\"222\"\n\t\"name\"\t\t\"NoDir\"\n}\n")
	if _, ok := readSteamAppManifest(
		filepath.Join(steamapps, "appmanifest_222.acf"), steamRoot, libraryRoot); ok {
		t.Error("缺 installdir 不该算有效游戏")
	}
}

// TestScanSteamAppsFromRoots 覆盖多库汇总：主库 + vdf 附加库、同一 appid 只留一份、
// 按名称排序（大小写不敏感）、非 manifest 文件不算。
func TestScanSteamAppsFromRoots(t *testing.T) {
	steamRoot := t.TempDir()
	extraLibrary := t.TempDir()

	mainSteamapps := filepath.Join(steamRoot, "steamapps")
	writeAppManifest(t, mainSteamapps, "1229490", "ULTRAKILL", "ULTRAKILL", "4", "100")
	writeAppManifest(t, mainSteamapps, "228980", "Steamworks Common Redistributables", "Steamworks Shared", "4", "0")
	writeFile(t, filepath.Join(mainSteamapps, "readme.txt"), "不是 manifest")

	extraSteamapps := filepath.Join(extraLibrary, "steamapps")
	writeAppManifest(t, extraSteamapps, "1229490", "ULTRAKILL", "ULTRAKILL", "4", "100") // 跨库重复
	writeAppManifest(t, extraSteamapps, "3527290", "PEAK", "PEAK", "4", "200")

	writeLibraryFoldersVDF(t, filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"), extraLibrary)

	apps, err := scanSteamAppsFromRoots(context.Background(), []string{steamRoot})
	if err != nil {
		t.Fatalf("扫描失败: %v", err)
	}

	want := []string{"PEAK", "Steamworks Common Redistributables", "ULTRAKILL"}
	got := make([]string, 0, len(apps))
	for _, app := range apps {
		got = append(got, app.Name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("游戏名 = %v，期望 %v", got, want)
	}
	// 重复 appid 保留先扫到的（主库）那份
	if apps[2].AppID != "1229490" || apps[2].LibraryRoot != filepath.Clean(steamRoot) {
		t.Errorf("重复 appid 的去重结果不对: %+v", apps[2])
	}
}

func TestScanSteamAppsFromRootsCancelled(t *testing.T) {
	steamRoot := t.TempDir()
	writeAppManifest(t, filepath.Join(steamRoot, "steamapps"), "1", "Game", "Game", "4", "0")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := scanSteamAppsFromRoots(ctx, []string{steamRoot}); err == nil || !strings.Contains(err.Error(), "取消") {
		t.Errorf("应报取消，实际 %v", err)
	}
}

// TestSteamCoverImageInRoots 用户要求的封面规则：按 appid 找目录 → 递归找文件名含
// 600x900 或 capsule 的图片；同档优先中文版；没有目录/没有图都返回空串（不报错）。
func TestSteamCoverImageInRoots(t *testing.T) {
	steamRoot := t.TempDir()
	cacheRoot := filepath.Join(steamRoot, "appcache", "librarycache")

	// ① 有 600x900 的中文版 + 英文版 + 一张 header（header 不该被选）
	appDir := filepath.Join(cacheRoot, "1229490")
	writeFile(t, filepath.Join(appDir, "header.jpg"), "header")
	writeFile(t, filepath.Join(appDir, "library_600x900.jpg"), "en")
	writeFile(t, filepath.Join(appDir, "library_600x900_schinese.jpg"), "cn")

	got, err := steamCoverImageInRoots(context.Background(), []string{steamRoot}, "1229490")
	if err != nil {
		t.Fatalf("找封面失败: %v", err)
	}
	if filepath.Base(got) != "library_600x900_schinese.jpg" {
		t.Errorf("应优先中文版 600x900，实际 %q", got)
	}

	// ② 只有嵌套目录里的 capsule → 也要递归找到
	capsuleDir := filepath.Join(cacheRoot, "3527290")
	writeFile(t, filepath.Join(capsuleDir, "logo.png"), "logo")
	writeFile(t, filepath.Join(capsuleDir, "assets", "deep", "library_capsule.jpg"), "capsule")

	got, err = steamCoverImageInRoots(context.Background(), []string{steamRoot}, "3527290")
	if err != nil {
		t.Fatalf("找封面失败: %v", err)
	}
	if filepath.Base(got) != "library_capsule.jpg" {
		t.Errorf("应递归找到 capsule，实际 %q", got)
	}

	// ③ 没有对应 appid 目录 → 空串（用户要求：没有文件夹就不用图像）
	if got, err = steamCoverImageInRoots(context.Background(), []string{steamRoot}, "9999999"); err != nil || got != "" {
		t.Errorf("没有目录时应返回空串，实际 %q err=%v", got, err)
	}

	// ④ 目录存在但没有 600x900/capsule 图 → 空串
	emptyDir := filepath.Join(cacheRoot, "777")
	writeFile(t, filepath.Join(emptyDir, "header.jpg"), "header")
	if got, err = steamCoverImageInRoots(context.Background(), []string{steamRoot}, "777"); err != nil || got != "" {
		t.Errorf("没有匹配图片时应返回空串，实际 %q err=%v", got, err)
	}
}

// TestSteamCoverScore 打分口径：600x900 > capsule；同档中文版优先。
func TestSteamCoverScore(t *testing.T) {
	if steamCoverScore("library_600x900.jpg") <= steamCoverScore("library_capsule.jpg") {
		t.Error("600x900 应该比 capsule 优先")
	}
	if steamCoverScore("library_600x900_schinese.jpg") <= steamCoverScore("library_600x900.jpg") {
		t.Error("中文版应该优先")
	}
	if !isSteamCoverImageName("library_600x900_schinese.jpg") || isSteamCoverImageName("library_hero.jpg") {
		t.Error("封面文件名筛选不对")
	}
}
