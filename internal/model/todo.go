// internal/model/todo.go
package model

import (
	"time"
)

// Todo 数据模型
type Todo struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Priority    int       `json:"priority"` // 0 1 2
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateTodoDTO 创建请求
type CreateTodoDTO struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

// UpdateTodoDTO 更新请求
type UpdateTodoDTO struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
}

// TodoStats 统计信息
type TodoStats struct {
	Total     int64 `json:"total"`
	Completed int64 `json:"completed"`
	Pending   int64 `json:"pending"`
}
