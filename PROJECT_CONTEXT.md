# WeedyBox — 跨工作区项目上下文（精简/机器可读）

> 用途：给**另一个工作区 / 另一个 agent 会话**快速加载本项目现状，避免重新通读源码。
> 本文件是「现状 + 决策 + 坑」的单一入口；`doc/*.md` 是逐函数行号速查（部分已过期，见 §9）。
> 记录时点：git `b621c2c`（main），工作区有 2 个未提交改动（见 §1.2）。所有数字均为**实测**，不是估计。

---

## 1. 项目一句话 + 当前状态

Wails v3 (Go) + Vue 3 + Pinia + UnoCSS 的无边框 Windows 桌面应用，目前只有一个功能域：**TODO 管理**（含批量操作、筛选、统计、双主题、系统托盘）。

### 1.1 技术栈与实测版本

| 层 | 技术 | 版本（实测） |
| --- | --- | --- |
| 桌面框架 | Wails v3 | `go.mod` 声明 `v3.0.0-beta.10`；**本机 CLI 实为 `v3.0.0-beta.20`** |
| 后端 | Go | 1.25.0（go.mod）/ 本机 toolchain 1.26.2 windows/amd64 |
| 数据库 | SQLite | 经 **Wails 内置 `pkg/services/sqlite`**（内部用 modernc.org/sqlite，纯 Go，无 CGO）；`modernc.org/sqlite v1.44.3` 在 go.mod 里是 **indirect** |
| 前端 | Vue / Pinia / vue-router | 3.5.41 / 2.3.1 / 4.6.4 |
| 构建 | Vite / vue-tsc | vite **`~8.2.2`（必须锁死，见 §7.1）** / typescript `~5.4.0` |
| 样式 | UnoCSS | `^66.8.1`（`presetUno` + `presetAttributify`(prefix `un-`) + `presetIcons`） |
| 图标 | lucide via `@iconify/json` | 构建期内联 data URI，**运行时零请求 / 离线可用** |
| 运行时 | Node / npm | v24.15.0 / 11.12.1 |

### 1.2 仓库与工作区状态

- 远程：`origin = https://github.com/WeedyMornbrize3/WeedyBox.git`，分支 `main`。
- 最新提交：`b621c2c 托盘点击改为「只唤回不隐藏」；移除关闭行为里的最小化选项`。
- **未提交改动（2 个文件，属清理性质）**：
  - `frontend/src/components/Sidebar.vue`：上导航顺序由 `首页/测试/TODO` 改为 `首页/TODO/测试`。
  - `internal/database/sql_service.go`：删掉日志里的 emoji（`🔧`/`⚠️`/`✅`）。
- **`frontend/bindings/` 与 `go.sum` 都被 `.gitignore` 忽略且不在版本控制内**（`.gitignore:18` 是 `go.sum`）。
  → 换机器 / 重新 clone 后**必须先**：`wails3 generate bindings -clean=true -ts -i`，否则前端找不到 `HideWindow` 等新方法。
  该命令恒输出 1 条无害警告：`package WeedyBox/internal/database: function types are not supported by encoding/json`。
  实测输出：`Processed: 275 Packages, 3 Services, 23 Methods, 0 Enums, 4 Models, 1 Event`。
- 本机环境坑：未装 Docker；`gh` CLI 已装但**未登录**；`task` CLI **未安装**（用 `wails3 task <name>` 替代）；系统代理 `127.0.0.1:7890` 会偶发拦截 TLS（`SELF_SIGNED_CERT_IN_CHAIN`），重试即可。
- 本机 `devicePixelRatio = 1.25`：**源码里的 `1px`/`2px` 是正确写法**，DevTools 显示 `0.8px`/`1.6px` 只是缩放后的计算值。
- 浏览器里跑 `npm run dev` **没有 Wails 后端**：所有写操作必然失败，只能验证「是否发请求 / 是否回滚 / UI 状态」，**不能验证落库**。

---

## 2. 目录结构（只列实际存在且重要的）

```
main.go                          应用入口：服务注册 / 窗口 / 系统托盘
internal/model/todo.go           Todo · CreateTodoDTO · UpdateTodoDTO · TodoStats
internal/database/sql_service.go SQLiteService：Execute / Query / QueryRow / GetLastInsertID / Transaction + initAllTables
internal/service/todo_service.go TodoService：全部 CRUD + 批量删除 + 统计
internal/service/getMeta/getFromDlsite.go  DLsite 元数据解析（MetaService，package getmeta）
internal/service/getMeta/meta_service.go   Wails 包装层 MetaApiService（搜索/抓取/下载封面到 preImg/）
internal/service/getMeta/testdata/        两个真实作品页 fixture（离线回归用）
internal/service/window_service.go WindowService：最小化 / 最大化 / 关闭 / 隐藏 / 显示 / 可见性
frontend/src/main.ts             Pinia + router + 三个 store 的 restoreState()
frontend/src/router/index.ts     hash 路由，MainLayout 下挂 4 个子页
frontend/src/stores/             todoStore.ts · Sidebar.ts · Theme.ts · App.ts
frontend/src/services/todoService.ts  Wails 绑定封装：todoApi（含 ensure() 兜底 null）
frontend/src/services/metaService.ts  Wails 绑定封装：metaApi（DLsite 搜索/抓取/下载封面）
frontend/src/components/todo/    TodoList · TodoItem · TodoInput · TodoStats · FilterBar · FilterChipGroup · TodoPrioritySelector
frontend/src/components/settings/ CloseActionSelect · ThemeModeSelector · ThemeToggle · LanguageSelect(占位) · SettingsSection/Item/Page · VersionInfo
frontend/src/components/         Sidebar.vue · TitleBar.vue
frontend/src/styles/global.css   三层 token 的**唯一色值来源**（浅色 :root / 深色 [data-theme="dark"]）
frontend/uno.config.ts           变量→类名映射 rules + shortcuts + theme.colors
frontend/scripts/                check-unocss-tokens.mjs · check-build-css.mjs（两个回归闸门）
frontend/bindings/               ⚠️ 自动生成、被 gitignore、勿手改
doc/                             README/backend/frontend/styles/conventions + need-fix.md/need-change.md（后者两个已清空为 0 字节但保留在版本控制）
```

---

## 3. 后端 API 契约（Go → TS）

服务注册在 `main.go:54-58`：`application.NewService(windowService / todoService / dbService)`。

### 3.1 TODO（`TodoService`，全部方法内建 3s `context.WithTimeout`）

