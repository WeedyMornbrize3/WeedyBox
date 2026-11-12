// internal/database/library_repo.go
//
// 游戏库（元数据抓取成果）的持久化层。
//
// 两条设计约束，都来自"以后不只抓 DLsite"：
//
//	1. 表是**来源无关**的：作品信息（id/名字/社团/…）放公共列，具体来源只体现在
//	   `source` 列的值上（当前只有 'dlsite'）。所以加新来源不需要加表，也不需要
//	   在库里存 "dlsite 特有字段" 的专属列 —— 原始数据整份进 raw_json。
//	2. 表是**惰性创建**的：这里每次操作前都 ensureLibraryTables()，而启动时的
//	   initAllTables 不建这两张表。没人往库里存东西时，数据库里就不存在这两张表
//	   （省掉空表占的 page）。代价是每次读写多一条 CREATE TABLE IF NOT EXISTS，
//	   幂等且极快，也顺手兼容"库文件是旧版本建的、还没有这些表"的情况。
//
// 这里只做「表 ↔ 结构体」的搬运，不含解析或格式化逻辑：
// 抓取在 internal/service/getMeta，编排在 internal/service/library_service.go。
package database

import (
	"context"
	"WeedyBox/internal/model"
	"fmt"
	"strings"
	"time"
)

// SourceDLsite 是当前唯一的元数据来源标识。
// 加新来源时在这里补常量，入库时把 source 传进来即可。
const SourceDLsite = "dlsite"

// workColumns 列顺序固定在这里，读写共用，免得多处手写漏列。
const workColumns = `source, work_id, name, maker, maker_id, series, series_id,
	work_type, age_rating, release_date, cover_url, source_url, raw_json, fetched_at,
	cover_dir, cover_path, cover_abs_path, cover_bytes, cover_suffix, created_at, updated_at`

// normalizeSource 统一来源标识：空值一律当 dlsite（历史数据/老调用都只可能是它）。
func normalizeSource(source string) string {
	s := strings.ToLower(strings.TrimSpace(source))
	if s == "" {
		return SourceDLsite
	}
	return s
}

// normalizeWorkID 统一编号：全大写、去空格（RJ01721474）。
func normalizeWorkID(workID string) string {
	return strings.ToUpper(strings.TrimSpace(workID))
}

// ensureLibraryTables 惰性建表（幂等）。
// 每个公开方法入口都要调一次 —— 这是"启动时少建两张空表"换来的代价。
func (s *SQLiteService) ensureLibraryTables(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}

	queries := []string{
		// 作品表：一条 = 一个来源下的一个作品（UNIQUE(source, work_id)）。
		// 公共字段（名称/社团/…)单独成列，方便列表页直接排序与筛选；
		// raw_json 存整份原始 json，来源特有的字段全在里面，不丢也不占公共列。
		`CREATE TABLE IF NOT EXISTS library_works (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL DEFAULT 'dlsite',
			work_id TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			maker TEXT DEFAULT '',
			maker_id TEXT DEFAULT '',
			series TEXT DEFAULT '',
			series_id TEXT DEFAULT '',
			work_type TEXT DEFAULT '',
			age_rating TEXT DEFAULT '',
			release_date TEXT DEFAULT '',
			cover_url TEXT DEFAULT '',
			source_url TEXT DEFAULT '',
			raw_json TEXT NOT NULL DEFAULT '{}',
			fetched_at TEXT DEFAULT '',
			cover_dir TEXT DEFAULT '',
			cover_path TEXT DEFAULT '',
			cover_abs_path TEXT DEFAULT '',
			cover_bytes INTEGER DEFAULT 0,
			cover_suffix TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source, work_id)
		)`,

		// 封面下载记录：每次「下载封面」写一行。
		// 下载实现刻意不覆盖同名文件（重名加 -2 / -3 后缀），所以同一作品可能有多行；
		// 作品表里的 cover_* 只指向最新那一行。
		`CREATE TABLE IF NOT EXISTS library_cover_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL DEFAULT 'dlsite',
			work_id TEXT NOT NULL,
			dir TEXT DEFAULT '',
			path TEXT DEFAULT '',
			abs_path TEXT DEFAULT '',
			bytes INTEGER DEFAULT 0,
			suffix TEXT DEFAULT '',
			cover_url TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_library_cover_files_work
			ON library_cover_files (source, work_id)`,
	}

	for _, query := range queries {
		if err := s.Execute(ctx, query); err != nil {
			return fmt.Errorf("初始化游戏库表失败: %w", err)
		}
	}
	return nil
}

