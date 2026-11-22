// internal/service/getMeta/getFromSteam_test.go
//
// Steam 源的解析口径与本地封面导入：
//   - appdetails → 作品 map 的字段映射（id/name/maker/workType/summary/…）；
//   - 非游戏条目（tool/dlc）要能被前端识别并跳过（steamType）；
//   - librarycache 里的封面复制进 covers/ 后要记账，重名不覆盖。
//
// 全部离线：用真实 appdetails 返回的 JSON 片段当 fixture，不发网络请求。
package getmeta

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
)

// steamFixture 是按 appdetails 真实结构裁出来的样本（schinese）。
const steamFixture = `{
  "1229490": {
    "success": true,
    "data": {
      "type": "game",
      "name": "ULTRAKILL",
      "steam_appid": 1229490,
      "required_age": 0,
      "is_free": false,
      "short_description": "一款快节奏的复古射击游戏。",
      "header_image": "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/1229490/header.jpg",
      "capsule_image": "https://shared.akamai.steamstatic.com/store_item_assets/steam/apps/1229490/capsule_231x87.jpg",
      "developers": ["Arsi \"Hakita\" Patala", "New Blood Interactive"],
      "publishers": ["New Blood Interactive"],
      "genres": [{"id": "1", "description": "动作"}],
      "categories": [{"id": 1, "description": "单人"}],
      "release_date": {"coming_soon": false, "date": "2020 年 9 月 3 日"},
      "platforms": {"windows": true, "mac": false, "linux": true},
      "price_overview": {"currency": "CNY", "initial": 9200, "final": 4600, "final_formatted": "¥ 46.00"},
      "recommendations": {"total": 100000},
      "achievements": {"total": 42},
      "metacritic": {"score": 90, "url": "https://www.metacritic.com/game/ultrakill/"}
    }
  }
}`

// TestParseSteamAppDetails 锁住字段映射：作品库现有组件不改也能显示。
func TestParseSteamAppDetails(t *testing.T) {
	work, err := parseSteamAppDetails("1229490", []byte(steamFixture))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if work["id"] != "1229490" || work["name"] != "ULTRAKILL" {
		t.Errorf("编号/名称不对: %v / %v", work["id"], work["name"])
	}
	if work["maker"] != `Arsi "Hakita" Patala / New Blood Interactive` {
		t.Errorf("maker = %q", work["maker"])
	}
	if work["workType"] != "游戏" || work["steamType"] != "game" {
		t.Errorf("类型映射不对: %v / %v", work["workType"], work["steamType"])
	}
	if work["releaseDate"] != "2020 年 9 月 3 日" {
		t.Errorf("releaseDate = %q", work["releaseDate"])
	}
	if work["summary"] != "一款快节奏的复古射击游戏。" {
		t.Errorf("summary = %q", work["summary"])
	}
	if work["coverUrl"] == "" || work["sourceUrl"] != "https://store.steampowered.com/app/1229490/" {
		t.Errorf("封面/来源页不对: %v / %v", work["coverUrl"], work["sourceUrl"])
	}
	if work["price"] != "¥ 46.00" {
		t.Errorf("价格 = %q", work["price"])
	}
	if work["fetchedAt"] == "" {
		t.Error("fetchedAt 为空")
	}

	tags, ok := work["tags"].([]string)
	if !ok || len(tags) < 4 {
		t.Fatalf("tags = %#v，期望至少 4 个（分类/类型/开发商/发行商）", work["tags"])
	}
	for _, want := range []string{"动作", "单人", "New Blood Interactive"} {
		found := false
		for _, tag := range tags {
			if tag == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tags 缺少 %q：%v", want, tags)
		}
	}

	outline, ok := work["outline"].([]map[string]string)
	if !ok || len(outline) == 0 {
		t.Fatalf("outline = %#v，期望有明细行", work["outline"])
	}
	if work["steam"] == nil {
		t.Error("完整原始数据（steam 键）不该丢")
	}
}