| 方法 | 签名 | 说明 |
| --- | --- | --- |
| `Create` | `(ctx, dto CreateTodoDTO) (*Todo, error)` | INSERT 后 `GetLastInsertID` 再 `GetByID` 回读 |
| `GetByID` | `(ctx, id int64) (*Todo, error)` | |
| `GetAll` | `(ctx) ([]Todo, error)` | `ORDER BY completed ASC, priority DESC, created_at DESC` |
| `GetByStatus` | `(ctx, completed bool) ([]Todo, error)` | |
| `Update` | `(ctx, id int64, dto UpdateTodoDTO) (*Todo, error)` | 动态拼 SET；空 updates 直接回读；自动写 `updated_at` |
| `ToggleComplete` | `(ctx, id int64) (*Todo, error)` | 单条 SQL：`CASE WHEN completed = 1 THEN 0 ELSE 1 END` |
| `Delete` | `(ctx, id int64) error` | |
| `DeleteByIDs` | `(ctx, ids []int64) error` | 动态拼 `IN (?,?,…)`，**一次 SQL**；空数组直接返回 |
| `DeleteAll` | `(ctx) error` | |
| `GetStats` | `(ctx) (TodoStats, error)` | `COUNT(*)` + 两个 `SUM(CASE…)`。**`:315` 有一行 `// bug need-fix` 注释但逻辑本身正确** |
| `GetPriorityStats` | `(ctx) (map[int]int64, error)` | 前端当前未使用 |

错误处理统一风格：`errors.Is(err, context.Canceled)` → 「…被取消」、`DeadlineExceeded` → 「…超时」、否则「…失败」。

### 3.2 窗口（`WindowService`，7 个方法，均已生成绑定）

`MinimizeWindow()` · `ToggleMaximizeWindow()` · `CloseWindow()` · `IsWindowMaximised() bool` · `HideWindow()` · `ShowWindow()` · `IsWindowVisible() bool`

> `HideWindow` 与 `MinimizeWindow` 的区别：最小化**仍留任务栏**，隐藏**完全不可见**、只能靠托盘唤回 —— 所以托盘必须在 `main.go` 里创建。

### 3.3 接口测试服务（`internal/service/getMeta/meta_service.go`，`MetaApiService`）

⚠️ **不做数据加工**：返回给前端的 `work` 是 `DLsiteWork` 的 **json 编码**（走 `toRawMap` 一个 `Marshal`+`Unmarshal` 往返，字段名与 json tag 完全一致），前端直接展示原始 JSON。这是刻意的——测试页的用途就是对着 DLsite 页面核对解析结果。

已在 `main.go:57` 注册：`metaService := &getmeta.MetaApiService{}`，`main.go:63` 加入 `applicationServices`（服务总数 4，方法 27）。结构体用 `&MetaApiService{}` 直接构造（不走 `NewMetaApiService()`），因此内部有 `ensure()` 惰性兜底 `meta` 字段。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `FetchWork` | `(ctx, req WorkRequest) (any, error)` | `workId` 优先，否则拿 `keyword`；关键词里含 RJ/RE/VJ 编号则按编号精确抓，否则模糊搜索后抓**第一个**候选。返回 `map[string]any` |
| `SearchWorks` | `(ctx, keyword string) (*searchResponse, error)` | 关键词含编号 → `kind="id"` 且带回 `work` 详情 + 列表候选；否则 `kind="keyword"`，**只返回候选列表不自动抓详情**（由用户决定看哪个） |
| `DownloadCover` | `(ctx, workID, coverURL string) (map[string]any, error)` | 下载封面**原样**存入相对路径 `preImg/`，返回 `{id, dir, path, absPath, bytes, suffix, coverUrl}`；重名加 `-2/-3` 后缀，**不覆盖已有文件** |
| `PreImgDir` | `() string` | `preImg/` 的绝对路径，前端用来显示"存到哪了" |

`WorkRequest{keyword, workId string}` 是前端入参——Wails 绑定不擅长可选参数，所以两个字段都给、服务端自己选。

**图片落地位置**：`const preImgDir = "preImg"` 是**相对路径**，实际落点取决于进程工作目录（开发态从仓库根目录跑 → `WeedyBox/preImg/`）。返回值同时给 `path`（相对，展示用）和 `absPath`（绝对，对照用）。后缀按**真实内容**判定（RIFF/WEBP、JPEG SOI、PNG 签名…）→ `Content-Type` → URL 扩展名 → `.jpg`；硬编码 `.jpg` 是错的，因为存在 `.webp` 封面。

**前端页面**：路由 `/testing`（name `Testing`；旧路径 `/test` 保留为重定向）。视图层 `frontend/src/views/test.vue` 只是个薄壳（与 `TodoPage.vue` → `TodoList.vue` 同一结构），实际内容在 **`frontend/src/components/test/MetaApiTestCard.vue`**。四张卡片：请求 / 精确命中（含封面预览与下载）/ 模糊搜索候选 / 原始 JSON（可复制）。
该页**自带滚动容器**：根元素为 `w-full max-w-4xl mx-auto h-full min-h-0 overflow-y-auto scrollbar-theme pr-2 flex flex-col gap-4 select-text`。原因是 `frontend/src/layouts/MainLayout.vue:6` 的 router-view 是 `overflow-hidden`，页面必须自己提供滚动；而 `frontend/src/styles/global.css:207-213` 用 `!important` 全局隐藏了滚动条，只有 `.scrollbar-theme` 能恢复（`scrollbar-width: thin` + 带 `!important` 的 `::-webkit-scrollbar`）。`pr-2` 给 8px 滚动条留位。实测：内容 1684px / 视口 881px 时 `scrollTop` 可正常设置，无第二层滚动条。

### 3.4 获取外部元数据（`internal/service/getMeta/getFromDlsite.go`，package `getmeta`）

抓 DLsite 作品页。**不依赖第三方 DLsite 库**（`go.felesatra.moe/dlsite` 已从 `go.mod` 移除，原因见 §10），只用 `net/http` + `goquery`。`MetaApiService`（§3.3）是它的 Wails 包装层。

