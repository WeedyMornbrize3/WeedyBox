// internal/service/todo_service.go
package service

import (
	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type TodoService struct {
	DB *database.SQLiteService
}

// ===== 辅助函数 =====

// rowToTodo 将 map 转换为 Todo
func (s *TodoService) rowToTodo(row map[string]interface{}) model.Todo {
	todo := model.Todo{
		ID:        row["id"].(int64),
		Title:     row["title"].(string),
		Priority:  int(row["priority"].(int64)),
		Completed: row["completed"].(int64) == 1,
	}

	if desc, ok := row["description"].(string); ok {
		todo.Description = desc
	}
	if createdAt, ok := row["created_at"].(string); ok {
		todo.CreatedAt = createdAt
	}
	if updatedAt, ok := row["updated_at"].(string); ok {
		todo.UpdatedAt = updatedAt
	}

	return todo
}

// ===== CRUD =====

// Create 创建 TODO
func (s *TodoService) Create(ctx context.Context, dto model.CreateTodoDTO) (*model.Todo, error) {
	if dto.Priority < 0 || dto.Priority > 2 {
		dto.Priority = 1
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `INSERT INTO todos (title, description, priority) VALUES (?, ?, ?)`
	if err := s.DB.Execute(ctx, query, dto.Title, dto.Description, dto.Priority); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("创建被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("创建超时: %w", err)
		}
		return nil, fmt.Errorf("创建 TODO 失败: %w", err)
	}

	id, err := s.DB.GetLastInsertID(ctx)
	if err != nil {
		return nil, err
	}

	return s.GetByID(ctx, id)
}

// GetByID 根据 ID 获取 TODO
func (s *TodoService) GetByID(ctx context.Context, id int64) (*model.Todo, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT id, title, description, priority, completed, created_at, updated_at FROM todos WHERE id = ?`
	row, err := s.DB.QueryRow(ctx, query, id)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("查询被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("查询超时: %w", err)
		}
		return nil, fmt.Errorf("查询 TODO 失败: %w", err)
	}

	todo := s.rowToTodo(row)
	return &todo, nil
}

// GetAll 获取所有 TODO
func (s *TodoService) GetAll(ctx context.Context) ([]model.Todo, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT id, title, description, priority, completed, created_at, updated_at FROM todos ORDER BY priority DESC, created_at DESC`
	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("查询被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("查询超时: %w", err)
		}
		return nil, fmt.Errorf("查询 TODO 失败: %w", err)
	}

	todos := make([]model.Todo, 0, len(rows))
	for _, row := range rows {
		todos = append(todos, s.rowToTodo(row))
	}
	return todos, nil
}

// GetByStatus 根据完成状态获取
func (s *TodoService) GetByStatus(ctx context.Context, completed bool) ([]model.Todo, error) {
	completedInt := 0
	if completed {
		completedInt = 1
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT id, title, description, priority, completed, created_at, updated_at FROM todos WHERE completed = ? ORDER BY priority DESC, created_at DESC`
	rows, err := s.DB.Query(ctx, query, completedInt)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("查询被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("查询超时: %w", err)
		}
		return nil, fmt.Errorf("查询 TODO 失败: %w", err)
	}

	todos := make([]model.Todo, 0, len(rows))
	for _, row := range rows {
		todos = append(todos, s.rowToTodo(row))
	}
	return todos, nil
}

// Update 更新 TODO
func (s *TodoService) Update(ctx context.Context, id int64, dto model.UpdateTodoDTO) (*model.Todo, error) {
	// 先检查是否存在
	if _, err := s.GetByID(ctx, id); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 构建动态 SQL
	updates := []string{}
	args := []interface{}{}

	if dto.Title != nil {
		updates = append(updates, "title = ?")
		args = append(args, *dto.Title)
	}
	if dto.Description != nil {
		updates = append(updates, "description = ?")
		args = append(args, *dto.Description)
	}
	if dto.Priority != nil {
		updates = append(updates, "priority = ?")
		args = append(args, *dto.Priority)
	}
	if dto.Completed != nil {
		updates = append(updates, "completed = ?")
		completedInt := 0
		if *dto.Completed {
			completedInt = 1
		}
		args = append(args, completedInt)
	}

	if len(updates) == 0 {
		return s.GetByID(ctx, id)
	}

	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := "UPDATE todos SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	if err := s.DB.Execute(ctx, query, args...); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("更新被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("更新超时: %w", err)
		}
		return nil, fmt.Errorf("更新 TODO 失败: %w", err)
	}

	return s.GetByID(ctx, id)
}

// ToggleComplete 切换完成状态
func (s *TodoService) ToggleComplete(ctx context.Context, id int64) (*model.Todo, error) {
	// 先检查是否存在
	if _, err := s.GetByID(ctx, id); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `UPDATE todos SET completed = CASE WHEN completed = 0 THEN 1 ELSE 0 END, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	if err := s.DB.Execute(ctx, query, id); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("切换状态被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("切换状态超时: %w", err)
		}
		return nil, fmt.Errorf("切换状态失败: %w", err)
	}

	return s.GetByID(ctx, id)
}

// Delete 删除 TODO
func (s *TodoService) Delete(ctx context.Context, id int64) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `DELETE FROM todos WHERE id = ?`
	err := s.DB.Execute(ctx, query, id)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("删除被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("删除超时: %w", err)
		}
		return fmt.Errorf("删除 TODO 失败: %w", err)
	}
	return nil
}

// DeleteAll 删除所有 TODO
func (s *TodoService) DeleteAll(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `DELETE FROM todos`
	err := s.DB.Execute(ctx, query)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("删除被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("删除超时: %w", err)
		}
		return fmt.Errorf("删除 TODO 失败: %w", err)
	}
	return nil
}

// ===== 统计 =====

// GetStats 获取统计信息
func (s *TodoService) rowToStats(row map[string]interface{}) (model.TodoStats, error) {
	stats := model.TodoStats{
		Total:     0,
		Completed: 0,
		Pending:   0,
	}
	if total, ok := row["total"].(sql.NullInt64); ok {
		stats.Total = total.Int64
	}
	if completed, ok := row["created_at"].(sql.NullInt64); ok {
		stats.Completed = completed.Int64
	}
	if pending, ok := row["updated_at"].(sql.NullInt64); ok {
		stats.Pending = pending.Int64
	}
	return stats, nil
}

// bug  need-fix
func (s *TodoService) GetStats() (model.TodoStats, error) {
	query := `
        SELECT 
            COALESCE(COUNT(*), 0) as total,
            COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as completed,
            COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending
        FROM todos
    `

	row, err := s.DB.QueryRow(application.Get().Context(), query)
	if err != nil {
		return model.TodoStats{}, err
	}
	return s.rowToStats(row)
}

// GetPriorityStats 按优先级统计
func (s *TodoService) GetPriorityStats(ctx context.Context) (map[int]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	query := `SELECT priority, COUNT(*) AS count FROM todos GROUP BY priority`
	rows, err := s.DB.Query(ctx, query)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("获取优先级统计被取消: %w", err)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("获取优先级统计超时: %w", err)
		}
		return nil, fmt.Errorf("获取优先级统计失败: %w", err)
	}

	stats := make(map[int]int64)
	for _, row := range rows {
		priority := int(row["priority"].(int64))
		count := row["count"].(int64)
		stats[priority] = count
	}
	return stats, nil
}
