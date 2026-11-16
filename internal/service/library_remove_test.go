// internal/service/library_remove_test.go
//
// 从游戏库删除作品时必须把入库时保存的封面图片一起删掉（用户 2026-11-16 要求）。
// 用真实临时目录 + 真实临时库，不 mock 文件系统。
package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
	"github.com/wailsapp/wails/v3/pkg/services/sqlite"
)

func newTestDB(t *testing.T) *database.SQLiteService {
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
	return &database.SQLiteService{DB: raw}
}

// Remove 要删掉这个作品的所有封面文件（含重名 -2/-3 的多份），再删库记录。
func TestRemoveDeletesCoverFiles(t *testing.T) {
	db := newTestDB(t)
	svc := &LibraryService{DB: db}
	ctx := context.Background()

	const workID = "RJ01234567"
	if err := db.UpsertLibraryWork(ctx, &model.LibraryWork{
		Source:  database.SourceDLsite,
		WorkID:  workID,
		Name:    "测试作品",
		RawJSON: `{"id":"RJ01234567"}`,
	}); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}

	dir := t.TempDir()
	paths := []string{
		filepath.Join(dir, "RJ01234567_cover.webp"),
		filepath.Join(dir, "RJ01234567_cover-2.webp"),
	}
	for i, path := range paths {
		if err := os.WriteFile(path, []byte("webp"), 0o644); err != nil {
			t.Fatalf("写封面文件失败: %v", err)
		}
		if err := db.RecordCoverFile(ctx, &model.CoverFile{
			Source:  database.SourceDLsite,
			WorkID:  workID,
			Dir:     "covers",
			Path:    "covers/" + filepath.Base(path),
			AbsPath: path,
			Bytes:   int64(4 + i),
			Suffix:  ".webp",
		}); err != nil {
			t.Fatalf("记录封面失败: %v", err)
		}
	}

	if err := svc.Remove(ctx, workID, database.SourceDLsite); err != nil {
		t.Fatalf("删除作品失败: %v", err)
	}

	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("封面文件应被删除: %s (err=%v)", path, err)
		}
	}
	if got, err := db.FindLibraryWork(ctx, database.SourceDLsite, workID); err != nil || got != nil {
		t.Errorf("库记录应已删除: got=%+v err=%v", got, err)
	}
	if files, err := db.ListCoverFiles(ctx, database.SourceDLsite, workID); err != nil || len(files) != 0 {
		t.Errorf("封面记录应已删除: files=%+v err=%v", files, err)
	}
}

// 文件已经被手动删掉时，Remove 仍应成功（os.IsNotExist 不算错误）。
func TestRemoveToleratesMissingCoverFile(t *testing.T) {
	db := newTestDB(t)
	svc := &LibraryService{DB: db}
	ctx := context.Background()

	const workID = "RJ01234567"
	if err := db.UpsertLibraryWork(ctx, &model.LibraryWork{
		Source:  database.SourceDLsite,
		WorkID:  workID,
		Name:    "测试作品",
		RawJSON: `{"id":"RJ01234567"}`,
	}); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}
	missing := filepath.Join(t.TempDir(), "gone.webp")
	if err := db.RecordCoverFile(ctx, &model.CoverFile{
		Source:  database.SourceDLsite,
		WorkID:  workID,
		Dir:     "covers",
		Path:    "covers/gone.webp",
		AbsPath: missing,
		Suffix:  ".webp",
	}); err != nil {
		t.Fatalf("记录封面失败: %v", err)
	}

	if err := svc.Remove(ctx, workID, database.SourceDLsite); err != nil {
		t.Fatalf("文件已不存在时删除仍应成功: %v", err)
	}
	if got, _ := db.FindLibraryWork(ctx, database.SourceDLsite, workID); got != nil {
		t.Errorf("库记录应已删除: %+v", got)
	}
}
