// internal/service/hero_image_test.go
//
// hero 大图缓存的两条底线（2026-11-27 用户要求）：
//  1. 本地没有缓存时才联网，取到的图落进 covers/ 并把目录/相对路径/绝对路径写进库；
//  2. 有缓存之后**完全不联网**（换一个无效 URL 也照样给图）。
// 顺带守住：模糊图直接复用这张 hero 缓存；移除作品时两张图一起删。
package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
)

// tinyJPEG 生成一张可解码的小图（HeroBlur 要真的解码它，不能用假字节）。
func tinyJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 16), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("生成测试图失败: %v", err)
	}
	return buf.Bytes()
}

// tinyWebP 是一张 1x1 的合法 webp（用来验证 webp 解码链路，不手拼字节）。
func tinyWebP(t *testing.T) []byte {
	t.Helper()
	const encoded = "UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAwA0JaQAA3AA/vuUAAA="
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("解码测试 webp 失败: %v", err)
	}
	return data
}

func seedHeroWork(t *testing.T, db *database.SQLiteService, workID string) {
	t.Helper()
	if err := db.UpsertLibraryWork(context.Background(), &model.LibraryWork{
		Source:  database.SourceDLsite,
		WorkID:  workID,
		Name:    "测试作品",
		RawJSON: `{"id":"` + workID + `","name":"测试作品"}`,
	}); err != nil {
		t.Fatalf("准备作品失败: %v", err)
	}
}

// 第一次联网取图并落盘 + 记库；第二次（哪怕给一个坏 URL）直接读缓存，请求数不再增长。
func TestHeroImage_DownloadOnceThenServeCache(t *testing.T) {
	t.Chdir(t.TempDir())

	jpg := tinyJPEG(t)
	requests := 0
	webp := tinyWebP(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch {
		case strings.HasSuffix(r.URL.Path, ".webp"):
			w.Header().Set("Content-Type", "image/webp")
			_, _ = w.Write(webp)
		default:
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(jpg)
		}
	}))
	defer server.Close()

	db := newTestDB(t)
	svc := &LibraryService{DB: db}
	ctx := context.Background()
	seedHeroWork(t, db, "RJ01234567")

	url := server.URL + "/hero.jpg"
	first, err := svc.HeroImage(ctx, "RJ01234567", database.SourceDLsite, url)
	if err != nil {
		t.Fatalf("第一次取 hero 失败: %v", err)
	}
	// 2026-11-27 用户要求「所有图片尽量改为 webp」：源站有同名 webp 就用它
	if !strings.HasPrefix(first, "data:image/webp;base64,") {
		t.Fatalf("应优先返回源站 webp 的 data URL，实际前缀 %q", dataURLPrefix(first))
	}
	if requests != 1 {
		t.Fatalf("第一次应只下载一次（.webp 探测即命中），实际 %d 次", requests)
	}

	row, err := db.FindLibraryWork(ctx, database.SourceDLsite, "RJ01234567")
	if err != nil || row == nil {
		t.Fatalf("读作品失败: %v", err)
	}
	if row.HeroDir != "covers" || !strings.HasPrefix(row.HeroPath, "covers/") ||
		!strings.HasSuffix(row.HeroAbsPath, filepath.Base(row.HeroPath)) ||
		row.HeroSuffix != ".webp" || row.HeroBytes != int64(len(webp)) || row.HeroURL != url {
		t.Fatalf("hero 落地信息没写全: %+v", row)
	}
	if _, err := os.Stat(row.HeroAbsPath); err != nil {
		t.Fatalf("hero 文件应已落盘: %v", err)
	}

	// 第二次：本地已有缓存 → 不给 URL 也照样返回，且不联网
	second, err := svc.HeroImage(ctx, "RJ01234567", database.SourceDLsite, server.URL+"/another.jpg")
	if err != nil {
		t.Fatalf("第二次取 hero 失败: %v", err)
	}
	if second != first {
		t.Fatal("缓存命中应与第一次返回同一张图")
	}
	if requests != 1 {
		t.Fatalf("缓存命中不该联网，实际请求 %d 次", requests)
	}
}

// 源站没有 webp 档位时回落原图（探测失败不能拖垮整张图）。
func TestHeroImage_FallsBackToOriginalWhenNoWebP(t *testing.T) {
	t.Chdir(t.TempDir())

	jpg := tinyJPEG(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if strings.HasSuffix(r.URL.Path, ".webp") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpg)
	}))
	defer server.Close()

	db := newTestDB(t)
	svc := &LibraryService{DB: db}
	seedHeroWork(t, db, "RJ01234567")

	got, err := svc.HeroImage(context.Background(), "RJ01234567", database.SourceDLsite, server.URL+"/only.jpg")
	if err != nil || got == "" {
		t.Fatalf("没有 webp 时应回落原图: got=%q err=%v", dataURLPrefix(got), err)
	}
	if requests != 2 {
		t.Fatalf("应先探 webp 再下原图（共 2 次请求），实际 %d 次", requests)
	}
	row, _ := db.FindLibraryWork(context.Background(), database.SourceDLsite, "RJ01234567")
	if row == nil || row.HeroSuffix != ".jpg" {
		t.Fatalf("回落后应存 jpg: %+v", row)
	}
}

