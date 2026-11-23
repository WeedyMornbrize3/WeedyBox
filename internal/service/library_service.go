// internal/service/library_service.go
//
// 「游戏库」的编排层：把**本地 SQLite 缓存**和**DLsite 联网抓取**接在一起。
//
// 分工：
//   - 抓取/解析      internal/service/getMeta（不知道数据库的存在）
//   - 持久化         internal/database/library_repo.go（只搬数据）
//   - 缓存与入库策略 ← 本文件：Fetch 只读（库里有的直接给，没有才联网，**不写库**）；
//     Store 才是"入库"（抓文本 + 下封面 + 写库），只有用户点「入库」时调用。
//
// 为什么单独一个服务而不是塞进 MetaApiService：
// 调试页要的恰恰是「每次真去 DLsite 抓一遍」，拿它来核对解析结果；
// 作品库要的则是「能缓存就缓存」。两种语义相反，混在一个方法里只能靠布尔开关，
// 反而谁都读不懂。所以 metaApi 保持"永远联网"，libraryApi 才是"缓存优先"。
package service

import (
	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
	getmeta "WeedyBox/internal/service/getMeta"
	"WeedyBox/internal/service/scan"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// LibraryService 是暴露给前端的游戏库服务。
type LibraryService struct {
	DB   *database.SQLiteService
	Meta *getmeta.MetaApiService
}

// FetchWorkRequest 是前端取一个作品的入参。
//
// WorkID 与 Keyword 二选一（两个都给时 WorkID 优先）；Refresh=true 表示
// 「我要最新的，别给我缓存」。可选参数在 Wails 绑定里不好表达，所以一律用结构体。
type FetchWorkRequest struct {
	Keyword string `json:"keyword"`
	WorkID  string `json:"workId"`
	Source  string `json:"source"` // 元数据来源，空 = dlsite
	Refresh bool   `json:"refresh"`
}

// ensureMeta 兜底：main.go 里如果不小心漏了字段，至少还能抓取（只是不缓存）。
func (s *LibraryService) ensureMeta() *getmeta.MetaApiService {
	if s.Meta == nil {
		s.Meta = getmeta.NewMetaApiService()
	}
	return s.Meta
}

// ===== 对外方法 =====

// Fetch 取一个作品：**缓存优先**（只读）。
//
//  1. 能确定编号时先查 library_works（按 source + work_id），命中就把 raw_json
//     还原成 map 返回（不打网络）；
//  2. 没命中（或 Refresh=true）才联网抓取；
//  3. **不写库**：搜索/浏览不该悄悄把作品塞进游戏库，入库是用户点「入库」的显式动作
//     （见 Store）；
//  4. 返回值里额外注入 cached / cover* 几个键，前端据此显示「缓存」标记与本地封面。
//
// 返回的 map 与 metaApi.FetchWork 的字段完全一致（就是同一份 DLsiteWork 的 json），
// 只是多了上面那几个键 —— 这样两个服务的返回类型对前端是兼容的。
func (s *LibraryService) Fetch(ctx context.Context, req FetchWorkRequest) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	keyword := strings.TrimSpace(req.Keyword)
	workID := strings.TrimSpace(req.WorkID)
	source := normalizeWorkSource(req.Source)

	// Steam 源：编号就是 appid，走商店接口（没有"按名字搜"这一步，搜索仍只属于 DLsite）
	if source == database.SourceSteam {
		return s.fetchSteamWork(ctx, workID, req.Refresh)
	}

	if workID == "" {
		if id, ok := getmeta.NormalizeDLsiteID(keyword); ok {
			workID = id
		}
	}

	// ① 缓存
	if workID != "" && !req.Refresh {
		row, err := s.DB.FindLibraryWork(ctx, source, workID)
		// 查缓存失败不该让整个页面不可用：退化成"没缓存"，继续联网抓。
		if err == nil && row != nil {
			// 只复用**完整**的 raw_json：旧版本写下的空壳（只有名字+封面）这里当作没缓存，
			// 联网重抓一份（只读、不写库）—— 否则详情页永远显示不出社团/形式/发售日。
			if work, ok := workFromRawJSON(row); ok && workIsComplete(work) {
				return decorate(work, row, true), nil
			}
			// raw_json 坏了或残缺 → 当作没缓存，重新抓一份
		}
	}

	// ② 联网（**只返回，不写库**：写库是「入库」按钮的事）
	data, err := s.ensureMeta().FetchWork(ctx, getmeta.WorkRequest{Keyword: keyword, WorkID: workID})
	if err != nil {
		return nil, err
	}
	work, ok := data.(map[string]any)
	if !ok {
		return nil, errors.New("后端返回了无法识别的作品数据")
	}

	// 强制刷新（refresh=true）也要把库里的记录挂回来：封面指针与 game_path 决定
	// 「入库 / 游戏启动」按钮和封面显示，不该因为"这次走了网络"就丢。
	row, _ := s.DB.FindLibraryWork(ctx, source, workID)
	return decorate(work, row, false), nil
}

