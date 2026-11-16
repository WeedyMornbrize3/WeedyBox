// internal/service/getMeta/cover_store_test.go
//
// 封面存储相关的两条底线：
//   - 旧目录 preImg/ 能自动迁移成 covers/（目录 + 数据库路径一起改，幂等）；
//   - 下载时优先探测同名 .webp，服务器没有就回落到原图。
package getmeta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
	"github.com/wailsapp/wails/v3/pkg/services/sqlite"
)

func TestWebPVariantURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://img.dlsite.jp/a_img_main.jpg", "https://img.dlsite.jp/a_img_main.webp"},
		{"https://img.dlsite.jp/a_img_main.JPG", "https://img.dlsite.jp/a_img_main.webp"},
		{"https://img.dlsite.jp/a_img_main.png?v=2", "https://img.dlsite.jp/a_img_main.webp?v=2"},
		{"https://img.dlsite.jp/a_img_main.webp", ""},
		{"data:image/gif;base64,R0lGOD", ""},
		{"https://img.dlsite.jp/noext", ""},
	}
	for _, tc := range cases {
		if got := webpVariantURL(tc.in); got != tc.want {
			t.Errorf("webpVariantURL(%q) = %q，期望 %q", tc.in, got, tc.want)
		}
	}
}

// newTestDB 起一个真实临时库（游戏库表会在第一次写入时惰性创建）。
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

// 迁移：preImg/ 改名成 covers/，库里 cover_* 与下载记录的路径一起替换；重复执行不出错。
func TestMigrateLegacyCoverDir(t *testing.T) {
	t.Chdir(t.TempDir()) // coversDir/preImg 是相对工作目录的，挪到临时目录里测

	if err := os.MkdirAll(legacyCoverDir, 0o755); err != nil {
		t.Fatalf("建旧目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyCoverDir, "RJ01234567_cover.jpg"), []byte("old"), 0o644); err != nil {
		t.Fatalf("写旧封面失败: %v", err)
	}
	legacyAbs, _ := filepath.Abs(filepath.Join(legacyCoverDir, "RJ01234567_cover.jpg"))

	db := newTestDB(t)
	ctx := context.Background()
	work := &model.LibraryWork{Source: database.SourceDLsite, WorkID: "RJ01234567", Name: "测试作品", RawJSON: `{"id":"RJ01234567"}`}
	if err := db.UpsertLibraryWork(ctx, work); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}
	if err := db.RecordCoverFile(ctx, &model.CoverFile{
		Source:  database.SourceDLsite,
		WorkID:  "RJ01234567",
		Dir:     legacyCoverDir,
		Path:    legacyCoverDir + "/RJ01234567_cover.jpg",
		AbsPath: legacyAbs,
		Bytes:   3,
		Suffix:  ".jpg",
	}); err != nil {
		t.Fatalf("准备封面记录失败: %v", err)
	}

	if err := MigrateLegacyCoverDir(ctx, db); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	if _, err := os.Stat(legacyCoverDir); !os.IsNotExist(err) {
		t.Fatalf("旧目录应该已被移走，实际 err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(coversDir, "RJ01234567_cover.jpg")); err != nil {
		t.Fatalf("文件应在新目录 %s 下: %v", coversDir, err)
	}

	got, err := db.FindLibraryWork(ctx, database.SourceDLsite, "RJ01234567")
	if err != nil || got == nil {
		t.Fatalf("读作品失败: %v", err)
	}
	if got.CoverDir != coversDir || strings.Contains(got.CoverPath, legacyCoverDir) || strings.Contains(got.CoverAbsPath, legacyCoverDir) {
		t.Fatalf("作品封面路径没迁移: dir=%q path=%q abs=%q", got.CoverDir, got.CoverPath, got.CoverAbsPath)
	}
	files, err := db.ListCoverFiles(ctx, database.SourceDLsite, "RJ01234567")
	if err != nil || len(files) != 1 {
		t.Fatalf("下载记录应保留一条: files=%+v err=%v", files, err)
	}
	if files[0].Dir != coversDir || strings.Contains(files[0].Path, legacyCoverDir) || strings.Contains(files[0].AbsPath, legacyCoverDir) {
		t.Fatalf("下载记录路径没迁移: %+v", files[0])
	}

	// 幂等：再跑一次不报错、不重复搬
	if err := MigrateLegacyCoverDir(ctx, db); err != nil {
		t.Fatalf("重复迁移不该报错: %v", err)
	}
}

// 下载优先 webp：同名 .webp 能用就用；没有就回落到原图（原图只下载一次，带重试）。
func TestFetchImagePreferringWebP(t *testing.T) {
	webpBytes := append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("WEBP")...)
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 'j', 'p', 'g'}

	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		switch r.URL.Path {
		case "/with-webp.webp":
			_, _ = w.Write(webpBytes)
		case "/with-webp.jpg", "/only-jpg.jpg":
			_, _ = w.Write(jpegBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	svc := &MetaApiService{}
	if _, ext, err := svc.fetchImagePreferringWebP(context.Background(), server.URL+"/with-webp.jpg"); err != nil || ext != ".webp" {
		t.Fatalf("有 webp 时应选 webp：ext=%q err=%v", ext, err)
	}
	requested = nil
	if _, ext, err := svc.fetchImagePreferringWebP(context.Background(), server.URL+"/only-jpg.jpg"); err != nil || ext != ".jpg" {
		t.Fatalf("没有 webp 时应回落原图：ext=%q err=%v", ext, err)
	}
	// 回落时应该先探过一次 webp，再下原图
	if len(requested) != 2 || requested[0] != "/only-jpg.webp" || requested[1] != "/only-jpg.jpg" {
		t.Fatalf("请求顺序不对: %v", requested)
	}
}
