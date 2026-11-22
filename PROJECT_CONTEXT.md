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
  实测输出：`Processed: 281 Packages, 7 Services, 45 Methods, 0 Enums, 15 Models, 1 Event`。
- 本机环境坑：未装 Docker；`gh` CLI 已装但**未登录**；`task` CLI **未安装**（用 `wails3 task <name>` 替代）；系统代理 `127.0.0.1:7890` 会偶发拦截 TLS（`SELF_SIGNED_CERT_IN_CHAIN`），重试即可。
- 本机 `devicePixelRatio = 1.25`：**源码里的 `1px`/`2px` 是正确写法**，DevTools 显示 `0.8px`/`1.6px` 只是缩放后的计算值。
- 浏览器里跑 `npm run dev` **没有 Wails 后端**：所有写操作必然失败，只能验证「是否发请求 / 是否回滚 / UI 状态」，**不能验证落库**。

---

## 2. 目录结构（只列实际存在且重要的）

```
main.go                          应用入口：服务注册 / 窗口 / 系统托盘
internal/model/todo.go           Todo · CreateTodoDTO · UpdateTodoDTO · TodoStats
internal/database/sql_service.go SQLiteService：Execute / Query / QueryRow / GetLastInsertID / Transaction + initAllTables（**只建 todos**，游戏库表惰性创建）
internal/database/library_repo.go 游戏库持久层：ensureLibraryTables（惰性建表 + 给旧库补 game_path 列）/ Find / List / Upsert / Delete / RecordCoverFile / GetGamePath / SetGamePath
internal/database/library_repo_test.go 游戏库仓储测试（惰性建表、先读先删、并发首用、来源隔离、封面指针、game_path 同表与旧库迁移）
internal/service/todo_service.go TodoService：全部 CRUD + 批量删除 + 统计
internal/service/library_service.go LibraryService：**缓存优先**取作品 / 列游戏库 / 删除 / 本地封面转 data URL
internal/service/dialog_service.go DialogService：系统「选择文件夹」/「选择游戏程序（.exe）」对话框（Wails v3 `app.Dialog.OpenFile()`）
internal/service/scan/            本地扫描（package `scan`，独立子包；**已注册为 Wails 服务 `ScanService`**）
internal/service/scan/scan.go     ScanService：一级子文件夹内的 .exe（通用）/ 一级子文件夹名+路径（特定）/ 从文件夹挑主程序 PickGameExecutable
internal/service/scan/steam*.go   Steam 游戏目录：steam.go 多库汇总 + steam_windows.go 注册表 / steam_other.go 非 Windows 空实现（build tag）
internal/service/scan/*_test.go   扫描边界（只读一层/不递归/坏输入/取消）、vdf 解析、多库汇总去重
internal/service/game/           游戏启动（package `game`，独立子包；**已注册为 Wails 服务 `GameService`**）
internal/service/game/game_service.go  GameService：Status（位置/exe 是否存在/是否运行 + 运行缓存）/ SetPath（校验并记录 exe 位置）/ Launch（按库里位置起独立进程）
internal/service/game/launch_windows.go / launch_other.go  起进程的平台实现（DETACHED_PROCESS，工作目录=exe 目录，返回 PID）
internal/service/game/process_windows.go / process_other.go  运行状态检测（Toolhelp 快照 + QueryFullProcessImageName 比对完整路径；非 Windows 恒 false）
internal/service/getMeta/getFromDlsite.go  DLsite 元数据解析（MetaService，package getmeta）
internal/service/getMeta/meta_service.go   Wails 包装层 MetaApiService（搜索/抓取/封面下载：入库图落 covers/（优先 webp）、搜索缩略图落 cache/，下载后记账到库）
internal/service/getMeta/cover_store.go    封面存储：covers/ 目录、preImg→covers 启动迁移（目录+库路径）、优先探测同名 .webp 下载
internal/service/getMeta/testdata/        两个真实作品页 fixture（离线回归用）
internal/service/window_service.go WindowService：最小化 / 最大化 / 关闭 / 隐藏 / 显示 / 可见性
frontend/src/main.ts             Pinia + router + store 的 restoreState()
frontend/src/router/index.ts     hash 路由，MainLayout 下挂 7 个子页（/ · /library · /works · /todo · /testing · /settings + /test 重定向）
frontend/src/stores/             todoStore.ts · metaStore.ts · libraryStore.ts · scanStore.ts · autoHeightStore.ts · Sidebar.ts · Theme.ts · App.ts
frontend/src/views/WorksPage.vue 作品库页面（**侧栏无入口**，只从游戏库点封面进入；见 §3.3）
frontend/src/services/todoService.ts  Wails 绑定封装：todoApi（含 ensure() 兜底 null）
frontend/src/services/metaService.ts  Wails 绑定封装：metaApi（**永远联网**：搜索/抓取 + 缩略图缓存 CacheCover）
frontend/src/services/libraryService.ts Wails 绑定封装：libraryApi（fetch **只读**缓存优先 · **storeIn 才入库** · list/remove/coverDataUrl · listTags/addTag/removeTag 自定义标签）
frontend/src/services/scanService.ts  Wails 绑定封装：scanApi（steamFolders / executables / subfolders / pickDirectory）
frontend/src/services/gameService.ts  Wails 绑定封装：gameApi（setPath / launch / pickExecutable，游戏启动）
frontend/src/utils/metaFormat.ts 元数据展示纯函数：toMessage / formatBytes / fileUrl / buildSummaryRows（+ MetaRow 类型）
frontend/src/utils/clipboard.ts 复制统一入口 copyText()：Clipboard API → execCommand 兜底，失败抛错（调用方提示）
frontend/src/components/meta/    作品库与调试页共用的高粒度积木（MetaSection/搜索行/封面/键值表/明细/标签…，见 §4.6）· MetaTagEditor.vue（作品详情「标签」：抓取标签与自定义标签合并编辑，hover 出 × 删除 + 末尾「+」原地输入 + 「保存」整份写库）
frontend/src/components/library/ ScanPanel.vue（游戏库「扫描添加」面板：来源/扫描方式/自动搜索进度/结果行勾选/全部入库·入库所选）
frontend/src/components/todo/    TodoList · TodoItem · TodoInput · TodoStats · FilterBar · FilterChipGroup · TodoPrioritySelector
frontend/src/components/settings/ CloseActionSelect · ThemeModeSelector · ThemeToggle · LanguageSelect(占位) · SettingsSection/Item/Page · VersionInfo · CacheCleanup（设置页「清理缓存」：进页面扫 cache/ 占用 + 一键清空） · TagLimitInput（标签显示数量 4–64）
frontend/src/components/         Sidebar.vue · TitleBar.vue
frontend/src/styles/global.css   三层 token 的**唯一色值来源**（浅色 :root / 深色 [data-theme="dark"]）
frontend/uno.config.ts           变量→类名映射 rules + shortcuts + theme.colors
frontend/scripts/                check-unocss-tokens.mjs · check-build-css.mjs（两个回归闸门）
frontend/bindings/               ⚠️ 自动生成、被 gitignore、勿手改
doc/                             README/backend/frontend/styles/conventions + need-fix.md/need-change.md（后者两个已清空为 0 字节但保留在版本控制）
```

---

## 3. 后端 API 契约（Go → TS）

服务注册在 `main.go:54-60`：`application.NewService(windowService / todoService / dbService / metaService / libraryService)`。

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

已在 `main.go` 注册：`metaService := &getmeta.MetaApiService{DB: dbService}`（持有 DB 是为了下载封面后记账进游戏库），`applicationServices` 加入后**服务总数 7、方法 45**（2026-11-12 的生成输出：281 Packages, 7 Services, 45 Methods, 15 Models）。结构体直接字面量构造（不走 `NewMetaApiService()`），因此内部有 `ensure()` 惰性兜底 `meta` 字段。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `FetchWork` | `(ctx, req WorkRequest) (any, error)` | `workId` 优先，否则拿 `keyword`；关键词里含 RJ/RE/VJ 编号则按编号精确抓，否则模糊搜索后抓**第一个**候选。返回 `map[string]any` |
| `SearchWorks` | `(ctx, keyword string, page int) (*searchResponse, error)` | 关键词含编号 → `kind="id"` 且带回 `work` 详情 + 列表候选；否则 `kind="keyword"`，返回**第 page 页**候选（不自动抓详情）。DLsite 每页固定 30 条（`per_page` 传 60/120 实测也只回 30），所以分页靠 `.../per_page/30/page/N/` 逐页取；响应里带 **`total`（页面 `data-search-result-count`，精确总数）** 与 `pageCount`（优先由 total 算出，取不到才退回分页块最大页码）—— **结果不再受 30 条限制** |
| `DownloadCover` | `(ctx, workID, coverURL string) (map[string]any, error)` | 下载封面存入相对路径 `covers/`，返回 `{id, dir, path, absPath, bytes, suffix, coverUrl}`。**优先探测同名 `.webp` 档位**（DLsite 通常有，体积更小；不引图像编码库，探测失败就回落到原图），重名加 `-2/-3` 后缀，**不覆盖已有文件**。日常由 `LibraryService.Store`（入库）和调试页调用 |
| `CacheCover` | `(ctx, workID, coverURL string) (string, error)` | **搜索缩略图**：缓存到相对路径 `cache/`（不存在就建），返回 `data:<mime>;base64,…`；已缓存过就不再联网。固定文件名 `<编号>_cover.<ext>`（允许覆盖，缓存语义） |
| `CoversDir` / `CacheDir` | `() string` | `covers/` / `cache/` 的绝对路径（后者顺带创建目录） |
| `ClearCacheDir` | `() error`（包级函数，`getmeta.ClearCacheDir`） | 清空 `cache/` 下的所有文件（目录保留，不存在算成功）。`main.go` 里挂在 `app.OnShutdown`：**每次退出软件前清掉搜索缩略图** |
| `CacheInfo` | `() (map[string]any, error)` | 扫描 `cache/`（`filepath.WalkDir`），返回 `{dir, files, bytes}`。**不联网、不建目录**：目录不存在就是 0。设置页进入时调一次，把占用显示在「清理缓存」按钮左边 |
| `ClearCache` | `() (map[string]any, error)` | 手动清空 `cache/`（复用 `ClearCacheDir` 的删除逻辑），返回 `{files, bytes, rest}`（删掉的文件数/字节数、删完剩余文件数）。设置页「清理缓存」按钮调它，成功后前端重新 `CacheInfo` 以真实占用为准 |
| `SearchSteamWorks` | `(ctx, term string) (map[string]any, error)` | 按名字搜 Steam 商店（`storesearch`，**最多 10 条、无分页**），返回 `{query, total, items:[{id,name,cover,price,type}]}`。游戏库「手动搜索」切到 Steam 来源时用 |
| `MatchSteamAppByFolder` | `(ctx, folderName string) (map[string]any, error)` | 拿**文件夹名**猜 appid：清洗（去括号注释/结尾点分版本号）→ 商店搜索（长词按 0.75/0.5/0.35 逐级截短重试）→ 归一化后「完全一致 > 公共前缀占比」打分（阈值 0.75，原声/试玩/工具类降权）。返回 `{matched, appId, name, cover}`；`matched=false` 是"猜不到"，**不是错误** |
| `FetchSteamStats` | `(ctx, appID string) (map[string]any, error)` | `/works` 的「Steam 动态」：GetItems 的 `reviews.summary_filtered` + `GetNumberOfCurrentPlayers` + `ISteamNews`（最近 3 条）。三项各自独立，全挂才报错 |

`WorkRequest{keyword, workId string}` 是前端入参——Wails 绑定不擅长可选参数，所以两个字段都给、服务端自己选。

**图片落地位置**：`const coversDir = "covers"` 是**相对路径**，实际落点取决于进程工作目录（开发态从仓库根目录跑 → `WeedyBox/covers/`）。旧版本叫 `preImg/`，`MetaApiService.ServiceStartup` 会调 `MigrateLegacyCoverDir` 自动改名并把库里 `cover_*` / `library_cover_files` 的旧路径一起替换（幂等）。返回值同时给 `path`（相对，展示用）和 `absPath`（绝对，对照用）。**格式策略**：不引图像编码库；下载时给 `jpg/jpeg/png` URL 拼出同名 `.webp` 候选并优先探测，拿到 webp 就用，服务器没有/不是 webp 就静默回落到原图（原样保存），后缀始终按**真实内容**判定（RIFF/WEBP、JPEG SOI、PNG 签名…）。

**缩略图缓存目录 `cache/`**（`const cacheDir = "cache"`，也是相对路径）与 `covers/` 分开：`covers/` 是**入库后的正式封面**（记账、不覆盖、不能丢）；`cache/` 只是搜索列表里瞄一眼用的临时图，可在退出时整体清掉（`getmeta.ClearCacheDir`），`.gitignore` 已忽略。**搜索结果每页 10 个，只给当前页下预览图**；DLsite 源数据每面 30 条，因此第 1~3 页用第 1 面、4~6 页用第 2 面……**用到哪面才取哪面**，翻到组末页（3/6/9…）时预取下一面（见 §4.3 `loadSourcePage` / `setCandidatePage`）。

**前端页面（两个）**：共用 `stores/metaStore.ts`（状态与请求）与 `components/meta/*`（积木），只是排布不同：