// fetchSteamWork 取一个 Steam 游戏：**缓存优先**（库里已有完整 raw_json 就直接读）。
//
// 与 DLsite 分支共用同一套「缓存优先、只读不写」语义，只是数据源换成
// MetaApiService.FetchSteamWork（appdetails）。编号必须是纯数字 appid。
func (s *LibraryService) fetchSteamWork(ctx context.Context, appID string, refresh bool) (map[string]any, error) {
	appID = getmeta.NormalizeSteamAppID(appID)
	if appID == "" {
		return nil, errors.New("Steam 来源需要纯数字的 appid（如 1229490）")
	}

	if !refresh {
		if row, err := s.DB.FindLibraryWork(ctx, database.SourceSteam, appID); err == nil && row != nil {
			if work, ok := workFromRawJSON(row); ok && workIsComplete(work) {
				return decorate(work, row, true), nil
			}
		}
	}

	work, err := s.ensureMeta().FetchSteamWork(ctx, appID)
	if err != nil {
		return nil, err
	}
	// 与 DLsite 分支同理：刷新也挂上库里的封面/位置指针
	row, _ := s.DB.FindLibraryWork(ctx, database.SourceSteam, appID)
	return decorate(work, row, false), nil
}

// StoreRequest 是「入库」的入参：来源 + 编号，外加**游戏启动位置的来源**。
//
// 三种情况（前端按扫描方式带进来，用户 2026-11-16 要求「入库时就把 exe 记下来」）：
//   - GamePath：按 exe 名扫到 / 手动搜索时用户选了程序 → 直接记录这个 exe；
//   - GameFolder + GameFolderName：按文件夹名扫到（Steam 库或本地文件夹）→
//     入库时到文件夹里挑主程序（scan.PickGameExecutable：排崩溃处理器、优先 Game.exe、
//     再选与文件夹名匹配字符最多的 exe）；
//   - 都为空：手动搜索且用户取消选择 → 位置留空，第一次启动时再让用户选。
type StoreRequest struct {
	WorkID         string `json:"workId"`
	Source         string `json:"source"`         // 空 = dlsite
	GamePath       string `json:"gamePath"`       // exe 绝对路径（已知时）
	GameFolder     string `json:"gameFolder"`     // 游戏文件夹绝对路径（按文件夹名匹配时）
	GameFolderName string `json:"gameFolderName"` // 文件夹名，用来在多个 exe 里挑主程序
}