| 方法 | 签名 |
| --- | --- |
| `NewMetaService(options ...MetaOption)` | 默认 `http.Client{Timeout: 20s}`、`tagLimit=24` |
| `WithHTTPClient(*http.Client)` / `WithTagLimit(int)` | 注入客户端 / 限制标签数 |
| `GetWorkByID` | `(ctx, id string) (*DLsiteWork, error)` —— 按 RJ/RE/VJ 取单个作品 |
| `SearchWorkByName` | `(ctx, keyword string) ([]DLsiteSearchItem, error)` —— 每项只有 `ID`/`Name`/`Cover` |
| `GetWorkByName` | `(ctx, keyword string) (*DLsiteWork, error)` —— 搜索取首个再抓详情 |
| `NormalizeDLsiteID` | `(raw string) (string, bool)` —— 抽编号，转大写、**保留前导零** |

`DLsiteWork` 字段（json tag 全 camelCase）：`ID` `Name` `AlternateID`(拼音 slug) `Maker` `MakerID` `Series` `SeriesID` `WorkType` `AgeRating` `FileFormats[]` `FileSize` `ReleaseDate` `Price` `Currency` `PriceUntil` `Rating` `RatingCount` `CoverURL` `Summary` `Tags[]` `Outline[]` `SourceURL` `FetchedAt`

两个真实作品的实测值（**离线 fixture，非猜测**）：

| 字段 | RJ01721474 | RJ01682332 |
| --- | --- | --- |
| 作品名 | 【简体中文版】ELF PARADISE～… | 【簡体中文版】乙女の花園…【CV.涼本あきほ】 |
| 社团 / ID | 青春×フェティシズム / `RG51931` | RaRo / `RG49556` |
| 系列 / 系列ID | `ELF PARADISE` / `SRI0000043774` | 空 / 空 |
| 作品形式 / 年龄指定 | ボイス・ASMR / `R18` | ボイス・ASMR / `全年齢` |
| 文件格式 / 容量 | `[WAV mp3 mp4同梱]` / `16.83GB` | `[WAV MP3同梱]` / 空 |
| 发售日 | `2026-09-26` | `2026-10-07` |
| 价格 / 币种 / 有效期 | `1540` / `JPY` / `2026-10-24T00:00:00+09:00` | `1650` / `JPY` / 空 |
| 评分 / 评价数 | `5` / `16` | `4.81` / `43` |
| 简介长度 / 标签数 | 4799 字符 / 10 个 | 2417 字符 / 10 个 |
| 封面 | `…RJ01721474_img_main_240x240.jpg` | `…RJ01664616_img_main.webp`（页面自带的值） |

`Outline[]` 额外原样带出：`作者` `シナリオ` `イラスト` `声優` `音楽` `その他` `対応言語` `ジャンル` —— 页面新增字段不用改后端。

**拿不到的**：曲目表与试听音频（`ol.work_tracklist_list` 在当前页面已不存在，改由 Vue 渲染）、销量/愿望单/评论数/折扣率、社团与系列的详情页、多语言版本对应关系、评论正文。

**七个坑（改之前必读）**：① 必须 HTTPS；② 前导零丢了必 404；③ 必须发 `Cookie: adultchecked=1; locale=ja`，否则 R18 页内容退化；④ 标题只取 `#work_name` 的文本（现代码里没有 `<a>`，第三方库 `Name` 恒空就是这个原因）；⑤ **价格/评分在 `<meta itemprop="…" content="…">` 的 content 属性上**，取 `.Text()` 只会得到空串；⑥ `SeriesID` 要从内嵌 JSON 正则取且必须判 `FindStringSubmatch` 长度（否则单作 panic）；⑦ **`CoverURL` 目前是最小缩略图**（先命中 `#work_thumb img`），更好的来源是 `og:image` / `itemprop="image"` 与 slider `data-src`，**尚未改**。

**图片可以直接爬（实测）**：`img.dlsite.jp` DNS 干净（Cloudflare `104.18.24.157` / `104.18.25.157`），**不需要 Cookie、不需要 Referer**，`curl --noproxy '*'` 直连即可（PowerShell `Invoke-WebRequest -OutFile` 会因 .NET 默认代理报 `NonInteractive mode`，不要用它）。同一封面 4 个档位：`_img_smp1.jpg` 1470×1000 / 808 KB > `_img_main.jpg` 560×420 / 222 KB（`og:image` 就是它）> `_img_main.webp` 38.9 KB > `_img_main_240x240.jpg` 45.9 KB。

**`CoverURL` 取大图的优先级（已修）**：旧实现扫 `img, source` 时先命中 `#work_thumb img` 的 `_img_main_240x240`，于是永远拿最小那张。现在 `extractCoverURL` 的顺序是 `meta[property="og:image"]` → `meta[itemprop="image"]` → `extractCoverURLWidest`（扫 `img,source`，用 `coverWidthScore` 按后缀宽度挑最大的：显式 `_WxH` 取 W，其次 `_smp`=1470 > `_img_main`=560）。实测两个 fixture 都从 240×240 升到 560×420：`RJ01721474_img_main.jpg` / `RJ01664616_img_main.jpg`。

**测试**：`getFromDlsite_test.go` 用 `testdata/work_RJ01721474.html`(154454 B) / `work_RJ01682332.html`(125550 B) 两个真实页面做**离线 fixture 回归**，断言具体值而非"非空"；两个样本互补（一个有系列名/文件容量/价格有效期，一个都没有），另有 `TestParseWorkRejectsEmptyPage`（结构变化必须报错而非返回空壳）与纯函数表驱动测试。`go test ./internal/service/getMeta/` 约 0.3s 全绿。

### 3.5 数据模型（`internal/model/todo.go`）

```go
type Todo struct {
    ID int64 `json:"id"`; Title string `json:"title"`
    Description string `json:"description,omitempty"`
    Priority int `json:"priority"`   // 0 低 / 1 中 / 2 高
    Completed bool `json:"completed"`
    CreatedAt, UpdatedAt time.Time
}
type CreateTodoDTO struct { Title string; Description string; Priority int }
type UpdateTodoDTO struct { Title, Description *string; Priority *int; Completed *bool }  // 全指针 = 局部更新
type TodoStats struct { Total, Completed, Pending int64 }
```

> `rowToTodo` / `rowToStats` 依赖 SQLite 返回 **`int64` / `time.Time`**，改列类型会 panic。

### 3.6 表结构（`internal/database/sql_service.go:45-53`）

`todos(id INTEGER PK AUTOINCREMENT, title TEXT NOT NULL, description TEXT, priority INTEGER DEFAULT 1, completed INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`

DB 文件 `WeedyBoxDB.db` 建在**可执行文件同目录**（`sqlite.Config{DBSource: "WeedyBoxDB.db"}`）；本机调试 11xx 字节级别。

---

## 4. 前端架构

### 4.1 数据流