| 路由 | 视图 | 定位与排布 |
| --- | --- | --- |
| `/testing`（name `Testing`；旧路径 `/test` 重定向） | `views/test.vue`（薄壳，只渲染 `MetaApiTestCard`） | **调试页**：单列四卡 —— 请求 / 精确命中（含封面预览与「入库」）/ 模糊搜索候选 / **原始 JSON（可复制，只在这里出现）**。抓取入口在这一页；按钮与作品库同语义（`meta.storeIn()`，会写库） |
| `/works`（name `Works`；**侧栏无入口**） | `views/WorksPage.vue` | **作品库**：**只剩作品信息**（单个 `MetaSection` + `MetaWorkOverview` + 【Steam 源另有 `MetaSteamStats`：评价/当前在线/最近公告】+ `MetaWorkDetails`）。没有搜索卡、没有候选栏、没有「封面保存位置」卡、没有「游戏库」卡。缓存命中时标题旁有「缓存」badge 与「重新抓取」按钮。只能从游戏库点封面进入 |
| `/library`（name `Library`） | `views/LibraryPage.vue` | **游戏库**：顶栏是**单行** `MetaSection`（只有标题 + 右侧「已入库 N 款」+ 刷新，无内容区；**原「搜索」按钮已移除**）；封面网格 `grid-cols-[repeat(auto-fill,minmax(150px,1fr))]`，hover 时半透明名称面板从下方淡入；点封面 → `openWork()` 跳 `/works` 并 `loadById`（缓存优先）；右上角删除按钮连点两下确认（**移开鼠标或失焦即撤销待确认态**）；**网格最后一位是透明圆形加号按钮**（`.lib-add`，3.5rem、`border-radius:50%`、透明底 + 虚线描边 + `i-lucide-plus`）。**添加游戏在这里做**：点那个加号圆钮弹居中「添加游戏」窗（`Teleport` 到 body，遮罩 `color-mix` + `backdrop-blur(8px)`，**只有 ✕ 与 Esc 能关，点遮罩不关**），窗内两种方式（`chip` 切换，默认「扫描添加」）：**扫描添加** = `components/library/ScanPanel.vue`（来源 Steam 库 / 选择文件夹，文件夹可选「按 exe 识别」或「按文件夹名」）扫出一批名字后**自动逐个搜索**（3 路并发，可「停止」/单行「重试」）；结果按扫描顺序一行一个（封面 + 匹配作品名 + RJ + 扫描名 + 社团·形式·发售日·容量，详情在搜索完后再补抓），行可勾选，工具条提供「全部入库」与「入库所选」（**串行**写库；已入库的行显示「已入库」且不再参与），点「详情」进 `/works` 核对；**手动搜索** = 原搜索行 + **「搜索来源」chip（DLsite / Steam）**：DLsite 支持 RJ 编号精确 / 作品名模糊（候选**每页 10 个**、分页条在候选下方、**只给当前页下 `cache/` 预览图**）；Steam 走 `storesearch`（候选 ≤10、无分页，缩略图用 `tiny_image` 直链），点「详情」按 appid 取商店详情；命中作品即关窗跳 `/works`； |

三页都**自带滚动容器**：`MainLayout.vue:6` 的 router-view 是 `overflow-hidden`，页面必须自己提供滚动；而 `global.css:207-213` 用 `!important` 全局隐藏了滚动条，只有 `.scrollbar-theme` 能恢复（`scrollbar-width: thin` + 带 `!important` 的 `::-webkit-scrollbar`）。`pr-2` 给 8px 滚动条留位。

⚠️ 但**数据不再直连 MetaApiService**：页面都走 `stores/metaStore.ts` → `libraryApi`（缓存优先、**只读**），只有 `/testing` 用 `refresh=true` 强制联网核对解析；`/testing` 的原始 JSON 因此会多出 `cached/workId/cover*` 等库侧键（后端注入，不是解析结果变化）。**现状：搜索/添加入口是 `/library` 网格末尾那个透明圆形加号按钮（弹出居中「添加游戏」窗：默认「扫描添加」—— 扫 Steam 库或本地游戏文件夹，扫完自动逐个搜索、结果按行排列，可勾选入库或一键全部入库；也可切「手动搜索」，顶栏不再有「搜索」按钮）；搜索、扫描与点开详情都只读，只有点「入库」才会抓文本 + 下封面写库 —— 作品因此只会在入库后出现在 `/library`，点卡片进 `/works` 看详情；`/testing` 只作调试核对。**

**「下载」字样已改为「入库」；入库后按钮不再显示「已入库」，而是替换成「游戏启动」**：`MetaDownloadButton` 两个阶段 —— 未入库时是 `入库` / `入库中`（原来叫「下载封面」，无封面 URL 时禁用）；已入库时（`stores/metaStore.ts` 的 `stored` = 本次会话入库过 `saved` 或后端注入的 `coverAbsPath` 非空）变成 `游戏启动` + play 图标，点击走 `metaStore.launchGame()` → 后端 `GameService.Launch` 用 **`library_works.game_path`** 里记录的 exe 启动独立进程。位置为空（老数据 / 手动搜索入库）或记录的文件已被移动时，后端返回 `missing=true`，前端立刻弹系统文件框选一次 exe（`DialogService.PickExecutable`，只筛 `.exe`），写进 `game_path` 后再启动；已设置过位置时按钮旁还有一个很轻的「更换位置」文字链接。真正的写入在 `LibraryService.Store`（抓文本 + 封面落 `covers/` + 写库 + 可选记录扫描到的 exe 路径），两个页面都走同一个 `metaStore.storeIn()`；前端不再有"只下图不写库"的独立入口。原来「已入库 + 禁用」的目的是挡掉重复下载（`writeFileUnique` 不覆盖同名文件，再点只会多出一份 `-2` 后缀的重复图）—— 现在已入库后根本没有入库按钮，这个目的同样达成。

### 3.7 游戏库服务（`internal/service/library_service.go`，`LibraryService`）

与 §3.3 的 `MetaApiService` **语义相反**（这是刻意的，见 §10）：`MetaApiService` 永远联网，`LibraryService` **缓存优先** —— 库里有的直接返回，没有才联网。**但读取不等于入库**：`Fetch` 只读不写，`Store` 才是"入库"（唯一会写库的方法）。所有方法都带 `source` 参数（当前唯一值 `dlsite`，为将来接别的源留位）。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `Fetch` | `(ctx, req FetchWorkRequest) (map[string]any, error)` | `FetchWorkRequest{keyword, workId, source, refresh}`。有编号且 `refresh=false` 时先查库；命中则用 `raw_json` 还原 + 注入 `cached=true` 与封面字段；未命中（或 `refresh=true`）走 `meta.FetchWork`，**只返回、不写库**（注入 `cached=false`） |
| `Store` | `(ctx, req StoreRequest) (map[string]any, error)` | **入库**：`StoreRequest{workId, source, gamePath, gameFolder, gameFolderName}`。库里已有**完整** `raw_json` 就复用（不再抓文本），`raw_json` 残缺（id/name 或任何详情字段都没有，即"空壳"）会重新联网抓；否则联网抓一份 → **校验 `workIsComplete`（编号+名称+至少一个详情字段）** → `UpsertLibraryWork` →（有位置时）`SetGamePath` 写进 `library_works.game_path` → `DownloadCover` 把封面落到 `covers/` 并记账。**启动位置在入库时就确定**（用户 2026-11-16 要求）：直接给 `gamePath`（按 exe 名扫到/手动搜索选好）就用它；只给 `gameFolder`+`gameFolderName`（按文件夹名扫到）就调 `scan.PickGameExecutable` 进文件夹挑主程序；都空（手动搜索取消选择）则位置留空、启动时再问。空壳直接报错、不写库（扫描添加一次入库几十部，不能污染游戏库）。封面已存在时幂等返回。写库失败**直接报错**（这一步就是在"载入数据库"） |
| `List` | `(ctx, source string) ([]LibraryWork, error)` | `source` 空 = 所有来源；`ORDER BY created_at DESC, id DESC` |
| `Remove` | `(ctx, workID, source string) error` | 删除库记录，并**删除该作品入库时保存在 `covers/` 的封面图片**（按 `library_cover_files` 的全部记录 + 作品表 `cover_abs_path` 去重；文件删不掉时报错、库记录保留以便重试） |
| `CoverDataURL` | `(ctx, workID, source string) string` | 读 `cover_abs_path` → `data:<mime>;base64,…`；文件不存在返回空串（不算错误） |
| `GetTags` | `(ctx, workID, source string) (model.TagSet, error)` | 读作品的标签集合 `{tags, edited}`。`edited=true` = 用户保存过整份列表（`work_tags` 里就是他定的那份，重抓不丢）；`false` = 只有旧版逐条加过的标签，前端还要与抓来的 `tags` 合并 |
| `SetTags` | `(ctx, workID, source string, tags []string) (model.TagSet, error)` | **唯一写标签的入口**：整份覆盖（清空 → 按顺序插入 → 记 `work_tag_edits` 标记，同一事务）。逐条清洗、丢空、去重；单条 ≤32 字符、最多 128 条，超了报错不截断；传空列表 = 用户删光（合法） |

**为什么封面用 base64 data URL 而不是 `file://`**：Wails webview 对本地文件协议的支持没实机验证过，data URL 一定可用；封面就几十~几百 KB，一次列表请求可接受。

**封面记账**：`MetaApiService.DownloadCover` 落盘后若持有 `DB` 就调 `RecordCoverFile` 写一行 `library_cover_files` 并把作品的 `cover_*` 指针指向它（`writeFileUnique` 会加 `-2/-3` 后缀，事后猜不出文件名，所以必须当时记账）。

**测试**：`internal/database/library_repo_test.go`（10 个用例）覆盖惰性建表、首写前先读/先删、**并发首次使用**（曾真实复现 `database is locked (5) (SQLITE_BUSY)`，见 §10）、来源隔离、封面指针更新。

### 3.8 本地扫描服务（`internal/service/scan/`，package `scan`，`ScanService`）

**已在 `main.go` 注册**为 Wails 服务 `ScanService`（`application.NewService(scanService)`，import `WeedyBox/internal/service/scan`）：绑定生成后前端经 `services/scanService.ts`（`scanApi`）调用，用于游戏库的「扫描添加」。**无状态**：根目录每次作为参数传入（游戏库可能挂多个根）。**放在独立子包 `scan`**（用户要求把扫描服务合并到一个文件夹），不要再往 `internal/service` 根目录塞扫描相关文件。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `ScanExecutablesInSubfolders` | `(ctx, root string) (GameScanResult, error)` | **通用**（靠 exe 认游戏，不管文件夹怎么命名）：读根目录 + 每个一级子文件夹各一层 → `GameScanResult{root, folders[], skipped[]}`；`folders[]` 每项 `{folderName, folderPath, executables[]}`，exe 为 `{fileName, name, path, size}`。**根目录自身的 .exe 不算、不递归、没有 .exe 的文件夹不进结果**；某个子文件夹读不动（权限/坏盘/被占用）只记进 `skipped` 并继续 |
| `ScanSubfolders` | `(ctx, root string) ([]Subfolder, error)` | **名字 + 绝对路径**（`Subfolder{name, path}`）：只读根目录一层，不打开子文件夹。给「按文件夹名扫描」用 —— 名字去搜索、路径留着**入库时进文件夹挑 exe** |
| `ScanSubfolderNames` | `(ctx, root string) ([]string, error)` | 同上的纯名字版本（内部复用 `ScanSubfolders`），保留给只要名单的调用方/测试 |
| `ScanSteamGameFolders` | `(ctx) ([]Subfolder, error)` | **Steam**：找本机 Steam 安装目录 → 读 `libraryfolders.vdf` 拿到全部库根 → 对每个 `<库根>\steamapps\common` **复用 `ScanSubfolders`**，返回游戏文件夹名 + 绝对路径。跨库去重（大小写不敏感）、按名排序（大小写不敏感，与资源管理器一致）。库盘没插/目录被删只跳过；一个 Steam 都找不到时返回哨兵错误 `ErrSteamNotFound`（区别于「装了但空库」） |
| `PickGameExecutable` | `(ctx, folderPath, folderName string) (string, error)` | **从游戏文件夹挑主程序**（用户 2026-11-16 规则）：先排除 `UnityCrashHandler64.exe`；有 `Game.exe` 优先；否则选**去掉扩展名的 exe 名与文件夹名最长公共子序列最长**的那个（并列取 ReadDir 顺序第一个）；没有可用 exe 返回空串 + nil（不阻断入库，启动时再让用户选） |

路径一律 `filepath.Abs` + `Clean` 成绝对路径（结果里的 `path` 将来要用来启动游戏，不能依赖进程工作目录）。`.exe` 判定大小写不敏感且用 `filepath.Ext`（`game.exe.txt` 不算）；`entry.IsDir()` 不够——Windows junction 的 `IsDir()` 是 false，要再 `Stat` 一次跟随链接。排序取 `os.ReadDir` 的按名顺序（**不按大小**，主程序不一定是最大 exe，选择交给调用方）。测试见 `scan/scan_test.go`：用真实 `t.TempDir()` 树断言「只读一层」的边界，不 mock 文件系统。

**Steam 安装目录的找法**（`scan/steam_windows.go`，build tag `windows`）：`HKCU\Software\Valve\Steam\SteamPath`（最准，值是正斜杠 `e:/steam`，要 `filepath.FromSlash`）→ `HKLM\SOFTWARE\Valve\Steam\InstallPath`（从 64 位进程读 32 位安装器写的键要带 **`WOW64_32KEY`**，所以 64/32 两个视图都试）→ `%ProgramFiles(x86)%\Steam` / `%ProgramFiles%\Steam` 兜底；注册表残留指向已删目录时由 `os.Stat` 筛掉。非 Windows 用 `scan/steam_other.go` 的空实现（本项目 GOOS 固定 windows，只是为了 `GOOS=darwin go build ./...` 不炸）。

