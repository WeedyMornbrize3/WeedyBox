# Steam 接口清单（2026-11-22 实测）

本文件记录 **WeedyBox 可能用到、且这次逐条实测过的 Steam 接口**。结论全部来自 2026-11-22 在本机（Windows 11、Go `net/http`、UA 伪装成 Chrome、无登录态、无 Web API key）直连的实测，不是抄文档。

代码里的入口常量在 [`internal/service/getMeta/steam_endpoints.go`](../internal/service/getMeta/steam_endpoints.go)，命名与本文件的「接口」一列一一对应；已经接入的实现见 [`internal/service/getMeta/steam_browse.go`](../internal/service/getMeta/steam_browse.go)。

> 为什么用 Go 探而不是 curl：本机的 PowerShell / `curl.exe` 都在 schannel 层失败（`AcquireCredentialsHandle failed: SEC_E_NO_CREDENTIALS`），而 Go 自带 `crypto/tls`，走同一张网卡能通。以后复测请写个几十行的 Go 程序，别在 PowerShell 里跟 TLS 较劲。
>
> 另外：**改这份文档请用编辑工具，不要用 PowerShell 的 `Get-Content`/`Set-Content`** —— 2026-11-22 就因为这个把整份中文文档写成了非法 UTF-8（PS 5.1 默认按 ANSI 读写）。

## 0.1 接入现状（2026-11-22 当天已接）

| 接口 | 落在哪 |
| --- | --- |
| `GetItems` | `steam_browse.go` 的 `fetchSteamBrowse` → Steam 标签（`enrichSteamWork`）+ 评价汇总 + `FetchSteamStats` |
| `GetMostPopularTags` | `steamTagNames`（tagid → 中文名，进程内缓存 24h） |
| `storesearch` | `SearchSteamWorks`（弹窗「搜索来源 = Steam」）+ `MatchSteamApp`（扫描「按文件夹名」猜 appid） |
| `GetNewsForApp` / `GetNumberOfCurrentPlayers` | `FetchSteamStats` → `/works` 的「Steam 动态」 |
| `appdetails` | 老路径，不变（单 appid） |
| `GetStoreCategories` | **还没用**（categoryid 能翻中文名，比 `type` 细，需要时再接） |

接入原则：**附加数据只加分** —— GetItems / 标签字典 / 公告任一失败都只是少一块信息，绝不影响 appdetails 主链与入库。

---

## 0. 结论速览

