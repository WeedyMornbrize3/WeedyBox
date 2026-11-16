// internal/service/getMeta/cover_store.go
//
// 封面在本地怎么存：目录名、旧目录迁移、以及「优先拿 WebP 版本」的下载策略。
//
// 目录从 preImg 改名为 covers（2026-11-16 用户要求）。旧版本装过的用户机器上
// 还留着 preImg/ 且数据库里记的是 preImg 路径，所以启动时做一次幂等迁移：
// 把目录改名（或合并）并把库里的路径一起替换。
//
// WebP：**不引入图像编码库**（用户要求「不需要额外导入库」）。DLsite 的封面
// 同一张图通常有同名 .webp 档位（实测 ..._img_main.jpg 旁就有 ..._img_main.webp，
// 体积小一个数量级），所以下载时先探测这个候选；服务器上没有 / 拿到的不是 webp
// 就静默回落到原来的 URL。全部走现成的 http 客户端，零额外依赖。
package getmeta

import (
	"context"
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"WeedyBox/internal/database"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// coversDir 是入库封面的本地目录（相对工作目录，与 cacheDir 一样）。
	coversDir = "covers"
	// legacyCoverDir 是改名前的旧目录，只用于启动时迁移。
	legacyCoverDir = "preImg"
)

// ServiceStartup 让 MetaApiService 也参与 Wails 的服务启动：
// 数据库服务在它之前注册（main.go 的顺序），所以这里 DB 已经打开，
// 可以安全地做「旧封面目录 + 旧数据库路径」的一次性迁移。
//
// 迁移失败只打日志、不阻断应用启动：最坏情况是旧封面图暂时显示不出来，
// 但新入库的封面会用新的 covers/ 正常保存。
func (s *MetaApiService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	if err := MigrateLegacyCoverDir(ctx, s.DB); err != nil {
		fmt.Printf("迁移旧封面目录失败（应用继续启动）: %v\n", err)
	}
	return nil
}