**库路径来自 `libraryfolders.vdf`**：现役位置 `<Steam>\steamapps\libraryfolders.vdf`（块内 `"path" "D:\\SteamLibrary"`），旧客户端在 `config\libraryfolders.vdf`，两处都读；旧格式（键是数字、值直接是路径）在没有 `path` 行时兜底。**不引 VDF 第三方库**：两条行级正则足够，反斜杠/引号转义手工还原。**依赖**：`golang.org/x/sys/windows/registry` —— 该模块本来就在 `go.mod`（indirect），这次只是变成 direct，**没有新增模块**。测试见 `scan/steam_test.go`（vdf 现役/旧版样本、多库汇总去重、vdf 在 config 下、主库重复列出、无 common、取消）。真实机器已烟测：注册表 `e:/steam` → 46 个游戏文件夹。

### 3.9 系统对话框（`internal/service/dialog_service.go`，`DialogService`）

已注册（第 7 个服务）。两个方法：

- `PickDirectory(title string) (string, error)`：`app.Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true)` 打开系统「选择文件夹」框，`AttachToWindow(app.Window.Current())` 让它贴着主窗口。前端封装是 `scanApi.pickDirectory()`，只在「扫描添加 → 选择文件夹」用。
- `PickExecutable(title string) (string, error)`：`CanChooseFiles(true).CanChooseDirectories(false).AllowsOtherFileTypes(false).AddFilter("可执行程序", "*.exe")`，给「游戏启动」第一次没有位置 / 更换位置时选 `.exe`。前端封装是 `gameApi.pickExecutable()`。

**用户取消返回空串 + nil**（不是错误），只有应用实例未就绪/对话框创建失败才报错。`PromptForSingleSelection` 内部走 `InvokeSync` 切到主线程，所以从服务 goroutine 调用不会死锁。为什么不塞进 WindowService：窗口操作与"问用户要一个路径"是两类事。**注意 `go.mod` 的 beta.10 里 `application.Get()` 是包级全局实例**，服务里直接取即可。

### 3.10 游戏启动服务（`internal/service/game/`，package `game`，`GameService`）

已注册为第 8 个服务（`application.NewService(gameService)`，import `WeedyBox/internal/service/game`）。位置就是 `library_works.game_path` 一列（用户要求「游戏路径和 library_repo.go 合并在一个表内」），不单开表；`GameService` 只负责校验与起进程，DB 读写仍走 `database/library_repo.go` 的 `GetGamePath` / `SetGamePath`。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `Status` | `(ctx, workID, source string) (GameStatus, error)` | 读位置 + 检测 exe 是否存在 + 是否运行，返回 `{workId, path, exists, running, pid}`。缓存命中先复核 PID（`processAlive` 比对完整映像路径，防 PID 复用）；失效/未命中就全量扫进程（快照按 exe 名粗筛 → `QueryFullProcessImageName` 比对**完整路径**），找到则把路径+PID 载入运行缓存。exe 不存在时清掉缓存并返回 `exists=false` |
| `SetPath` | `(ctx, req SetGamePathRequest) (GamePathState, error)` | `SetGamePathRequest{workId, path, source}`。`normalizeGamePath`：非空 → `filepath.Abs` + `Clean` → `os.Stat` 必须存在且**不是文件夹** → `DB.SetGamePath`（只更新已在游戏库里的作品；作品不在库里时报错，不顺手造空壳行）。返回 `{workId, path, exists:true}` |
| `Launch` | `(ctx, workID, source string) (GameLaunchResult, error)` | 读 `game_path`：空 → `{started:false, missing:true, message:"还没有记录…"}`；文件不存在 → `missing:true`（**不报 error**，前端据此弹选 exe 框）；否则 `startDetached` 起独立进程 → `{started:true, running:true, path}`，并把 PID 载入运行缓存。进程起不来（权限/损坏）→ error |

启动实现（`launch_windows.go`）：`exec.Command(exe)` + `cmd.Dir = filepath.Dir(exe)`（很多游戏按相对路径找资源）+ `SysProcAttr.CreationFlags = windows.DETACHED_PROCESS`（独立进程、关掉 WeedyBox 不带走游戏、不闪控制台）+ `Process.Release()`，返回 PID。非 Windows 用 `launch_other.go` 空平台实现（build tag），只为 `GOOS=darwin` 检查不炸。

**运行检测**（`process_windows.go`）：**不引 gopsutil**（项目约定不新增依赖；`golang.org/x/sys` 本来就在 go.mod）——`CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS)` 枚举进程，先用 `ExeFile` 基名粗筛，再 `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` + `QueryFullProcessImageName` 比对完整路径（同名 exe 不会误判）。`GameService` 持 `mu sync.Mutex` + `running map[归一化小写路径]pid` 运行缓存：查询命中先 `processAlive` 复核，游戏退出/文件删除就从缓存移除。前端进入作品详情调一次 `Status`，`WorksPage` 每 5 秒轮询 + 窗口 focus 时刷新，按钮在「游戏启动 ↔ 运行中」之间实时切换。测试见 `game/game_service_test.go`（路径校验、SetPath 写入、Upsert 不丢位置、未入库拒绝、Launch `missing` 分支、Status 无位置/文件被删清缓存、运行缓存）与 `process_windows_test.go`（真起一个 `cmd.exe`/ping 进程，验证按完整路径能认出运行中，查完即杀）。

### 3.11 Steam 元数据源（2026-11-21）