| 接口 | 地址 | key | 实测 | 能扩展什么 |
| --- | --- | --- | --- | --- |
| appdetails | `store.steampowered.com/api/appdetails` | 免 | ✅ 200，**一次只能一个 appid** | 已在用：入库详情 |
| storesearch | `store.steampowered.com/api/storesearch/` | 免 | ✅ 200，**最多 10 条、按词匹配** | **已接**：弹窗「搜索来源 = Steam」；扫描文件夹名 → appid |
| GetItems | `api.steampowered.com/IStoreBrowseService/GetItems/v1/` | 免（文档写必填） | ✅ 200，一次 100 个 id | **已接**：玩家标签 + 评价汇总（600×900 封面 URL 还没用） |
| GetMostPopularTags | `api.steampowered.com/IStoreService/GetMostPopularTags/v1/` | 免 | ✅ 200，约 15 KB | **已接**：tagid → 中文名（进程内缓存 24h） |
| GetStoreCategories | `api.steampowered.com/IStoreBrowseService/GetStoreCategories/v1/` | 免 | ✅ 200 | categoryid → 中文名（判定试玩版/DLC/工具）；**暂未用** |
| GetNewsForApp | `api.steampowered.com/ISteamNews/GetNewsForApp/v2/` | 免 | ✅ 200 | **已接**：`/works` 的「Steam 动态」公告 |
| GetNumberOfCurrentPlayers | `api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/` | 免 | ✅ 200 | **已接**：`/works` 的「Steam 动态」在线人数 |
| search/suggest | `store.steampowered.com/search/suggest` | 免 | ✅ 200（HTML 片段） | 名字联想，`data-ds-appid` 直接带 appid（可选，未用） |
| featured / featuredcategories | `store.steampowered.com/api/featured` | 免 | ✅ 200 | 商店推荐位，与本软件无关，**不留** |
| **appreviews** | `store.steampowered.com/appreviews/<id>` | — | ❌ **302 → 商店首页，拿不到 JSON** | 不要用，评价改用 GetItems 的 `reviews` |
| GetDLCForApps | `api.steampowered.com/IStoreBrowseService/GetDLCForApps/v1/` | 需 | ❌ 401 | 需 key |
| IStoreService/GetAppList | `api.steampowered.com/IStoreService/GetAppList/v1/` | 需 | ❌ 403 | 需 key（全量 appid→名字） |
| GetOwnedGames | `api.steampowered.com/IPlayerService/GetOwnedGames/v1/` | 需 | ❌ 401 | 需 key + steamid + 隐私公开 |
| GetSchemaForGame | `api.steampowered.com/ISteamUserStats/GetSchemaForGame/v2/` | 需 | ❌ 400（缺 key） | 需 key |
| GetGlobalAchievementPercentagesForApp | `api.steampowered.com/ISteamUserStats/…/v2/` | 需 | ❌ 403 | 需 key |
| ISteamApps/GetAppList | `api.steampowered.com/ISteamApps/GetAppList/v2/` | — | ❌ **404 已下线** | 用 storesearch 代替 |
| SteamSpy | `steamspy.com/api.php` | — | ❌ 403（Cloudflare 挑战页） | 第三方，不引 |
| 商店页 | `store.steampowered.com/app/<id>/` | 免 | ✅ 200（HTML） | 「在浏览器打开商店页」 |

---

## 1. 已在用：appdetails

```
GET https://store.steampowered.com/api/appdetails?appids=<appid>&l=schinese&cc=CN
```

- 返回 `{"<appid>":{"success":true,"data":{…}}}`；失败时 `success:false`，`data` 缺失。
- `l=schinese&cc=CN` 决定语言与货币（价格是人民币分）。
- **一次只能查一个 appid**：`appids=1229490,1145360` → **HTTP 400 `null`**（实测）。要批量就循环，别指望逗号。
- `filters=` 参数**实测被忽略**（传 `filters=basic` 仍返回全量），不要依赖它裁字段。
- 字段见 `getFromSteam.go` 的 `steamAppDetails`；完整原始 JSON 已经存进 `raw_json` 的 `steam` 键，`/works` 改版时直接读，不必重抓。
- 官方没公布限速；社区经验值约 200 次/5 分钟（**未实测**）。接入新功能时按「≤ 1 次/秒 + 优先命中缓存」写。

## 2. 建议保留的接口

### 2.1 storesearch —— 按名字搜 Steam（**已接**）

```
GET https://store.steampowered.com/api/storesearch/?term=<关键词>&l=schinese&cc=CN
```

返回：

```json
{"total":10,"items":[{"type":"app","name":"Hades II","id":1145350,
  "price":{"currency":"CNY","initial":10800,"final":10800},
  "tiny_image":"https://…/capsule_sm_120.jpg",
  "metascore":"","platforms":{"windows":true,"mac":false,"linux":false},
  "streamingvideo":false}]}
```

- **最多 10 条**（`total` 也就是 10），没有分页参数；想要更多只能换关键词。
- `id` 就是 appid，`tiny_image` 可直接当候选缩略图。
- **按词匹配，不是随便怎么切都能中**：`黑神话：悟空` ✅、`黑神话` ✅（9 条）、`黑神话悟空` ❌ 0 条、`黑神话悟` ❌ 0 条；`艾尔登法环` ✅ 而 `艾尔登法环黄金树幽影` ❌ 0 条；`ULTRAKILL` ✅ 而 `ULTRAKILL v1.0` ❌ 0 条。所以拿文件夹名去搜时必须**逐级截短重试**（见 2.2 的匹配策略）。
- 国区不上架的游戏搜不到（`荒野大镖客2` 各种写法都是 0 条），这是地区限制，不是匹配失败。
- 已用于：① 游戏库搜索弹窗「搜索来源 = Steam」；② 扫描「按文件夹名」猜 appid。

