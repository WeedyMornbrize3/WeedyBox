// internal/database/library_repo.go
//
// 游戏库（元数据抓取成果）的持久化层。
//
// 两条设计约束，都来自"以后不只抓 DLsite"：
//
//  1. 表是**来源无关**的：作品信息（id/名字/社团/…）放公共列，具体来源只体现在
//     `source` 列的值上（当前只有 'dlsite'）。所以加新来源不需要加表，也不需要
//     在库里存 "dlsite 特有字段" 的专属列 —— 原始数据整份进 raw_json。
//  2. 表是**惰性创建**的：这里每次操作前都 ensureLibraryTables()，而启动时的
//     initAllTables 不建这两张表。没人往库里存东西时，数据库里就不存在这两张表
//     （省掉空表占的 page）。代价是每次读写多一条 CREATE TABLE IF NOT EXISTS，
//     幂等且极快，也顺手兼容"库文件是旧版本建的、还没有这些表"的情况。
//
// 这里只做「表 ↔ 结构体」的搬运，不含解析或格式化逻辑：
// 抓取在 internal/service/getMeta，编排在 internal/service/library_service.go。
package database

import (
	"WeedyBox/internal/model"
	"context"
	"fmt"
	"strings"
	"time"
)

// SourceDLsite / SourceSteam 是当前支持的元数据来源标识。
// 加新来源时在这里补常量，入库时把 source 传进来即可（表结构来源无关）。
const (
	SourceDLsite = "dlsite"
	SourceSteam  = "steam" // Steam 商店（appid 当编号），封面取本机 appcache\librarycache
)

// LaunchMode* 是「怎么启动这个游戏」的取值（library_works.launch_mode）：
//
//   - LaunchModeSteam：跑 steam://rungameid/<appid>，由 Steam 客户端拉起游戏
//     （只有 steam 来源才有意义，steam 来源**默认**用它）；
//   - LaunchModeExe：直接跑本机记录的游戏程序（game_path），其它来源默认用它。
//
// 空串 = 还没选过，按来源取默认值（见 game 服务的 resolveLaunchMode）。
const (
	LaunchModeSteam = "steam"
	LaunchModeExe   = "exe"
)

// workColumns 列顺序固定在这里，读写共用，免得多处手写漏列。
const workColumns = `source, work_id, name, maker, maker_id, series, series_id,
	work_type, age_rating, release_date, cover_url, source_url, raw_json, fetched_at,
	cover_dir, cover_path, cover_abs_path, cover_bytes, cover_suffix, game_path, launch_mode,
	hero_dir, hero_path, hero_abs_path, hero_bytes, hero_suffix, hero_url,
	quit_game_time_stamp, last_launch_at, play_time_total_seconds,
	created_at, updated_at`

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
			game_path TEXT DEFAULT '',
			launch_mode TEXT DEFAULT '',
			hero_dir TEXT DEFAULT '',
			hero_path TEXT DEFAULT '',
			hero_abs_path TEXT DEFAULT '',
			hero_bytes INTEGER DEFAULT 0,
			hero_suffix TEXT DEFAULT '',
			hero_url TEXT DEFAULT '',
			quit_game_time_stamp DATETIME DEFAULT '',
			last_launch_at DATETIME DEFAULT '',
			play_time_total_seconds INTEGER DEFAULT 0,
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

		// 按天的游玩时长：一天一个桶（sessions 跨午夜时切开分别累加）。
		// 与 library_works 上的 play_time_total_seconds 同时写：总数用于动作栏秒开，
		// 日桶留给以后「游戏统计」页做每天/每周曲线。
		`CREATE TABLE IF NOT EXISTS work_play_time (
			source TEXT NOT NULL DEFAULT 'dlsite',
			work_id TEXT NOT NULL,
			day TEXT NOT NULL,
			seconds INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (source, work_id, day)
		)`,
	}

	for _, query := range queries {
		if err := s.Execute(ctx, query); err != nil {
			return fmt.Errorf("初始化游戏库表失败: %w", err)
		}
	}

	// 旧库文件里 library_works 可能还没有后来新增的列
	// （2026-11 的 game_path「游戏启动」、2026-11-23 的 launch_mode「启动方式」、
	// 2026-11-27 的 hero_*「hero 图本地缓存」与游玩统计三列）。
	// CREATE TABLE IF NOT EXISTS 不会补列，所以这里显式迁移一次：
	// 一条 PRAGMA 查清现有列，只 ALTER 缺的那几列（每次读写都要走，别做成一列一次）。
	return s.ensureLibraryWorkColumns(ctx, libraryWorkColumns)
}