// 没有缓存、也没有可用 URL：返回空串 + nil（页面退成渐变底，不是错误）。
func TestHeroImage_NoSourceReturnsEmpty(t *testing.T) {
	t.Chdir(t.TempDir())
	db := newTestDB(t)
	svc := &LibraryService{DB: db}

	got, err := svc.HeroImage(context.Background(), "RJ09999999", database.SourceDLsite, "")
	if err != nil {
		t.Fatalf("没有 hero 不该报错: %v", err)
	}
	if got != "" {
		t.Fatalf("没有源时应返回空串，实际 %q", got)
	}
}

// 模糊图直接用 covers/ 里的 hero 缓存（不再单独下载），盘上只留一张 v4 文件。
func TestHeroBlur_UsesCachedHero(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("covers", 0o755); err != nil {
		t.Fatalf("建封面目录失败: %v", err)
	}
	heroPath := filepath.Join("covers", "hero_dlsite_RJ01234567.jpg")
	if err := os.WriteFile(heroPath, tinyJPEG(t), 0o644); err != nil {
		t.Fatalf("写 hero 缓存失败: %v", err)
	}

	svc := &LibraryService{DB: newTestDB(t)}
	blur, err := svc.HeroBlur(context.Background(), "RJ01234567", database.SourceDLsite, "")
	if err != nil {
		t.Fatalf("生成模糊图失败: %v", err)
	}
	if !strings.HasPrefix(blur, "data:image/jpeg;base64,") {
		t.Fatalf("模糊图应是 JPEG data URL，实际前缀 %q", dataURLPrefix(blur))
	}

	matches, err := filepath.Glob(filepath.Join("covers", "hero_blur_v4_*.jpg"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("盘上应只留一张 v4 模糊图: %v %v", matches, err)
	}
}

// webp 的 hero 缓存也要能生成模糊图（2026-11-27 「所有图片尽量改为 webp」后，
// hero 缓存可能是 webp；模糊图只引官方 x/image 的解码器，不引编码库）。
func TestHeroBlur_DecodesWebPHero(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("covers", 0o755); err != nil {
		t.Fatalf("建封面目录失败: %v", err)
	}
	heroPath := filepath.Join("covers", "hero_dlsite_RJ01234567.webp")
	if err := os.WriteFile(heroPath, tinyWebP(t), 0o644); err != nil {
		t.Fatalf("写 webp hero 缓存失败: %v", err)
	}

	svc := &LibraryService{DB: newTestDB(t)}
	blur, err := svc.HeroBlur(context.Background(), "RJ01234567", database.SourceDLsite, "")
	if err != nil {
		t.Fatalf("webp 源生成模糊图失败: %v", err)
	}
	if !strings.HasPrefix(blur, "data:image/jpeg;base64,") {
		t.Fatalf("模糊图应是 JPEG data URL，实际前缀 %q", dataURLPrefix(blur))
	}
}

// 从游戏库移除作品时，hero 原图与模糊图都要删掉（与封面同一口径）。
func TestRemoveDeletesHeroCache(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("covers", 0o755); err != nil {
		t.Fatalf("建封面目录失败: %v", err)
	}
	heroPath := filepath.Join("covers", "hero_dlsite_RJ01234567.jpg")
	blurPath := filepath.Join("covers", "hero_blur_v4_dlsite_RJ01234567_12345678.jpg")
	for _, path := range []string{heroPath, blurPath} {
		if err := os.WriteFile(path, []byte("jpg"), 0o644); err != nil {
			t.Fatalf("写缓存文件失败: %v", err)
		}
	}

	db := newTestDB(t)
	svc := &LibraryService{DB: db}
	seedHeroWork(t, db, "RJ01234567")

	if err := svc.Remove(context.Background(), "RJ01234567", database.SourceDLsite); err != nil {
		t.Fatalf("移除作品失败: %v", err)
	}
	for _, path := range []string{heroPath, blurPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("缓存文件应被删除: %s (err=%v)", path, err)
		}
	}
}

// dataURLPrefix 取前 40 个字符，失败信息里不至于把整张 base64 打出来。
func dataURLPrefix(value string) string {
	if len(value) <= 40 {
		return value
	}
	return value[:40]
}