```
组件 → stores/todoStore.ts → services/todoService.ts → bindings/…/todoservice.ts
     → Go service.TodoService → database.SQLiteService → SQLite
```

`services/todoService.ts` 导出一个 `todoApi`：`getAll · getByID · create · update · toggleComplete · delete · deleteMany(ids) · getStats`，用本地 `ensure<T>(value, message)` 把绑定层的 `T | null` 收成 `T`。

### 4.2 `stores/todoStore.ts`（Pinia setup store，id `todo`）

- **State**：`todos` · `loading` · `error` · `stats` · `curPriority`(默认 1) · `filters{completed,priority,time}`
- **Gettters**：`completedTodos` · `pendingTodos` · `highPriorityTodos` · `completionRate` · **`filteredTodos`（模板渲染只用这一个）** · `isFilterActive` · `isFilteredEmpty`
- **筛选类型**：`CompletedFilter='pending'|'completed'|'all'`（默认 `'pending'`）、`PriorityFilter=number|null`、`TimeFilter='all'|'today'|'week'|'month'`；选项常量 `COMPLETED_OPTIONS` / `PRIORITY_OPTIONS` / `TIME_OPTIONS` / `TIME_RANGE_DAYS{today:0,week:7,month:30}`
- **Actions**：
  - `loadTodos()`：`Promise.all([getAll, getStats])`
  - `addTodo(title, description?)`：`unshift` + `loadStats`
  - `toggleComplete(id)` / `updateTodo(id, updates)`：就地替换返回对象
  - `deleteTodo(id)`：本地 filter + `loadStats`
  - **`deleteMany(ids)`**：去重 → 记录 `{index,todo}` 快照 → 乐观移除 → 失败时**按原索引升序插回**恢复顺序 → 成功 `loadStats`
  - **`completeMany(ids, completed)`**：只处理「状态与目标不同」的条目，逐条复用 `todoApi.update`，就地替换；失败时 `loadTodos()` 对齐真实状态
  - `loadStats()` · `init()` · `resetFilters()`

### 4.3 其余 store

| store | id | 关键点 |
| --- | --- | --- |
| `stores/Sidebar.ts` | `sidebar` | `barWidth=72`、`barWidthCollapsed=250`；**命名反直觉**：`isCollapsed=true` → 宽度 250。localStorage 键 `sidebar-collapsed` |
| `stores/Theme.ts` | `theme` | `ThemeMode='light'|'dark'|'system'`(默认 system)；`effectiveTheme`；`applyTheme()` 用 `documentElement[data-theme="dark"]`；localStorage `theme-mode`；`setupSystemListener()` 监听 `prefers-color-scheme` |
| `stores/App.ts` | `app` | `CloseAction='ask'|'hide'|'close'`（默认 `ask`）；localStorage `close-action`；`VALID` 数组判定，**非法值/旧值 `minimise` 一律回落 `ask`**；导出 `CLOSE_ACTION_OPTIONS`；`setCloseAction` + `restoreState` + `watch` 兜底落盘 |

> ⚠️ 命名不一致：`Sidebar.ts`/`Theme.ts` 大写开头，`todoStore.ts`/`App.ts` 小写/大写混用 —— 新增 store 时 import 大小写要**手写对**。

### 4.4 services/ 封装层（不是 store，是所有后端调用的唯一入口）

| 文件 | 导出 | 说明 |
| --- | --- | --- |
| `frontend/src/services/todoService.ts` | `todoApi` | `getAll · getByID · create · update · toggleComplete · delete · deleteMany(ids) · getStats`；用本地 `ensure<T>(value, message)` 把绑定层的 `T | null` 收成 `T` |
| `frontend/src/services/metaService.ts` | `metaApi` | `searchWorks(keyword) · fetchWork(keyword, workId?) · downloadCover(workId, coverUrl) · preImgDir()`；同样用 `ensure()` 兜底 null |

> `downloadCover` 的返回值后端是 `map[string]any`，绑定层生成的是 `{ [k: string]?: any }`，需要在封装层 `as DownloadResult` 收窄（否则 `vue-tsc` 报 TS2345）。

### 4.5 TODO 组件公开接口

`TodoItem.vue`（`props: todo, isUpdating?, isDeleting?, selected?`；`emits: toggle[id], delete[id], select[id]`）—— 共 3 个 emit，**没有 `error` emit**。

`TodoList.vue`（编排层，558 行）持有：
- `selectedIds: number[]`、`visibleIds`、`allVisibleSelected`、`toggleSelected`、`toggleSelectAll`、`clearSelection`
- `armed: 'complete' | 'delete' | null`（**互斥**，同一时刻只允许一个按钮待确认）、`completing`、`deleting`
- `uncompleteMode = filters.completed === 'completed'` → `completeActionLabel` 反转为「取消完成」
- 筛选面板 `isFilterOpen`，localStorage 键 `todo-tool-panel`（值是 `'filter'` 或 `''`）；`activeFilterCount`
- 两个关键 `watch`：`watch(selectedIds, disarm)`；**切换任一筛选 → `clearSelection() + disarm()`**（防止「看不见的选中项被批量操作带上」）

---

## 5. 交互模型（当前最终形态，逐项都是用户明确要求过的）

