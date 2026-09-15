<div align="center">

# WeedyBox

**Wails v3 + Vue 3 + UnoCSS + SQLite 桌面应用**

想到什么做什么，目前只做了 TODO，当作练手与学习。

</div>

---

## 项目简介

WeedyBox 是一个基于 **Wails v3** 的跨平台桌面应用（当前**只构建 Windows**）。
后端 Go 直连本地 SQLite（`modernc.org/sqlite`，**纯 Go 实现，无需 CGO**），
前端 Vue 3 + Pinia + UnoCSS，界面为无边框窗口 + 自绘标题栏。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 桌面框架 | Wails v3（`v3.0.0-beta.10`） |
| 后端 | Go 1.25+ |
| 数据库 | SQLite（`modernc.org/sqlite`，无 CGO） |
| 前端 | Vue 3.5 + Vue Router 4 + Pinia 2 |
| 样式 | UnoCSS 66（`presetUno` / `presetAttributify` / `presetIcons`） |
| 构建 | Vite 8 + vue-tsc |

## 功能

- **TODO 管理**：新增 / 完成切换 / 删除，支持描述（可多行）
- **优先级**：低 / 中 / 高，列表按 优先级 ⭢ 创建时间 排序
- **筛选栏**：按 完成状态 / 优先级 / 创建时间 过滤，默认只显示**未完成**；横向排列、可横向滚动
- **统计**：总计 / 已完成 / 待完成 / 完成率
- **主题**：浅色 / 深色 / 跟随系统，配置持久化到 `localStorage`
- **侧边栏**：可折叠，宽度带过渡动画
- **窗口**：无边框 + 自绘标题栏（最小化 / 最大化 / 关闭）

## 环境要求