### 2.2 GetItems —— 一次拿一批 appid 的元数据（**已接：标签 + 评价**）

```
GET https://api.steampowered.com/IStoreBrowseService/GetItems/v1/?input_json=<URL 编码后的 JSON>
```

`input_json`：

```json
{"ids":[{"appid":1229490}],
 "context":{"language":"schinese","country_code":"CN"},
 "data_request":{"include_tag_count":20,"include_reviews":true}}
```

- `ids[]` 里每项可以是 `appid` / `packageid` / `bundleid` / `tagid` / `creatorid` / `hubcategoryid`。
- `data_request` 可用开关（xPaw 文档 + 实测都在）：`include_assets` `include_assets_without_overrides` `include_release` `include_platforms` `include_all_purchase_options` `include_best_purchase_option` `include_screenshots` `include_trailers` `include_ratings` `include_tag_count`(int) `include_reviews` `include_basic_info` `include_supported_languages` `include_full_description` `include_included_items` `include_links` `include_extra_details` `apply_user_filters` `include_optin_registration_tags`。
- **实测免 key 可用**（文档把 `key` 标成 required，但匿名请求照样 200）——这是未公开行为，接入时必须有容错：真被收紧就退回原来的行为（WeedyBox 里就是"标签退回 appdetails 那套"）。
- **实测一次 100 个 id 全部返回**（`store_items` 100 条；测试里 100 个 id 是由 30 个 appid 循环拼的，所以只证明了**数量没被截断**）。上限没探到，建议按 ≤ 100 批。
- 返回（`response.store_items[]`）实测字段：

| 字段 | 说明 |
| --- | --- |
| `item_type` / `type` | 0 = app / 1 = package；`type` 0 是 game，1 是 demo/dlc 之类 |
| `name` `store_url_path` `appid` `visible` `is_early_access` | 基础 |
| `basic_info.short_description` `developers[]` `publishers[]` | 简介、开发商、发行商 |
| `tagids[]` + `tags[{tagid,weight}]` | **玩家标签**（带权重），中文名用 2.3 的字典换；WeedyBox 按权重降序排在最前 |
| `categories.supported_player_categoryids` `feature_categoryids` `controller_categoryids` | 单人/多人、功能分类 |
| `reviews.summary_filtered.review_count/percent_positive/review_score/review_score_label` | 评价汇总，`label` 随 `context.language` 本地化（实测 `好评如潮`）；另有 `summary_language_specific`（按语言） |
| `assets.asset_url_format` + `main_capsule` `small_capsule` `header` `hero_capsule` `library_capsule` `library_capsule_2x` `library_hero` | **竖版 600×900 就在 `library_capsule`**（WeedyBox 还没用，见 §2.2 末尾） |
| `release.steam_release_date`（Unix 秒） `is_early_access` | 发售时间 |
| `platforms.windows` `vr_support` `steam_deck_compat_category` … | 平台 / Deck 兼容 |
| `best_purchase_option` / `purchase_options[]` | 价格（`final_price_in_cents`、`formatted_final_price`、折扣率、bundle 列表） |
| `screenshots.all_ages_screenshots[]` / `mature_content_screenshots[]` | 截图文件名 |
| `full_description_bbcode` `supported_languages` `trailers` `links` `included_items` `extra_details` | 大字段，按需再开 |

- **封面 URL 拼法**（实测 HTTP 200）：

  ```
  https://shared.akamai.steamstatic.com/store_item_assets/ +
  strings.ReplaceAll(assets.asset_url_format, "${FILENAME}", assets.library_capsule)
  ```

  `asset_url_format` 形如 `steam/apps/1229490/${FILENAME}?t=1792046802`，拼出来就是 600×900 的 `library_600x900.jpg`。**注意**：WeedyBox 的封面仍按用户要求只取本机 `appcache\librarycache`，没有就不使用图像；联网兜底与那条决定冲突，没做。