1. **行左侧 checkbox = 「选择」**，不表示完成。选中行呈 `border-brand bg-brand-light`。
2. **「完成」是行右侧独立图标按钮**，位于**删除按钮左侧**：未完成 → `i-lucide-circle`；已完成 → `i-lucide-circle-check-big`；带 `aria-pressed`，**单击即生效、不要求连点**。
3. **单条删除 = 连点两下**：第一下进待确认态（`i-lucide-check` + 红底白字 + `scale-110`），第二下才 `emit('delete')`；`@mouseleave`/`@blur` 自动撤销；提交中显示 `i-lucide-loader-circle spin`，另有 `STUCK_TIMEOUT_MS = 4000` 兜底复位 + `watch(() => props.isDeleting)` 回落。
4. **两个批量按钮**（完成/取消完成、删除全部）也是**连点两下**，仅当 `selectedIds.length > 0` 时出现，均带数量徽标。
5. **切换筛选清空所有选择**（理由见 4.4）。
6. **「已完成」筛选下**批量按钮语义反转成「取消完成」。
7. **行内按钮静息不透明度 `opacity-75`**（不是 0.4）：这是两套主题下同时满足「有意义图标 ≥3:1」的最小值 —— 浅色 3.49~4.01、深色 4.03~4.49；0.4 时仅 1.78:1。提亮交给 `group-hover:opacity-100` / `group-focus-within:opacity-100` / `focus-visible:opacity-100`。
8. **三个批量按钮严格同构**（`.bulk-btn`，只有 accent 语义色不同）：默认透明底 + accent 描边/文字 → hover 填 accent + 反色文字 → 待确认 `.is-armed` 填 accent + 外描边环。accent：完成 `--color-success`、**取消完成 `--color-warning`（「回退」语义与「推进」区分）**、删除 `--color-danger`。反色文字走 `--bulk-on-accent`。数量徽标 `color-mix(in srgb, currentColor 22%, transparent)`。
9. **关闭行为**：标题栏 X 按 `appStore.closeAction` 分流 —— `ask` 弹出标题栏内小面板（**刻意不是模态框**，因为这是两个不同意图的二选一），选项只有「隐藏到托盘」/「退出程序」+「记住我的选择」；`hide`/`close` 直接执行。独立的最小化按钮**不受影响**。
10. **托盘左键 = 只唤回、永不隐藏**（`main.go:102-112`）：`win.IsVisible()` 时直接 `return`；否则 `tray.PositionWindow(win, 0)` + `Show()` + `UnMinimise()` + `Focus()`。不可用 Wails 默认的 `ToggleWindow`（它的语义是「显示则隐藏、否则显示」，会导致手滑再点一下窗口又没了）。隐藏统一走标题栏 X 或设置里的「隐藏到托盘」。

---

## 6. 设计系统（三层 token）

层级：**`global.css` CSS 变量（唯一色值）→ `uno.config.ts` rules（变量→类名）→ shortcuts（组合）**。`uno.config.ts` 里**不允许出现硬编码色值**。

### 6.1 已核验的对比度定稿（WCAG AA 4.5:1，均为实测）

| 组合 | 值 | 对比度 |
| --- | --- | --- |
| 浅色 `text-primary` | `#0f172a` | 17.9:1 |
| 浅色 `text-secondary` | `#334155` | 10.4:1 |
| 浅色 `text-tertiary` | `#475569` | 7.6:1 |
| 浅色 `text-muted` | `#64748b` | 4.8:1（仅辅助小字） |
| 浅色品牌（文字/填充统一） | `#0f766e` | 对白 5.47:1 / 对底 5.25:1 |
| 浅色 success / warning / danger | `#15803d` / `#b45309` / `#dc2626` | 5.02 / 5.02 / 4.83:1 |
| 深色品牌 | `#2dd4bf` | 9.21:1 |
| 深色 success / warning / danger | `#4ade80` / `#fbbf24` / `#f87171` | 9.84 / 10.27 / 6.20:1 |

**关键坑（不要重犯）**：
- 设计系统推荐的 `#0D9488`(teal-600) **作文字只有 3.74:1、白字压其上也只有 3.74:1**，两者都不达标 → 文字/填充统一用 teal-700。
- **深色下 success/brand/danger 都是亮色，白字压上去只有 1.74 / 1.86 / 2.77:1** → 必须翻转成深色文字：`--color-text-on-brand:#04211f`、`--color-text-on-success:#04211f`、`--color-text-on-danger:#2a0a0a`；`--color-check-mark` 同步为 `#04211f`。
- `html,body,#app` 的 `background:#101418` 是**启动底色**（与窗口 `BackgroundColour` 一致，避免启动闪白），**不是主题色**，两套主题下都保持深色；界面底色由 `#app` 内的 `bg-primary` 覆盖。

### 6.2 动效 token（按「变化幅度」分档，不按元素类型）

`--motion-fast:160ms`（颜色/透明度）· `--motion-base:220ms`（位移/缩放）· `--motion-slow:280ms`（展开收起/进出场）；缓动 `--ease-enter:cubic-bezier(0.16,1,0.3,1)`（进入减速）/ `--ease-leave:cubic-bezier(0.4,0,1,1)`（离开加速）。`prefers-reduced-motion: reduce` 下全部压到 0.01ms。

### 6.3 图标尺寸 token（替代混用的 `text-sm`/`text-xl`）

`icon-xs 14px` · `icon-sm 16px` · `icon-md 20px` · `icon-lg 24px` · `icon-xl 32px`（每个含 `w/h` + `text-[Npx]` + `leading-none`）。

### 6.4 半透明 / 颜色类只能用**已定义的具体类名**

rules 是**精确字符串匹配**：`bg-secondary/50`、`text-primary/70`、`w-full-16px`、`will-change-[width]` 全部 **unmatched 且静默丢弃**。需要半透明请用 `bg-secondary-soft` 这类已定义类。

---

## 7. 必须遵守的工程约定（踩过坑才有的）

### 7.1 依赖与版本

