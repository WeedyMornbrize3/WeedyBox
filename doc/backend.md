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

## 获取元数据 `internal/service/getMeta/getFromDlsite.go`（DLsite，package `getmeta`）

抓取 DLsite 作品页（音声 / 游戏）。**不依赖任何第三方 DLsite 库**，只用 `net/http` + `goquery`；`go.felesatra.moe/dlsite` 已从 `go.mod` 移除。
⚠️ **尚未在 `main.go` 注册为服务**，所以前端绑定里还没有它。

| 函数 | 行 | 说明 |
| --- | --- | --- |
| `NewMetaService(options ...MetaOption)` | 126 | 默认 `http.Client{Timeout: 20s}`、`tagLimit=24` |
| `WithHTTPClient(*http.Client)` / `WithTagLimit(int)` | 116 / 121 | 可注入客户端、限制标签数 |
| `(*MetaService) GetWorkByID(ctx, id)` | 143 | 按 RJ/RE/VJ 取单个作品 → `*DLsiteWork` |
| `(*MetaService) SearchWorkByName(ctx, keyword)` | 160 | 搜索，返回 `[]DLsiteSearchItem` |
| `(*MetaService) GetWorkByName(ctx, keyword)` | 185 | 搜索取首个再抓详情 |
| `NormalizeDLsiteID(raw)` | 430 | 抽 RJ/RE/VJ 编号，转大写、**保留前导零** |

解析内部：`parseWork` :250、`parseOutline` :291、`extractSummary` :326、`extractCoverURL` :336、`extractTags` :365、`parseSearchItem` :397、`fetchDocument` :216（发 `Cookie: adultchecked=1; locale=ja`）。

### 能拿到的数据（`DLsiteWork`，json tag 全 camelCase）

`ID` `Name` `AlternateID`(拼音 slug) `Maker` `MakerID` `Series` `SeriesID` `WorkType` `AgeRating` `FileFormats[]` `FileSize` `ReleaseDate` `Price` `Currency` `PriceUntil` `Rating` `RatingCount` `CoverURL` `Summary` `Tags[]` `Outline[]` `SourceURL` `FetchedAt`

实测（`:64`）：

| 字段 | RJ01721474 | RJ01682332 |
| --- | --- | --- |
| 作品名 | 【简体中文版】ELF PARADISE～… | 【簡体中文版】乙女の花園…【CV.涼本あきほ】 |
| 社团 / ID | 青春×フェティシズム / `RG51931` | RaRo / `RG49556` |
| 系列 / 系列ID | `ELF PARADISE` / `SRI0000043774` | 空 / 空 |
| 作品形式 / 年龄 | ボイス・ASMR / `R18` | ボイス・ASMR / `全年齢` |
| 文件格式 / 容量 | `[WAV mp3 mp4同梱]` / `16.83GB` | `[WAV MP3同梱]` / 空 |
| 发售日 | `2026-09-26` | `2026-10-07` |
| 价格 / 币种 / 有效期 | `1540` / `JPY` / `2026-10-24T00:00:00+09:00` | `1650` / `JPY` / 空 |
| 评分 / 评价数 | `5` / `16` | `4.81` / `43` |
| 简介长度 | 4799 字符 | 2417 字符 |
| 标签 | 10 个 | 10 个 |

`Outline[]` 额外原样带出：`作者` `シナリオ` `イラスト` `声優` `音楽` `その他` `対応言語` `ジャンル`（页面加字段不用改后端）。

`SearchWorkByName` 每项只有 `ID` / `Name` / `Cover`。

### 拿不到的

曲目表与试听音频（`ol.work_tracklist_list` 在当前页面**已不存在**，Vue 渲染）、销量/愿望单/评论数/折扣率、社团与系列的详情页、多语言版本对应关系、评论正文。

### 踩坑（改之前必读）

1. **必须 HTTPS**；`http://` 会被重定向，部分网络下直接超时。
2. **前导零不能丢**：`RJ01721474` 丢了 0 → 404「該当作品がありません」。
3. **必须发 `Cookie: adultchecked=1; locale=ja`**，否则 R18 页内容退化。
4. **标题只取 `#work_name` 的文本**，不要往里找 `<a>`（现代码没有 `<a>`，第三方库 `Name` 恒空就是这个原因）。
5. **价格 / 评分在 `<meta itemprop="…" content="…">` 的 content 属性上**，取 `.Text()` 得到空串 —— 用 `metaContent` :445。
6. `SeriesID` 只在内嵌 JSON 里，`extractSeriesID` :452 要判 `FindStringSubmatch` 长度，否则单作会 panic。
7. **`CoverURL` 目前取到的是最小缩略图**（`_img_main_240x240`）：`extractCoverURL` 先命中 `#work_thumb img`。更好的来源是 `<meta property="og:image">` / `itemprop="image"`（`_img_main.jpg` 560×420）与 slider 的 `data-src`（`_img_smp1.jpg` 1470×1000）。**尚未修改**。

### 图片可以直接爬（实测）

`img.dlsite.jp` DNS 干净（Cloudflare `104.18.24.157` / `104.18.25.157`），**不需要 Cookie、不需要 Referer**，`curl --noproxy '*'` 直连即可（PowerShell `Invoke-WebRequest -OutFile` 会因 .NET 默认代理报 `NonInteractive mode`，别用）。同一封面的 4 个档位：

| 档位 | 尺寸 | 体积 |
| --- | --- | --- |
| `modpub/…/RJ01721474_img_smp1.jpg` | 1470×1000 | 808 KB |
| `modpub/…/RJ01721474_img_main.jpg` | 560×420 | 222 KB（`og:image` 就是它） |
| `modpub/…/RJ01721474_img_main.webp` | 同尺寸 | 38.9 KB |
| `resize/…/RJ01721474_img_main_240x240.jpg` | 240×240 | 45.9 KB |

### 测试 `getFromDlsite_test.go`

离线 fixture 回归：`testdata/work_RJ01721474.html`(154454 B) / `work_RJ01682332.html`(125550 B) 是真实页面，断言具体值（不断言"非空"）。两个样本互补：一个有系列名/文件容量/价格有效期，一个都没有。另有 `TestParseWorkRejectsEmptyPage`（页面结构变化必须报错而非返回空壳）与纯函数表驱动测试。

## 前端可调用的绑定

`frontend/bindings/WeedyBox/internal/service/`（**自动生成，勿手改**）
- `TodoService.*`：`Create, GetByID, GetAll, GetByStatus, Update, ToggleComplete, Delete, DeleteAll, GetStats`
- `WindowService.*`：`MinimizeWindow, ToggleMaximizeWindow, CloseWindow, IsWindowMaximised`

导出名规则：Go 方法名首字母大写即前端函数名。