// libraryColumn 是给旧库文件的 library_works 补的一列。
type libraryColumn struct {
	name string
	ddl  string
}

// libraryWorkColumns 是 library_works 上"后来新增"的全部列（幂等补列清单）。
// 加新列时：这里的 ddl、上面的 CREATE TABLE、workColumns、rowToLibraryWork 四处一起改。
var libraryWorkColumns = []libraryColumn{
	{"game_path", `ALTER TABLE library_works ADD COLUMN game_path TEXT DEFAULT ''`},
	{"launch_mode", `ALTER TABLE library_works ADD COLUMN launch_mode TEXT DEFAULT ''`},
	{"hero_dir", `ALTER TABLE library_works ADD COLUMN hero_dir TEXT DEFAULT ''`},
	{"hero_path", `ALTER TABLE library_works ADD COLUMN hero_path TEXT DEFAULT ''`},
	{"hero_abs_path", `ALTER TABLE library_works ADD COLUMN hero_abs_path TEXT DEFAULT ''`},
	{"hero_bytes", `ALTER TABLE library_works ADD COLUMN hero_bytes INTEGER DEFAULT 0`},
	{"hero_suffix", `ALTER TABLE library_works ADD COLUMN hero_suffix TEXT DEFAULT ''`},
	{"hero_url", `ALTER TABLE library_works ADD COLUMN hero_url TEXT DEFAULT ''`},
	{"quit_game_time_stamp", `ALTER TABLE library_works ADD COLUMN quit_game_time_stamp DATETIME DEFAULT ''`},
	{"last_launch_at", `ALTER TABLE library_works ADD COLUMN last_launch_at DATETIME DEFAULT ''`},
	{"play_time_total_seconds", `ALTER TABLE library_works ADD COLUMN play_time_total_seconds INTEGER DEFAULT 0`},
}