// Store 把作品**真正收进游戏库**（只有用户点「入库」时才走这里）：
//
//   - 文本：联网抓一份元数据（库里已有**完整** raw_json 时直接复用，不重复下载），写进 library_works；
//   - 图片：DownloadCover 优先下载 webp 并落到 covers/，并把落地位置记进库。
//
// **入库前校验**（扫描添加会一次入库几十部，必须挡住空壳）：抓回来的数据没有编号或名称
// 就报错、不写库；库里已有的 raw_json 如果残缺（旧数据/解析退化）也会重新抓一份。
// 这样"扫描入库"和手动入库写进库里的作品信息是同一份完整元数据。
//
// 与 Fetch 的分工：Fetch 只读不写，Store 才写 —— 搜索、点开详情都不会再自动入库。
// 幂等：已入库且封面也在了，直接返回库里那份（不会重复下图；writeFileUnique 不覆盖，
// 重复调用只会多出 -2 后缀的重复文件）。
func (s *LibraryService) Store(ctx context.Context, req StoreRequest) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second) // 要抓文本 + 下图，给足时间
	defer cancel()

	if s.DB == nil {
		return nil, errors.New("数据库未就绪")
	}
	workID := strings.ToUpper(strings.TrimSpace(req.WorkID))
	if workID == "" {
		return nil, errors.New("作品编号为空，无法入库")
	}
	source := normalizeWorkSource(req.Source)
	// 入库时就确定要记录的游戏程序位置（直接给 exe / 从文件夹里挑 / 留空）
	gamePath := s.resolveStoreGamePath(ctx, req)

	// Steam 源：元数据来自商店接口，封面来自本机 appcache\librarycache，走单独一条路径
	if source == database.SourceSteam {
		return s.storeSteamWork(ctx, workID, gamePath)
	}

	// ① 库里已有**完整**文本：不再抓；封面也在了就直接返回。
	//   raw_json 残缺（没有编号/名称）时走联网重抓，避免把空壳一直复用下去。
	if row, err := s.DB.FindLibraryWork(ctx, source, workID); err == nil && row != nil {
		if work, ok := workFromRawJSON(row); ok && workIsComplete(work) {
			// 重新入库时带上扫描到的 exe 路径：顺手把启动位置更新掉
			if perr := s.saveGamePath(ctx, source, workID, gamePath); perr != nil {
				return nil, perr
			}
			if strings.TrimSpace(row.CoverAbsPath) == "" {
				if coverURL := strOf(work["coverUrl"]); coverURL != "" {
					if _, derr := s.ensureMeta().DownloadCover(ctx, workID, coverURL); derr != nil {
						return nil, fmt.Errorf("封面下载失败: %w", derr)
					}
				}
			}
			saved, ferr := s.DB.FindLibraryWork(ctx, source, workID)
			if ferr == nil && saved != nil {
				return decorate(work, saved, true), nil
			}
			return decorate(work, row, true), nil
		}
	}

	// ② 全新作品：联网抓文本
	data, err := s.ensureMeta().FetchWork(ctx, getmeta.WorkRequest{WorkID: workID})
	if err != nil {
		return nil, err
	}
	work, ok := data.(map[string]any)
	if !ok {
		return nil, errors.New("后端返回了无法识别的作品数据")
	}
	// 少了编号/名称就是空壳（解析退化/页面改版）：宁可不入库，也不能污染游戏库
	if !workIsComplete(work) {
		return nil, fmt.Errorf("作品 %s 的元数据不完整（缺少编号或名称），已取消入库", workID)
	}

	row := rowFromWork(source, work)
	// 入库失败必须报给用户：这一步就是在"载入数据库"，不能像浏览那样只打日志
	if err := s.DB.UpsertLibraryWork(ctx, &row); err != nil {
		return nil, fmt.Errorf("写入游戏库失败: %w", err)
	}

	// 扫描添加时顺手记录游戏启动位置（放在封面下载之前：封面失败不该连位置一起丢）
	if err := s.saveGamePath(ctx, source, workID, gamePath); err != nil {
		return nil, err
	}

	// ③ 封面：优先下载同名 webp 并落到 covers/ 记账（文本已入库，这里失败时前端会提示重试）
	if coverURL := strOf(work["coverUrl"]); coverURL != "" {
		if _, derr := s.ensureMeta().DownloadCover(ctx, workID, coverURL); derr != nil {
			return nil, fmt.Errorf("元数据已入库，但封面下载失败: %w", derr)
		}
	}

	if saved, ferr := s.DB.FindLibraryWork(ctx, source, workID); ferr == nil && saved != nil {
		return decorate(work, saved, false), nil
	}
	return decorate(work, nil, false), nil
}

