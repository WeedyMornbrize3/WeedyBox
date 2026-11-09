// internal/database/library_repo_test.go
//
// 游戏库仓储层的测试。为什么这些用例不可省：表是**惰性创建**的（启动时不建，
// 第一次读写才建），意味着「表还不存在」是一个经常出现的正常状态。用户明确提醒过
// 「使用表时可能会导致数据库异常」，所以这里重点覆盖：
//
//	1. 全新库里先**读**、先**删**（表尚不存在）不能报错；
//	2. 多个 goroutine 同时首次使用（并发 CREATE TABLE IF NOT EXISTS）不能炸；
//	3. 重复入库不会产生重复行，且不会把已下载的封面指针覆盖掉；
//	4. 来源（source）隔离：同一个编号在不同来源下是两条独立记录。
package database

import (
	"context"
	"WeedyBox/internal/model"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/services/sqlite"
)

// newTestService 起一个真实的临时库（不用 :memory:，因为要覆盖"文件里还没有表"的情形）。
func newTestService(t *testing.T) *SQLiteService {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	raw := sqlite.NewWithConfig(&sqlite.Config{DBSource: path})
	if err := raw.Open(); err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		_ = raw.Close()
		// t.TempDir 会自动清理，这里只是显式忽略 close 错误
		_ = os.Remove(path)
	})

	return &SQLiteService{DB: raw}
}

// tableExists 直接查 sqlite_master，用来断言"表到底建没建"。
func tableExists(t *testing.T, s *SQLiteService, name string) bool {
	t.Helper()
	rows, err := s.Query(context.Background(),
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name)
	if err != nil {
		t.Fatalf("查询 sqlite_master 失败: %v", err)
	}
	return len(rows) > 0
}

func sampleWork() model.LibraryWork {
	return model.LibraryWork{
		Source:      SourceDLsite,
		WorkID:      "rj01721474", // 故意小写，验证入库前会归一化成大写
		Name:        "【简体中文版】エルフのパラダイス",
		Maker:       "青春×フェティシズム",
		MakerID:     "RG51931",
		Series:      "エルフシリーズ",
		SeriesID:    "SRI0001",
		WorkType:    "アドベンチャー",
		AgeRating:   "成人向け",
		ReleaseDate: "2025-10-31",
		CoverURL:    "https://img.dlsite.jp/main.jpg",
		SourceURL:   "https://www.dlsite.com/maniax/work/=/product_id/RJ01721474.html",
		RawJSON:     `{"id":"RJ01721474","name":"【简体中文版】エルフのパラダイス"}`,
		FetchedAt:   "2026-10-18T11:49:37+08:00",
	}
}

// 全新库第一次就「读」：表还不存在，必须平平静静返回"没有缓存"。
func TestFindLibraryWork_FirstReadCreatesTables(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if tableExists(t, s, "library_works") {
		t.Fatal("前置条件失败：全新库里不该已经有 library_works")
	}

	work, err := s.FindLibraryWork(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("首次读取不该报错，实际: %v", err)
	}
	if work != nil {
		t.Fatalf("空库里不该查到记录，实际: %+v", work)
	}
	if !tableExists(t, s, "library_works") {
		t.Fatal("读一次之后应该已经把表建好")
	}
	if !tableExists(t, s, "library_cover_files") {
		t.Fatal("读一次之后封面记录表也该建好")
	}
}

// 全新库第一次就「列」：同样不能报错，返回空切片而不是 nil。
func TestListLibraryWorks_FirstList(t *testing.T) {
	s := newTestService(t)

	works, err := s.ListLibraryWorks(context.Background(), "")
	if err != nil {
		t.Fatalf("首次列表不该报错，实际: %v", err)
	}
	if works == nil {
		t.Fatal("应该返回空切片而不是 nil（前端会直接 .length）")
	}
	if len(works) != 0 {
		t.Fatalf("空库里不该有记录，实际 %d 条", len(works))
	}
}

// 全新库第一次就「删」：最容易被忽略的分支（DELETE 打在不存在的表上）。
func TestDeleteLibraryWork_FirstDelete(t *testing.T) {
	s := newTestService(t)

	if err := s.DeleteLibraryWork(context.Background(), SourceDLsite, "RJ01721474"); err != nil {
		t.Fatalf("首次删除不该报错，实际: %v", err)
	}
}