// MigrateLegacyCoverDir 把旧版 preImg/ 目录改名成 covers/，并把库里记录的旧路径同步替换。
//
// 幂等：没有 preImg/ 时什么都不做；重复执行也只是每条 REPLACE 一次。
// DB 可以是 nil（调试/测试只迁移目录）；表不存在时跳过对应的 SQL
// —— 游戏库表是惰性创建的，没入过库的机器上本来就没有这些表。
func MigrateLegacyCoverDir(ctx context.Context, db *database.SQLiteService) error {
	if err := migrateCoverDirectory(); err != nil {
		return err
	}
	if db == nil {
		return nil
	}

	// 先问清楚哪些表存在，别对着不存在的表执行 UPDATE。
	rows, err := db.Query(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name IN ('library_works', 'library_cover_files')`)
	if err != nil {
		return fmt.Errorf("检查封面相关表失败: %w", err)
	}

	for _, row := range rows {
		table := fmt.Sprint(row["name"])
		switch table {
		case "library_works", "library_cover_files":
		default:
			continue
		}
		legacy, err := tableHasLegacyCoverPath(ctx, db, table)
		if err != nil {
			return err
		}
		if !legacy {
			continue // 已经迁移过，不重复写整张表
		}
		if err := db.Execute(ctx, migrateCoverPathSQL(table)); err != nil {
			return fmt.Errorf("迁移 %s 的封面路径失败: %w", table, err)
		}
	}
	return nil
}

// tableHasLegacyCoverPath 检查这张表里是否还有 preImg 字样（幂等迁移用：
// 没有旧路径就跳过 UPDATE，避免每次启动都把所有行重写一遍）。
func tableHasLegacyCoverPath(ctx context.Context, db *database.SQLiteService, table string) (bool, error) {
	var query string
	if table == "library_cover_files" {
		query = `SELECT 1 FROM library_cover_files
			WHERE instr(dir, 'preImg') > 0 OR instr(path, 'preImg') > 0 OR instr(abs_path, 'preImg') > 0
			LIMIT 1`
	} else {
		query = `SELECT 1 FROM library_works
			WHERE instr(cover_dir, 'preImg') > 0 OR instr(cover_path, 'preImg') > 0 OR instr(cover_abs_path, 'preImg') > 0
			LIMIT 1`
	}
	rows, err := db.Query(ctx, query)
	if err != nil {
		return false, fmt.Errorf("检查 %s 的旧封面路径失败: %w", table, err)
	}
	return len(rows) > 0, nil
}

// migrateCoverPathSQL 生成把三个路径列里的 preImg 换成 covers 的 UPDATE。
// 列名各表一致（cover_dir/cover_path/cover_abs_path 与 dir/path/abs_path 不同，
// 所以按表名拼两份）。
func migrateCoverPathSQL(table string) string {
	if table == "library_cover_files" {
		return `UPDATE library_cover_files SET
			dir = REPLACE(dir, 'preImg', 'covers'),
			path = REPLACE(path, 'preImg', 'covers'),
			abs_path = REPLACE(abs_path, 'preImg', 'covers')`
	}
	return `UPDATE library_works SET
		cover_dir = REPLACE(cover_dir, 'preImg', 'covers'),
		cover_path = REPLACE(cover_path, 'preImg', 'covers'),
		cover_abs_path = REPLACE(cover_abs_path, 'preImg', 'covers')`
}

// migrateCoverDirectory 负责文件系统这一半：preImg/ → covers/。
//
// 如果 covers/ 已经存在（上一次迁移只做了一半 / 用户手动建过），就逐个文件合并，
// 目标同名文件保留 covers/ 里的那份，最后把空的 preImg/ 删掉。
func migrateCoverDirectory() error {
	info, err := os.Stat(legacyCoverDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 已经迁移过 / 从来没有下载过封面
		}
		return fmt.Errorf("检查旧封面目录失败: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	if _, err := os.Stat(coversDir); os.IsNotExist(err) {
		if err := os.Rename(legacyCoverDir, coversDir); err != nil {
			return fmt.Errorf("把旧封面目录改为 %s 失败: %w", coversDir, err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("检查封面目录 %s 失败: %w", coversDir, err)
	}

	entries, err := os.ReadDir(legacyCoverDir)
	if err != nil {
		return fmt.Errorf("读取旧封面目录失败: %w", err)
	}
	for _, entry := range entries {
		src := filepath.Join(legacyCoverDir, entry.Name())
		dst := filepath.Join(coversDir, entry.Name())
		if _, err := os.Stat(dst); err == nil {
			continue // 新目录已经有同名文件，保留新的
		}
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("迁移封面文件 %s 失败: %w", src, err)
		}
	}
	if remaining, _ := os.ReadDir(legacyCoverDir); len(remaining) == 0 {
		_ = os.Remove(legacyCoverDir)
	}
	return nil
}

// fetchImagePreferringWebP 下载一张图：优先探测同名 .webp 版本，拿不到再下原图。
//
// 「探测」只发一次请求（不重试），避免服务器没有 webp 时白白多等两轮；
// 探测失败/返回的不是 webp 都静默回落到原 URL 的正常下载（带重试）。
func (s *MetaApiService) fetchImagePreferringWebP(ctx context.Context, rawURL string) ([]byte, string, error) {
	if candidate := webpVariantURL(rawURL); candidate != "" {
		client := s.ensure().client
		if data, ext, err := s.tryFetchImage(ctx, client, candidate); err == nil && strings.EqualFold(ext, ".webp") {
			return data, ".webp", nil
		}
	}
	return s.fetchImage(ctx, rawURL)
}

// webpVariantURL 由图片 URL 推出同名的 .webp 候选；推不出来返回空串。
//
// 只把扩展名 jpg/jpeg/png 换成 webp（DLsite 的档位就是同名的），
// 查询串原样保留；已经是 webp、data URL 或没有可换的扩展名时不探测。
func webpVariantURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.HasPrefix(rawURL, "data:") {
		return ""
	}

	cut := len(rawURL)
	if i := strings.IndexAny(rawURL, "?#"); i >= 0 {
		cut = i
	}
	base, suffix := rawURL[:cut], rawURL[cut:]

	ext := pathpkg.Ext(base)
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png":
		return strings.TrimSuffix(base, ext) + ".webp" + suffix
	}
	return ""
}