// storeSteamWork 把一个 Steam 游戏收进游戏库（source='steam'，编号=appid）。
//
// 与 DLsite 的 Store 同一套策略：
//   - 库里已有完整 raw_json 就不再抓（只补位置/封面）；
//   - 否则联网抓一份 → 校验完整性 → 写库；
//   - 位置在入库时就记录（扫描添加带进来的安装目录 → PickGameExecutable）。
//
// 封面**不联网**：Steam 的 2:3 竖版封面在本机 `appcache\librarycache\<appid>\`，
// 找到就复制进 covers/ 并记账；**找不到就不使用图像**（用户明确要求），不算错误。
func (s *LibraryService) storeSteamWork(ctx context.Context, appID, gamePath string) (map[string]any, error) {
	appID = getmeta.NormalizeSteamAppID(appID)
	if appID == "" {
		return nil, errors.New("Steam 游戏编号无效（应为纯数字 appid）")
	}

	// ① 库里已有完整元数据：不再抓；顺手更新位置、补封面
	if row, err := s.DB.FindLibraryWork(ctx, database.SourceSteam, appID); err == nil && row != nil {
		if work, ok := workFromRawJSON(row); ok && workIsComplete(work) {
			if perr := s.saveGamePath(ctx, database.SourceSteam, appID, gamePath); perr != nil {
				return nil, perr
			}
			if strings.TrimSpace(row.CoverAbsPath) == "" {
				if _, ierr := s.importSteamCover(ctx, appID); ierr != nil {
					return nil, ierr
				}
			}
			if saved, ferr := s.DB.FindLibraryWork(ctx, database.SourceSteam, appID); ferr == nil && saved != nil {
				return decorate(work, saved, true), nil
			}
			return decorate(work, row, true), nil
		}
	}

	// ② 抓商店详情（appdetails）
	work, err := s.ensureMeta().FetchSteamWork(ctx, appID)
	if err != nil {
		return nil, err
	}
	if !workIsComplete(work) {
		return nil, fmt.Errorf("Steam 游戏 %s 的元数据不完整（缺少名称或详情字段），已取消入库", appID)
	}

	row := rowFromWork(database.SourceSteam, work)
	row.WorkID = appID
	if err := s.DB.UpsertLibraryWork(ctx, &row); err != nil {
		return nil, fmt.Errorf("写入游戏库失败: %w", err)
	}

	// 扫描添加时顺手记录游戏启动位置（放在封面之前：封面失败不该连位置一起丢）
	if err := s.saveGamePath(ctx, database.SourceSteam, appID, gamePath); err != nil {
		return nil, err
	}

	// ③ 本机封面：librarycache 里有就导入，没有就留空
	if _, ierr := s.importSteamCover(ctx, appID); ierr != nil {
		return nil, ierr
	}

	if saved, ferr := s.DB.FindLibraryWork(ctx, database.SourceSteam, appID); ferr == nil && saved != nil {
		return decorate(work, saved, false), nil
	}
	return decorate(work, nil, false), nil
}

// importSteamCover 把 Steam 缓存的封面复制进 covers/ 并记账，返回源文件路径。
// 找不到封面返回空串 + nil（用户要求：没有对应文件夹就不使用图像）。
func (s *LibraryService) importSteamCover(ctx context.Context, appID string) (string, error) {
	image, err := scan.SteamCoverImage(ctx, appID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(image) == "" {
		return "", nil
	}
	if _, ierr := s.ensureMeta().ImportCoverFile(ctx, database.SourceSteam, appID, image); ierr != nil {
		return "", fmt.Errorf("元数据已入库，但导入 Steam 封面失败: %w", ierr)
	}
	return image, nil
}

// normalizeWorkSource 统一来源标识：空值一律当 dlsite（与 database 的口径一致）。
func normalizeWorkSource(source string) string {
	s := strings.ToLower(strings.TrimSpace(source))
	if s == "" {
		return database.SourceDLsite
	}
	return s
}

// List 列出游戏库里所有作品（最近入库的在前）。source 传空串 = 所有来源。
func (s *LibraryService) List(ctx context.Context, source string) ([]model.LibraryWork, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if s.DB == nil {
		return []model.LibraryWork{}, nil
	}
	return s.DB.ListLibraryWorks(ctx, source)
}

// Remove 从游戏库删掉一条作品，并删除它入库时保存在 covers/ 的封面图片。
//
// 删文件放在删库记录之前：文件删不掉（被占用/权限）时直接报错、库记录保留，
// 用户还能重试；反过来先删记录会让图片变成谁也找不到的孤儿文件。
// 一次入库可能下过多次封面（重名 -2/-3），所以按 library_cover_files 的记录逐个删，
// 不只删作品表里的最新指针。
func (s *LibraryService) Remove(ctx context.Context, workID, source string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if s.DB == nil {
		return errors.New("数据库未就绪")
	}
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return errors.New("作品编号为空")
	}
	if strings.TrimSpace(source) == "" {
		source = database.SourceDLsite
	}

	covers, err := s.DB.ListCoverFiles(ctx, source, workID)
	if err != nil {
		return fmt.Errorf("读取封面记录失败: %w", err)
	}
	row, ferr := s.DB.FindLibraryWork(ctx, source, workID)
	if ferr != nil {
		return ferr
	}

	// 作品表里的 cover_abs_path 可能指向一条没有下载记录的旧文件，也一并删；
	// 路径去重（Windows 大小写不敏感）。
	paths := make([]string, 0, len(covers)+1)
	seen := make(map[string]struct{}, len(covers)+1)
	addPath := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		key := strings.ToLower(path)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		paths = append(paths, path)
	}
	for _, cover := range covers {
		addPath(cover.AbsPath)
	}
	if row != nil {
		addPath(row.CoverAbsPath)
	}

	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("删除封面图片 %s 失败: %w", path, err)
		}
	}
	return s.DB.DeleteLibraryWork(ctx, source, workID)
}