- 已用于：`enrichSteamWork`（标签 + 评价汇总 + 原始数据 `steamBrowse`）、`FetchSteamStats`（评价）。
- **匹配策略（文件夹名 → appid）**：清洗（去【】/[]/() 注释、去结尾 `v1.0` 这类点分版本号、下划线换空格）→ 搜索词按 0.75 / 0.5 / 0.35 逐级截短、用第一个有结果的词 → 归一化（只留字母数字，含中文）后打分：完全一致 = 2，否则 = 公共前缀长度 / 文件夹名长度且要求前缀 ≥ 3 字、占比 ≥ 0.6，最终阈值 0.75；「原声 / 试玩 / 工具 / 升级包」等陪跑条目降权。实测：`黑神话悟空→2358720`、`艾尔登法环黄金树幽影→2778580`、`ULTRAKILL v1.0→1229490`、`Hades→1145360`、`Stardew_Valley→413150`。

### 2.3 GetMostPopularTags —— tagid → 中文名（**已接**）

```
GET https://api.steampowered.com/IStoreService/GetMostPopularTags/v1/?language=schinese
```

返回 `{"response":{"tags":[{"tagid":492,"name":"独立"},{"tagid":19,"name":"动作"},…]}}`，约 15 KB、免 key、一次拿全。GetItems 的 `tagids` / `tags[].tagid` 靠它翻译；`language` 换语言即得对应文字。WeedyBox 在进程内缓存 24 小时（`steamTagCache`）。

### 2.4 GetStoreCategories —— categoryid → 中文名（暂未用）

```
GET https://api.steampowered.com/IStoreBrowseService/GetStoreCategories/v1/?language=schinese
```

返回 `{"response":{"categories":[{"categoryid":10,"type":0,"internal_name":"Game Demo","display_name":"游戏试用版",…}]}}`。用来把 GetItems 的 `categories.*` 翻译成中文，或按 `internal_name` 识别 Demo/试玩、工具、DLC（比 appdetails 的 `type` 更细）。

### 2.5 GetNewsForApp —— 更新公告（**已接**）

```
GET https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/?appid=1229490&count=3&maxlength=200
```

返回 `{"appnews":{"appid":…,"newsitems":[{"gid","title","url","author","contents","feedlabel","date","feedname"}],"count":N}}`，`date` 是 Unix 秒，`contents` 受 `maxlength` 截断（BBcode 纯文本）。免 key。WeedyBox 取最近 3 条显示在 `/works` 的「Steam 动态」里。

### 2.6 GetNumberOfCurrentPlayers —— 在线人数（**已接**）

```
GET https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=1229490
```

`{"response":{"player_count":3199,"result":1}}`。免 key、极轻，`/works` 的「Steam 动态」里显示。`result != 1` 时视为失败。

---

## 3. 需要 Web API key 的（默认不接）

这些实测都直接拒绝：`GetDLCForApps` 401、`IStoreService/GetAppList` 403、`GetOwnedGames` 401、`GetSchemaForGame` 400「Required parameter 'key' is missing」、`GetGlobalAchievementPercentagesForApp` 403。

- key 来源：<https://steamcommunity.com/dev/apikey>（要登录 Steam 账号，且账号需绑定手机/消费过）。
- 值得考虑的唯一场景：**「导入我 Steam 账号里的游戏」**——`IPlayerService/GetOwnedGames`（`key` + `steamid`，还要求用户把「游戏详情」设成公开）。成本高（要用户填 key、要处理隐私设置）、收益是「不需要先把游戏装在盘上」。暂时不做，记在这里备查。
- `IStoreService/GetAppList` 有 key 才能拿全量 appid→名字；没有它时「名字 → appid」就用 storesearch（10 条够用）。

