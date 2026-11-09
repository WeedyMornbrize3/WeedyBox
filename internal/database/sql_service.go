// internal/database/sql_service.go
package database

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/sqlite"
)

// SQLiteService 通用数据库服务（使用 Wails 内置 SQLite，无需 CGO）
type SQLiteService struct {
	DB *sqlite.SQLiteService

	// mu 串行化所有 SQL 语句。
	//
	// 为什么必须要有：SQLite 在同一文件上被多个连接并发写时会直接返回
	// `database is locked (5) (SQLITE_BUSY)`。游戏库的表是**惰性创建**的
	// （第一次读写才 CREATE TABLE IF NOT EXISTS），所以并发首次使用天然就是
	// 一次 DDL 竞争 —— 无锁时实测 8 个 goroutine 同时首用必定有若干次 BUSY。
	// 桌面端所有访问都来自同一个进程、同一个人，串行化的代价可以忽略；
	// 而且 Query 是一次性把行全部取完再返回，持锁期间不会把锁泄漏给调用方。
	mu sync.Mutex
}

// ===== 实现 application.Service 接口 =====

func (s *SQLiteService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	log.Println("初始化数据库服务...")
	// 直接使用 s.DB.Execute 测试连接
	if err := s.DB.Open(); err != nil {
		// 如果连接失败，尝试重新创建
		log.Printf("数据库连接失败，尝试重新创建: %v", err)
		// 注意：在 Wails v3 中，ServiceStartup 时连接可能还未建立
		// 解决方法：在窗口创建后再初始化
	}
	if err := initAllTables(application.Get().Context(), s); err != nil {
		log.Fatal("初始化表失败", err)
	}
	log.Println("数据库服务已注册")
	return nil
}

func (s *SQLiteService) ServiceShutdown(ctx context.Context, options application.ServiceOptions) error {
	log.Println("🔧 关闭数据库服务...")
	return nil
}

// ===== 表初始化 =====

func initAllTables(ctx context.Context, s *SQLiteService) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS todos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT,
			priority INTEGER DEFAULT 1,
			completed INTEGER DEFAULT 0,  -- 0=false, 1=true
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// ── 游戏库的表**不在这里建** ──────────────────────────────────────
		// 库里的作品是"下载/入库"才会有的数据，没人用的时候建两张空表只是白占
		// 数据库页（SQLite 每张表至少一个 page）。所以 game library 的两张表
		// 改为**惰性创建**：第一次真正读写时才建，见 library_repo.go 的
		// ensureLibraryTables()。这里只留一个注释，避免下次有人以为漏了。
	}

	for _, query := range queries {
		if err := s.DB.Execute(query); err != nil {
			return err
		}
	}
	return nil
}

// ===== 通用执行方法 =====

func (s *SQLiteService) Execute(ctx context.Context, query string, args ...interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.DB.Execute(query, args...)
}

func (s *SQLiteService) Query(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.DB.QueryContext(ctx, query, args...)
}

func (s *SQLiteService) QueryRow(ctx context.Context, query string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := s.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("没有找到记录")
	}
	return rows[0], nil
}

func (s *SQLiteService) GetLastInsertID(ctx context.Context) (int64, error) {
	rows, err := s.Query(ctx, "SELECT last_insert_rowid() AS id")
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("无法获取插入的 ID")
	}

	switch v := rows[0]["id"].(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case float64:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("未知 ID 类型: %T", v)
	}
}

func (s *SQLiteService) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := s.Execute(ctx, "BEGIN TRANSACTION"); err != nil {
		return err
	}
	if err := fn(ctx); err != nil {
		_ = s.Execute(ctx, "ROLLBACK")
		return err
	}
	return s.Execute(ctx, "COMMIT")
}