**需求原话**：「steam源添加(meta)根据游戏id( ...\Steam\steamapps目录下有appmanifest_id相关文件 扫描id 并通过 https://store.steampowered.com/api/appdetails?appids=id 获取json数据 优先复用已有逻辑)进入相关文件夹 并无限递归查找含有 600x900 或 capsule 字样的图片文件 用作封面(游戏库封面展示图像格式也全部变为2:3等比图样 包括steam源和 DLsite源 和以后所有可能添加源) 另外 作品信息显示界面未来要作修改 暂时不进行改动」；补充：「查找封面图片文件夹目录在...\Steam\appcache\librarycache内 通过已有逻辑查找 目录下含有游戏id名称的文件夹 进入查找相关图像 没有文件夹则不适用图像」。

- **来源**：`database.SourceSteam = "steam"`，编号 = appid（纯数字）。与 dlsite 同表（`library_works` 不变结构），`UNIQUE(source, work_id)` 天然隔离。
- **扫描**（`internal/service/scan/steam_apps.go`）：`ScanSteamApps()` 读每个库 `steamapps\appmanifest_<appid>.acf`，解析 appid / name / installdir / StateFlags / SizeOnDisk。**StateFlags 按位判 4**（实测 R.E.P.O. 是 6 = 已安装|待更新，写 `==4` 会漏）；跨库按 appid 去重、按名称排序。库根来自 `steamLibraryRoots()`（主库 + `libraryfolders.vdf`，现役与旧版位置都读）。
- **元数据**（`internal/service/getMeta/getFromSteam.go`）：`MetaService.FetchSteamApp` → `store.steampowered.com/api/appdetails?appids=<id>&l=schinese&cc=CN`；`parseSteamAppDetails` 解析外层 `{"<appid>":{success,data}}`，`buildSteamWork` 映射成作品 map：`id/name/maker`(开发商)`/workType`(类型中文)`/releaseDate/summary/coverUrl`(header_image)`/tags`(类型+分类+开发商+发行商)`/outline`(开发商/发行商/平台/评测/成就/Metacritic)`/sourceUrl/fetchedAt`，外加 `steamType`（原始 type，前端据此跳过 tool/dlc）与 `steam`（完整 appdetails，作品界面以后重做时直接读）。**genres.id 是字符串、categories.id 是数字**，两个 id 都不声明，只用 description。
- **分流**（`internal/service/library_service.go`）：`Fetch` / `Store` 先看 `source`；steam 走 `fetchSteamWork` / `storeSteamWork`，**缓存优先、Fetch 只读、Store 才写库**的语义与 dlsite 完全一致。位置仍走 `resolveStoreGamePath`（扫描给的安装目录 → `scan.PickGameExecutable`）。
- **封面**（`scan.SteamCoverImage` + `getmeta.ImportCoverFile`）：在 `<Steam>\appcache\librarycache` 下找名字含 appid 的目录（精确同名优先）→ **不限深度**递归找文件名含 `600x900` 或 `capsule` 的图片，打分 `600x900`(100) > `capsule`(40) > `_schinese`(+20)，同分越浅越优先；**没有对应目录 / 没有候选图就返回空串**（用户要求：没有文件夹就不用图像，不联网兜底）。找到后复制进 `covers/`（沿用不覆盖、重名 `-2/-3`）并记账，`library_cover_files.source='steam'`。
- **前端**：`scanStore` 的 Steam 来源改用 `scanApi.steamApps()`，行内带 `source:'steam'` + `appId`，直接 `libraryApi.fetch('', appid, false, 'steam')` 取详情（不再拿文件夹名去 DLsite 搜）；`steamType !== 'game'` 的行标「已跳过」、不给入库。`metaStore` 增加 `source` 状态，`loadById(id, refresh, fromSource?)` 把来源贯穿到入库/启动/标签/封面调用；`stored` 判定增加 `libraryUpdatedAt`（Steam 可能没有封面，只看 `coverAbsPath` 会误判未入库）。
- **封面比例**：`LibraryPage.vue` 网格封面 `aspect-[4/3]` → **`aspect-[2/3]`**（`object-cover`），Steam / DLsite / 以后所有来源统一。`/works` 作品信息界面本次**未改**（用户明确以后单独改）。
- **实测**（本机 `E:\steam`）：19 份 appmanifest → 19 条；17 条在 librarycache 找到封面（`Beat Banger` 与本机 `Steamworks Common Redistributables` 没有对应图），嵌套 hash 目录里的 `library_capsule.jpg` 也能递归命中。测试：`scan/steam_apps_test.go`（manifest 解析/位判断/多库去重/封面递归与打分）、`getMeta/getFromSteam_test.go`（字段映射/失败分支/封面导入记账）。
- **标签改走 GetItems（2026-11-22，用户要求「tag 的获取尝试使用别的接口」）**：`getFromSteam.go` 抓到 appdetails 后调 `enrichSteamWork`（`steam_browse.go`）——① 用 `IStoreBrowseService/GetItems/v1`（`include_tag_count:20` + `include_reviews`）拿**玩家标签**（tagid + 权重），经 `IStoreService/GetMostPopularTags`（进程内缓存 24h）换成中文名，**按权重降序排在前面**，原来 appdetails 的「类型/分类/开发商/发行商」接在后面去重（`mergeSteamTags`，上限 `steamTagLimit=64`）；② 评价汇总写进 `work["steamReviews"]` 并补一行 outline「Steam 评价：好评如潮 97%（N 篇）」；③ GetItems 原始数据留在 `work["steamBrowse"]`。**这一步任何失败都只是"少一点信息"**，不报错、不影响入库。
- **手动搜索支持 Steam（2026-11-22）**：`MetaApiService.SearchSteamWorks`（storesearch，10 条、无分页）→ 前端弹窗的「搜索来源」chip（DLsite / Steam）→ 点候选按 appid 走 `libraryApi.fetch('', appid, false, 'steam')`。Steam 候选的缩略图用 `tiny_image` 直链（公网 CDN，不需要 `cache/`；`CacheCover` 只认 DLsite 编号）。
- **扫描「按文件夹名」先匹配 Steam（2026-11-22）**：`scanStore.searchOne` 先调 `MatchSteamAppByFolder`（清洗 → storesearch → 逐级截短 → 归一化打分），命中就把该行改成 Steam 条目（`source='steam'` + `appId`，`detail` 改「Steam · id」），猜不到再走原来的 DLsite 名字搜索。目的：Steam 源原本只能靠 `appmanifest`（= 必须已安装）识别。联网烟测：`黑神话悟空→2358720`、`艾尔登法环黄金树幽影→2778580`、`ULTRAKILL v1.0→1229490`、`Hades→1145360`、`Stardew_Valley→413150`，而 `荒野大镖客2`（国区不上架）与随机词正确返回"不匹配"。
- **/works「Steam 动态」（2026-11-22）**：`components/meta/MetaSteamStats.vue`，只在 `meta.source === 'steam'` 时渲染：评价（label + 好评率 + 篇数）、当前在线、最近 3 条公告（标题外链 + 日期）。数据 `MetaApiService.FetchSteamStats`，`applyWork` 时自动取一次，头部有「刷新」。

### 3.12 Steam 公开接口清单（2026-11-22 调研，**已部分接入**）

用户要求「在网上搜索相关api信息 把可能可以扩展当前软件的api保留」。逐条实测（本机 PowerShell / `curl.exe` 都在 schannel 层失败，改用 Go `net/http` 探测），结论写进 [`doc/steam-api.md`](./doc/steam-api.md)，入口常量在 `internal/service/getMeta/steam_endpoints.go`。

- **已接入（2026-11-22 当天接着做完的三条 + 标签源）**：`GetItems`（Steam 标签 + 评价汇总，见 §3.11）、`GetMostPopularTags`（tagid → 中文名）、`storesearch`（手动搜索的 Steam 来源 + 扫描「按文件夹名」猜 appid）、`GetNewsForApp` + `GetNumberOfCurrentPlayers`（`/works` 的 Steam 动态）。代码在 `steam_browse.go`，测试 `steam_browse_test.go`。
- **免 key 可用但暂未用**：`GetStoreCategories`（categoryid → 中文名，比 appdetails 的 type 更细）、`search/suggest`（HTML 片段）、`featured/featuredcategories`（商店推荐位，与本软件无关）。
- **不能忘的两个实测坑**：`appdetails` **一次只能一个 appid**（逗号拼多个 → `400 null`）、`filters=` 参数被忽略；`storesearch` **最多 10 条且按词匹配**（`黑神话悟空` 0 条、`黑神话` 9 条 → 所以文件夹名匹配要"逐级截短"）。
- **不可用**：`appreviews`（匿名请求一律 `302` 到商店首页；换区、加 cookie、加 `X-Requested-With`/`Referer`、改 POST 全都不行 → 评价改用 GetItems 的 `reviews.summary_filtered`）；`ISteamApps/GetAppList/v2` 已 `404` 下线；SteamSpy 返回 Cloudflare 挑战页。
- **要 Web API key（默认不接）**：`IStoreService/GetAppList`、`IPlayerService/GetOwnedGames`、`GetDLCForApps`、`GetSchemaForGame`、`GetGlobalAchievementPercentagesForApp`。
- **还没做、也仍需用户点头的**：联网封面兜底（`assets.library_capsule` 现成 600×900）—— 与用户「librarycache 没有就不要封面」的决定冲突，未经同意不做。

### 3.4 获取外部元数据（`internal/service/getMeta/getFromDlsite.go`，package `getmeta`）

抓 DLsite 作品页。**不依赖第三方 DLsite 库**（`go.felesatra.moe/dlsite` 已从 `go.mod` 移除，原因见 §10），只用 `net/http` + `goquery`。`MetaApiService`（§3.3）是它的 Wails 包装层。

| 方法 | 签名 |
| --- | --- |
| `NewMetaService(options ...MetaOption)` | 默认 `http.Client{Timeout: 20s}`、`tagLimit=24` |
| `WithHTTPClient(*http.Client)` / `WithTagLimit(int)` | 注入客户端 / 限制标签数 |
| `GetWorkByID` | `(ctx, id string) (*DLsiteWork, error)` —— 按 RJ/RE/VJ 取单个作品 |
| `SearchWorkByName` | `(ctx, keyword string) ([]DLsiteSearchItem, error)` —— 第一页候选（`SearchResultPage(ctx, keyword, page)` 返回该页 30 条 + 总页数 + 总数）；每项只有 `ID`/`Name`/`Cover`。⚠️ 缩略图在页面里是 Vue 组件 `<thumb-with-ng-filter-block :thumb-candidates="['//img…webp','//img…jpg']">`，**不是** `<img>` 的 `src`/`data-src`；块内那个 `<img>` 只是弹层占位图（`src` 是内联 gif data URL）。`extractSearchItemCover` 因此优先读 `:thumb-candidates` 的第一项并补成 `https://`，退路才看传统图片属性且跳过 `data:` 占位 —— 曾经的"搜索结果不出图、点详情才有图"就是这个坑 |
| `GetWorkByName` | `(ctx, keyword string) (*DLsiteWork, error)` —— 搜索取首个再抓详情 |
| `NormalizeDLsiteID` | `(raw string) (string, bool)` —— 抽编号，转大写、**保留前导零** |

`DLsiteWork` 字段（json tag 全 camelCase）：`ID` `Name` `AlternateID`(拼音 slug) `Maker` `MakerID` `Series` `SeriesID` `WorkType` `AgeRating` `FileFormats[]` `FileSize` `ReleaseDate` `Price` `Currency` `PriceUntil` `Rating` `RatingCount` `CoverURL` `Summary` `Tags[]` `Outline[]` `SourceURL` `FetchedAt`

两个真实作品的实测值（**旧模板**时代的离线记录；旧 fixture 已随工作区裁剪删除，现役新模板见下方「九个坑」与测试说明）：

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

**标签（`Tags`）的拼接规则（2026-11-12，用户要求）**：来源 = 分类（`dd a[href*="/genre/"]`，旧页面 `.main_genre a`）+ 作品形式（按 `dd` 里的 a/span 逐个取）+ 社团名 + 作者 + 剧情 + 插画 + 声优（按元素取，一个声优一个元素）+ 年龄指定/販売形式。全部经 `splitTagText` 按**空白字符和 `/`** 切分（`双声道立体声/人头麦` → 两个、`みたかりん 野上菜月` → 两个），再去重、按 `defaultTagLimit=64` 截断。**多值字段必须按元素取**：中文页面里相邻元素之间没有空白，取整段 `dd.Text()` 会粘成 `异种X造孽命令…` / `みたかりん野上菜月`。前端设置页的「标签显示数量」（4–64，默认 24，`stores/App.ts` 的 `tagLimit`）只控制**显示**多少个，超出折成 `+N`。

**九个坑（改之前必读，2026-11-12 复核）**：① 必须 HTTPS；② 前导零丢了必 404；③ 必须发 `Cookie: adultchecked=1; locale=zh-cn`（代码里写死中文；R18 不退化，且页面标签变成中文 —— **只认日文标签会全部取空**）；④ 标题只取 `#work_name` 的文本（现代码里没有 `<a>`）；⑤ **页面模板 2026-11 换新版**：`#work_outline` 表格、`#work_maker`、`<meta itemprop="price/ratingValue">` 全部消失，改用 `div.c-productInfo` 的 `dt.c-productInfo__key → dd.c-productInfo__value`（信息表）+ `<script type="application/ld+json">` 的 Product（价格/评分/社团/拼音/简介）。**只认旧选择器时解析结果就是"只有名字+封面"的空壳**（`raw_json` 约 280B）—— 用户报"扫描入库只有封面和作品名"的根因就在这里，旧库里已有空壳也会被 `workIsComplete` 判为不完整、重新抓取；⑥ 标签文字随 locale 变（`社团名/サークル名`、`发售日/販売日`、`文件容量/ファイル容量`…），取值一律走 `outlineValueAny` 多语言候选；genre 用 `dd a[href*="/genre/"]` 识别，不认标签；⑦ `SeriesID` 旧页面取内嵌 JSON，新页面取 `title_id/SRI…` 链接；`MakerID` 取 `maker_id/RG…` 链接或 JSON-LD `brand.url`；⑧ 必须判 `FindStringSubmatch` 长度（否则单作 panic）；⑨ `CoverURL` 已改为 `og:image` 优先。

**图片可以直接爬（实测）**：`img.dlsite.jp` DNS 干净（Cloudflare `104.18.24.157` / `104.18.25.157`），**不需要 Cookie、不需要 Referer**，`curl --noproxy '*'` 直连即可（PowerShell `Invoke-WebRequest -OutFile` 会因 .NET 默认代理报 `NonInteractive mode`，不要用它）。同一封面 4 个档位：`_img_smp1.jpg` 1470×1000 / 808 KB > `_img_main.jpg` 560×420 / 222 KB（`og:image` 就是它）> `_img_main.webp` 38.9 KB > `_img_main_240x240.jpg` 45.9 KB。

**`CoverURL` 取大图的优先级（已修）**：旧实现扫 `img, source` 时先命中 `#work_thumb img` 的 `_img_main_240x240`，于是永远拿最小那张。现在 `extractCoverURL` 的顺序是 `meta[property="og:image"]` → `meta[itemprop="image"]` → `extractCoverURLWidest`（扫 `img,source`，用 `coverWidthScore` 按后缀宽度挑最大的：显式 `_WxH` 取 W，其次 `_smp`=1470 > `_img_main`=560）。实测两个 fixture 都从 240×240 升到 560×420：`RJ01721474_img_main.jpg` / `RJ01664616_img_main.jpg`。

**测试**：`new_template_test.go` 用两个 fixture —— `work_new_RJ01232863.html`（日文标签的精简页面）与 `work_new_zh_RJ01232863.html`（**真实中文页面**，去掉 style/script 后约 81KB），分别锁住「日文标签」与「中文标签（应用实际发的 locale）」两条路径，断言具体值；另有 `TestParseWorkRejectsPageWithoutName`（结构变化必须报错而非返回空壳）。旧的 `work_RJ01721474.html` / `work_RJ01682332.html` 已随工作区裁剪删除。`go test ./internal/service/getMeta/` 全绿。实测（2026-11-12 联网）：RJ01232863 的社团/系列/形式/分级/容量/发售日/价格/评分/简介/标签全部解析成功。

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

`internal/model/library.go`：

```go
type LibraryWork struct {   // 来源无关：公共列 + 来源专有字段整份进 RawJSON
    Source string           // "dlsite"（当前唯一值）
    WorkID string           // RJ01721474
    Name, Maker, MakerID, Series, SeriesID, WorkType, AgeRating, ReleaseDate,
    CoverURL, SourceURL, RawJSON string
    FetchedAt, CreatedAt, UpdatedAt time.Time
    CoverDir, CoverPath, CoverAbsPath, CoverSuffix string
    CoverBytes int64
}
type CoverFile struct {     // 每次下载一行，作品表的 cover_* 指向最新一行
    ID int64; Source, WorkID, Dir, Path, AbsPath, Suffix, CoverURL string
    Bytes int64; CreatedAt time.Time
}
```

### 3.6 表结构

**`todos`**（`internal/database/sql_service.go:45-53`）：`todos(id INTEGER PK AUTOINCREMENT, title TEXT NOT NULL, description TEXT, priority INTEGER DEFAULT 1, completed INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`

**游戏库两张表**：**不在 `initAllTables` 里建**，改由 `internal/database/library_repo.go` 的 `ensureLibraryTables(ctx)` 在**每次读写前** `CREATE TABLE IF NOT EXISTS`（幂等）——这是用户要求的「当需要保存到库中时 即是需要下载数据才新建表 节约空间」。

- `library_works`：`id PK / source / work_id / name / maker / maker_id / series / series_id / work_type / age_rating / release_date / cover_url / source_url / raw_json / fetched_at / cover_dir / cover_path / cover_abs_path / cover_bytes / cover_suffix / **game_path** / created_at / updated_at`，唯一键 `UNIQUE(source, work_id)`。`game_path` = 这个作品在本机上的游戏程序（.exe）绝对路径，给「游戏启动」用；与元数据同表但**不被 Upsert 覆盖**（ON CONFLICT SET 里不含它）。旧库文件没有这一列 —— `ensureLibraryTables` 末尾的 `ensureGamePathColumn` 用 `PRAGMA table_info` 检测后 `ALTER TABLE … ADD COLUMN game_path TEXT DEFAULT ''` 补齐（并发重复 ALTER 按已存在忽略）。
- `library_cover_files`：`id PK / source / work_id / dir / path / abs_path / bytes / suffix / cover_url / created_at`，索引 `idx_library_cover_files_work(source, work_id)`
- `work_tags`（作品标签，**惰性建表**，由 `tag_repo_test.go` 覆盖）：`id PK / source / work_id / tag / created_at`，`UNIQUE(source, work_id, tag)`，索引 `idx_work_tags_work(source, work_id)`。存的是"用户整理后的整份列表"（抓取标签与自定义标签合并的结果），顺序 = `id` 递增。
- `work_tag_edits`（标签已改写标记，同一处惰性建表）：`source + work_id` 主键 + `updated_at`，有行 = 这个作品的标签被用户整份改写过。为什么需要它：用户把标签**全删光**时 `work_tags` 也是空的，光看行数分不清「从没整理过」与「整理成空列表」。
- 两张标签表都刻意不挂在 `library_works` 上：作品库详情页可以打开还没入库的作品，那里的标签也要能存。

`UpsertLibraryWork` 用 `INSERT … ON CONFLICT(source, work_id) DO UPDATE SET …`，且**不覆盖 `cover_*` / `game_path` 列**（封面由 `RecordCoverFile` 维护，启动位置由 `SetGamePath` / 扫描入库维护；重新抓元数据不该把它们清空）。

DB 文件 `WeedyBoxDB.db` 建在**可执行文件同目录**（`sqlite.Config{DBSource: "WeedyBoxDB.db"}`）。

---

## 4. 前端架构

### 4.1 数据流

```
组件 → stores/todoStore.ts → services/todoService.ts → bindings/…/todoservice.ts
     → Go service.TodoService → database.SQLiteService → SQLite

组件 → stores/metaStore.ts → services/libraryService.ts → bindings/…/libraryservice.ts
     → Go service.LibraryService ─┬→ database/library_repo.go → SQLite（缓存命中）
                                  └→ getmeta.MetaApiService → DLsite（未命中/强制刷新）

组件 → stores/metaStore.ts → services/gameService.ts → bindings/…/gameservice.ts
     → Go game.GameService → database/library_repo.go（读/写 library_works.game_path）→ 启动 exe
```

调试页（`/testing`）是唯一例外：它用 `refresh=true` 让 `LibraryService` 跳过缓存，等价于直连解析器。

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
| `stores/App.ts` | `app` | `CloseAction='ask'|'hide'|'close'`（默认 `ask`）；localStorage `close-action`；`VALID` 数组判定，**非法值/旧值 `minimise` 一律回落 `ask`**；导出 `CLOSE_ACTION_OPTIONS`；`setCloseAction` + `restoreState` + `watch` 兜底落盘。**另有 `tagLimit`**（作品标签显示数量，默认 24，范围 4–64，localStorage `tag-limit`，导出 `TAG_LIMIT_MIN/MAX/DEFAULT`，`setTagLimit` 会自动夹取；注：2026-11-22 起 `/works` 的标签由 `MetaTagEditor` 全量显示可编辑，这个数量只作用于调试页的标签云） |
| `stores/metaStore.ts` | `meta` | 元数据（DLsite / Steam 双源）：State `keyword·loading·downloading·launching·error·status·work·saved·coversPath·candidateCovers(id→dataURL)·sourcePages(DLsite页号→30条)·sourcePageCount·searchTotal·pageLoading·candidatePage·raw·cached·libraryUpdatedAt·coverAbsPath·gamePath·gamePathExists·gameRunning·gameWarning·**tagList（编辑稿：抓取标签 + 自定义标签合并）·tagSaved（上次写库的那份）·tagsEdited·tagsSaving·tagError**·**searchSource('dlsite'\|'steam'，弹窗搜哪个源)·steamCandidates·steamSearchTotal·steamStats·steamStatsLoading**`；Getters `rawJson·hasResult·stored·launchBlocked（记了路径但 exe 不存在）·tagsDirty（编辑稿与已保存不同，决定「保存」按钮显不显示）·candidates·candidateTotal·candidatePageCount·pagedCandidates`；Actions `runSearch·**runSteamSearch**·**setSearchSource**·**loadSteamStats()（Steam 动态：评价/在线/公告；applyWork 自动取一次）**·loadById·searchBySample·loadCandidateCovers·loadSourcePage·setCandidatePage·storeIn()（**入库**）·launchGame()（**游戏启动**；无位置时先弹选 exe 框再启动，启动后标记 running）·changeGamePath()（更换 exe）·refreshGameStatus()（查 exe 是否存在 + 是否运行，页面轮询用）·**loadTags（载入时：edited 就用库里那份，否则抓取 tags 打底 + 旧自定义标签追加）·addTag / removeTag（只改编辑稿）·saveTags（整份写库）**·refreshCoversDir·ensureCoversDir·clearResult`。导出 `META_SAMPLES`。**缓存优先**（走 `libraryApi`）；调试页传 `refresh=true` 强制联网。`keyword` 不参与"打开作品"（`loadById` 不写回），搜索弹窗开/关都会清空它（见 §10）。**DLsite 候选分页每页 10 条**（30/10=3，第 1~3 页对应 DLsite 第 1 面…用到哪面才取哪面）；**Steam 候选最多 10 条、无分页**（缩略图用 `tiny_image` 直链，不进 `cache/`） |
| `stores/libraryStore.ts` | `library` | 游戏库列表：State `works·loading·error·covers(id→dataURL)·removingId·loaded`；Getters `count`；Actions `load(force=false)·loadCovers()·coverFor(work)·remove(work)·refreshCover(workId, source)·has(workId, source?)`。**列表不按来源过滤**（`libraryApi.list()`，dlsite/steam 一起显示），封面 key 用 `来源:编号` |
| `stores/scanStore.ts` | `scan` | 游戏库「扫描添加」：State `source('steam'\|'folder')·mode('exe'\|'folder')·folderPath·scanning·picking·error·status·entries({name,detail,origin,path?,folderPath?})·done`；Getters `count`；Actions `setSource·setMode·pickFolder()（原生选目录，取消返回空串不动路径）·scan()`。**只读磁盘、不联网**；点结果里的名字由 LibraryPage 调 `meta.runSearch(name)`。**扫描结果记录来源**：`origin='exe'` 带 exe 绝对路径（入库直接记）、`origin='folder'` 带文件夹绝对路径（入库时后端进文件夹挑主程序）；批量入库时随 `libraryApi.storeIn(workId, {gamePath/gameFolder/gameFolderName})` 写进 `library_works.game_path`。**2026-11-22 起 `origin='folder'` 的行先调 `matchSteamApp(name)` 猜 appid**（`matchSteamEntry`）：命中就改成 Steam 条目（`source='steam'`+`appId`+`detail='Steam · id'`）走商店详情，猜不到才回退 DLsite 名字搜索 |
| `stores/autoHeightStore.ts` | `autoHeight` | `<Transition>` 的高度自适应钩子（原 `composables/useAutoHeight.ts`，已按约定搬进 stores）：`duration`（读 `--motion-slow`）+ `enter·afterEnter·leave·afterLeave`；用法 `@enter="collapse.enter"`，`const collapse = useAutoHeightStore()` |

> ⚠️ 命名不一致：`Sidebar.ts`/`Theme.ts` 大写开头，`todoStore.ts`/`App.ts`/`metaStore.ts`/`libraryStore.ts` 混用 —— 新增 store 时 import 大小写要**手写对**。

### 4.4 services/ 封装层（不是 store，是所有后端调用的唯一入口）

| 文件 | 导出 | 说明 |
| --- | --- | --- |
| `frontend/src/services/todoService.ts` | `todoApi` | `getAll · getByID · create · update · toggleComplete · delete · deleteMany(ids) · getStats`；用本地 `ensure<T>(value, message)` 把绑定层的 `T | null` 收成 `T` |
| `frontend/src/services/metaService.ts` | `metaApi` | **永远联网**：`searchWorks(keyword, page=1)`（返回该页 30 条 + `pageCount`）` · fetchWork(keyword, workId?) · coversDir() · cacheCover(workId, coverUrl)`（后者返回 `cache/` 里的缩略图 data URL）` · searchSteam(term)`（Steam 商店候选 ≤10 条）` · matchSteamApp(folderName)`（文件夹名 → appid，`matched=false` 不是错误）` · steamStats(appId)`（评价/在线/公告）；同样用 `ensure()` 兜底 null |
| `frontend/src/services/libraryService.ts` | `libraryApi` | `fetch(keyword, workId?, refresh=false, source='dlsite')`（**只读**缓存优先；`source='steam'` 时 `workId` 是 appid） · `storeIn(workId, options?, source='dlsite')`（**入库，唯一写库入口**；`options` = `{gamePath?, gameFolder?, gameFolderName?}`） · `list(source?)`（空 = **所有来源**） `· remove(workId, source?) · coverDataUrl(workId, source?)` · `getTags(workId, source?) · setTags(workId, tags, source?)`（**标签整份读写**：`TagSet{tags, edited}`，编辑过程不写库）；导出 `LibrarySource='dlsite'|'steam'`、`LIBRARY_SOURCE_DLSITE/STEAM` 与 `LibraryMeta`（= `WorkMeta` + `cached/workId/cover*/gamePath`） |
| `frontend/src/services/scanService.ts` | `scanApi` | `steamApps()`（**Steam 源**：`SteamApp[]` = appid/名称/安装目录/库根，来自 appmanifest） · `steamFolders()`（旧的 `common` 子文件夹列表，仍保留） · `executables(root)`（返回 `{root, folders[], skipped[]}`） · `subfolders(root)`（返回 `Subfolder[]`） · `pickDirectory(title?)`（原生文件夹选择框，取消返回空串）；导出 `ScannedGame{name,detail,origin,path?,folderPath?,source?,appId?}` 与 `Subfolder` |
| `frontend/src/services/gameService.ts` | `gameApi` | `status(workId, source?)`（返回 `{path, exists, running, pid}`，进入详情/轮询用） · `setPath(workId, path, source?)`（校验并写入 `library_works.game_path`） · `launch(workId, source?)`（返回 `{started, running, path, missing, message}`） · `pickExecutable(title?)`（原生选 `.exe` 框，取消返回空串） |

> 绑定层返回的是 `map[string]any` / `{ [k: string]?: any }`，封装层用 `as LibraryMeta` 收窄（否则 `vue-tsc` 报 TS2345）。注意 `frontend/bindings/` 是**生成物、被 gitignore**：用 `wails3 generate bindings -clean=true -ts -i` 生成（实测 beta.24 `-ts -i` 下服务与 models 产出的是 **`.ts`**，只有个别 runtime 包是 `.js`；`tsconfig` 的 `exclude` 只覆盖 `**/*.js`）。新克隆仓库后先跑一次生成命令，缺文件会直接解析失败。

### 4.5 `components/meta/*` 积木（作品库与调试页共用）

| 组件 | 作用 |
| --- | --- |
| `MetaSection` | 卡片外壳（`section-header` + 图标/标题/`#actions` 插槽 + `bodyClass` 内容区），全站唯一的"卡片长相"定义。**没有默认插槽内容时不渲染 body**，并把上下内边距交给 header（`pt-4` → `py-3`）—— 用于「只有标题 + 右侧动作」的顶栏卡片，避免空 body 把卡片撑高 |
| `MetaSearchBar` | 搜索行（`input-theme` + `btn-brand`），受控 `v-model` + `@submit`，不发请求 |
| `MetaSampleChips` / `MetaMessage` | 快捷示例 chips / 一行提示（`tone=info|error|success`，字面量表映射图标与颜色） |
| `MetaCover` / `MetaCoverFigure` | 封面缩略图（`sm` 44px / `md` 140×105，失败退占位图，`src` 变化复位）/ 封面 + URL 标注 |
| `MetaKeyValueList` / `MetaTags` / `MetaSubsection` / `MetaTextBlock` | 两列键值表（`dense`/`scrollable`）/ 不可点标签云（`limit` 控制显示个数，超出折 `+N`）/ 小标题块 / 长文本块（`whitespace-pre-wrap` + 限高） |
| `MetaTagEditor` | 作品详情「标签」：**抓取标签与自定义标签合并成一份** `chip` 列表 —— hover 标签右上角浮出 14px 圆形 ×（`title`/`aria-label` 提示删除，hover 填红），末尾固定一颗同款「+」（点开原地变输入框：先保持原宽、随输入变宽；回车/失焦提交、空输入无视、Esc 取消）；删除/新增**只改 store 的编辑稿**，`tagsDirty` 时标题右侧出现「保存」，点它才整份写库。数据 `metaStore.tagList / tagsDirty / addTag / removeTag / saveTags`（后端 `work_tags` + `work_tag_edits`） |
| `MetaWorkOverview` / `MetaWorkDetails` | 详情上半（封面+名称+关键字段+标签+入库/启动按钮+回执，`props` 含 `stored/gamePath/launching/showTags`，`emits: download/launch/changePath`；`showTags=false` 时不出标签云 —— 作品库把标签统一交给 `MetaTagEditor`）/ 下半（summary + outline） |
| `MetaDownloadButton` / `MetaSavedNotice` / `MetaCoversPath` | **入库 / 游戏启动按钮**：未入库 = `入库`（无封面禁用）/ `入库中`；已入库 = `游戏启动`（play 图标；`running` 时「运行中」+ activity 图标并禁用；`launching` 时「启动中」）+ 有位置时旁边一个轻量「更换位置」文字按钮（`i-lucide-folder-search`，运行中/启动中禁用）。**记了位置但 exe 不存在时启动按钮禁用**（页面同时给提示），路径选择沿用原系统文件框 / 成功回执 + `file://` 打开 / 封面落地目录（失败静默留空） |
| `MetaCandidateList` / `MetaPager` / `MetaRawJson` / `MetaEmptyState` | 候选行（`@select` 出编号）/ 分页条（`page · pageCount · total` → `@change`，页码显示前 3 + 当前 + 后 3 并带省略号，另有输入页码跳转，复用 `chip`/`chip-active`）/ 原始 JSON + 复制 / 空态 |

拆分层级：**store = 拿数据 → 积木 = 长什么样 → views = 放哪**。加新页面只需组合积木，不用再写请求。

### 4.6 TODO 组件公开接口

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
11. **点击卡片 = 复制「标题 + 描述」**（用户要求）：`TodoItem` 根 `<div>` 绑 `@click`，拼成 `标题\n描述` 后 emit `copy`，由 `TodoList.handleCopy` 走 `utils/clipboard.ts` 的 `copyText()`（Clipboard API 失败退 `execCommand`）并弹 toast「文本已复制到剪贴板」。**刻意不加 `cursor-pointer`** —— 指针保持默认，整行不暗示可点；选择框加 `@click.stop`、行内按钮本来就有 `.stop`，点它们不会误复制。

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
- **不新增依赖模块**：`frontend/package.json`、根 `package.json` 自设计系统重构以来未变。`go.mod` 的模块集合也没变，但 `golang.org/x/sys` 因 Steam 扫描要用 `windows/registry`，从 indirect 提升为 **direct require**（仍是 v0.47.0，本地早已存在，无新下载；`go mod tidy -diff` 为空）。
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
| 用于 | 搜索框（`components/meta/MetaSearchBar.vue`，作品库与调试页共用） | TODO 添加框（`components/todo/TodoInput.vue`） |

**控件高度统一走 `control-h`**（`uno.config.ts:161`，`h-[2.2rem]` = **35.2px**）：`btn-brand`（`uno.config.ts:164`）、`input-theme`、`TodoInput.vue` 的外壳三处都挂它，因此三者同高。⚠️ **挂了固定高度就不要再写纵向 padding** —— 总高已由 `control-h` 决定，`py-*` 不会再改变盒高，只会把内容挤出盒子（`TodoInput.vue` 内层 input 因此不带 `py-*`，垂直居中交给外壳的 `items-center`）。历史教训：输入框曾用 `py-2.5`、按钮用 `py-2`，高度由「内容高 + 纵向 padding」撑出，实测 41.6px vs 35.2px，同一行 form（`items-end`）底部对齐时顶部错开 6.4px。

两者实测盒模型**逐项相同**：`border-radius: 10px`、`padding: 10px 12px`、`font-size: 14px`、`line-height: 20px`、`border: 0.8px solid var(--color-border)`、总高 **35.2px**；聚焦时边框都变 `var(--color-brand)`。**任一项（内边距 / 圆角 / 字号 / 行高 / 边框色 / 聚焦色 / 高度）都必须在两处同步修改**，否则两个输入框的圆角与高度会分叉。

**提交按钮统一走 `btn-brand`**（`uno.config.ts:164`）：TODO 的「添加」与作品库/调试页的「搜索」类串同为 `btn-brand flex items-center gap-1.5 shrink-0`（搜索按钮在 `components/meta/MetaSearchBar.vue`），TODO 额外带 `disabled:opacity-50 disabled:cursor-not-allowed`（短输入要能看出来不可点，`btn-brand` 本身不含禁用态）。`HomePage.vue:10` 也用它 —— 即全站按钮已统一到这一个 shortcut，不要再手写 `rounded-lg px-4 py-2 ...`。

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
| `frontend/src/views/test.vue` | 薄壳（只渲染 `MetaApiTestCard`），DLsite **调试页**（路由 `/testing`，见 §3.3） |
| `frontend/src/views/WorksPage.vue` | 作品库：**只剩作品信息**（详情卡）+ **「标签」**（`MetaTagEditor`：抓取标签与自定义标签合并成一份，hover 右上角 × 删除、末尾「+」原地添加，改完点「保存」整份写进 `work_tags`；概览卡的标签云在这里关掉 `:show-tags="false"`），缓存优先。**侧栏无入口**，只从游戏库点封面进入（用户要求）。搜索入口在游戏库网格末尾的加号圆钮（「添加游戏」窗里的「手动搜索」）。**尚未做**收藏、列表分页、批量入库、历史记录；从游戏库进来后没有显式「返回游戏库」按钮（靠侧栏点游戏库） |
| `frontend/src/views/LibraryPage.vue` | 游戏库（路由 `/library`）：已入库列表 + 删除 + 跳详情 + 网格末尾透明圆形加号按钮（打开「添加游戏」窗：扫描添加 / 手动搜索，候选每页 10 个）。点封面进 `/works` 后可「游戏启动」（`library_works.game_path`）。**尚未做**游戏库列表排序/分页、批量删除、在网格卡片上直接启动/打开所在文件夹、封面缺失时自动补下 |
| `internal/service/library_service.go` | 游戏库只跑了离线单元测试（`internal/database/library_repo_test.go`），**Wails 端的真实接线未实机验证** |
| `internal/service/getMeta/*_test.go` + `testdata/` | 本工作区当前是**已从磁盘删除（未提交）**状态（属用户「裁剪」清理），恢复：`git checkout -- internal/service/getMeta` |
| `internal/service/todo_service.go:315` | 一行 `// bug need-fix` 注释（`GetStats`），逻辑本身正确，属陈旧标记 |
| `main.go:62` | `Description` 仍是 `"A demo of using raw HTML & CSS"` |
| `build/config.yml` | 仍是模板（`companyName="My Company"`、`productName="My Product"`、`version 0.0.1`） |

### 9.2 需**实机**确认（无头浏览器验证不到）

- **托盘**：托盘图标是否出现在通知区域、左键是否只唤回不隐藏、窗口已显示时再点托盘应无反应、右键菜单「显示主窗口/退出程序」是否可用 —— 浏览器观察不到 Windows 通知区域。
- **DLsite 测试页（`/testing`）**：浏览器里没有 Wails 后端，`/wails/runtime` 一律 404，因此**所有真实调用都必须实机验证**（用户明确说「由于网络限制因素 不需要测试 完成后我手动测试」，故本地未跑任何联网调用）。需要确认的点：
  1. 按 `RJ01721474` 精确抓取 → 元数据是否与页面一致、`coverUrl` 是否已是 `_img_main.jpg`（560×420 那一档，不再是 240×240）；
  2. 封面在 webview 里能否直接显示（`Referer` 已设为 `https://www.dlsite.com/`，`<img referrerpolicy="no-referrer">`）；
  3. 点「入库」是否真的在**工作目录下**生成 `covers/`，文件名形如 `RJ01721474_cover.webp`（DLsite 有同名 webp 档位时应优先拿到 webp；没有 webp 时回落到 `.jpg` 等原格式）；卡片里显示的是 `absPath`，以那个为准；入库成功后按钮应立刻**变成「游戏启动」**（不再是「已入库 + 禁用」），且**退出再进 `/works` 仍是「游戏启动」**（此时靠后端注入的 `coverAbsPath`，不是会话内缓存）；
  4. 模糊搜索（如 `エルフ パラダイス`）是否返回候选列表；
  5. 「打开已保存的图片」的 `file:///` 链接在 Wails webview 里是否可用（**把握最低的一项**，不可用不影响主流程）；
  6. **搜索候选缩略图**：首次搜索会把图缓存到工作目录的 `cache/`（目录自动创建），列表里的小图应能显示；删掉 `cache/` 再搜一次仍能显示（重新下载）；结果多于 10 个时出现分页条，翻到哪页才下哪页的图（看 `cache/` 文件数增加）；
  7. **退出清缓存**：正常退出应用后 `cache/` 应为空（目录还在），`covers/` 里的入库封面不能被删；
  8. **只有点「入库」才进游戏库**：搜到的作品不应出现在 `/library`；点「入库」后 `covers/` 生成图、库里多出一张带封面的卡片，`/works` 的按钮变成「游戏启动」。**入库时就记录 exe**：扫描添加「按 exe 识别」直接记那个 exe；「按文件夹名 / Steam 库」由后端进文件夹挑主程序（排除 UnityCrashHandler64、优先 Game.exe、否则取与文件夹名匹配最多的）；手动搜索入库会先弹 `.exe` 选择框，**取消也能正常入库**（位置留空）。再点「游戏启动」：已有位置就起进程；没位置/之前取消过则这时弹选择框，选完存库并启动。**启动后按钮几秒内变「运行中」并禁用；手动关掉游戏后（≤5 秒轮询）变回「游戏启动」。把 exe 挪走/删掉后再进详情：按钮应变为禁用并显示"游戏程序已被删除或移动了位置，请点「更换位置」…"，用「更换位置」重选后可正常启动并清除提示。**
  9. **设置页「清理缓存」**：进入设置页应显示 `cache/` 当前占用（先搜几次让 `cache/` 里有图）；点「清理缓存」后占用归零、`cache/` 里文件消失、`covers/` 里的入库封面不受影响；没有缓存时按钮应禁用并显示「当前无缓存」。

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
| `covers` 用**相对路径**而不是 `os.UserConfigDir()` 之类的绝对路径 | 用户原话「保存在工作区目录下的 preImg 使用相对路径」（2026-11-16 目录改名 `covers`，策略不变）。代价是落点随进程工作目录变化（开发态是仓库根），所以返回值同时给 `path`（相对）与 `absPath`（绝对），界面上直接显示 `absPath`，不留"到底存哪了"的歧义 |
| 下载封面**不覆盖**已有文件，同名加 `-2/-3` 后缀 | 同一部作品可能被反复入库（库记录被删后重收、调试页重复点），覆盖会把上一次下到的图悄悄冲掉；`O_EXCL` 还能顺带防止两次并发请求写同一个文件 |
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
| 作品库与调试页**共用** `stores/metaStore.ts` 与 `components/meta/*`，而不是各写一份 | 用户要求「MetaApiTestCard.vue 内容高度组件化，方便重新排列使用」。拆三层：**store 拿数据 / 积木定长相 / views 定排布**。两页因此只是同一批积木的两种组合：调试页单列四卡（含原始 JSON），作品库双列（详情+候选栏）。请求、错误兜底、封面下载等逻辑只有一处，改一次两页同步 |
| 状态放 **Pinia store** 而不是 composable（`composables/useMetaSearch.ts` 已删） | 用户明确要求「本项目使用了 pinia 所以状态管理相关尽量写在 stores 文件夹中」。附带好处：同一个 store 实例被两个页面共享，切页不丢搜索结果 |
| 游戏库表**来源无关**（`source` 列 + `raw_json`），而不是 `dlsite_works` 专表 | 用户要求「表存储元数据来源 来源于dlsite则为dlsite字段 剩下存储为id等作品信息」。公共列存跨来源可比的元数据（id/名称/社团/系列/发售日），来源特有字段整份塞 `raw_json`；唯一键 `UNIQUE(source, work_id)`，将来接别的源加一行常量即可，不动表结构 |
| 游戏库表**惰性创建**（`ensureLibraryTables` 每次读写前 `CREATE TABLE IF NOT EXISTS`），不在 `initAllTables` 里建 | 用户要求「当需要保存到库中时 即是需要下载数据才新建表 节约空间」。代价是每次多条 DDL 语句（幂等，代价可忽略）；换来的是从不用游戏库的用户库里不会凭空多两张空表 |
| `SQLiteService` 的 `Execute`/`Query` 加 `mu sync.Mutex` **串行化** | 惰性建表让「并发首次使用」天然撞上 DDL 竞争，实测 `go test ./internal/database/` 报 `database is locked (5) (SQLITE_BUSY)` —— 正是用户提醒的「使用表时可能会导致数据库异常」。SQLite 多连接并发写同一文件必 BUSY；桌面端串行化开销可忽略，`Query` 一次性取完行不把锁泄给调用方 |
| 游戏库封面走 **base64 `data:` URL**（`CoverDataURL`），不用 `file://` | Wails webview 对本地文件协议的支持没实机验证过（§9.2 第 5 项把握最低），data URL 一定可用；封面几十~几百 KB，一次列表请求可接受 |
| **删除库记录不删磁盘图片**（已于 2026-11-16 改为删除入库封面） | 图片是用户自己的文件，应用删库记录属于"清理索引"，顺手删文件是不可逆的越权操作。**这条已按用户新要求推翻**：现在只删本应用入库时保存的 `covers/` 文件，见下面「封面目录改名 + 删图」决策；用户自己另存/转换的图片仍不受影响 |
| 缓存优先放**新服务** `LibraryService`，不往 `MetaApiService` 加开关 | 两个语义相反：调试页要**永远联网**（否则解析改动被缓存掩盖），作品库/游戏库要**缓存优先**。混在一个方法里只能靠布尔开关，调用方永远搞不清这次到底走没走网络；两个服务各自语义单一，前端 `metaApi` / `libraryApi` 一眼可辨 |
| **`composables/` 目录取消**，唯一残留的 `useAutoHeight.ts` 改名 `stores/autoHeightStore.ts`（`useAutoHeightStore`），`TodoList.vue` 改 `collapse.enter` 等 | 用户要求「本项目 vue3 已经引入了 pinia 所以 composable 应该封装进 store 中 记得调整相关使用了该函数的库的相对路径及其名称」。项目里共享逻辑只认 stores 一个入口，避免两套目录各放一半 |
| **作品库 `/works` 在侧栏没有入口**，只能从游戏库点封面进入（路由保留） | 用户要求「保留 router 中游戏库相关前端 删去作品库（仅仅只是在 router 中 后面要复用作品库相关 component）」，随后明确「不在 sidebar 显示 而是通过游戏库进入」。于是：路由留着（游戏库 `openWork()` 要跳它）、侧栏只留「游戏库」，导航里不会出现两个"库"让人分不清；`WorksPage.vue` 与 `components/meta/*` 一行没删 |
| 游戏库卡片 = **封面网格 + hover 浮现名称 + 右上角删除**，点封面进作品库 | 用户要求「预览图横向排列、在预览图下方显示名称（hover 时半透明界面从下方淡入）、点击预览图进入原本的作品库界面、右上角添加删除按钮」。实现要点：半透明背景只能走 scoped 的 `color-mix(in srgb, var(--color-bg-base) 88%, transparent)`（UnoCSS rules 精确匹配，`bg-secondary/50` 会被静默丢弃）；名称面板用 `translate-y-full → group-hover:translate-y-0` 做"从下淡入"；删除按钮与封面按钮是**同级兄弟**（button 不能嵌套 button），用 `@click.stop` 隔离点击。**连点两下的待确认态在 `@mouseleave` / `@blur` 时自动撤销**（用户要求「在移开时取消二次确认 仿照 todoitem 的删除按钮逻辑」，`cancelArm()` 与 TodoItem 的 `cancelDelete()` 同款）。外观也与 TodoItem 对齐：形状走 `btn-row`，静息 `opacity-75`、hover/聚焦提亮到 100，颜色状态（红描边 → hover 填红 → armed 填红 + `scale-110`、文字用 `--color-text-on-danger`）写在 `.lib-del`，**与 `.row-btn-delete` 必须同步改** |
| **搜索移到游戏库顶部的居中弹窗**，作品库不再有搜索卡 | 用户要求「移除上方作品库·DLsite元数据卡片（搜索功能保留 并添加在游戏库界面的上方 通过按钮显示搜索 通过悬浮窗弹出提示搜索 弹窗在中央 显示时周围模糊）」。因此 `/works` 只剩作品信息（也去掉了「封面保存位置」「游戏库」两张卡）；弹窗用 `Teleport to="body"` + `z-[1300]` 避免被页面层级/overflow 裁掉，遮罩 `color-mix(...) + backdrop-filter: blur(8px)`；**关闭只留 ✕ 与 Esc，点遮罩不关**（用户要求「取消…点击搜索卡片以外区域会退出搜索界面的逻辑」，避免手滑点旁边就把搜索关掉） |
| 「下载封面」文案改成**「入库」** | 用户要求「下载封面字样改为"入库"功能不变（下载文本数据 图片 并加入库 载入数据库相关数据）」。`MetaDownloadButton` 默认 `入库` / `入库中`。**注：随后用户又要求"搜索结果不能自动入库"，语义已升级为"点入库才写库"（见下面一行），本行只保留文案变更的历史** |
| **搜索/浏览只读，只有「入库」才写库** | 用户要求「搜索内容只有在点击入库后才会下载封面并且在游戏库中显示 在搜索内容后点开就会自动入库 去除该逻辑 文本和图片在点击入库后才下载并载入数据库中」。`LibraryService.Fetch` 去掉 upsert（搜索、点开详情都不再写库），新增 `Store` 负责"抓文本 + 下封面到 covers + 写库"，前端只有 `metaStore.storeIn()` 调它。代价：打开未入库作品每次都联网（没有缓存可命中），换来的是游戏库只装用户显式收的作品 |
| **搜索缩略图缓存在 `cache/`，与 `covers/` 分开** | 用户要求「并且显示在搜索界面的缩略图(载入缓存文件夹cache中 没有就创建一个)」。`MetaApiService.CacheCover` 把候选小图落到工作目录的相对路径 `cache/`（不存在就 `MkdirAll`）并返回 data URL；文件名固定 `<编号>_cover.<ext>`（缓存语义，允许覆盖，已缓存不重复联网）。与 `covers/` 的区别：`covers/` 是入库后的正式存档（`-2/-3` 不覆盖），`cache/` 可以整目录删掉重下 |
| 游戏库顶栏压成**单行**：标题 + 右侧「已入库 N 款」+ 搜索 + 刷新 | 用户要求「将游戏库界面顶部卡片文字高度居中 并缩减卡片高度 并在游戏库右边显示已经入库多少款游戏」。做法：`MetaSection` 在没有默认插槽内容时不渲染 body，header 上下内边距改 `py-3`（有 body 时仍是 `pt-4`），卡片高度因此只剩「按钮 35.2px + 24px」；标题/数量/按钮同处 `items-center` 的 header，天然垂直居中。数量用 `count`（`withCoverCount` 随之删除，页面上不再有"多少部已下载封面"）。之后按用户要求把「刷新」调到「搜索」左边 |
| **搜索列表直接显示预览图**（不再"点详情才有图"） | 用户要求「原本要点击详情才加载预览图 现在在搜索界面直接显示预览图」，并明确要载入 `cache/`。根因不在前端：DLsite 搜索页的缩略图是 Vue 组件 `:thumb-candidates="['//img…']"`，旧的 `parseSearchItem` 抓的是块内那个 `src="data:image/gif…"` 的弹层占位图 → `Cover` 是个内联 gif，`CacheCover` 拿到 `data:` URL 直接放弃（http 请求不认这个 scheme），所以列表永远空白、cache 里也没有文件。修复：解析 `:thumb-candidates`（`extractSearchItemCover`），`CacheCover` 显式跳过 `data:` 占位图；前端仍用 `covers[id]`（cache → data URL）优先、远程直链兜底 |
| 搜索结果**每页 10 条**，只给当前页下预览图；**退出时清空 `cache/`** | 用户要求「搜索卡片分页 每10个搜索结果为一页 只有当前页面需要加载预览图 以节约网络资源 每次退出软件前 清除cache中文件」。分页状态放 store（`candidatePage` / `pagedCandidates` / `setCandidatePage`），`setCandidatePage` 只对当前页调 `loadCandidateCovers`，已缓存过的直接命中 `cache/` 不再联网；清理挂在 Go 的 `app.OnShutdown` → `getmeta.ClearCacheDir()`（只删 `cache/` 里的文件，`covers/` 不动）。分页条本身在 `MetaPager`：前 3 页 / 当前页 / 后 3 页 + 输入页码跳转（用户后续要求「前三页和后三页可以跳转 并支持输入数字跳转到相应页面」） |
| 搜索**不再受 30 条限制**：按 DLsite 源数据页懒加载（1~3 页取第 1 面，4~6 页取第 2 面…） | 用户要求「检查搜索结果是否有限制 如果有则取消 可能是30个搜索结果加载限制」，随后明确「DLsite搜索界面每一面只有30条 能否在本页面在3页内时加载DLsite源数据页第一面 后三的倍数页加载第二面」，并要求「首先获取搜索结果个数调整页面数量 在需要加载源数据第二面内容时才重新爬取加载」。实测 `per_page` 无效（传 60/120 仍只回 30 条），只能翻页：`SearchWorks` 加 `page` 入参，返回 `pageCount` 与**精确总数** `total`（页面 `data-search-result-count`，如 1710 → 57 页）；前端 `sourcePages` 存"DLsite 页号→30 条"，总页数直接由精确总数算，`ceil(前端页/3)` 决定需要哪一面，**用到才取**；翻到第 3/6/9…页时静默预取下一面 |
| 「入库」按钮在**已入库后变 `已入库` + 禁用**（**已被下面「游戏启动」一行取代**；判定 `stored` 的规则沿用） | 用户要求「作评库界面的入库按钮 在打开界面时如果已经入库则显示已入库 并将按钮的状态改为不可按下」。判定用**库里已有封面文件**（后端注入的 `coverAbsPath` 非空，或本次会话刚下过 `saved`），而不是"在游戏库里" —— 元数据随抓取就入库、`/works` 又只从游戏库进入，按"在库里"判定会让按钮永远不可点、封面永远下不来。禁用还顺手挡掉重复下载：`writeFileUnique` 不覆盖同名文件，重名加 `-2/-3` 后缀，多点一次就多一份重复图 |
| 已入库后**不再显示「已入库」，替换为「游戏启动」** | 用户要求「已经入库游戏的入库按钮 现在是在入库后变为已入库并将按钮状态改为不可按 改为入库后不显示替换为游戏启动 具体逻辑为按钮组件添加逻辑判断 调用数据库中存储的游戏位置启动(如果没有就新建相关条目或新建表)」，随后明确「游戏路径和 library_repo.go 合并在一个表内」。实现：①位置就是 `library_works.game_path` 一列（不单开表）—— 与元数据同表，但 `UpsertLibraryWork` 的 `ON CONFLICT SET` 不含它（重抓元数据/重新入库不冲掉位置），旧库文件由 `ensureGamePathColumn` 用 `PRAGMA table_info` + `ALTER TABLE ADD COLUMN` 补列；②`MetaDownloadButton` 按 `stored` 分支：未入库 = `入库` / `入库中`，已入库 = `游戏启动`（play 图标；`launching` 时「启动中」），不再有禁用态；③点击走 `metaStore.launchGame()` → `GameService.Launch` 用记录的绝对路径起独立进程（`cmd.Dir` = exe 所在目录、`DETACHED_PROCESS`、`Process.Release()`）；④没位置/文件已不在时后端返回 `missing=true`（不报错），前端弹 `DialogService.PickExecutable` 选一次 `.exe` → `SetPath` 写库 → 再启动；已设置位置时按钮旁另有轻量「更换位置」；⑤扫描添加「按 exe 识别」的每行携带 `path`，`Store({gamePath})` 入库时写入位置，所以扫描入库的游戏第一次点「游戏启动」就能直接跑。原「已入库 + 禁用」防重复下载的目的，现在由"已入库后根本没有入库按钮"达成 |
| **启动按钮先检测 exe + 运行状态，运行中/文件丢失都禁用** | 用户要求「进入相关界面时启动按钮检测 首先检测游戏的路径文件exe文件是否存在(如果不存在提示用户游戏已经删除或变更位置 并将原本按钮改为不可按下状态 路径选择复用原本ui 不需要改变) 使用gopsutil等库(可以不是该库 自行判断使用什么库合适 用来读取特定程序运行状态) 通过数据库的路径信息 如果游戏路径正确则判断游戏是否运行(运行就将游戏路径载入缓存中 方便实时查看游戏运行状态 不再运行如果路径在缓存中 就将路径移除缓存) 如果运行便显示运行中相关字样 没有运行就显示运行相关字样」。实现：①新方法 `GameService.Status(workID)` 返回 `{path, exists, running, pid}`；进入 `/works`（`applyWork`）检测一次 + 页面每 5 秒轮询 + 窗口 focus 刷新；②exe 不存在 → 清运行缓存，前端 `gameWarning` 提示"游戏程序已被删除或移动了位置"，`launchBlocked` 让启动按钮禁用，「更换位置」仍可用（沿用原系统文件框，UI 未改）；③运行检测**不引 gopsutil**：`x/sys/windows` 的 Toolhelp 快照按 exe 名粗筛 + `QueryFullProcessImageName` 比对完整路径（同名 exe 不误判），`GameService` 用 `running map[小写绝对路径]pid` 做运行缓存，缓存命中先 `processAlive` 复核（防 PID 复用），失效/未命中再全量扫；④运行中按钮显示「运行中」并禁用（防重复启动），游戏退出后轮询自动切回「游戏启动」；`Launch` 成功后把 PID 直接载入缓存 |
| `loadById` **不把编号写回搜索框**，且搜索弹窗**每次打开都清空输入框** | 用户要求「在打开游戏库中游戏时 搜索界面会自动填入作品rj号 取消该逻辑」。两处一起收拾：①`loadById` 原本顺手 `keyword = id`；②`runSearch` 会把搜过的关键词（常是 RJ）留在 store 里，弹窗重开就带出来。现在 `openAddGame()`（原 `openSearch()`）与 `finishSearch()` 都清 `meta.keyword`，搜索框只反映"这次开窗后自己输入的"。代价：不再"沿用上次关键词方便反复调整"；失败的搜索仍留在框里，便于改错 |
| 游戏库顶栏**去掉「搜索」按钮**，添加游戏改成网格末尾的**透明圆形加号按钮** | 用户要求「在游戏库卡片的游戏卡片最后加一个加号button以添加游戏 加号替换为原本headbar搜索按钮功能 headbar搜索按钮去除」，随后补充「添加游戏为透明圆形ui」。实现：`.lib-add` 是 3.5rem（56px）见方、`border-radius:50%`、**全程透明底**（静息与 hover 都不填色）的圆形按钮 —— 虚线 `--color-border-dark` 描边 + 居中 `i-lucide-plus`，hover / 键盘聚焦只把描边与图标换成品牌色；它放在网格最后一位的 `flex-center` 容器里（刻意不做成 4:3 方块，圆形能把"入口"和作品卡区分开）。虚线边框整条写在 scoped 样式里：UnoCSS 的 `border-*` rules 都是 `1px solid` 简写，再叠 `border-dashed` 会互相覆盖（同属性只留一条）。空库时网格仍然渲染（`v-if="works.length \|\| !loading"`），保证唯一的添加入口始终可见，空库提示改成一行文字（原 `MetaEmptyState` 在本页不再使用） |
| **搜索栏为空时点「搜索」只提示「搜索栏不能为空」，绝不跳详情页** | 用户报 bug：「点击搜索按钮 如果搜索栏为空会弹出上次的搜索结果详细页 修改逻辑 改为点击搜索按钮时 如果搜索栏为空 提示搜索栏不能为空」。根因不是没做空值校验（`runSearch` 本来就有早退），而是**调用方拿 `meta.work` 当"搜索成功"的标志**：空关键词早退时不会清 `work`，上一次搜到的作品还留在 store 里，于是 `submitSearch` 的 `if (meta.work) finishSearch()` 照样跳 `/works`。修法两层：①`runSearch` 改成返回 `boolean`（这次是否真的发起了搜索），空值统一提示 `搜索栏不能为空` 并返回 `false`；②`LibraryPage.submitSearch` 只认返回 `true` 才看 `work`。提示复用弹窗里现成的 `MetaMessage tone="error"`，不另做 toast。**同类坑：判断"有没有结果"不能用 store 里的残留状态。** |
| 设置页新增「清理缓存」：进页扫描 `cache/` 占用显示在按钮旁，一键清空（`covers/` 不动） | 用户要求「设置界面添加清理缓存功能(将缓存cache内所有文件删去 并在进入设置界面时运行dir命令 扫描cache文件夹大小 并显示在清理缓存功能旁边)」。分两层：①后端 `MetaApiService` 加 `CacheInfo`（`filepath.WalkDir` 统计 `{dir,files,bytes}`，目录不存在算 0，不联网、不建目录）与 `ClearCache`（复用 `ClearCacheDir` 的删除逻辑，回 `{files,bytes,rest}`）；②前端新组件 `components/settings/CacheCleanup.vue`，`onMounted` 调 `CacheInfo` —— 设置页是**动态导入的路由页、没有 keep-alive**，每次进入都会重新挂载，所以 `onMounted` 就等价于"进入设置界面时扫描"，不必额外写路由钩子。按钮旁显示 `占用 X（N 个文件）`（无缓存时显示「当前无缓存」并禁用），清理后**重新统计一次**、以真实占用为准（`rest > 0` 说明有文件删不掉，能立刻看见）。按钮颜色沿用项目的破坏性语义（透明底 + danger 描边/图标 → hover 填红 + `--color-text-on-danger`），反馈是按钮下方一行 4 秒自动消失的小字；**缓存是可再生的临时图，所以刻意不做连点两下确认**（与 TODO/游戏库的删除区别开），也不建模态框。 |
| **游戏库「添加游戏」改为双模式：扫描添加（新）+ 手动搜索（原流程）** | 用户要求「将这些函数整合到前端调用 功能是添加游戏的新方法 原先搜索改为手动搜索 将获取到的游戏名称导入到搜索函数获取相关游戏信息 并让用户决定是否入库」。实现：`main.go` 注册 `ScanService`（第 6 个服务）+ 新增 `DialogService.PickDirectory`（第 7 个服务，原生选文件夹）；后端扫描逻辑不变（只读、不联网、不写库）。前端新增 `services/scanService.ts`（`scanApi`）、`stores/scanStore.ts`（来源/方式/路径/结果）、`components/library/ScanPanel.vue`；`LibraryPage` 弹窗标题改「添加游戏」，顶部 `chip` 双模式切换，默认「扫描添加」：来源 Steam（一键扫）或本地文件夹（路径输入 + 「选择…」原生框 + 「按 exe 识别 / 按文件夹名」），扫出名字后**自动逐个搜索**（3 路并发，可「停止」/单行「重试」），结果按扫描顺序一行一个（封面 + 匹配作品名 + RJ + 扫描名），行可勾选，工具条提供「全部入库」/「入库所选」（**串行**写库；已入库的行显示「已入库」且不再参与），点「详情」进 `/works` 核对；「手动搜索」保留原搜索行。扫描结果留在独立 store（关窗不丢）。**弹窗内容整体可滚动**（`max-h-[80vh]` + body `overflow-y-auto`），避免结果列表与候选同屏时撑破弹窗。 |
| 扫描结果改为**全部直接搜索 + 一键全部入库 / 选择部分入库** | 用户要求「扫描添加的点文件去搜索改为全部直接搜索 并将搜索结果排列展示用户 用户决定是否添加」「分为一键全部入库 和选择部分入库」。实现：`scanStore` 扫完自动 `searchAll()`（3 路并发、`searchToken` 代次作废旧请求、可停止、单行重试）；每行 `ScanMatch{state, workId, workName, cover, selected, stored, storeError}`，匹配到作品时用 `library.has()` 预标「已入库」；工具栏「全选 / 全部入库(N) / 入库所选(N)」，批量走 `storeMany()` **串行**调 `libraryApi.storeIn`（别让几十个"抓文本+下图"同时压网络与 SQLite），每行独立报错；成功后 `library.load(true)` 让关窗后网格立刻出现。入库返回的完整元数据回填到行，库里已有的行直接读库里的列。**详情补全**：搜索完成后再对匹配行逐条 `libraryApi.fetch('', id)`（缓存优先、只读）抓详情，`match.info = 社团 · 作品形式 · 发售日 · 容量` 显示在行里（失败显示「详情获取失败」原因，不影响入库）。封面直接用 DLsite 远程 URL（`MetaCover` 自带失败占位），不批量下 `cache/`。 |
| 入库前校验元数据**完整性**（扫描入库也要拿到作品信息） | 用户要求「扫描入库的游戏也应获取作品相关信息」。`LibraryService.Store` 本来就按编号抓详情写库；这次加一道底线校验 `workIsComplete`（`id` 与 `name` 都非空）：抓回来是空壳就报错、**不写库**；库里已有 `raw_json` 若是残缺的（旧数据/解析退化）也不复用，改为联网重抓。扫描行在入库成功后用返回的 `maker/workType/releaseDate` 回填一行摘要，让"信息已抓到并保存"肉眼可见。测试：`internal/service/library_service_test.go`（`TestWorkIsComplete` 7 例、`TestRowFromWorkCarriesWorkInfo`）。 |
| **标签：多值切分 + 更多来源 + 数量可配 + 用户自定义标签** | 用户要求「标签分割有问题 连在一起了 尝试使用空字符和/分割标签 并进行检查 并且标签含有json中的 声优 分类 插画 剧情 作者 社团名 且标签的数量可以由用户自定义 在设置界面中 且用户可在作品详细页中自行添加标签」。三块：①解析：`extractTags` 的标签来源扩充到 分类/作品形式/社团名/作者/剧情/插画/声优，多值字段**按元素取**（中文页元素间无空白，整段 `dd.Text()` 会粘成「异种X造孽…」「みたかりん野上菜月」），最后用 `splitTagText` 按空白与 `/` 切分、去重，`defaultTagLimit` 放宽到 64；②显示：设置页「标签显示数量」`TagLimitInput`（4–64，默认 24）写进 `stores/App.ts` 的 `tagLimit`，`MetaTags` 按 `limit` 截断并折 `+N`；③自定义：新表 `work_tags`（惰性建表、`UNIQUE(source,work_id,tag)`）+ `LibraryService.ListTags/AddTag/RemoveTag` + 前端 `MetaUserTags`（详情页输入/移除，存后端，与抓取标签分开、重抓不丢）。测试：`TestSplitTagText`、`TestParseWorkTagsIncludeDetailFields`、`TestWorkTagsLazyTableAndCRUD`、`TestNormalizeUserTag`。**注：③的「逐条加/删」与 `MetaUserTags` 已被 2026-11-22「标签改成合并 + 原地编辑 + 显式保存」一行取代**（标签改为整份保存，`ListTags/AddTag/RemoveTag` 换成 `GetTags/SetTags`）。 |
| **DLsite 换模板后"只有封面和名字"的根因修复** | 用户报「依然没有抓取到详细信息 只有封面和作品名 应该抓取详细页的信息 检查是否抓取错误」。排查：DB 里扫描入库的行 `raw_json` 只有 6 个键（约 280B），maker/workType/releaseDate 全空；实测页面发现 2026-11 DLsite 换新版（Vue + JSON-LD），旧的 `#work_outline` / `#work_maker` / `<meta itemprop>` 全部失效；并且代码发的是 `locale=zh-cn`，页面标签变成中文（社团名/发售日/文件容量），只认日文标签同样取空。修法：`parseWork` 双模板兼容（旧 `#work_outline` + 新 `div.c-productInfo` 的 dt/dd），价格/评分/社团/拼音/简介改读 JSON-LD Product，标签用 `outlineValueAny` 多语言候选，genre 用 `href*="/genre/"` 识别、`SeriesID` 取 `title_id/SRI…`、`MakerID` 取 `maker_id/RG…`/`brand.url`。配套：`workIsComplete` 收紧为「id+name+至少一个详情字段」，`Fetch` 与 `Store` 遇到旧空壳都当缓存未命中重抓（`/works` 立刻能显示详情，重新入库会把库里的空壳覆盖）。测试：`getMeta/new_template_test.go` + 两个 fixture（日文标签精简页、中文真实页）。 |

| **入库时就记录 exe（按扫描来源决定路径）** | 用户要求「在游戏入库时便存储相关exe 具体为记录结果通过什么文件夹名或exe搜索获得 如果通过文件夹名获取 读取该文件夹内exe记录位置(存在多个exe名则选择匹配文件夹名字字符最多的那个 首先排除含有UnityCrashHandler64.exe文件 优先选择Game.exe文件为路径) 如果通过exe名称搜索获得则直接使用改exe路径 手动搜索添加则在入库时便向用户请求路径 如果不提供便在需要启动时才请求路径」。实现：①扫描结果每行记 `origin`（exe/folder）+ `path`/`folderPath`；`ScanSubfolders` / `ScanSteamGameFolders` 改为返回名字+绝对路径（`Subfolder`）；②`StoreRequest` 增加 `gameFolder`/`gameFolderName`，`LibraryService.resolveStoreGamePath` 决定位置：直接给 exe 用之，给文件夹则调 `scan.PickGameExecutable`（排除 `UnityCrashHandler64.exe` → 优先 `Game.exe` → 否则选 exe 名与文件夹名最长公共子序列最长者）；挑不到/都没给就留空，**不阻断入库**；③手动搜索入库（`metaStore.storeIn`）先弹系统文件框选 exe，取消则照常入库但位置留空、启动时再问；已记录位置的不重复问；④扫描批量入库不再提示，路径随 `storeIn` 写入 `library_works.game_path`。 |

| **封面目录 preImg → covers；下载优先 webp；删库记录时删图** | 用户要求「preImg文件夹改名为covers(记得更改相关路径) 入库时保存的本地图像格式改为webp 在从游戏库移除游戏时 同时删除入库后保存的图像」，随后明确「不需要额外导入库 在下载资源到本地时优先检查是否可以下载webp格式图像」。实现：①目录/方法/前端全部改名（`coversDir`、`CoversDir`、`coversPath`、`MetaCoversPath`）；`MigrateLegacyCoverDir` 在 `MetaApiService.ServiceStartup` 里把旧 `preImg/` 改名（已存在则逐文件合并）并 REPLACE 库里 `cover_*` / `library_cover_files` 的旧路径，幂等；②不引编码库：`fetchImagePreferringWebP` 给 jpg/jpeg/png URL 拼同名 `.webp` 候选，单次探测，拿到 webp 就用，否则回落原图（搜索缩略图同样优先 webp）；③`LibraryService.Remove` 先按 `ListCoverFiles` + `cover_abs_path` 去重删磁盘文件（`os.IsNotExist` 忽略），再删库记录 —— 文件删不掉时保留记录可重试；④`covers/` 为空也不删目录，后续入库继续用。 |

| **新增 Steam 元数据源（appid + 商店 appdetails + 本机 librarycache 封面）** | 用户要求「steam源添加(meta)根据游戏id…扫描id 并通过 appdetails 获取json数据 优先复用已有逻辑」「查找封面图片文件夹目录在...\Steam\appcache\librarycache内…目录下含有游戏id名称的文件夹 进入查找相关图像 没有文件夹则不适用图像」。实现：`database.SourceSteam='steam'`（编号=appid）；扫描读 `appmanifest_<id>.acf`（StateFlags **按位**判 4，6 也算已安装）；详情走 `MetaApiService.FetchSteamWork`（`l=schinese&cc=CN`，归一化成与 DLsite 同一套作品 map，完整 appdetails 原样留在 `steam` 键）；`LibraryService.Fetch/Store` 按 source 分流，缓存优先/只读/只 Store 写库的语义不变；封面从 `appcache\librarycache\<appid>\` 递归（不限深度）找名字含 `600x900`/`capsule` 的图，600x900 优先、中文版其次、越浅越好，**找不到就不用封面**（不联网兜底）；`ImportCoverFile` 复制进 `covers/` 并记账。前端 Steam 行按 appid 直接取详情，`steamType !== 'game'` 的行跳过。**不复用 `steamFolders` 的"按文件夹名去 DLsite 搜"**：Steam 游戏的元数据固定来自 Steam（用户：「一个游戏数据固定一个来源」）。 |

| **游戏库封面统一 2:3** | 用户要求「游戏库封面展示图像格式也全部变为2:3等比图样 包括steam源和 DLsite源 和以后所有可能添加源」。`LibraryPage.vue` 封面容器 `aspect-[4/3]` → `aspect-[2/3]` + `object-cover`（DLsite 的 4:3 图居中裁掉两边，Steam 的 `library_600x900` 原生 2:3 不裁）；卡片 key / 封面缓存 key 都改成 `来源:编号`，为以后多来源做准备。**作品信息界面 `/works` 本次不动**（用户：「作品信息显示界面未来要作修改 暂时不进行改动」）。 |

| **Steam 接口调研只「保留入口」，不接入** |用户要求「在网上搜索相关api信息 把可能可以扩展当前软件的api保留」。做法：①`doc/steam-api.md` 逐条写清参数/返回/坑/能扩展什么，全部标 2026-11-22 实测（含反例：appreviews 302、GetAppList 404、appdetails 多 appid 400、steamspy 403）；②`internal/service/getMeta/steam_endpoints.go` 收一组免 key 常量，`steamAppDetailsFmt` 改由 `steamAppDetailsAPIURL` 派生（单一事实来源），除此之外**零改动**——不注册服务、不加绑定、不写库。理由：这些接口只是候选，先接入会带出用户没要的 UI/表结构改动；而把「哪个接口还能信」沉淀成文档，下次接的时候不用重新试错（本次最贵的一步是 PowerShell/curl 在 schannel 层不通，得用 Go 探）。 |

| **Steam 标签改走 GetItems；手动搜索 / 扫描 / 作品页补齐 Steam**（2026-11-22） | 用户要求「tag 的获取尝试使用别的接口（不是原来的 appdetails）」，并确认「改动三条 + tag 换接口」「手动搜索可以是 steam 也可以是 DLsite」。落点：①`steam_browse.go` 用 `IStoreBrowseService/GetItems`（`include_tag_count:20` + `include_reviews`）取**玩家标签**与评价汇总，`IStoreService/GetMostPopularTags` 做 tagid→中文名（进程内缓存 24h），标签按权重降序排在 appdetails 的分类标签前面、去重截断到 `steamTagLimit`；②`MetaApiService` 新增 `SearchSteamWorks` / `MatchSteamAppByFolder` / `FetchSteamStats`（bindings 已重生成，65 个方法）；③前端弹窗加「搜索来源」chip（DLsite / Steam）、`scanStore` 对「按文件夹名」的行先猜 Steam、`/works` 加 `MetaSteamStats`。**所有 Steam 附加数据都是"能拿到就更好"**：GetItems／标签字典／公告任一失败只退化成少一块信息，不影响 appdetails 主链与入库。匹配阈值 0.75 + 查询词逐级截短，是实测「storesearch 按词匹配」（`黑神话悟空` 0 条 / `黑神话` 9 条）逼出来的。 |

| **标签改成「合并 + 原地编辑 + 显式保存」**（2026-11-22，两轮需求定稿） | 第一轮用户要求「鼠标指针hover到标签右上角显示x号提示删除 用户点击从数据库中删除标签 添加标签显示改为在标签排列位置的最后排列css格式与原标签格式相同的占位+号 点击后用户可以输入文字添加标签(位置在+号标签中) 如果输入为空则无视本次修改 如果不为空则添加输入内容为标签到占位+号左侧 删去原本的添加标签前端内容」；第二轮澄清「我的标签删除 我的标签与抓取标签合并 … 删掉后不是隐藏 而是改写数据库标签 但不是每删一个就改写数据库 有改动后添加保存按钮 点击后进行标签的重新写入 且添加标签的ui点击后应保持原有大小 更具输入动态扩展标签大小」。落点：①**合并**：删除独立的「我的标签」区块，抓取来的 `tags` 与 `work_tags` 合成**一份**编辑稿 `metaStore.tagList`（载入时：`GetTags.edited=true` 直接用库里那份，否则抓取 tags 打底 + 旧自定义标签追加、去重）；概览卡的标签云在 `/works` 关掉（`MetaWorkOverview.showTags=false`，调试页仍显示），避免同页两处标签。②**UI**：标签是 `chip`，`li` 加 `position:relative`，× 是 14px 圆形绝对定位在 `top/right:-5px`，静息 `opacity:0` + `pointer-events:none`（不占点击区、防误删），`.tag-chip:hover / :focus-within / .tag-remove:focus-visible` 才浮出，`title`+`aria-label` 提示删除，hover 填 danger 红（沿用游戏库/TODO 删除的破坏性配色）；末尾固定一颗同款 `chip` 的「+」（`i-lucide-plus icon-sm`），点它把该 `li` 换成 `chip` 外壳的输入框（`!border-brand` 表聚焦），`display:inline-grid` + 隐藏镜像 span 叠同一网格单元 → **先保持 `+` 的原宽（`min-width:2.375rem`），再随输入内容变宽**；回车/失焦提交、Esc 取消，空输入直接收起不写库。③**批量保存**：增删只改编辑稿，`tagsDirty`（与 `tagSaved` 逐项比较，顺序也算）时标题右侧出现「保存」；点它调 `SetTags` **整份覆盖**（清空 → 按顺序插入 → 记 `work_tag_edits` 标记，同一事务），返回的清洗结果作为新基准。④**持久化语义**：`edited=true` 后重新抓取元数据**不会**冲掉用户整理的标签（`work.tags` 只用于未整理过的初始合并）；`work_tag_edits` 的存在是为了区分「从没整理过」与「整理成空列表」（后者也要显示为空）。测试：`TestWorkTagsSaveAndEditFlag`（覆盖/保序/空列表/来源隔离）、`TestNormalizeTagSet`（去重保序、超长超量报错）。 |

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