// TestParseSteamAppDetailsFailures not-found / 下架 / 数据为空都要报错，不能返回空壳。
func TestParseSteamAppDetailsFailures(t *testing.T) {
	cases := map[string]string{
		"没有该 appid": `{"999": {"success": false}}`,
		"success 为 false": `{"999": {"success": false}}`,
		"data 是空数组":      `{"999": {"success": true, "data": []}}`,
		"返回不是 json":      `oops`,
	}
	for name, body := range cases {
		if _, err := parseSteamAppDetails("999", []byte(body)); err == nil {
			t.Errorf("%s：应该报错", name)
		}
	}
}

// TestBuildSteamWorkRequiresName 名称是底线（作品库靠它显示），缺了就报错。
func TestBuildSteamWorkRequiresName(t *testing.T) {
	if _, err := buildSteamWork("1", []byte(`{"type": "game"}`)); err == nil {
		t.Error("缺名称应该报错")
	}
}

// TestNormalizeSteamAppID 只接受整串数字，避免把 RJ 编号之类误当 appid。
func TestNormalizeSteamAppID(t *testing.T) {
	cases := map[string]string{
		"1229490":    "1229490",
		"  3527290":  "3527290",
		"RJ01721474": "",
		"appid 123":  "",
		"":           "",
		"12a3":       "",
	}
	for input, want := range cases {
		if got := NormalizeSteamAppID(input); got != want {
			t.Errorf("NormalizeSteamAppID(%q) = %q，期望 %q", input, got, want)
		}
	}
}

// TestFetchSteamAppRejectsInvalidID 非法编号在发请求之前就报错（不把 RJ 编号发去商店）。
func TestFetchSteamAppRejectsInvalidID(t *testing.T) {
	svc := NewMetaService()
	if _, err := svc.FetchSteamApp(context.Background(), "RJ01721474"); err == nil {
		t.Fatal("非法 appid 应该直接报错")
	}
}

// TestImportCoverFile 本地封面（Steam librarycache）复制进 covers/ 并记账：
// 文件名取编号、后缀按真实内容、作品表的 cover_* 指针要更新。
func TestImportCoverFile(t *testing.T) {
	t.Chdir(t.TempDir()) // coversDir 是相对工作目录的

	src := filepath.Join(t.TempDir(), "library_600x900_schinese.jpg")
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 'j', 'p', 'g'}
	if err := os.WriteFile(src, jpegBytes, 0o644); err != nil {
		t.Fatalf("准备源文件失败: %v", err)
	}

	db := newTestDB(t)
	ctx := context.Background()
	if err := db.UpsertLibraryWork(ctx, &model.LibraryWork{
		Source: database.SourceSteam,
		WorkID: "1229490",
		Name:   "ULTRAKILL",
	}); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}

	svc := &MetaApiService{DB: db}
	result, err := svc.ImportCoverFile(ctx, database.SourceSteam, "1229490", src)
	if err != nil {
		t.Fatalf("导入封面失败: %v", err)
	}
	if result["suffix"] != ".jpg" {
		t.Errorf("后缀 = %v，期望 .jpg（按真实内容判定）", result["suffix"])
	}
	relPath, _ := result["path"].(string)
	if !strings.HasPrefix(relPath, coversDir+"/") || filepath.Ext(relPath) != ".jpg" {
		t.Errorf("落地路径不对: %q", relPath)
	}
	if _, err := os.Stat(filepath.Join(coversDir, "1229490_cover.jpg")); err != nil {
		t.Errorf("covers/ 下应有 1229490_cover.jpg: %v", err)
	}

	row, err := db.FindLibraryWork(ctx, database.SourceSteam, "1229490")
	if err != nil || row == nil {
		t.Fatalf("读作品失败: %v", err)
	}
	if row.CoverAbsPath == "" || row.CoverSuffix != ".jpg" || row.CoverBytes != int64(len(jpegBytes)) {
		t.Errorf("封面指针没记账: %+v", row)
	}

	// 再导入一次：不覆盖已有文件（与 DownloadCover 一致）
	second, err := svc.ImportCoverFile(ctx, database.SourceSteam, "1229490", src)
	if err != nil {
		t.Fatalf("重复导入失败: %v", err)
	}
	if second["path"] == result["path"] {
		t.Error("重复导入应生成 -2 后缀的新文件，而不是覆盖")
	}
	if _, err := os.Stat(filepath.Join(coversDir, "1229490_cover-2.jpg")); err != nil {
		t.Errorf("应有 1229490_cover-2.jpg: %v", err)
	}
}