// 入库 → 读取：编号被归一化成大写，列值回来要对得上。
func TestUpsertAndFindLibraryWork(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	work := sampleWork()
	if err := s.UpsertLibraryWork(ctx, &work); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	got, err := s.FindLibraryWork(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if got == nil {
		t.Fatal("入库后应该能查到")
	}
	if got.WorkID != "RJ01721474" {
		t.Errorf("work_id 应为大写，实际 %q", got.WorkID)
	}
	if got.Source != SourceDLsite {
		t.Errorf("source 应为 %q，实际 %q", SourceDLsite, got.Source)
	}
	if got.Name != work.Name || got.MakerID != work.MakerID {
		t.Errorf("字段没存对: %+v", got)
	}
	if got.RawJSON != work.RawJSON {
		t.Errorf("raw_json 应原样保留，实际 %q", got.RawJSON)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at 应由数据库填上")
	}
}

// 重复入库：只更新不新增，且**不清掉**已下载的封面指针。
func TestUpsertLibraryWork_NoDuplicateAndKeepsCover(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	work := sampleWork()
	if err := s.UpsertLibraryWork(ctx, &work); err != nil {
		t.Fatalf("首次入库失败: %v", err)
	}

	cover := &model.CoverFile{
		Source:   SourceDLsite,
		WorkID:   "RJ01721474",
		Dir:      "preImg",
		Path:     "preImg/RJ01721474_cover.jpg",
		AbsPath:  `F:\Code\WeedyBox\preImg\RJ01721474_cover.jpg`,
		Bytes:    12345,
		Suffix:   ".jpg",
		CoverURL: "https://img.dlsite.jp/main.jpg",
	}
	if err := s.RecordCoverFile(ctx, cover); err != nil {
		t.Fatalf("记录封面失败: %v", err)
	}

	// 再抓一次（模拟刷新缓存），名称变了
	work.Name = "改名了"
	if err := s.UpsertLibraryWork(ctx, &work); err != nil {
		t.Fatalf("二次入库失败: %v", err)
	}

	all, err := s.ListLibraryWorks(ctx, SourceDLsite)
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("同一作品重复入库应只有 1 行，实际 %d 行", len(all))
	}
	if all[0].Name != "改名了" {
		t.Errorf("名称应被更新，实际 %q", all[0].Name)
	}
	if all[0].CoverAbsPath != cover.AbsPath {
		t.Errorf("刷新元数据不该清掉封面指针，实际 %q", all[0].CoverAbsPath)
	}
}

