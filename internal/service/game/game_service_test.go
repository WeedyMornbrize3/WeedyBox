// internal/service/game/game_service_test.go
//
// 只测「不真的启动游戏」就能覆盖的部分：路径归一化/校验、位置读写、
// 以及「位置没设/文件没了」时 Launch 返回 Missing 而不是报错
// （这个分支决定了前端要不要弹选择框，最容易回归）。
package game

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
	"github.com/wailsapp/wails/v3/pkg/services/sqlite"
)

// newTestGameService 起一个真实临时库（与 database 包的测试同一套做法：
// 不用 :memory:，因为要覆盖"文件里还没有表"的惰性建表路径）。
func newTestGameService(t *testing.T) *GameService {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	raw := sqlite.NewWithConfig(&sqlite.Config{DBSource: path})
	if err := raw.Open(); err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	t.Cleanup(func() {
		_ = raw.Close()
		_ = os.Remove(path)
	})
	return &GameService{DB: &database.SQLiteService{DB: raw}}
}

// writeTempExe 造一个"看起来像游戏程序"的文件（测试不会执行它）。
func writeTempExe(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "game.exe")
	if err := os.WriteFile(path, []byte("MZ fake"), 0o644); err != nil {
		t.Fatalf("写临时 exe 失败: %v", err)
	}
	return path
}

// seedLibraryWork 先让作品存在于游戏库 —— 启动位置是 library_works 上的一列，
// 只有入库过的作品才能记录（与线上流程一致）。
func seedLibraryWork(t *testing.T, s *GameService, workID string) {
	t.Helper()
	work := &model.LibraryWork{
		Source:  database.SourceDLsite,
		WorkID:  workID,
		Name:    "测试作品",
		RawJSON: `{"id":"` + workID + `","name":"测试作品"}`,
	}
	if err := s.DB.UpsertLibraryWork(context.Background(), work); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}
}

func TestNormalizeGamePath(t *testing.T) {
	exe := writeTempExe(t)
	dir := filepath.Dir(exe)

	t.Run("空路径报错", func(t *testing.T) {
		if _, err := normalizeGamePath("  "); err == nil {
			t.Fatal("空路径应该报错")
		}
	})

	t.Run("不存在的文件报错", func(t *testing.T) {
		if _, err := normalizeGamePath(filepath.Join(dir, "nope.exe")); err == nil {
			t.Fatal("不存在的文件应该报错")
		}
	})

	t.Run("文件夹报错", func(t *testing.T) {
		if _, err := normalizeGamePath(dir); err == nil {
			t.Fatal("文件夹不是游戏程序，应该报错")
		}
	})

	t.Run("存在的文件归一化成绝对路径", func(t *testing.T) {
		got, err := normalizeGamePath(exe)
		if err != nil {
			t.Fatalf("合法 exe 不该报错: %v", err)
		}
		if got != filepath.Clean(exe) {
			t.Fatalf("期望 %q，实际 %q", filepath.Clean(exe), got)
		}
	})
}

