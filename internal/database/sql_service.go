// internal/database/sql_service.go
package database

import (
	"context"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/sqlite"
)

// SQLiteService 通用数据库服务（使用 Wails 内置 SQLite，无需 CGO）
type SQLiteService struct {
	DB *sqlite.SQLiteService
}

// ===== 实现 application.Service 接口 =====

func (s *SQLiteService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	log.Println("🔧 初始化数据库服务...")
	// 直接使用 s.DB.Execute 测试连接
	if err := s.DB.Open(); err != nil {
		// 如果连接失败，尝试重新创建
		log.Printf("⚠️ 数据库连接失败，尝试重新创建: %v", err)
		// 注意：在 Wails v3 中，ServiceStartup 时连接可能还未建立
		// 解决方法：在窗口创建后再初始化
	}
	if err := initAllTables(application.Get().Context(), s); err != nil {
		log.Fatal("初始化表失败", err)
	}
	log.Println("✅ 数据库服务已注册")
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
			completed INTEGER DEFAULT 0,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
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
	return s.DB.Execute(query, args...)
}

func (s *SQLiteService) Query(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	return s.DB.QueryContext(ctx, query, args...)
}

func (s *SQLiteService) QueryRow(ctx context.Context, query string, args ...interface{}) (map[string]interface{}, error) {
	rows, err := s.DB.QueryContext(ctx, query, args...)
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