| 依赖 | 版本 | 说明 |
| --- | --- | --- |
| [Go](https://go.dev/dl/) | **1.25+** | `go.mod` 要求 `go 1.25.0` |
| [Node.js](https://nodejs.org/en/download/) | 20+（开发用 v24） | 需带 npm |
| [Wails3 CLI](https://v3.wails.io/quick-start/installation/) | `v3.0.0-beta.20` | 见下方安装步骤 |
| [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/) | — | **仅 Windows 需要**；Win11 通常已内置，可用 `wails3 doctor` 检查 |

> 确保 `%USERPROFILE%\go\bin` 已加入 `PATH`（Wails3 CLI 会安装到那里）。
> 验证：`$env:PATH -split ';' | Where-Object { $_ -like '*\go\bin' }`

## 安装

### 1. 安装 Wails3 CLI

```powershell
go install -v github.com/wailsapp/wails/v3/cmd/wails3@latest
```

如需与本项目开发环境一致的版本：

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.20
```

安装后**重开终端**，然后检查环境：

```powershell
wails3 doctor
```

### 2. 克隆并安装依赖

```powershell
git clone https://github.com/WeedyMornbrize3/WeedyBox.git
cd WeedyBox

# 前端依赖
cd frontend
npm install
cd ..
```

> Go 侧依赖（含 Wails 模块）会在首次构建时由 `go mod tidy` 自动拉取，无需手动执行。

## 运行

### 开发模式（推荐，前后端热重载）

```powershell
wails3 dev
```

会自动完成：装前端依赖 → 生成前后端绑定（bindings）→ 生成图标 → 构建后端 → 启动 Vite 开发服务器（默认端口 **9245**）→ 打开桌面窗口。

### 生产构建

```powershell
wails3 build
```

产物：**`bin/weedybox.exe`**

### 打包安装程序

```powershell
wails3 package
```

产物：`bin/WeedyBox-amd64-installer.exe`（NSIS 安装包）

### 只跑前端（调试 UI 用）

```powershell
cd frontend
npm run dev
```

浏览器打开 `http://127.0.0.1:9245`。

> ⚠️ 此时**没有 Wails 后端**，涉及数据库的请求会失败（列表为空），只能用于调样式与布局。

## 常用命令

| 命令 | 说明 |
| --- | --- |
| `wails3 dev` | 开发模式（热重载） |
| `wails3 build` | 生产构建 → `bin/weedybox.exe` |
| `wails3 package` | 打包 Windows 安装程序 |
| `wails3 doctor` | 检查环境依赖 |
| `wails3 task --list` | 列出所有可用任务 |
| `cd frontend; npm run dev` | 只跑前端 Vite |
| `cd frontend; npm run build` | 前端构建（含 `vue-tsc` 类型检查） |

> 本项目内置了 task 运行器，若本机没装独立 `task` CLI，用 `wails3 task <name>` 即可。

## 项目结构

```
WeedyBox/
├── main.go                  # 应用入口：注册服务、创建窗口
├── go.mod / go.sum          # Go 依赖
├── Taskfile.yml             # 构建任务（已裁剪为仅 Windows）
├── build/                   # 构建资源与各平台 Taskfile
│   ├── config.yml           # 应用元信息 / dev 配置
│   └── windows/             # Windows 图标、manifest、NSIS 脚本
├── internal/
│   ├── model/todo.go        # 数据结构（Todo / DTO / Stats）
│   ├── database/            # SQLite 通用执行层
│   └── service/             # 业务服务（Todo / Window）
├── frontend/
│   ├── src/
│   │   ├── views/           # 页面（首页 / 设置 / TODO / 测试）
│   │   ├── components/      # 组件（todo / settings）
│   │   ├── stores/          # Pinia（todo / sidebar / theme）
│   │   ├── services/        # Wails 绑定封装
│   │   ├── layouts/         # 布局骨架
│   │   ├── router/          # 路由表
│   │   └── styles/          # CSS 变量主题
│   ├── bindings/            # ⚠️ 自动生成，勿手改
│   └── uno.config.ts        # UnoCSS 规则与 shortcuts
└── doc/                     # 源码文档（函数位置速查）
```

数据库文件 `WeedyBoxDB.db` 会在首次运行时于**可执行文件同目录**创建，表结构由 `internal/database/sql_service.go` 的 `initAllTables` 自动初始化。

## 文档

`doc/` 目录下有按模块整理的源码速查文档：

| 文件 | 内容 |
| --- | --- |
| [`doc/README.md`](doc/README.md) | 索引、目录速查、数据流 |
| [`doc/backend.md`](doc/backend.md) | Go 后端：模型 / 数据库 / 服务函数表 |
| [`doc/frontend.md`](doc/frontend.md) | 前端：路由 / store / 组件 / API |
| [`doc/styles.md`](doc/styles.md) | CSS 变量、UnoCSS 规则与 shortcuts |
| [`doc/conventions.md`](doc/conventions.md) | 构建约定、命名规范、已知问题 |

## 已知问题

| 位置 | 说明 |
| --- | --- |
| `internal/service/todo_service.go` | `GetStats` 处有 `bug need-fix` 标记 |
| `frontend/src/components/settings/LanguageSelect.vue` | 语言选择仅占位，未实现 |
| `frontend/src/components/todo/TodoItem.vue` | 描述只读展示，暂不支持编辑（后端 `Update` 已支持 `description`） |
| `frontend/src/views/test.vue` | 测试页，可删 |
| `main.go` | `Description` 仍为 Wails 模板默认值 |

## 排错

**`open build/darwin/icon.icns: The system cannot find the path specified`**（或在 `build/darwin/icons.icns`）

`wails3 generate icons` **没有「只生成 Windows」的开关**——它总会写一个 mac 图标，区别只在写到哪里：

| 写法 | 实际落点 |
| --- | --- |
| 传 `-macfilename darwin/icons.icns` | `build/darwin/icons.icns` ✓ |
| 不传 `-macfilename` | CLI 内部默认值 `build/darwin/icon.icns`；任务以 `dir: build` 运行，会去找 `build/build/darwin/`，**必然失败** |

而 `generate:icons` 是 `windows:build` 的依赖，所以它一失败，**`wails3 build` 与 `wails3 dev` 全都起不来**。

本仓库已处理：图标任务显式传 `-macfilename darwin/icons.icns` 并先 `mkdir -p darwin`，
同时用 **`build/darwin/.gitkeep`** 保证该目录在全新 clone 后依然存在。

> 如果你把 `build/darwin/` 整个删掉且没有 `.gitkeep`，构建就会报这个错。
> 解决：`mkdir build/darwin && touch build/darwin/.gitkeep`，或从仓库恢复该文件。

**`wails3: command not found`**

Wails3 CLI 装在 `%USERPROFILE%\go\bin`，确认该目录在 `PATH` 中，并**重开终端**。

**首次构建要联网**

`go.sum` 被 `.gitignore` 忽略（不在仓库里），首次构建时由 `go mod tidy` 自动重新生成；
`npm install` 也需要联网下载依赖。之后即可离线构建。

**设置 / TODO 页样式不生效（只有首页正常）**

这是 `vite` 版本漂移导致的 UnoCSS HMR 回归（上游 [unocss#5331](https://github.com/unocss/unocss/issues/5331)）：
动态导入的路由页新出现的工具类无法送达浏览器（服务端已生成，浏览器拿不到）。

**本项目已修复**：`frontend/package.json` 把 `vite` 锁为 `~8.2.2`，且 **`frontend/package-lock.json` 已纳入 git**，
正常 `npm install` 会精确装到 8.2.2。

若仍遇到，请核对版本：

```powershell
node -e "console.log(require('./frontend/node_modules/vite/package.json').version)"
```

- 若是 **8.3.x** → 删除 `frontend/node_modules` 后重新 `npm install`（lock 已锁版本）。
- 若已是 **8.2.2** 仍异常 → 完整刷新页面；dev server 有内存缓存，必要时重启 `wails3 dev`。

> 注意：**不要把 `frontend/package-lock.json` 加进 `.gitignore`**，否则版本会漂回 8.3.x 并复现该问题。

**列表能显示但数据为空 / 前端报错**

只跑 `npm run dev` 时没有 Wails 后端，属正常现象，请用 `wails3 dev`。