func TestSetPath_WritesLibraryWorkColumn(t *testing.T) {
	s := newTestGameService(t)
	ctx := context.Background()
	seedLibraryWork(t, s, "RJ01234567")
	exe := writeTempExe(t)

	// 编号故意用小写：入库/查询都会归一化成大写
	state, err := s.SetPath(ctx, SetGamePathRequest{WorkID: "rj01234567", Path: exe})
	if err != nil {
		t.Fatalf("设置游戏位置失败: %v", err)
	}
	if state.WorkID != "RJ01234567" {
		t.Fatalf("编号应该转大写，实际 %q", state.WorkID)
	}
	if state.Path != filepath.Clean(exe) || !state.Exists {
		t.Fatalf("设置后的状态不对: %+v", state)
	}

	// 位置与元数据同表：从 library_works 的 game_path 列读得出来
	got, err := s.DB.GetGamePath(ctx, database.SourceDLsite, "RJ01234567")
	if err != nil {
		t.Fatalf("读取游戏位置失败: %v", err)
	}
	if got != filepath.Clean(exe) {
		t.Fatalf("读回来的位置不对：期望 %q，实际 %q", filepath.Clean(exe), got)
	}

	// 再设置一次（同一个作品换 exe）：覆盖那一格，而不是另起一行
	second := writeTempExe(t)
	if _, err := s.SetPath(ctx, SetGamePathRequest{WorkID: "RJ01234567", Path: second}); err != nil {
		t.Fatalf("更新游戏位置失败: %v", err)
	}
	rows, err := s.DB.Query(ctx,
		`SELECT game_path FROM library_works WHERE source = ? AND work_id = ?`,
		database.SourceDLsite, "RJ01234567")
	if err != nil {
		t.Fatalf("查询游戏库失败: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("同一作品应该只有一行库记录，实际 %d 行", len(rows))
	}
	if textOf(rows[0]["game_path"]) != filepath.Clean(second) {
		t.Fatalf("位置没有被更新，实际 %v", rows[0]["game_path"])
	}
}

// 重新抓取/重新入库（UpsertLibraryWork）不能把已经设置好的启动位置冲掉。
func TestUpsertLibraryWork_KeepsGamePath(t *testing.T) {
	s := newTestGameService(t)
	ctx := context.Background()
	seedLibraryWork(t, s, "RJ01234567")
	exe := writeTempExe(t)

	if _, err := s.SetPath(ctx, SetGamePathRequest{WorkID: "RJ01234567", Path: exe}); err != nil {
		t.Fatalf("设置游戏位置失败: %v", err)
	}

	refreshed := &model.LibraryWork{
		Source:  database.SourceDLsite,
		WorkID:  "RJ01234567",
		Name:    "刷新后的作品名",
		RawJSON: `{"id":"RJ01234567","name":"刷新后的作品名","maker":"某社团"}`,
	}
	if err := s.DB.UpsertLibraryWork(ctx, refreshed); err != nil {
		t.Fatalf("刷新元数据失败: %v", err)
	}

	got, err := s.DB.GetGamePath(ctx, database.SourceDLsite, "RJ01234567")
	if err != nil {
		t.Fatalf("读取游戏位置失败: %v", err)
	}
	if got != filepath.Clean(exe) {
		t.Fatalf("刷新元数据后位置丢了：期望 %q，实际 %q", filepath.Clean(exe), got)
	}
}

// 作品不在游戏库里时不许记录位置（否则会给库凭空造一条空壳记录）。
func TestSetPath_RejectsWorkNotInLibrary(t *testing.T) {
	s := newTestGameService(t)
	exe := writeTempExe(t)

	if _, err := s.SetPath(context.Background(), SetGamePathRequest{WorkID: "RJ01234567", Path: exe}); err == nil {
		t.Fatal("没入库的作品不该能记录启动位置")
	}
}

// 位置没设置：Launch 必须返回 Missing（前端据此弹选择框），而不是报错。
func TestLaunch_NoPathIsMissing(t *testing.T) {
	s := newTestGameService(t)
	seedLibraryWork(t, s, "RJ01234567")

	result, err := s.Launch(context.Background(), "RJ01234567", "")
	if err != nil {
		t.Fatalf("没有位置不该报错: %v", err)
	}
	if result.Started || !result.Missing {
		t.Fatalf("期望 Started=false / Missing=true，实际 %+v", result)
	}
}

// 记录的位置文件被移动/删除：同样返回 Missing，让用户重新选一次。
func TestLaunch_FileGoneIsMissing(t *testing.T) {
	s := newTestGameService(t)
	ctx := context.Background()
	seedLibraryWork(t, s, "RJ01234567")
	exe := writeTempExe(t)

	// 直接写库（绕过 SetPath 的存在性校验），模拟"记录还在、文件已经没了"
	if err := s.DB.SetGamePath(ctx, database.SourceDLsite, "RJ01234567", exe); err != nil {
		t.Fatalf("写位置失败: %v", err)
	}
	if err := os.Remove(exe); err != nil {
		t.Fatalf("删除临时 exe 失败: %v", err)
	}

	result, err := s.Launch(ctx, "RJ01234567", "")
	if err != nil {
		t.Fatalf("文件不存在不该当启动错误: %v", err)
	}
	if result.Started || !result.Missing {
		t.Fatalf("期望 Started=false / Missing=true，实际 %+v", result)
	}
	if result.Message == "" {
		t.Fatal("Missing 时应该给一句说明，前端要显示给用户")
	}
}

// textOf 把 sqlite 返回的 TEXT 列转成字符串（测试里够用）。
func textOf(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// Status：还没设置位置 → Path/Exists/Running 全 false（前端据此允许第一次点启动去选程序）。
func TestStatus_NoPath(t *testing.T) {
	s := newTestGameService(t)
	seedLibraryWork(t, s, "RJ01234567")

	status, err := s.Status(context.Background(), "RJ01234567", "")
	if err != nil {
		t.Fatalf("没有位置不该报错: %v", err)
	}
	if status.Path != "" || status.Exists || status.Running {
		t.Fatalf("没有位置时状态应全 false，实际 %+v", status)
	}
}

// Status：exe 存在但没运行 → Exists=true / Running=false。
func TestStatus_ExistingNotRunning(t *testing.T) {
	s := newTestGameService(t)
	ctx := context.Background()
	seedLibraryWork(t, s, "RJ01234567")
	exe := writeTempExe(t)
	if err := s.DB.SetGamePath(ctx, database.SourceDLsite, "RJ01234567", exe); err != nil {
		t.Fatalf("写位置失败: %v", err)
	}

	status, err := s.Status(ctx, "RJ01234567", "")
	if err != nil {
		t.Fatalf("读状态失败: %v", err)
	}
	if !status.Exists {
		t.Fatalf("文件存在，Exists 应为 true: %+v", status)
	}
	if status.Running {
		t.Fatalf("测试用的假 exe 不可能在运行: %+v", status)
	}
}

// Status：exe 被删除 → Exists=false，且缓存里残留的路径必须被移除（用户要求）。
func TestStatus_FileGoneClearsRunningCache(t *testing.T) {
	s := newTestGameService(t)
	ctx := context.Background()
	seedLibraryWork(t, s, "RJ01234567")
	exe := writeTempExe(t)
	if err := s.DB.SetGamePath(ctx, database.SourceDLsite, "RJ01234567", exe); err != nil {
		t.Fatalf("写位置失败: %v", err)
	}
	s.rememberRunning(exe, 4242) // 模拟"上次查询时还在运行"

	if err := os.Remove(exe); err != nil {
		t.Fatalf("删除临时 exe 失败: %v", err)
	}

	status, err := s.Status(ctx, "RJ01234567", "")
	if err != nil {
		t.Fatalf("读状态失败: %v", err)
	}
	if status.Exists || status.Running {
		t.Fatalf("文件已删除，应 Exists=false / Running=false: %+v", status)
	}
	if _, ok := s.cachedRunning(normalizePathKey(exe)); ok {
		t.Fatal("文件不存在时，运行缓存里的路径应被移除")
	}
}

// 运行缓存的读写/移除：路径大小写不敏感，重复写覆盖同一键。
func TestRunningCache(t *testing.T) {
	s := newTestGameService(t)
	path := `D:\Games\RJ01234567\Game.EXE`

	s.rememberRunning(path, 100)
	pid, ok := s.cachedRunning(normalizePathKey(`d:\games\rj01234567\game.exe`))
	if !ok || pid != 100 {
		t.Fatalf("缓存应命中同一路径（大小写不敏感），实际 pid=%d ok=%v", pid, ok)
	}

	s.rememberRunning(path, 200)
	if pid, _ := s.cachedRunning(normalizePathKey(path)); pid != 200 {
		t.Fatalf("重复写应覆盖 PID，实际 %d", pid)
	}

	s.forgetRunning(path)
	if _, ok := s.cachedRunning(normalizePathKey(path)); ok {
		t.Fatal("移除后缓存不该还有这条路径")
	}
}
