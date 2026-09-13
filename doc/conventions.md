# 约定与注意事项

## 构建 / 运行

| 命令 | 说明 |
| --- | --- |
| `wails3 dev` | 开发模式（热重载前后端） |
| `wails3 build` | 生产构建 → `bin/weedybox.exe` |
| `wails3 package` | 打包（Windows → NSIS 安装包） |
| `task dev` | 等价 `wails3 dev -config ./build/config.yml -port 9245` |
| `task build` / `task package` / `task run` | 分发到 `build/<GOOS>/Taskfile.yml`，**本项目已裁剪为只有 windows** |
| `npm run dev` | 仅跑前端 Vite（端口 9245，`strictPort`） |
| `npm run build` | `vue-tsc && vite build`（含类型检查） |

> 本机 `task` CLI 未安装，直接用 `wails3 build` / `wails3 task <name>` 即可（`wails3 task` 内置 task 运行器）。

### 只构建 Windows

`Taskfile.yml` 的 `includes` **只挂载了 `common` 与 `windows`**，`GOOS` 固定为 `windows`：

- `darwin` / `linux` / `ios` / `android` 的 include 已移除，因此 `task build` / `package` / `run` 只会分发到 `windows:build` 等。
- 不再支持 `wails3 build GOOS=linux` 这类跨平台覆盖（没有对应 include 可分发）。
- 被裁剪平台对应的 `build/<os>/` 目录（`android`/`ios`/`linux` 为空壳，`darwin` 已删除）**不在版本控制内**，
  需要重建某平台时用 `wails3 update build-assets` 重新生成，再把 include 加回来、并把 `GOOS` 改回 `'{{.GOOS | default OS}}'`。
- 验证：`wails3 task --list` 只应出现 `windows:*` 与 `common:*` 命名空间；
  构建日志中会出现 `[windows:generate:syso]` 与 `[windows:build:native] ... -o "bin/weedybox.exe"`。

#### ⚠️ 图标任务已改为只出 Windows `.ico`

`build/Taskfile.yml` 的 `generate:icons` **不能**加回 mac 参数：

```yaml
cmds:
  - wails3 generate icons -input appicon.png -windowsfilename windows/icon.ico
```

- 原模板是 `... -macfilename darwin/icons.icns -iconcomposerinput appicon.icon -macassetdir darwin`。
  由于 `build/darwin/` 已裁剪，该命令会报
  **`open darwin/icons.icns: The system cannot find the path specified`**，
  而 `generate:icons` 是 `windows:build` 的依赖，**会直接卡死 Windows 构建与 `wails3 dev`**。
- 只删 `-macfilename` 也不行：CLI 默认 Mac 路径是 `build/darwin/icon.icns`，任务以 `dir: build` 运行，照样失败。
- `wails3 generate icons` **不会自动创建目标目录**，目录不存在即报错。

#### 已删除的无用任务

以下 5 个任务依赖被裁掉的目录，已从 `build/Taskfile.yml` 删除：

| 任务 | 原依赖 |
| --- | --- |
| `common:build:docker` | `build/docker/Dockerfile.server` |
| `common:run:docker` | 同上 |
| `common:setup:docker` | `build/docker/Dockerfile.cross` |
| `common:ios:device:list` | `xcrun`（macOS 专有） |
| `common:ios:run:device` | `build/ios/` + `xcrun` |

根 `Taskfile.yml` 中转发到它们的 `setup:docker` / `build:docker` / `run:docker` 也已一并删除。

> 仍保留 `common:build:server` / `common:run:server`：它们用 `go build -tags server` **纯 Go 编译、不依赖 Docker**，
> 在本机可用（产出 Windows 可执行文件），不属于"坏任务"。

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