// FindLibraryWork 按（来源, 编号）查一条。
// **查不到返回 (nil, nil)** —— "没有缓存"是正常分支而非错误，调用方据此决定要不要联网抓。
func (s *SQLiteService) FindLibraryWork(ctx context.Context, source, workID string) (*model.LibraryWork, error) {
	if s == nil || s.DB == nil {
		return nil, nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return nil, nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return nil, err
	}

	row, err := s.QueryRow(ctx,
		`SELECT `+workColumns+` FROM library_works WHERE source = ? AND work_id = ?`,
		normalizeSource(source), workID)
	if err != nil {
		// QueryRow 查不到时返回"没有找到记录"，按无缓存处理
		if strings.Contains(err.Error(), "没有找到记录") {
			return nil, nil
		}
		return nil, fmt.Errorf("查询作品缓存失败: %w", err)
	}
	work := rowToLibraryWork(row)
	return &work, nil
}

// ListLibraryWorks 列出游戏库作品，最近入库的在前。
// source 传空串表示"所有来源"。
func (s *SQLiteService) ListLibraryWorks(ctx context.Context, source string) ([]model.LibraryWork, error) {
	if s == nil || s.DB == nil {
		return []model.LibraryWork{}, nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return nil, err
	}

	query := `SELECT ` + workColumns + ` FROM library_works ORDER BY created_at DESC, id DESC`
	args := []interface{}{}
	if strings.TrimSpace(source) != "" {
		query = `SELECT ` + workColumns + ` FROM library_works WHERE source = ? ORDER BY created_at DESC, id DESC`
		args = append(args, normalizeSource(source))
	}

	rows, err := s.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询游戏库失败: %w", err)
	}

	works := make([]model.LibraryWork, 0, len(rows))
	for _, row := range rows {
		works = append(works, rowToLibraryWork(row))
	}
	return works, nil
}

// UpsertLibraryWork 写入或更新一条作品。
//
// 用 ON CONFLICT 而不是先查后写：抓取是幂等的，重复抓同一部作品就是"刷新它的缓存"，
// 而且这样不会因为并发/重试插出两行（UNIQUE(source, work_id) 兜底）。
// **不覆盖 cover_* 列**：封面是独立下载的，刷新元数据不该把已下好的图丢掉。
func (s *SQLiteService) UpsertLibraryWork(ctx context.Context, w *model.LibraryWork) error {
	if s == nil || s.DB == nil || w == nil {
		return nil
	}
	workID := normalizeWorkID(w.WorkID)
	if workID == "" {
		return fmt.Errorf("work_id 为空，无法入库")
	}
	source := normalizeSource(w.Source)
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}

	query := `INSERT INTO library_works
		(source, work_id, name, maker, maker_id, series, series_id, work_type, age_rating,
		 release_date, cover_url, source_url, raw_json, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source, work_id) DO UPDATE SET
			name = excluded.name,
			maker = excluded.maker,
			maker_id = excluded.maker_id,
			series = excluded.series,
			series_id = excluded.series_id,
			work_type = excluded.work_type,
			age_rating = excluded.age_rating,
			release_date = excluded.release_date,
			cover_url = excluded.cover_url,
			source_url = excluded.source_url,
			raw_json = excluded.raw_json,
			fetched_at = excluded.fetched_at,
			updated_at = CURRENT_TIMESTAMP`

	if err := s.Execute(ctx, query,
		source, workID, w.Name, w.Maker, w.MakerID, w.Series, w.SeriesID,
		w.WorkType, w.AgeRating, w.ReleaseDate, w.CoverURL, w.SourceURL,
		w.RawJSON, w.FetchedAt,
	); err != nil {
		return fmt.Errorf("写入作品失败: %w", err)
	}
	return nil
}