- **`frontend/package-lock.json` 必须入库**（`vite` 锁 `~8.2.2`）。忽略它会让 clone 重新解析到 vite **8.3.0**，触发 UnoCSS HMR 回归（上游 [unocss#5331](https://github.com/unocss/unocss/issues/5331)）：**动态导入的路由页（设置/TODO）样式不生效，首页正常**。症状判定：`node -e "console.log(require('./frontend/node_modules/vite/package.json').version)"`。
- **`go.sum` 被 `.gitignore:18` 忽略且不在版本控制内** —— 首次构建需联网由 `go mod tidy` 重新生成。
- **全程零新增依赖**：`frontend/package.json`、根 `package.json`、`go.mod` 自设计系统重构以来未变。
- **桌面端离线优先**：不引联网字体（Plus Jakarta Sans 只在 font-family 里作为可选首选项，实际回退到系统栈），图标走 `presetIcons` 构建期内联。

### 7.2 UnoCSS 七类静默失效（本项目的核心坑，改样式前必读）

1. **`border-color` 不能塞简写值**：`{'border-color':'1px solid X'}` → `.border-x{border-color:1px solid X}` 非法 CSS、整条作废。
2. **只设 `border-color` 等于没有边框**：preflight 把 `*,::before,::after` 重置为 `border-width:0;border-style:solid`；若元素另有非零 `border-width`，`border-style` 会退回 UA 初始值 **`outset`** → 渲染成 1.6px 的 3D 伪立体边框。**结论：每条 border-* 规则必须自带 width+style+color，一律用 `border` 简写。**
3. **`border` 工具类只输出 `border-width`**：`<button>` 的 UA 默认 `border-style` 是 `outset` → 按钮上必须写 `border border-solid`。
4. **同一元素上两条改同一属性的类只保留一条**：模板同时写 `border-l-theme-light` 与 `border-l-brand` 时后者**根本不会生成** → 基础样式放组件 `<style scoped>`，UnoCSS 只负责 `group-hover:border-l-brand`。
5. **shortcuts 输出在 rules 之前；rules 之间按字母序（同权重后者胜）**：`.text-danger` 排在 `.text-white` 之后会压掉白字 → **同一属性只由一处决定**。
6. **过长的 shortcut 链会被静默截断**（十余条 utility 串成一个 shortcut 后只剩两条，`width`/`appearance`/焦点环全丢）。
7. **没有 `placeholder-<color>` 工具类**：`placeholder-tertiary` 会被静默丢弃，只在构建时留一行 `[unocss] unmatched utility "placeholder-tertiary" in shortcut "input-theme"` 警告，元素上完全看不出异常。正确写法是 `placeholder:text-tertiary` 变体（`input-theme` 已修）。

**单侧边框规则的正确写法**：必须**显式把其余三边归零**，否则整框简写会重置四边。现有三条：`border-b-theme-light`、`border-r-sidebar`、`border-l-brand`。

**输入框有两套实现，CSS 必须成对改（改一个就改另一个）**：

| | `input-theme`（`uno.config.ts:190` shortcut） | `TodoInput.vue` 复合输入框 |
| --- | --- | --- |
| 结构 | `<input class="input-theme">`，盒模型全在 shortcut 里 | 外层壳 `div` 承边框/圆角/底色，内层无边框 `input` |
| 边框 | `border-theme` + `focus:border-brand` | 壳 `border-theme` + `focus-within:border-brand` |
| 用于 | 测试页搜索框（`components/test/MetaApiTestCard.vue:24`） | TODO 添加框（`components/todo/TodoInput.vue`） |

**控件高度统一走 `control-h`**（`uno.config.ts:161`，`h-[2.2rem]` = **35.2px**）：`btn-brand`（`uno.config.ts:164`）、`input-theme`、`TodoInput.vue` 的外壳三处都挂它，因此三者同高。⚠️ **挂了固定高度就不要再写纵向 padding** —— 总高已由 `control-h` 决定，`py-*` 不会再改变盒高，只会把内容挤出盒子（`TodoInput.vue` 内层 input 因此不带 `py-*`，垂直居中交给外壳的 `items-center`）。历史教训：输入框曾用 `py-2.5`、按钮用 `py-2`，高度由「内容高 + 纵向 padding」撑出，实测 41.6px vs 35.2px，同一行 form（`items-end`）底部对齐时顶部错开 6.4px。

两者实测盒模型**逐项相同**：`border-radius: 10px`、`padding: 10px 12px`、`font-size: 14px`、`line-height: 20px`、`border: 0.8px solid var(--color-border)`、总高 **35.2px**；聚焦时边框都变 `var(--color-brand)`。**任一项（内边距 / 圆角 / 字号 / 行高 / 边框色 / 聚焦色 / 高度）都必须在两处同步修改**，否则两个输入框的圆角与高度会分叉。`TodoInput.vue` 顶部与 `MetaApiTestCard.vue` 分别写了互相指向的注释。

**提交按钮统一走 `btn-brand`**（`uno.config.ts:164`）：TODO 的「添加」与测试页的「搜索」类串同为 `btn-brand flex items-center gap-1.5 shrink-0`，TODO 额外带 `disabled:opacity-50 disabled:cursor-not-allowed`（短输入要能看出来不可点，`btn-brand` 本身不含禁用态）。`HomePage.vue:10` 也用它 —— 即全站按钮已统一到这一个 shortcut，不要再手写 `rounded-lg px-4 py-2 ...`。

### 7.3 布局高度链（缺一环就整页溢出）

`presetUno` **不带** `border-box` reset → `global.css:15-19` 手写 `*,*::before,*::after{box-sizing:border-box}`；再加 `html,body,#app{height:100%;overflow:hidden}`，沿链 `h-screen` / `min-h-0` / `flex-1` / `overflow-hidden`。实测：加 `border-box` 后 root 高度 1048→1018，列表底部从「超出视口 36px、最后一条滚不到」变为完整落在视口内。

滚条：全局 `::-webkit-scrollbar{display:none!important}` 隐藏；局部用 `.scrollbar-theme` 恢复（**必须也带 `!important`**，因为「带 `!important` 的声明永远胜过不带的」，与选择器权重无关）。滚动条伪元素只能写在纯 CSS 里 —— UnoCSS utility 无法生成 `::-webkit-scrollbar` 这种伪元素选择器。

### 7.4 代码约定

- Go 方法**首字母大写**才会导出到前端绑定。
- **`frontend/bindings/` 只读**，改 Go 签名后重新生成。
- Go 返回指针/切片 → 前端绑定类型是 `T | null`，统一在 `services/todoService.ts` 的 `ensure()` 兜底。
- 所有 `TodoService` 方法内建 3s `context.WithTimeout`，并区分 `Canceled` / `DeadlineExceeded` 文案。
- 前端 store 文件命名不统一（见 4.3），import 大小写手写。
- 用户明确要求：**`doc/need-fix.md` 与 `doc/need-change.md` 可以清空内容，但文件必须保留在版本控制中**（当前均为 0 字节）。

---

## 8. 构建 / 验证工作流

### 8.1 命令

| 命令 | 说明 |
| --- | --- |
| `wails3 dev` | 开发模式（装依赖 → 生成绑定 → 生成图标 → 构建后端 → Vite 端口 **9245** `strictPort` → 开窗口） |
| `wails3 build` | 生产构建 → `bin/weedybox.exe` |
| `wails3 package` | NSIS 安装包 → `bin/WeedyBox-amd64-installer.exe` |
| `wails3 task <name>` | 等价 `task <name>`（本机未装独立 `task` CLI） |
| `cd frontend; npm run dev` | 只跑前端（**无后端**，只能调样式） |
| `cd frontend; npm run build` | `vue-tsc && vite build --mode production` |
| `cd frontend; npm run check:tokens` | `node scripts/check-unocss-tokens.mjs` |
| `cd frontend; npm run check:build` | `node scripts/check-build-css.mjs`（**先 `vite build` 再跑**） |
| `go build ./...` · `go vet . ./internal/...` | Go 侧静态检查 |
| `npx vue-tsc` | 前端类型检查 |

### 8.2 两个回归闸门（改动样式后必跑）

- `frontend/scripts/check-unocss-tokens.mjs`：逐个 `uno.parseToken` 打印解析结果；刻意保留 `bg-secondary/50`、`will-change-[width]` 两个「已知应失效」样例作对照。
- `frontend/scripts/check-build-css.mjs`：断言构建产物 —— 关键类是否生成（约 54~58 个）、无 `border-color:1px solid` 污染、17 项布局链与无障碍断言、**边框规则自带三件套**、**单侧边框置零其余边**、「同侧边框类叠加」扫描、「border 缺 border-style」扫描、**搜索框/表单组**（`input-theme` 底座 + 聚焦两条 border 规则、`::placeholder{color:var(--color-text-tertiary)}` 存在、反向断言无 `.placeholder-<color>` 无效类、`.input-theme` 含 border 的规则必须恰好 2 条）、**控件高度一致性组**（`control-h`/`input-theme`/`btn-brand` 三条规则都必须含 `height:2.2rem` 且**不得带纵向 padding**；扫 `TodoInput.vue` 源码确认外壳挂了 `control-h`、内层 `input` 没写 `py-*`）。
  注意：**断言压缩后的 CSS 不能拿带 `{}` 的整串去 `includes`** —— minify 会把选择器分组写成 `.input-theme:focus,[input-theme=""]:focus{…}`，必须按「规则块」切分（`css.match(/[^{}]*\.input-theme[^{}]*\{[^}]*\}/g)`）再判断。
- 已知噪音：产物 JS 里的 `⚠️` 来自 `@wailsio/runtime` 的控制台提示，**不是界面 emoji**。

### 8.3 调试手段与测试教训

- **用 chrome-devtools MCP 读 `getComputedStyle` 的计算样式**，不要只看源码；可一次确认有无 emoji、图标尺寸是否精确命中、token 是否按主题解析、间距节奏。
- **Vite dev server 会提供陈旧的 CSS/模块转换缓存**（曾服务 9528 字符 vs 实际 11942，导致浏览器一直解析旧 token、反复误判「代码没生效」）→ **删 `frontend/node_modules/.vite` 并重启 dev server**。判断手段：枚举 `document.styleSheets` 命中的规则 + 直接 `fetch` dev server 该文件比对长度。
- 注入测试数据：
  ```js
  const pinia = document.querySelector('#app').__vue_app__.config.globalProperties.$pinia
  const st = pinia._s.get('todo')
  // 测完必须清空，避免污染用户的 SQLite：
  st.$patch({ todos: [], stats: { total: 0, completed: 0, pending: 0 } })
  ```
- 反复踩到的测试坑：Vue DOM 更新异步（派发事件后必须 `await`）；合成事件测 `:checked` 会与手动赋值叠加导致计数翻倍；`:hover` 伪类无法用合成事件触发；颜色过渡中途取样会读到中间值（已填充却读到 `transparent`），需等 ≥500ms。
- **沙箱限制**：`spawn EPERM`（不允许 Node 捕获子进程输出）；PowerShell 中文输出有 mojibake（判断文件内容必须用 read 工具）；PowerShell 不支持 `\u{...}` 正则与 here-string `<<<`。

---

## 9. 已知问题与文档陈旧点（接手前先看这里）

### 9.1 未完成 / 待办

| 位置 | 状态 |
| --- | --- |
| `frontend/src/components/settings/LanguageSelect.vue` | 仅占位，i18n 未实现 |
| `frontend/src/components/todo/TodoItem.vue` | 描述**只读展示，尚不能编辑**（后端 `Update` 已支持 `description`） |
| `frontend/src/views/test.vue` | **已不是占位页**，且已是薄壳（只渲染 `MetaApiTestCard`）；DLsite 接口测试卡片在 `frontend/src/components/test/MetaApiTestCard.vue`（路由 `/testing`，见 §3.3） |
| `internal/service/todo_service.go:315` | 一行 `// bug need-fix` 注释（`GetStats`），逻辑本身正确，属陈旧标记 |
| `main.go:62` | `Description` 仍是 `"A demo of using raw HTML & CSS"` |
| `build/config.yml` | 仍是模板（`companyName="My Company"`、`productName="My Product"`、`version 0.0.1`） |

### 9.2 需**实机**确认（无头浏览器验证不到）

- **托盘**：托盘图标是否出现在通知区域、左键是否只唤回不隐藏、窗口已显示时再点托盘应无反应、右键菜单「显示主窗口/退出程序」是否可用 —— 浏览器观察不到 Windows 通知区域。
- **DLsite 测试页（`/testing`）**：浏览器里没有 Wails 后端，`/wails/runtime` 一律 404，因此**所有真实调用都必须实机验证**（用户明确说「由于网络限制因素 不需要测试 完成后我手动测试」，故本地未跑任何联网调用）。需要确认的点：
  1. 按 `RJ01721474` 精确抓取 → 元数据是否与页面一致、`coverUrl` 是否已是 `_img_main.jpg`（560×420 那一档，不再是 240×240）；
  2. 封面在 webview 里能否直接显示（`Referer` 已设为 `https://www.dlsite.com/`，`<img referrerpolicy="no-referrer">`）；
  3. 点「下载封面」是否真的在**工作目录下**生成 `preImg/`，文件名形如 `RJ01721474_cover.jpg`；卡片里显示的是 `absPath`，以那个为准；
  4. 模糊搜索（如 `エルフ パラダイス`）是否返回候选列表；
  5. 「打开已保存的图片」的 `file:///` 链接在 Wails webview 里是否可用（**把握最低的一项**，不可用不影响主流程）。

### 9.3 `doc/*.md` 的已确认错误（本文件 §5/§6/§7 优先于它们）

- `doc/styles.md`「无效工具类」表**有两行是错的**：`top-50%` 与 `right-2` **实际合法生成**（旧构建产物 CSS 里也有这两条规则）。真正 unmatched 的是 `w-full-16px`、`bg-secondary/50`、`text-primary/70`、`border-theme/50`、`will-change-[width]`。
- `doc/frontend.md` 称 `desc-hint` 是 textarea 的 flex 同级元素 —— 模板里**根本不存在**（只在 `<style scoped>` 里，是死代码）。
- `doc/frontend.md` 提到 MainLayout 有 `overflow-y-auto` —— 实际 `frontend/src/layouts/MainLayout.vue:6` 只有 `overflow-hidden`。
- `doc/conventions.md` 称 `global.css:8` 的 body 背景硬编码深色 —— 实际在 `global.css:29`（且是设计上的启动底色，非缺陷）。
- `doc/conventions.md` 与 `README.md` 称数据库走 `modernc.org/sqlite` —— 实际走 **Wails 内置 `pkg/services/sqlite`**，modernc 只是传递依赖（indirect）。
- `doc/conventions.md` 的「扫描范围」列表里 `vendor/`、`frontend/build/`、`build/{android,ios,linux,docker}/`、`.wails/`、`wails-build/`、`logs/`、`.vscode/`、`.idea/`、`dist/` **本机均不存在**；实际被忽略且存在的只有 `node_modules/`、`bin/`、`.task/`、`frontend/dist/`、`frontend/node_modules/`、`frontend/bindings/`、`build/darwin/`、根 `WeedyBoxDB.db`、`go.sum`。
- `README.md` 与 `doc/conventions.md` 称 `go.sum` 被忽略 —— **正确**（早前「go.sum 已被跟踪」的判断是错的，实测 `git ls-files go.sum` 为空）。

---

## 10. 决策日志（为什么是现在这样，改动前请先读）

| 决策 | 理由 |
| --- | --- |
| 关闭行为只有 `ask/hide/close`，**没有 `minimise`** | 标题栏左侧已有独立最小化按钮，放在关闭行为里是重复入口，且让「关闭」这个决定含糊 |
| 接口测试服务**不做数据加工**，`work` 原样返回 `DLsiteWork` 的 json | 测试页的用途就是「对着 DLsite 页面核对解析结果」。任何二次加工（改字段名、拍平、只挑几个字段）都会让「后端解析错了」和「展示层挑错了」变得无法区分。格式化交给前端 `JSON.stringify(raw, null, 2)` |
| `preImg` 用**相对路径**而不是 `os.UserConfigDir()` 之类的绝对路径 | 用户明确要求「保存在工作区目录下的 preImg 使用相对路径」。代价是落点随进程工作目录变化（开发态是仓库根），所以返回值同时给 `path`（相对）与 `absPath`（绝对），界面上直接显示 `absPath`，不留"到底存哪了"的歧义 |
| 下载封面**不覆盖**已有文件，同名加 `-2/-3` 后缀 | 测试页会被反复点击，覆盖会把上一次下到的图悄悄冲掉；`O_EXCL` 还能顺带防止两次并发请求写同一个文件 |
| 托盘左键**只唤回不隐藏** | Wails 默认 `ToggleWindow` 是「可见则隐藏、否则显示」，会让「唤回」后手滑再点一下窗口又消失 |
| 关闭询问用**标题栏内小面板**而非模态框 | 这是「两个不同意图的二选一」，用模态打断不值得 |
| 行左侧 checkbox 改为**选择**、完成移为行右侧独立按钮 | 原设计里「勾选」同时承担完成与选择，语义冲突；且旧版勾选框 `w-0 h-0` 完全隐藏、键盘不可操作 |
| 批量按钮**连点两下确认**、且 `armed` 互斥 | 用户明确要求「不展开面板、直接执行但需连点两下」；互斥是防止「以为在确认完成、其实点的是删除」 |
| 「选择范围」面板**整套移除** | 用户要求批量按钮直接执行、不展开面板 |
| 弃用 `go.felesatra.moe/dlsite`，自写 `internal/service/getMeta/getFromDlsite.go` | 该库的 `Work.Name` **恒为空**：它只找 `#work_name` 里的 `<a>`，而现代码没有 `<a>`；且它用裸 `http.Get` 无超时、不发成人确认 Cookie、`TrackList` 因选择器已失效也恒为空。自写版与直接 `net/http`+`goquery` 相比没有额外依赖，反而能删掉一整个模块树 |
| DLsite 解析器用**离线 fixture** 测试 | 本机 `www.dlsite.com` DNS 被污染（解析到 Meta 的 IP 段），真实抓取每次卡 21 秒还可能失败；把两个真实页面存进 `testdata/` 后 `go test` 0.3 秒稳定通过，且改解析代码后立刻知道有没有弄坏 |
| **切换筛选清空选择** | 筛选会换掉可见条目，残留旧选择会让「看不见的选中项被批量操作带上」，在「已完成」语义反转下极易误操作 |
| 「取消完成」用 warning 橙 | 与「推进」（success 绿）在语义上区分「回退」 |
| 行内按钮静息 `opacity-75` 而非 0.4 | 0.4 时图标对比度仅 1.78:1；0.75 是两套主题同时 ≥3:1 的最小值 |
| 优先级溢出修复：`TodoStats` 改 `flex-1 min-w-0` | 原 `flex-shrink-0` 让统计栏（约 271px）与优先级选择器（172px）都拒绝收缩，容器 <455px 时溢出而统计栏自身的 `flex-wrap` 永远得不到触发；实测溢出区间 380~452px、越界最多 75px，修复后 320–660px 全范围零溢出 |
| 描述块基础左竖线放 scoped 样式、hover 变色放 UnoCSS | 同一元素两条改 `border-left` 的类会被 UnoCSS 只保留一条（见 §7.2.4） |
| emoji 全面换 lucide | emoji 跨平台渲染不一致、无法用主题变量控色；lucide 经 `presetIcons` 构建期内联，离线可用且零运行时请求 |
| 不引外部字体 | 桌面端离线优先 |
| 图标尺寸提为 5 档 token | 原代码混用 `text-sm`/`text-xl` 控图标大小，语义错位 |

---

## 11. 相关提交（自设计系统重构起，倒序）

`b621c2c` 托盘点击只唤回不隐藏；移除关闭行为里的最小化 ·
`81d25bc` 新增「隐藏到托盘」+ 系统托盘图标 ·
`1bd902a` 标题栏 X 首次点击询问，可在设置页修改 ·
`be098be` 切换筛选清空选择；统一三个批量按钮风格 ·
`30b6305` 修复完成按钮伪立体边框；边框宽度统一整数 px；补回归扫描 ·
`701b4e2` 完成按钮 UI 对齐删除按钮；「已完成」筛选下语义反转为取消完成 ·
`6d4793f` 交互模型调整：选择与完成分离，批量收敛为两个按钮 ·
`b8c3df8` 新增「删除范围」面板（后已移除） ·
`f159ca3` UI 排版调整：极简取向 + 统一过渡动效 + 统计/筛选按需显现 ·
`21607b0` 修复描述块 hover 出现方框 ·
`e6a42bf` 修复 need-fix / need-change ·
`23f9770` 修复边框不显示 ·
`aa8f800` 设计系统重构：Flat Design + Teal 主题，图标全面替换 emoji
