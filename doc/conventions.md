# 约定与注意事项

## 构建 / 运行

| 命令 | 说明 |
| --- | --- |
| `wails3 dev` | 开发模式（热重载前后端） |
| `wails3 build` | 生产构建 |
| `task dev` | 等价 `wails3 dev -config ./build/config.yml -port 9245` |
| `task build` / `task package` / `task run` | 按 `GOOS` 分发到 `build/<os>/Taskfile.yml` |
| `npm run dev` | 仅跑前端 Vite（端口 9245，`strictPort`） |
| `npm run build` | `vue-tsc && vite build`（含类型检查） |

Go 模块：`WeedyBox`，Wails v3 `v3.0.0-beta.10`，SQLite 走 `modernc.org/sqlite`（**纯 Go，无需 CGO**）。

## 扫描范围

`.gitignore` 已排除、**无需阅读**的目录：

`node_modules/`、`bin/`、`dist/`、`.task/`、`vendor/`、`frontend/dist|build|node_modules|bindings/`、`build/{android,darwin,ios,linux,docker}/`、`*.db`、`*.exe`、`.env`、`.vscode/`、`.idea/`、`logs/`

> `build/` 下只有 `config.yml` 与 `windows/`、`appicon*` 属于模板配置；其余平台目录是 Wails 生成物。

## 代码约定

- **命名**：Go 方法首字母大写才会导出到前端绑定。
- **前端后缀**：store 文件用大写开头（`Sidebar.ts` `Theme.ts`）与 `todoStore.ts` 不一致，**新增 store 需手动 import 对应大小写**。
- **bindings 只读**：`frontend/bindings/` 由 Wails 生成，改 Go 签名后重新生成，勿手改。
- **返回 null**：Go 返回指针/切片的接口，前端绑定类型为 `T | null`，统一在 `services/todoService.ts` 的 `ensure()` 兜底。
- **超时**：所有 `TodoService` 方法内建 3 秒 `context.WithTimeout`，并区分 `context.Canceled` / `DeadlineExceeded` 错误文案。
- **类型断言**：`rowToTodo` / `rowToStats` 依赖 SQLite 返回的 `int64` / `time.Time`，改列类型会 panic。

## 已知待办

| 位置 | 问题 |
| --- | --- |
| `internal/service/todo_service.go:284` | 注释标记 `bug need-fix`（`GetStats`） |
| `frontend/src/components/settings/LanguageSelect.vue` | 仅占位，未实现 |
| `frontend/src/components/todo/TodoItem.vue` | 描述只读展示，**尚不能编辑**（后端 `Update` 已支持 `description`） |
| `frontend/src/views/test.vue` | 测试页，可删 |
| `README.md` | 仍为 Wails 模板原文，未更新为项目说明 |
| `main.go:56` | `Description` 仍是 `"A demo of using raw HTML & CSS"` |
| `global.css:8` | `body` 背景硬编码 `#1A1A22`（深色值），浅色主题下启动会闪深色 |