// DeleteLibraryWork 把一条作品连同它的封面下载记录一起删掉。
// 磁盘上的图片**不删** —— 那是用户自己的文件，库里删条记录不该顺手删文件。
func (s *SQLiteService) DeleteLibraryWork(ctx context.Context, source, workID string) error {
	if s == nil || s.DB == nil {
		return nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return fmt.Errorf("work_id 为空")
	}
	source = normalizeSource(source)
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}

	if err := s.Execute(ctx,
		`DELETE FROM library_cover_files WHERE source = ? AND work_id = ?`, source, workID); err != nil {
		return fmt.Errorf("删除封面记录失败: %w", err)
	}
	if err := s.Execute(ctx,
		`DELETE FROM library_works WHERE source = ? AND work_id = ?`, source, workID); err != nil {
		return fmt.Errorf("删除作品失败: %w", err)
	}
	return nil
}

// RecordCoverFile 记一次封面下载：先插一行下载记录，再把这个作品的
// cover_* 指针更新为最新文件。
//
// 作品不在库里时只插下载记录（UPDATE 影响 0 行）—— 调试页可能只下了张图、
// 没走游戏库的入库流程，这种情况不该凭空造一条作品。
func (s *SQLiteService) RecordCoverFile(ctx context.Context, f *model.CoverFile) error {
	if s == nil || s.DB == nil || f == nil {
		return nil
	}
	workID := normalizeWorkID(f.WorkID)
	if workID == "" {
		return nil
	}
	source := normalizeSource(f.Source)
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}

	insert := `INSERT INTO library_cover_files (source, work_id, dir, path, abs_path, bytes, suffix, cover_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if err := s.Execute(ctx, insert, source, workID, f.Dir, f.Path, f.AbsPath, f.Bytes, f.Suffix, f.CoverURL); err != nil {
		return fmt.Errorf("记录封面文件失败: %w", err)
	}

	update := `UPDATE library_works SET
		cover_dir = ?, cover_path = ?, cover_abs_path = ?, cover_bytes = ?, cover_suffix = ?,
		updated_at = CURRENT_TIMESTAMP
		WHERE source = ? AND work_id = ?`
	if err := s.Execute(ctx, update, f.Dir, f.Path, f.AbsPath, f.Bytes, f.Suffix, source, workID); err != nil {
		return fmt.Errorf("更新作品封面指针失败: %w", err)
	}
	return nil
}

// ===== 行 → 结构体 =====

// rowToLibraryWork 把一行 map 摊成 LibraryWork。
// SQLite 驱动返回的 TEXT 是 string、INTEGER 是 int64、DATETIME 是 time.Time；
// 但可空列可能是 nil，所以逐列做安全取值（直接类型断言会在空列上 panic）。
func rowToLibraryWork(row map[string]interface{}) model.LibraryWork {
	return model.LibraryWork{
		Source:       textOf(row["source"]),
		WorkID:       textOf(row["work_id"]),
		Name:         textOf(row["name"]),
		Maker:        textOf(row["maker"]),
		MakerID:      textOf(row["maker_id"]),
		Series:       textOf(row["series"]),
		SeriesID:     textOf(row["series_id"]),
		WorkType:     textOf(row["work_type"]),
		AgeRating:    textOf(row["age_rating"]),
		ReleaseDate:  textOf(row["release_date"]),
		CoverURL:     textOf(row["cover_url"]),
		SourceURL:    textOf(row["source_url"]),
		RawJSON:      textOf(row["raw_json"]),
		FetchedAt:    textOf(row["fetched_at"]),
		CoverDir:     textOf(row["cover_dir"]),
		CoverPath:    textOf(row["cover_path"]),
		CoverAbsPath: textOf(row["cover_abs_path"]),
		CoverBytes:   intOf(row["cover_bytes"]),
		CoverSuffix:  textOf(row["cover_suffix"]),
		CreatedAt:    timeOf(row["created_at"]),
		UpdatedAt:    timeOf(row["updated_at"]),
	}
}

func textOf(v interface{}) string {
	switch value := v.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func intOf(v interface{}) int64 {
	switch value := v.(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func timeOf(v interface{}) time.Time {
	switch value := v.(type) {
	case time.Time:
		return value
	case string:
		// 兜底：有些驱动会把 DATETIME 给成字符串
		if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

// ===== 作品自定义标签 =====
//
// 单独一张表、也惰性创建。刻意不挂在 library_works 上（外键/同一行）：作品库详情页
// 可以打开一个还没入库的作品，用户在那里加的标签也要能存下来 —— 标签的归属是
// 「来源 + 编号」，和"是否已入库"无关。

// ensureWorkTagsTable 惰性建表（幂等）。
func (s *SQLiteService) ensureWorkTagsTable(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	queries := []string{
		`CREATE TABLE IF NOT EXISTS work_tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL DEFAULT 'dlsite',
			work_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source, work_id, tag)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_work_tags_work ON work_tags (source, work_id)`,
	}
	for _, query := range queries {
		if err := s.Execute(ctx, query); err != nil {
			return fmt.Errorf("初始化作品标签表失败: %w", err)
		}
	}
	return nil
}

// ListWorkTags 列出某个作品的自定义标签（按添加顺序）。
func (s *SQLiteService) ListWorkTags(ctx context.Context, source, workID string) ([]string, error) {
	if s == nil || s.DB == nil {
		return []string{}, nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return []string{}, nil
	}
	if err := s.ensureWorkTagsTable(ctx); err != nil {
		return nil, err
	}
	rows, err := s.Query(ctx,
		`SELECT tag FROM work_tags WHERE source = ? AND work_id = ? ORDER BY id ASC`,
		normalizeSource(source), workID)
	if err != nil {
		return nil, fmt.Errorf("查询作品标签失败: %w", err)
	}
	tags := make([]string, 0, len(rows))
	for _, row := range rows {
		if tag, ok := row["tag"].(string); ok && tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

// AddWorkTag 添加一个标签。重复添加不报错（UNIQUE + INSERT OR IGNORE）。
func (s *SQLiteService) AddWorkTag(ctx context.Context, source, workID, tag string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	tag = strings.TrimSpace(tag)
	if workID == "" || tag == "" {
		return fmt.Errorf("作品编号或标签为空")
	}
	if err := s.ensureWorkTagsTable(ctx); err != nil {
		return err
	}
	if err := s.Execute(ctx,
		`INSERT OR IGNORE INTO work_tags (source, work_id, tag) VALUES (?, ?, ?)`,
		normalizeSource(source), workID, tag); err != nil {
		return fmt.Errorf("添加标签失败: %w", err)
	}
	return nil
}

// RemoveWorkTag 删除一个标签；不存在也算成功（幂等）。
func (s *SQLiteService) RemoveWorkTag(ctx context.Context, source, workID, tag string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	tag = strings.TrimSpace(tag)
	if workID == "" || tag == "" {
		return nil
	}
	if err := s.ensureWorkTagsTable(ctx); err != nil {
		return err
	}
	if err := s.Execute(ctx,
		`DELETE FROM work_tags WHERE source = ? AND work_id = ? AND tag = ?`,
		normalizeSource(source), workID, tag); err != nil {
		return fmt.Errorf("删除标签失败: %w", err)
	}
	return nil
}
