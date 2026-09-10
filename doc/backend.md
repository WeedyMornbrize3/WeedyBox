# 后端 (Go)

## 入口 `main.go`

| 函数 / 代码 | 说明 |
| --- | --- |
| `main()` :33 | 注册服务 → `application.New` → 建窗口 → `app.Run()` |
| 服务注册 :36-52 | `WindowService` / `TodoService` / `database.SQLiteService` 注入 |
| 窗口参数 :66-81 | 1200×741，无边框，最小 900×556 |
| `assets` :20 | `//go:embed all:frontend/dist` 嵌入前端产物 |

## 数据模型 `internal/model/todo.go`

| 类型 | 行 | 字段 |
| --- | --- | --- |
| `Todo` | 9 | `ID, Title, Description, Priority(0/1/2), Completed, CreatedAt, UpdatedAt` |
| `CreateTodoDTO` | 20 | `Title, Description, Priority` |
| `UpdateTodoDTO` | 27 | `*Title, *Description, *Priority, *Completed`（指针=可选字段） |
| `TodoStats` | 35 | `Total, Completed, Pending` |

## 数据库层 `internal/database/sql_service.go`

| 函数 | 行 | 说明 |
| --- | --- | --- |
| `ServiceStartup(ctx, opts)` | 20 | 服务启动钩子：`DB.Open()` + `initAllTables` |
| `ServiceShutdown(ctx, opts)` | 36 | 服务关闭钩子 |
| `initAllTables(ctx, s)` | 43 | 建表：`todos`（IF NOT EXISTS） |
| `Execute(ctx, query, args...)` | 66 | 执行写操作（增删改） |
| `Query(ctx, query, args...)` | 70 | 返回多行 `[]map[string]interface{}` |
| `QueryRow(ctx, query, args...)` | 74 | 返回单行，无记录时报错「没有找到记录」 |
| `GetLastInsertID(ctx)` | 85 | `SELECT last_insert_rowid()`，兼容 int64/int/float64 |
| `Transaction(ctx, fn)` | 106 | BEGIN → fn → COMMIT，出错 ROLLBACK |

> 表结构：`todos(id PK AUTOINCREMENT, title NOT NULL, description, priority DEFAULT 1, completed DEFAULT 0, created_at, updated_at)`

## Todo 服务 `internal/service/todo_service.go`

### 辅助

| 函数 | 行 | 说明 |
| --- | --- | --- |
| `rowToTodo(row)` | 21 | `map` → `model.Todo`（需类型断言） |
| `rowToStats(row)` | 266 | 统计行 → `model.TodoStats` |

### CRUD（均带 3 秒 `context.WithTimeout`）

| 函数 | 行 | 说明 |
| --- | --- | --- |
| `Create(ctx, dto)` | 45 | INSERT → `GetLastInsertID` → `GetByID` |
| `GetByID(ctx, id)` | 69 | 单条查询 |
| `GetAll(ctx)` | 90 | 全部，排序 `completed ASC, priority DESC, created_at DESC` |
| `GetByStatus(ctx, completed)` | 114 | 按完成状态过滤 |
| `Update(ctx, id, dto)` | 143 | 动态拼 SQL，仅更新非 nil 字段，附 `updated_at` |
| `ToggleComplete(ctx, id)` | 199 | `completed = CASE WHEN 1 THEN 0 ELSE 1 END` |
| `Delete(ctx, id)` | 226 | 按 ID 删除 |
| `DeleteAll(ctx)` | 245 | 清空表 |

### 统计

| 函数 | 行 | 说明 |
| --- | --- | --- |
| `GetStats(ctx)` | 286 | 一次查询出 total / completed / pending |
| `GetPriorityStats(ctx)` | 314 | `map[priority]count` |

> ⚠️ :284 标记 `bug need-fix`：`GetStats` 已被绑定导出，签名勿随意改动。

## 窗口服务 `internal/service/window_service.go`

| 函数 | 行 | 说明 |
| --- | --- | --- |
| `MinimizeWindow()` | 13 | 最小化 |
| `ToggleMaximizeWindow()` | 20 | 最大化 / 还原切换 |
| `CloseWindow()` | 32 | 关闭窗口 |
| `IsWindowMaximised()` | 39 | 返回是否最大化 |

> 结构体字段 `Window *application.WebviewWindow` 在 `main.go:83` 赋值。

## 前端可调用的绑定

`frontend/bindings/WeedyBox/internal/service/`（**自动生成，勿手改**）
- `TodoService.*`：`Create, GetByID, GetAll, GetByStatus, Update, ToggleComplete, Delete, DeleteAll, GetStats`
- `WindowService.*`：`MinimizeWindow, ToggleMaximizeWindow, CloseWindow, IsWindowMaximised`

导出名规则：Go 方法名首字母大写即前端函数名。
