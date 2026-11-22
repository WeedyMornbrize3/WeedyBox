// internal/service/getMeta/steam_endpoints.go
//
// Steam 公开接口清单（2026-11-22 逐条实测，结论与返回样例见 doc/steam-api.md）。
//
// 这里只「保留入口」：常量本身不参与任何流程，也没有注册服务、没有前端绑定
// （项目准则：新能力先不接，除非用户明确要求）。以后要扩展时直接从这组常量拼 URL，
// 别再手抄一遍地址，也别再把 appreviews 那种已失效的接口当可用接口。
package getmeta

const (
	// steamWebAPIBaseURL Steam Web API 的根（IStoreBrowseService / IStoreService 等）。
	steamWebAPIBaseURL = "https://api.steampowered.com"

	// ===== store.steampowered.com（商店站，无需 Web API key）=====

	// steamAppDetailsAPIURL 商店详情，已在用（getFromSteam.go 的 steamAppDetailsFmt 由它派生）。
	// ?appids=<appid>&l=schinese&cc=CN；**一次只能一个 appid**，逗号拼多个实测返回 400 null。
	steamAppDetailsAPIURL = steamStoreBaseURL + "/api/appdetails"

	// steamStoreSearchAPIURL 按名字模糊搜索：返回 appid + 名称 + 价格 + 缩略图，**最多 10 条**。
	// ?term=<关键词>&l=schinese&cc=CN
	// 用途：游戏库搜索弹窗加 Steam 来源；扫描到的文件夹名先查这里拿 appid。
	steamStoreSearchAPIURL = steamStoreBaseURL + "/api/storesearch/"

	// steamStoreSuggestAPIURL 搜索联想：返回 HTML 片段，每条带 data-ds-appid；要解析 HTML，仅作备选。
	// ?term=<关键词>&f=games&cc=CN&l=schinese&v=2
	steamStoreSuggestAPIURL = steamStoreBaseURL + "/search/suggest"

	// ===== api.steampowered.com（以下均实测免 key；文档标 required 的 key 实际可省）=====

	// steamBrowseItemsAPIURL 批量元数据（清单里最值得留的一个）。
	// ?input_json=<URL 编码的 JSON>
	//   {"ids":[{"appid":…}],"context":{"language":"schinese","country_code":"CN"},
	//    "data_request":{"include_basic_info":true,"include_assets":true,"include_release":true,
	//                    "include_platforms":true,"include_ratings":true,"include_tag_count":20,
	//                    "include_reviews":true}}
	// 实测一次 100 个 id 全部返回（数量没被截断）；可拿中文标签 tagid、评价汇总（数量/好评率/本地化标签）、
	// 600x900 竖版封面文件名（assets.library_capsule）、发售日、价格、截图。
	steamBrowseItemsAPIURL = steamWebAPIBaseURL + "/IStoreBrowseService/GetItems/v1/"

	// steamStoreCategoriesAPIURL categoryid → 中文名（试玩版/DLC/工具判定比 appdetails 的 type 更细）。
	// ?language=schinese
	steamStoreCategoriesAPIURL = steamWebAPIBaseURL + "/IStoreBrowseService/GetStoreCategories/v1/"

	// steamPopularTagsAPIURL tagid → 中文名，配合 GetItems 的 tags/tagids 出中文用户标签。
	// ?language=schinese（一次约 15 KB 拿全）
	steamPopularTagsAPIURL = steamWebAPIBaseURL + "/IStoreService/GetMostPopularTags/v1/"

	// steamNewsAPIURL 更新公告。?appid=<>&count=<>&maxlength=<>
	steamNewsAPIURL = steamWebAPIBaseURL + "/ISteamNews/GetNewsForApp/v2/"

	// steamCurrentPlayersAPIURL 当前在线人数。?appid=<>
	steamCurrentPlayersAPIURL = steamWebAPIBaseURL + "/ISteamUserStats/GetNumberOfCurrentPlayers/v1/"

	// 已实测不可用，别再写回来（细节见 doc/steam-api.md §4）：
	//   - store.steampowered.com/appreviews/<appid>?json=1 → 302 到商店首页，匿名拿不到 JSON；
	//     要评价请用 GetItems 的 reviews.summary_filtered。
	//   - api.steampowered.com/ISteamApps/GetAppList/v2 → 404（方法已下线）。
	//   - IStoreService/GetAppList、IPlayerService/GetOwnedGames、GetDLCForApps、
	//     ISteamUserStats/GetSchemaForGame 等一律要 Web API key（401/403）。
)