// ===== 作品标签 =====
//
// 作品详情页只有**一份**标签列表：抓取来的 tags 与用户自己加的标签在这里被当成
// 同一份东西让用户整理（删掉不想要的、补上自己的），点「保存」后整份写进 work_tags，
// 并标记为"已改写"—— 之后重新抓取元数据也不会把它冲掉（与"自己加的标签重抓不丢"
// 是同一条原则）。归属是「来源 + 编号」，所以对还没入库的作品也能整理。

// maxUserTagLength 是单个标签的长度上限（按字符数，不是字节）。
const maxUserTagLength = 32

// maxUserTagCount 是一次保存的标签条数上限（抓取来的标签本身 ≤64，留出追加空间）。
const maxUserTagCount = 128

// normalizeUserTag 标签清洗：去掉首尾空白，中间连续空白压成一个空格。
func normalizeUserTag(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}

// normalizeTagSet 清洗用户提交的整份标签列表：逐条 normalize、丢空值、按首次出现去重。
// 太长/太多直接报错而不是静默截断 —— 用户能看见是哪一条出了问题，改完再保存。
func normalizeTagSet(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		tag := normalizeUserTag(item)
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > maxUserTagLength {
			return nil, fmt.Errorf("标签「%s」超过 %d 个字符", tag, maxUserTagLength)
		}
		if _, dup := seen[tag]; dup {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	if len(out) > maxUserTagCount {
		return nil, fmt.Errorf("标签最多 %d 个", maxUserTagCount)
	}
	return out, nil
}

// GetTags 读一个作品的标签集合。
//
//   - Edited=true：用户保存过整份列表，Tags 就是他定的那一份（work_tags 里的全部）；
//   - Edited=false：还没整理过，Tags 里只有旧版本逐条加过的标签（可能为空），
//     前端要再与抓取来的 tags 合并成初始列表。
func (s *LibraryService) GetTags(ctx context.Context, workID, source string) (model.TagSet, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if s.DB == nil {
		return model.TagSet{Tags: []string{}}, nil
	}
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return model.TagSet{}, errors.New("作品编号为空")
	}
	edited, err := s.DB.WorkTagsEdited(ctx, source, workID)
	if err != nil {
		return model.TagSet{}, err
	}
	tags, err := s.DB.ListWorkTags(ctx, source, workID)
	if err != nil {
		return model.TagSet{}, err
	}
	if tags == nil {
		tags = []string{}
	}
	return model.TagSet{Tags: tags, Edited: edited}, nil
}

// SetTags 用整份列表覆盖保存这个作品的标签（用户点「保存」时才调用）。
// 返回保存后的集合（已清洗、已去重），前端据此更新"已保存"基准。
func (s *LibraryService) SetTags(ctx context.Context, workID, source string, tags []string) (model.TagSet, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if s.DB == nil {
		return model.TagSet{}, errors.New("数据库未就绪")
	}
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return model.TagSet{}, errors.New("作品编号为空")
	}
	cleaned, err := normalizeTagSet(tags)
	if err != nil {
		return model.TagSet{}, err
	}
	if err := s.DB.SaveWorkTags(ctx, source, workID, cleaned); err != nil {
		return model.TagSet{}, err
	}
	return model.TagSet{Tags: cleaned, Edited: true}, nil
}