// 来源隔离：同一编号在两个来源下是两条记录。
func TestLibraryWorks_SourceIsolation(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	dlsite := sampleWork()
	if err := s.UpsertLibraryWork(ctx, &dlsite); err != nil {
		t.Fatalf("dlsite 入库失败: %v", err)
	}

	other := sampleWork()
	other.Source = "steam" // 假装以后接了别的来源
	other.Name = "同一个编号，不同来源"
	if err := s.UpsertLibraryWork(ctx, &other); err != nil {
		t.Fatalf("steam 入库失败: %v", err)
	}

	all, err := s.ListLibraryWorks(ctx, "")
	if err != nil {
		t.Fatalf("列全部失败: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("两个来源应各留一行，实际 %d 行", len(all))
	}

	onlyDlsite, err := s.ListLibraryWorks(ctx, SourceDLsite)
	if err != nil {
		t.Fatalf("按来源列表失败: %v", err)
	}
	if len(onlyDlsite) != 1 || onlyDlsite[0].Name != dlsite.Name {
		t.Fatalf("按来源过滤结果不对: %+v", onlyDlsite)
	}

	if got, err := s.FindLibraryWork(ctx, "steam", "RJ01721474"); err != nil || got == nil {
		t.Fatalf("应能按来源查到 steam 那条，err=%v got=%+v", err, got)
	}
}

// 同一作品下载两次封面：下载记录两行，作品表的指针跟着最新那次走。
func TestRecordCoverFile_KeepsHistoryPointsToLatest(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	work := sampleWork()
	if err := s.UpsertLibraryWork(ctx, &work); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	first := &model.CoverFile{Source: SourceDLsite, WorkID: "RJ01721474", Dir: "preImg",
		Path: "preImg/RJ01721474_cover.jpg", AbsPath: `F:\a\RJ01721474_cover.jpg`, Bytes: 100, Suffix: ".jpg"}
	second := &model.CoverFile{Source: SourceDLsite, WorkID: "RJ01721474", Dir: "preImg",
		Path: "preImg/RJ01721474_cover-2.jpg", AbsPath: `F:\a\RJ01721474_cover-2.jpg`, Bytes: 200, Suffix: ".webp"}

	for _, cover := range []*model.CoverFile{first, second} {
		if err := s.RecordCoverFile(ctx, cover); err != nil {
			t.Fatalf("记录封面失败: %v", err)
		}
	}

	rows, err := s.Query(ctx, `SELECT COUNT(*) AS n FROM library_cover_files WHERE work_id = ?`, "RJ01721474")
	if err != nil {
		t.Fatalf("统计封面记录失败: %v", err)
	}
	if n := intOf(rows[0]["n"]); n != 2 {
		t.Fatalf("应留下 2 条下载记录，实际 %d", n)
	}

	got, err := s.FindLibraryWork(ctx, SourceDLsite, "RJ01721474")
	if err != nil || got == nil {
		t.Fatalf("查询作品失败: err=%v", err)
	}
	if got.CoverAbsPath != second.AbsPath {
		t.Errorf("指针应指向最新一次下载，实际 %q", got.CoverAbsPath)
	}
	if got.CoverBytes != 200 || got.CoverSuffix != ".webp" {
		t.Errorf("封面元信息应是最后一次的，实际 bytes=%d suffix=%q", got.CoverBytes, got.CoverSuffix)
	}
}

// 作品不在库里时下封面：只记下载记录，不该凭空造一条作品。
func TestRecordCoverFile_NoWorkRow(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	cover := &model.CoverFile{Source: SourceDLsite, WorkID: "RJ99999999", Dir: "preImg",
		Path: "preImg/RJ99999999_cover.jpg", AbsPath: `F:\a\RJ99999999_cover.jpg`, Bytes: 1, Suffix: ".jpg"}
	if err := s.RecordCoverFile(ctx, cover); err != nil {
		t.Fatalf("记录封面失败: %v", err)
	}

	if got, _ := s.FindLibraryWork(ctx, SourceDLsite, "RJ99999999"); got != nil {
		t.Fatalf("不该凭空造作品行，实际 %+v", got)
	}
}

// 空编号/空服务：一律当无操作，不能 panic。
func TestLibraryRepo_EmptyInputs(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	if got, err := s.FindLibraryWork(ctx, "", ""); err != nil || got != nil {
		t.Fatalf("空编号应返回 nil,nil，实际 got=%+v err=%v", got, err)
	}
	if err := s.UpsertLibraryWork(ctx, &model.LibraryWork{WorkID: "  "}); err == nil {
		t.Fatal("空 work_id 入库应报错（否则会写一行无用数据）")
	}
	if err := s.DeleteLibraryWork(ctx, SourceDLsite, ""); err == nil {
		t.Fatal("空 work_id 删除应报错")
	}

	var nilService *SQLiteService
	if got, err := nilService.FindLibraryWork(ctx, SourceDLsite, "RJ1"); err != nil || got != nil {
		t.Fatalf("nil 服务应安全返回，实际 got=%+v err=%v", got, err)
	}
	if works, err := nilService.ListLibraryWorks(ctx, ""); err != nil || works == nil {
		t.Fatalf("nil 服务列表应返回空切片，实际 err=%v", err)
	}
}

// 并发首次使用：多个 goroutine 同时撞上"表还没建"，CREATE TABLE IF NOT EXISTS
// 必须幂等且不互相报错（这是惰性建表最大的风险点）。
func TestLibraryRepo_ConcurrentFirstUse(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers*2)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// 故意先读（会触发建表），再写
			if _, err := s.FindLibraryWork(ctx, SourceDLsite, "RJ0000000"+string(rune('0'+idx))); err != nil {
				errs <- err
				return
			}
			work := sampleWork()
			work.WorkID = "RJ0000000" + string(rune('0'+idx))
			work.Name = "并发写入"
			if err := s.UpsertLibraryWork(ctx, &work); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("并发首用报错: %v", err)
	}

	all, err := s.ListLibraryWorks(ctx, SourceDLsite)
	if err != nil {
		t.Fatalf("列表失败: %v", err)
	}
	if len(all) != workers {
		t.Fatalf("应有 %d 条记录，实际 %d 条", workers, len(all))
	}
}