## 4. 实测不可用（别再花时间）

### 4.1 `store.steampowered.com/appreviews/<appid>?json=1` —— 302 到商店首页

2026-11-22 实测（这是本次最反直觉的一条）：

- `?json=1`、加 `filter=recent|all|summary`、加 `language=all`、加 `cursor=*`、加 `cc=CN/US/JP` + `l=schinese/english/japanese`、带 `Cookie: SteamCountry=…`、带 `X-Requested-With: XMLHttpRequest` + `Referer`、甚至改成 **POST 表单** —— **全部** `302 Location: https://store.steampowered.com/`，跟随之后拿到的是商店首页 HTML（`text/html`），不是 JSON。
- 结论：匿名调用已经拿不到评论列表。**要用评价，就取 GetItems 的 `reviews.summary_filtered`**（数量 + 好评率 + 本地化标签），足够 `/works` 显示「好评如潮 97%（218393）」这种一行。
- 想拿评论正文（单条内容）目前只能爬商店页/社区页 HTML，成本高、易碎，不建议。

### 4.2 `ISteamApps/GetAppList/v2` —— 已下线

`404`，正文写得很直白：`Method 'GetAppList' not found in interface 'ISteamApps'`。全量列表只剩需要 key 的 `IStoreService/GetAppList`。

### 4.3 SteamSpy / 其它第三方

`steamspy.com` 返回 Cloudflare 挑战页（403）。本项目一直不引第三方数据源，保持。

---

## 5. 和现有设计的结合点

1. ~~扫描「按文件夹名」时先查 Steam~~ **已做（2026-11-22）**：`scanStore.matchSteamEntry` → `MatchSteamAppByFolder`。
2. ~~游戏库搜索弹窗加 Steam 来源~~ **已做（2026-11-22）**：弹窗「搜索来源」chip。
3. ~~`/works` 加评价/在线/公告~~ **已做（2026-11-22）**：`MetaSteamStats`。
4. **中文用户标签**：`tagids` + `GetMostPopularTags` —— 已做，标签按权重排在原有分类标签之前。之后再想加「按标签筛选游戏库」可以直接用这批中文标签。
5. **批量刷新**：一次 GetItems 拿 100 个的 `basic_info`+`release`，比逐个 appdetails 快一个数量级；适合以后做「刷新游戏库元数据」。
6. **联网封面兜底**（`assets.library_capsule`，现成 600×900）：**与用户「librarycache 没有就不要封面」的决定冲突**，只在用户明确改口后再做。

---

## 6. 复测方法（下次照抄）

```go
// GOCACHE 指到工作区里，避免沙箱拒绝写 %LOCALAPPDATA%\go-build
// $env:GOCACHE='F:\Code\WeedyBox\.tmp\gocache'; $env:CGO_ENABLED='0'; go run .\main.go
req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/126.0")
```

`curl` 等价命令（在能通 TLS 的机器上）：

```bash
curl 'https://store.steampowered.com/api/appdetails?appids=1229490&l=schinese&cc=CN'
curl 'https://store.steampowered.com/api/storesearch/?term=Hades&l=schinese&cc=CN'
curl 'https://api.steampowered.com/IStoreBrowseService/GetItems/v1/?input_json=%7B%22ids%22%3A%5B%7B%22appid%22%3A1229490%7D%5D%2C%22context%22%3A%7B%22language%22%3A%22schinese%22%2C%22country_code%22%3A%22CN%22%7D%2C%22data_request%22%3A%7B%22include_tag_count%22%3A20%2C%22include_reviews%22%3Atrue%7D%7D'
curl 'https://api.steampowered.com/IStoreService/GetMostPopularTags/v1/?language=schinese'
curl 'https://api.steampowered.com/ISteamNews/GetNewsForApp/v2/?appid=1229490&count=2&maxlength=80'
curl 'https://api.steampowered.com/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=1229490'
```

xPaw 的接口字典（参数/字段的权威来源）：<https://steamapi.xpaw.me/IStoreBrowseService>