// ensureLibraryWorkColumns 给 library_works 补齐一批列（幂等）：先 PRAGMA 查有没有，没有才 ALTER。
//
// 为什么用 PRAGMA 先查再加列，而不是直接 ALTER 然后吃掉错误：
// 直接 ALTER 在每次读写前都会失败一次，靠字符串匹配错误码判断「字段已存在」
// 既脆弱又吵；PRAGMA table_info 是一次极快的内存查询，先把事实问清楚更稳。
// 并发首次读写时两个 goroutine 可能都判断"缺列"、都去 ALTER，后一个会报
// duplicate column —— 这种竞争按已存在处理即可。
func (s *SQLiteService) ensureLibraryWorkColumns(ctx context.Context, columns []libraryColumn) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	rows, err := s.Query(ctx, `PRAGMA table_info(library_works)`)
	if err != nil {
		return fmt.Errorf("读取游戏库表结构失败: %w", err)
	}
	existing := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		existing[textOf(row["name"])] = struct{}{}
	}
	for _, column := range columns {
		if _, ok := existing[column.name]; ok {
			continue
		}
		if err := s.Execute(ctx, column.ddl); err != nil {
			// 竞争窗口：另一个调用已经加上了这一列
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				continue
			}
			return fmt.Errorf("为游戏库表添加 %s 列失败: %w", column.name, err)
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

	// quit_game_time_stamp 只在**首次入库**时写：初始值 = 添加时间
	// （用户 2026-11-27：「model 添加 quitGameTimeStamp 初始值为添加时间」）。
	// ON CONFLICT 的 SET 里刻意不含它 —— 之后这一格只由 game 服务的启动/退出改写。
	quitAt := strings.TrimSpace(w.QuitGameTimeStamp)
	if quitAt == "" {
		quitAt = formatTimeStamp(time.Now())
	}

	query := `INSERT INTO library_works
		(source, work_id, name, maker, maker_id, series, series_id, work_type, age_rating,
		 release_date, cover_url, source_url, raw_json, fetched_at, quit_game_time_stamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		w.RawJSON, w.FetchedAt, quitAt,
	); err != nil {
		return fmt.Errorf("写入作品失败: %w", err)
	}
	return nil
}

// DeleteLibraryWork 把一条作品连同它的封面下载记录、游戏启动位置一起删掉。
// 磁盘上的图片与游戏程序**不删** —— 那是用户自己的文件，库里删条记录不该顺手删文件。
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

// ListCoverFiles 列出某个作品的所有封面落地记录（按下载顺序）。
//
// 从游戏库删除作品时要按这些记录逐个删磁盘文件：一次入库可能下过多次
// （重名会加 -2/-3），只删作品表里的最新指针会留下旧图。
func (s *SQLiteService) ListCoverFiles(ctx context.Context, source, workID string) ([]model.CoverFile, error) {
	if s == nil || s.DB == nil {
		return []model.CoverFile{}, nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return []model.CoverFile{}, nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx,
		`SELECT id, source, work_id, dir, path, abs_path, bytes, suffix, cover_url, created_at
		 FROM library_cover_files WHERE source = ? AND work_id = ? ORDER BY id ASC`,
		normalizeSource(source), workID)
	if err != nil {
		return nil, fmt.Errorf("查询封面记录失败: %w", err)
	}

	files := make([]model.CoverFile, 0, len(rows))
	for _, row := range rows {
		files = append(files, model.CoverFile{
			ID:        intOf(row["id"]),
			Source:    textOf(row["source"]),
			WorkID:    textOf(row["work_id"]),
			Dir:       textOf(row["dir"]),
			Path:      textOf(row["path"]),
			AbsPath:   textOf(row["abs_path"]),
			Bytes:     intOf(row["bytes"]),
			Suffix:    textOf(row["suffix"]),
			CoverURL:  textOf(row["cover_url"]),
			CreatedAt: timeOf(row["created_at"]),
		})
	}
	return files, nil
}

// ===== 游戏启动位置 =====
//
// 位置就是 library_works 上的一列 game_path（不单开表）：一个库记录对应一个本机
// 游戏程序，生命周期和这条记录绑在一起 —— 删库记录时自动消失，重新抓元数据时
// 不会被 Upsert 覆盖（ON CONFLICT 的 SET 里刻意不含 game_path）。
//
// 旧库文件（2026-11 之前建的）没有这一列，由 ensureGamePathColumn 惰性补上。

// GetGamePath 读一个作品的游戏启动位置；没有记录/还没设置时返回空串 + nil。
//
// 「没有设置」是正常状态（老库作品、只入了元数据还没指定程序），不是错误 ——
// 前端据此决定第一次点「游戏启动」时要不要弹系统文件框让用户选 exe。
func (s *SQLiteService) GetGamePath(ctx context.Context, source, workID string) (string, error) {
	if s == nil || s.DB == nil {
		return "", nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return "", nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return "", err
	}

	rows, err := s.Query(ctx,
		`SELECT game_path FROM library_works WHERE source = ? AND work_id = ?`,
		normalizeSource(source), workID)
	if err != nil {
		return "", fmt.Errorf("查询游戏位置失败: %w", err)
	}
	if len(rows) == 0 {
		return "", nil
	}
	return strings.TrimSpace(textOf(rows[0]["game_path"])), nil
}

// SetGamePath 写入/更新一个作品的游戏启动位置（幂等：重复设置就是覆盖那一格）。
//
// 只更新**已在游戏库里**的作品：库里没有这条记录时报错，而不是顺手插一条
// 只有路径的空壳（那会让游戏库网格里冒出一个没有名字/封面的作品）。
func (s *SQLiteService) SetGamePath(ctx context.Context, source, workID, path string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	path = strings.TrimSpace(path)
	if workID == "" {
		return fmt.Errorf("work_id 为空，无法记录游戏位置")
	}
	if path == "" {
		return fmt.Errorf("游戏位置为空")
	}
	source = normalizeSource(source)
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}
	row, err := s.FindLibraryWork(ctx, source, workID)
	if err != nil {
		return err
	}
	if row == nil {
		return fmt.Errorf("作品 %s 不在游戏库中，无法记录启动位置（请先入库）", workID)
	}

	if err := s.Execute(ctx,
		`UPDATE library_works SET game_path = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE source = ? AND work_id = ?`,
		path, source, workID); err != nil {
		return fmt.Errorf("记录游戏位置失败: %w", err)
	}
	return nil
}

// ===== 游戏启动方式 =====
//
// launch_mode 和 game_path 同表同生命周期：空 = 没选过，按来源取默认值
// （steam 源默认走 steam://rungameid，其它源默认跑本地 exe）。

// GetLaunchMode 读一个作品的启动方式；没选过时返回空串 + nil（不是错误）。
func (s *SQLiteService) GetLaunchMode(ctx context.Context, source, workID string) (string, error) {
	if s == nil || s.DB == nil {
		return "", nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return "", nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return "", err
	}

	rows, err := s.Query(ctx,
		`SELECT launch_mode FROM library_works WHERE source = ? AND work_id = ?`,
		normalizeSource(source), workID)
	if err != nil {
		return "", fmt.Errorf("查询启动方式失败: %w", err)
	}
	if len(rows) == 0 {
		return "", nil
	}
	return strings.ToLower(strings.TrimSpace(textOf(rows[0]["launch_mode"]))), nil
}

// SetLaunchMode 写入/更新一个作品的启动方式（mode 只接受 steam / exe / 空串）。
//
// 与 SetGamePath 一样只更新**已在游戏库里**的作品：库里没有这条记录时报错，
// 不顺手插一条只有启动方式的空壳。
func (s *SQLiteService) SetLaunchMode(ctx context.Context, source, workID, mode string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	mode = strings.ToLower(strings.TrimSpace(mode))
	if workID == "" {
		return fmt.Errorf("work_id 为空，无法记录启动方式")
	}
	if mode != "" && mode != LaunchModeSteam && mode != LaunchModeExe {
		return fmt.Errorf("不支持的启动方式: %q（只能是 steam / exe / 空）", mode)
	}
	source = normalizeSource(source)
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}
	row, err := s.FindLibraryWork(ctx, source, workID)
	if err != nil {
		return err
	}
	if row == nil {
		return fmt.Errorf("作品 %s 不在游戏库中，无法记录启动方式（请先入库）", workID)
	}

	if err := s.Execute(ctx,
		`UPDATE library_works SET launch_mode = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE source = ? AND work_id = ?`,
		mode, source, workID); err != nil {
		return fmt.Errorf("记录启动方式失败: %w", err)
	}
	return nil
}

// ===== 游玩统计（quitGameTimeStamp / lastLaunchAt / playTimeTotalSeconds）=====
//
// 三列都在 library_works 上（与 game_path / launch_mode 同一个思路：生命周期跟着
// 库记录走，Upsert 刷新元数据不会碰它们）。日桶另起 work_play_time。
//
// 语义（2026-11-27 用户要求）：
//   - quit_game_time_stamp：入库时 = 添加时间；启动时临时写成启动时间（计时起点）；
//     退出时写成真正的退出时间 —— 退出那格与启动那格相减就是本次游玩时长；
//   - last_launch_at：上一次启动时间，启动时写、退出后保留（动作栏显示它）；
//   - play_time_total_seconds：累计运行总时长（秒）。

// GamePlayStats 是一个作品的动作栏要显示的游玩统计（全从库里读）。
type GamePlayStats struct {
	QuitGameTimeStamp    time.Time
	LastLaunchAt         time.Time
	PlayTimeTotalSeconds int64
}

// GetGamePlayStats 读一个作品的游玩统计；没入过库/没有记录时返回零值 + nil。
//
// quit 这一格在**老库记录**里是空的（列是后补的）：按用户要求「初始值为添加时间」，
// 这里回落到 created_at，而不是返回零值。
func (s *SQLiteService) GetGamePlayStats(ctx context.Context, source, workID string) (GamePlayStats, error) {
	var stats GamePlayStats
	if s == nil || s.DB == nil {
		return stats, nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return stats, nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return stats, err
	}

	rows, err := s.Query(ctx,
		`SELECT quit_game_time_stamp, last_launch_at, play_time_total_seconds, created_at
		 FROM library_works WHERE source = ? AND work_id = ?`,
		normalizeSource(source), workID)
	if err != nil {
		return stats, fmt.Errorf("查询游玩统计失败: %w", err)
	}
	if len(rows) == 0 {
		return stats, nil
	}

	stats.QuitGameTimeStamp = timeOf(rows[0]["quit_game_time_stamp"])
	if stats.QuitGameTimeStamp.IsZero() {
		stats.QuitGameTimeStamp = timeOf(rows[0]["created_at"])
	}
	stats.LastLaunchAt = timeOf(rows[0]["last_launch_at"])
	stats.PlayTimeTotalSeconds = intOf(rows[0]["play_time_total_seconds"])
	return stats, nil
}

// MarkGameLaunched 记录「这个作品刚刚启动」：quit 与 lastLaunchAt 都写成启动时间。
//
// 为什么启动就要盖掉 quit：用户要求「在启动时临时记录 quitGameTimeStamp，退出时和
// 新的 quitGameTimeStamp 做差值」—— 启动这一格就是本次的计时起点；应用中途被杀掉时，
// 下次也能从这一格恢复计时（game 服务里 quit == lastLaunchAt 即"还在跑"）。
// **不动 updated_at**：那一列被界面当作「入库时间」显示，启动游戏不该改它。
func (s *SQLiteService) MarkGameLaunched(ctx context.Context, source, workID string, at time.Time) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return fmt.Errorf("work_id 为空，无法记录启动时间")
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}
	if err := s.Execute(ctx,
		`UPDATE library_works SET quit_game_time_stamp = ?, last_launch_at = ?
		 WHERE source = ? AND work_id = ?`,
		formatTimeStamp(at), formatTimeStamp(at), normalizeSource(source), workID); err != nil {
		return fmt.Errorf("记录游戏启动时间失败: %w", err)
	}
	return nil
}

// RecordGameSession 结算一次游戏会话：更新退出时间、累计总时长、并按天累加日桶。
// seconds 与 days 由 game 服务算好（它负责"跨午夜切到第二天"的规则）。
func (s *SQLiteService) RecordGameSession(
	ctx context.Context,
	source, workID string,
	quitAt time.Time,
	seconds int64,
	days []model.PlayTimeDay,
) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return fmt.Errorf("work_id 为空，无法记录游玩时长")
	}
	if seconds < 0 {
		seconds = 0
	}
	src := normalizeSource(source)
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}

	return s.Transaction(ctx, func(ctx context.Context) error {
		if err := s.Execute(ctx,
			`UPDATE library_works
			 SET quit_game_time_stamp = ?, play_time_total_seconds = play_time_total_seconds + ?
			 WHERE source = ? AND work_id = ?`,
			formatTimeStamp(quitAt), seconds, src, workID); err != nil {
			return fmt.Errorf("记录游戏退出时间失败: %w", err)
		}
		for _, day := range days {
			day.Day = strings.TrimSpace(day.Day)
			if day.Day == "" || day.Seconds <= 0 {
				continue
			}
			if err := s.Execute(ctx,
				`INSERT INTO work_play_time (source, work_id, day, seconds) VALUES (?, ?, ?, ?)
				 ON CONFLICT(source, work_id, day) DO UPDATE SET
					seconds = work_play_time.seconds + excluded.seconds,
					updated_at = CURRENT_TIMESTAMP`,
				src, workID, day.Day, day.Seconds); err != nil {
				return fmt.Errorf("累加 %s 的游玩时长失败: %w", day.Day, err)
			}
		}
		return nil
	})
}

// ListPlayTimeDays 列出某个作品按天的游玩时长（日期升序）。
// 目前动作栏只显示总时长，日桶先存着，留给「游戏统计」页以后做趋势。
func (s *SQLiteService) ListPlayTimeDays(ctx context.Context, source, workID string) ([]model.PlayTimeDay, error) {
	days := []model.PlayTimeDay{}
	if s == nil || s.DB == nil {
		return days, nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return days, nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return nil, err
	}
	rows, err := s.Query(ctx,
		`SELECT day, seconds FROM work_play_time WHERE source = ? AND work_id = ? ORDER BY day ASC`,
		normalizeSource(source), workID)
	if err != nil {
		return nil, fmt.Errorf("查询每日游玩时长失败: %w", err)
	}
	for _, row := range rows {
		days = append(days, model.PlayTimeDay{
			Day:     textOf(row["day"]),
			Seconds: intOf(row["seconds"]),
		})
	}
	return days, nil
}

// ===== hero 大图缓存 =====
//
// 一个作品只留当前这一张：换图时 game/library 服务会把旧文件删掉再写新指针。
// 只更新**已在游戏库里**的作品；未入库的作品也可能缓存过 hero（文件在 covers/），
// 那种情况 UPDATE 影响 0 行，不算错误。

// SetWorkHeroImage 记录一个作品的 hero 图落地位置（目录 / 相对路径 / 绝对路径都写）。
func (s *SQLiteService) SetWorkHeroImage(ctx context.Context, source, workID string, rec *model.HeroImageFile) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	if workID == "" || rec == nil {
		return nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}
	if err := s.Execute(ctx,
		`UPDATE library_works SET
			hero_dir = ?, hero_path = ?, hero_abs_path = ?, hero_bytes = ?, hero_suffix = ?, hero_url = ?
		 WHERE source = ? AND work_id = ?`,
		rec.Dir, rec.Path, rec.AbsPath, rec.Bytes, rec.Suffix, rec.URL,
		normalizeSource(source), workID); err != nil {
		return fmt.Errorf("记录 hero 图位置失败: %w", err)
	}
	return nil
}

// ClearWorkHeroImage 清掉一个作品的 hero 指针（文件删除失败时也不影响库状态）。
func (s *SQLiteService) ClearWorkHeroImage(ctx context.Context, source, workID string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return nil
	}
	if err := s.ensureLibraryTables(ctx); err != nil {
		return err
	}
	return s.Execute(ctx,
		`UPDATE library_works SET
			hero_dir = '', hero_path = '', hero_abs_path = '', hero_bytes = 0, hero_suffix = '', hero_url = ''
		 WHERE source = ? AND work_id = ?`,
		normalizeSource(source), workID)
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
		GamePath:     textOf(row["game_path"]),
		LaunchMode:   strings.ToLower(strings.TrimSpace(textOf(row["launch_mode"]))),
		HeroDir:      textOf(row["hero_dir"]),
		HeroPath:     textOf(row["hero_path"]),
		HeroAbsPath:  textOf(row["hero_abs_path"]),
		HeroBytes:    intOf(row["hero_bytes"]),
		HeroSuffix:   textOf(row["hero_suffix"]),
		HeroURL:      textOf(row["hero_url"]),

		QuitGameTimeStamp:    textOf(row["quit_game_time_stamp"]),
		LastLaunchAt:         textOf(row["last_launch_at"]),
		PlayTimeTotalSeconds: intOf(row["play_time_total_seconds"]),

		CreatedAt: timeOf(row["created_at"]),
		UpdatedAt: timeOf(row["updated_at"]),
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

// timeOf 把行里的时间列转成 time.Time。
// 驱动对 DATETIME 列可能给 time.Time，也可能给字符串，所以两种都认；
// 新写入的时间戳统一是 RFC3339（带本地时区），老数据是 CURRENT_TIMESTAMP
// 的 "2006-01-02 15:04:05"（UTC）。
func timeOf(v interface{}) time.Time {
	switch value := v.(type) {
	case time.Time:
		return value
	case string:
		return parseTimeStamp(value)
	}
	return time.Time{}
}

// parseTimeStamp 解析库里存的时间戳文本；认不出来返回零值。
func parseTimeStamp(text string) time.Time {
	text = strings.TrimSpace(text)
	if text == "" {
		return time.Time{}
	}
	// RFC3339(Nano) 是本项目自己写的格式；后面几个兼容 CURRENT_TIMESTAMP /
	// modernc 驱动可能带回来的其它写法。
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

// formatTimeStamp 把时间写成 RFC3339（**带本地时区**），与前端 new Date() 的口径一致。
// 零值写空串 —— 界面上"没有这个时间"比 0001-01-01 更好判断。
func formatTimeStamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ===== 作品标签 =====
//
// 两张表都惰性创建，且都刻意不挂在 library_works 上（外键/同一行）：作品库详情页
// 可以打开一个还没入库的作品，用户在那里整理的标签也要能存下来 —— 标签的归属是
// 「来源 + 编号」，和"是否已入库"无关。
//
//   - work_tags      —— 标签行，id 递增 = 保存时的顺序；
//   - work_tag_edits —— 「这个作品的标签被用户整份改写过」的标记（每作品一行）。
//
// 为什么需要标记：整份改写后 work_tags 里存的就是权威列表；但"用户把标签全删光"
// 同样会让 work_tags 变空，光看行数分不清「从没整理过」与「整理成空列表」——
// 前者要拿抓取来的 tags 当前置内容，后者必须显示为空。

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
		`CREATE TABLE IF NOT EXISTS work_tag_edits (
			source TEXT NOT NULL,
			work_id TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (source, work_id)
		)`,
	}
	for _, query := range queries {
		if err := s.Execute(ctx, query); err != nil {
			return fmt.Errorf("初始化作品标签表失败: %w", err)
		}
	}
	return nil
}

// ListWorkTags 列出某个作品的标签（按保存顺序，即 work_tags.id 递增）。
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

// SaveWorkTags 用整份列表**覆盖**某个作品的标签（用户点「保存」时才走这里）：
// 先清空、再按顺序插入、最后记一行"已改写"标记 —— 三步在同一个事务里，
// 中途失败不会留下半份标签、也不会出现"标记了但其实没写进去"的状态。
//
// 传空列表是合法操作（用户把标签全删光）：work_tags 清空但标记保留，表示
// "这个作品的标签就是空"，而不是"从没整理过"。
func (s *SQLiteService) SaveWorkTags(ctx context.Context, source, workID string, tags []string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("数据库未就绪")
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return fmt.Errorf("作品编号为空")
	}
	if err := s.ensureWorkTagsTable(ctx); err != nil {
		return err
	}
	src := normalizeSource(source)
	return s.Transaction(ctx, func(ctx context.Context) error {
		if err := s.Execute(ctx,
			`DELETE FROM work_tags WHERE source = ? AND work_id = ?`, src, workID); err != nil {
			return fmt.Errorf("清空旧标签失败: %w", err)
		}
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if err := s.Execute(ctx,
				`INSERT OR IGNORE INTO work_tags (source, work_id, tag) VALUES (?, ?, ?)`,
				src, workID, tag); err != nil {
				return fmt.Errorf("写入标签失败: %w", err)
			}
		}
		if err := s.Execute(ctx,
			`INSERT OR REPLACE INTO work_tag_edits (source, work_id, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)`,
			src, workID); err != nil {
			return fmt.Errorf("标记标签已改写失败: %w", err)
		}
		return nil
	})
}

// WorkTagsEdited 这个作品的标签是否被用户整份改写过（work_tag_edits 里有行）。
func (s *SQLiteService) WorkTagsEdited(ctx context.Context, source, workID string) (bool, error) {
	if s == nil || s.DB == nil {
		return false, nil
	}
	workID = normalizeWorkID(workID)
	if workID == "" {
		return false, nil
	}
	if err := s.ensureWorkTagsTable(ctx); err != nil {
		return false, err
	}
	rows, err := s.Query(ctx,
		`SELECT COUNT(*) AS n FROM work_tag_edits WHERE source = ? AND work_id = ?`,
		normalizeSource(source), workID)
	if err != nil {
		return false, fmt.Errorf("查询标签标记失败: %w", err)
	}
	if len(rows) == 0 {
		return false, nil
	}
	// 驱动可能给 int64 / int / float64，逐个认
	switch v := rows[0]["n"].(type) {
	case int64:
		return v > 0, nil
	case int:
		return v > 0, nil
	case float64:
		return v > 0, nil
	}
	return false, nil
}
