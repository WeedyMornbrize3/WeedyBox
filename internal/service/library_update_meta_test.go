// internal/service/library_update_meta_test.go
//
// 编辑页保存元数据（LibraryService.UpdateMeta）：用真实临时库，锁住几条关键约定 ——
//   - 打补丁语义：只改传来的键，空值 = 删掉这一项；
//   - 公共列跟着 raw_json 一起变（游戏库列表、详情页读的是列）；
//   - 本地信息不被冲掉：封面指针、game_path、launch_mode；
//   - 编号不可改、名称不可清空、未入库的作品不能编辑。
package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
)

// newTestDB 定义在 library_remove_test.go（真实临时库文件），这里直接复用。

// TestUpdateMetaPatchesRawJSONAndColumns 改名称/发售日、清空价格：
// 返回的 map 与库里的列都应是最新值，未提到的字段保持不变。
func TestUpdateMetaPatchesRawJSONAndColumns(t *testing.T) {
	db := newTestDB(t)
	svc := &LibraryService{DB: db}
	ctx := context.Background()

	const workID = "RJ01721474"
	raw := map[string]any{
		"id":          workID,
		"name":        "旧名字",
		"maker":       "旧社团",
		"releaseDate": "2025-01-01",
		"price":       "1540",
		"summary":     "旧简介",
		"fetchedAt":   "2026-11-12T00:00:00Z",
	}
	blob, _ := json.Marshal(raw)
	if err := db.UpsertLibraryWork(ctx, &model.LibraryWork{
		Source:  database.SourceDLsite,
		WorkID:  workID,
		Name:    "旧名字",
		Maker:   "旧社团",
		RawJSON: string(blob),
	}); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}

	got, err := svc.UpdateMeta(ctx, UpdateMetaRequest{
		WorkID: workID,
		Source: database.SourceDLsite,
		Fields: map[string]any{
			"name":        "新名字",
			"releaseDate": "2026-06-18",
			"price":       "", // 清空这一项
			// "id" 与未提到的键都不该被改
			"id": "RJ00000000",
		},
	})
	if err != nil {
		t.Fatalf("保存元数据失败: %v", err)
	}

	if got["name"] != "新名字" || got["releaseDate"] != "2026-06-18" {
		t.Errorf("返回值没带上修改: name=%v releaseDate=%v", got["name"], got["releaseDate"])
	}
	if _, has := got["price"]; has {
		t.Errorf("price 应被清空，仍存在: %v", got["price"])
	}
	if got["id"] != workID {
		t.Errorf("编号不应被改，实际 %v", got["id"])
	}
	if got["summary"] != "旧简介" || got["maker"] != "旧社团" {
		t.Errorf("未提到的字段被改动了: summary=%v maker=%v", got["summary"], got["maker"])
	}

	row, err := db.FindLibraryWork(ctx, database.SourceDLsite, workID)
	if err != nil || row == nil {
		t.Fatalf("重新读取作品失败: %v", err)
	}
	if row.Name != "新名字" || row.ReleaseDate != "2026-06-18" {
		t.Errorf("公共列没跟着更新: name=%q releaseDate=%q", row.Name, row.ReleaseDate)
	}
	if strings.Contains(row.RawJSON, "1540") {
		t.Errorf("raw_json 里仍残留被清空的 price: %s", row.RawJSON)
	}
	if !strings.Contains(row.RawJSON, "新名字") {
		t.Errorf("raw_json 没写入新名称: %s", row.RawJSON)
	}
	if row.FetchedAt == "" {
		t.Errorf("抓取时间不该因为编辑而丢失: %+v", row)
	}
}

// TestUpdateMetaKeepsLocalInfo 封面指针、游戏位置、启动方式是本地信息，
// 改元数据不能把它们冲掉（UpsertLibraryWork 的 SET 里刻意不含这几列）。
func TestUpdateMetaKeepsLocalInfo(t *testing.T) {
	db := newTestDB(t)
	svc := &LibraryService{DB: db}
	ctx := context.Background()

	const workID = "RJ01721333"
	raw := map[string]any{"id": workID, "name": "作品", "maker": "社团"}
	blob, _ := json.Marshal(raw)
	if err := db.UpsertLibraryWork(ctx, &model.LibraryWork{
		Source:  database.SourceDLsite,
		WorkID:  workID,
		Name:    "作品",
		Maker:   "社团",
		RawJSON: string(blob),
	}); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}
	if err := db.RecordCoverFile(ctx, &model.CoverFile{
		Source: database.SourceDLsite, WorkID: workID,
		Dir: "covers", Path: "covers/a.webp", AbsPath: "/tmp/a.webp", Suffix: ".webp",
	}); err != nil {
		t.Fatalf("记录封面失败: %v", err)
	}
	if err := db.SetGamePath(ctx, database.SourceDLsite, workID, "/games/a.exe"); err != nil {
		t.Fatalf("记录游戏位置失败: %v", err)
	}
	if err := db.SetLaunchMode(ctx, database.SourceDLsite, workID, database.LaunchModeExe); err != nil {
		t.Fatalf("记录启动方式失败: %v", err)
	}

	if _, err := svc.UpdateMeta(ctx, UpdateMetaRequest{
		WorkID: workID,
		Fields: map[string]any{"summary": "新简介"},
	}); err != nil {
		t.Fatalf("保存元数据失败: %v", err)
	}

	row, err := db.FindLibraryWork(ctx, database.SourceDLsite, workID)
	if err != nil || row == nil {
		t.Fatalf("重新读取作品失败: %v", err)
	}
	if row.CoverAbsPath != "/tmp/a.webp" || row.GamePath != "/games/a.exe" || row.LaunchMode != database.LaunchModeExe {
		t.Errorf("本地信息被元数据编辑冲掉了: cover=%q path=%q mode=%q",
			row.CoverAbsPath, row.GamePath, row.LaunchMode)
	}
}

// TestUpdateMetaRejects 名称不能清空、未入库的作品不能编辑。
func TestUpdateMetaRejects(t *testing.T) {
	db := newTestDB(t)
	svc := &LibraryService{DB: db}
	ctx := context.Background()

	const workID = "RJ01721000"
	if err := db.UpsertLibraryWork(ctx, &model.LibraryWork{
		Source: database.SourceDLsite, WorkID: workID, Name: "作品",
		RawJSON: `{"id":"RJ01721000","name":"作品"}`,
	}); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}

	if _, err := svc.UpdateMeta(ctx, UpdateMetaRequest{
		WorkID: workID,
		Fields: map[string]any{"name": "  "},
	}); err == nil {
		t.Error("清空名称应该报错")
	}

	if _, err := svc.UpdateMeta(ctx, UpdateMetaRequest{
		WorkID: "RJ09999999",
		Fields: map[string]any{"name": "别的作品"},
	}); err == nil {
		t.Error("未入库的作品应该报错，而不是凭空造一条记录")
	}
}