// CoverDataURL 把游戏库里记录的本地封面读成 data URL，给 <img> 直接显示。//
// 为什么不用 file://：Wails 的 webview 对本地文件协议限制不确定（调试页的
// file:// 链接至今未实机验证过），改走 base64 是确定能显示的方案；封面本来就只有几百 KB。
// 文件不存在（被手动删了）时返回空串 —— 那是"没有缓存封面"，不该当错误抛给用户。
func (s *LibraryService) CoverDataURL(ctx context.Context, workID, source string) (string, error) {
	if s.DB == nil {
		return "", nil
	}
	if strings.TrimSpace(source) == "" {
		source = database.SourceDLsite
	}
	row, err := s.DB.FindLibraryWork(ctx, source, workID)
	if err != nil || row == nil || strings.TrimSpace(row.CoverAbsPath) == "" {
		return "", nil
	}

	data, err := os.ReadFile(row.CoverAbsPath)
	if err != nil || len(data) == 0 {
		return "", nil
	}

	mime := mimeBySuffix(row.CoverSuffix)
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// SteamHeroImage 读取本机 Steam 客户端缓存的 hero 背景图，转成 data URL 给详情页当大背景。
//
// 用户要求（2026-11-23）：「背景图如果是 steam 源使用本地扫描函数扫描含 hero 字样图片，
// 如果没有就使用 api 获取的预览图填充」。所以这里**找不到就返回空串 + nil**（不是错误），
// 由前端决定回落到商店接口的预览图；扫描规则见 scan.SteamHeroImage。
// 为什么转 data URL 而不是给文件路径：与封面同一原因（Wails webview 对 file:// 没有
// 实机验证过）；hero 图实测 200~450KB，base64 后仍是一次可接受的传输。
func (s *LibraryService) SteamHeroImage(ctx context.Context, appID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	image, err := scan.SteamHeroImage(ctx, getmeta.NormalizeSteamAppID(appID))
	if err != nil || strings.TrimSpace(image) == "" {
		return "", err
	}
	data, err := os.ReadFile(image)
	if err != nil || len(data) == 0 {
		return "", nil
	}
	mime := mimeBySuffix(filepath.Ext(image))
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// ===== 内部实现 =====

// resolveStoreGamePath 决定这次入库要记录的游戏程序位置（用户 2026-11-16 要求「入库时便存储相关 exe」）：
//
//  1. 直接给了 exe（按 exe 名扫到 / 手动搜索时用户选了程序）→ 用它；
//  2. 只给了游戏文件夹（按文件夹名扫到，Steam 库或本地文件夹）→ 进文件夹挑主程序：
//     先排除 UnityCrashHandler64.exe，有 Game.exe 优先，再选与文件夹名匹配字符最多的 exe；
//  3. 什么都没有（手动搜索且用户取消选择）→ 空串，第一次启动时再让用户选。
//
// 挑不到 exe 不阻断入库：位置留空即可，别因为一个探测失败把整部作品卡在库外。
func (s *LibraryService) resolveStoreGamePath(ctx context.Context, req StoreRequest) string {
	if path := strings.TrimSpace(req.GamePath); path != "" {
		return path
	}
	folder := strings.TrimSpace(req.GameFolder)
	if folder == "" {
		return ""
	}
	picked, err := scan.PickGameExecutable(ctx, folder, req.GameFolderName)
	if err != nil {
		return ""
	}
	return picked
}

// saveGamePath 记录「这个作品在本机上的游戏程序位置」（扫描添加带进来的 exe 路径）。
//
// 空路径 = 这一次没有带位置（手动搜索入库），**不覆盖**库里可能已经设置好的位置 ——
// 用户之前手动选过的 exe 不该因为再入一次库就丢掉。
// 路径统一成绝对路径再落库：启动进程的工作目录是可执行文件所在目录，
// 库里存相对路径换一个上下文就失效。
func (s *LibraryService) saveGamePath(ctx context.Context, source, workID, raw string) error {
	path := strings.TrimSpace(raw)
	if path == "" {
		return nil
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = filepath.Clean(abs)
	}
	if err := s.DB.SetGamePath(ctx, source, workID, path); err != nil {
		return fmt.Errorf("元数据已入库，但记录游戏位置失败: %w", err)
	}
	return nil
}

// decorate 把缓存信息挂到作品 map 上。
// 前端的 WorkMeta 有 `[key: string]: unknown` 索引签名，多出来的键不会报错，
// 缺字段也不会 —— 所以走"加法"而不是另造一个返回结构体：两个服务的返回形状保持一致。
func decorate(work map[string]any, row *model.LibraryWork, cached bool) map[string]any {
	work["cached"] = cached
	if row == nil {
		return work
	}
	work["workId"] = row.WorkID
	work["libraryCreatedAt"] = row.CreatedAt
	work["libraryUpdatedAt"] = row.UpdatedAt
	work["coverDir"] = row.CoverDir
	work["coverPath"] = row.CoverPath
	work["coverAbsPath"] = row.CoverAbsPath
	work["coverBytes"] = row.CoverBytes
	work["coverSuffix"] = row.CoverSuffix
	// 游戏启动位置与元数据同表（library_works.game_path），一起带给前端：
	// 详情页据此决定「入库」按钮是否已经变成「游戏启动」并显示在哪个程序上。
	work["gamePath"] = row.GamePath
	// 启动方式（library_works.launch_mode，2026-11-23）：空 = 按来源取默认
	// （steam 源走 steam://rungameid，其它走本地 exe）。详情页据此显示"通过 Steam / 本地程序"。
	work["launchMode"] = row.LaunchMode
	return work
}

// workFromRawJSON 把库里的 raw_json 还原成 map（即当初抓到的 DLsiteWork json）。
func workFromRawJSON(row *model.LibraryWork) (map[string]any, bool) {
	raw := strings.TrimSpace(row.RawJSON)
	if raw == "" || raw == "{}" {
		return nil, false
	}
	work := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &work); err != nil {
		return nil, false
	}
	return work, true
}

// workIsComplete 判断一份元数据足不足以入库/复用缓存：编号与名称是底线，
// 另外**至少要有一个详情字段**（社团 / 作品形式 / 发售日 / 简介 / 容量）。
//
// 为什么要这层校验：2026-11 DLsite 换模板后，旧解析器只认得出名字与封面，
// 写进库的就是"只有编号+名称+封面"的空壳（raw_json 仅 6 个键）。如果只校验编号+名称，
// 这种空壳会被一直当成"完整数据"复用，用户永远看不到详细信息。
func workIsComplete(work map[string]any) bool {
	if strings.TrimSpace(strOf(work["id"])) == "" || strings.TrimSpace(strOf(work["name"])) == "" {
		return false
	}
	for _, key := range []string{"maker", "workType", "releaseDate", "summary", "fileSize"} {
		if strings.TrimSpace(strOf(work[key])) != "" {
			return true
		}
	}
	return false
}

// rowFromWork 从抓取结果里抽出入库需要的列。
// 只挑查询/展示用得到的字段；完整数据全部留在 raw_json 里。
// source 由调用方传入（当前恒为 dlsite），这样将来换个抓取来源也不用改这里。
func rowFromWork(source string, work map[string]any) model.LibraryWork {
	return model.LibraryWork{
		Source:      source,
		WorkID:      strings.ToUpper(strOf(work["id"])),
		Name:        strOf(work["name"]),
		Maker:       strOf(work["maker"]),
		MakerID:     strOf(work["makerId"]),
		Series:      strOf(work["series"]),
		SeriesID:    strOf(work["seriesId"]),
		WorkType:    strOf(work["workType"]),
		AgeRating:   strOf(work["ageRating"]),
		ReleaseDate: strOf(work["releaseDate"]),
		CoverURL:    strOf(work["coverUrl"]),
		SourceURL:   strOf(work["sourceUrl"]),
		RawJSON:     jsonOf(work),
		FetchedAt:   strOf(work["fetchedAt"]),
	}
}

func strOf(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func jsonOf(work map[string]any) string {
	blob, err := json.Marshal(work)
	if err != nil {
		return "{}"
	}
	return string(blob)
}

func mimeBySuffix(suffix string) string {
	switch strings.ToLower(suffix) {
	case ".webp":
		return "image/webp"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".avif":
		return "image/avif"
	case ".bmp":
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}
