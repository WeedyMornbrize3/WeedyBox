// internal/service/library_service.go
//
// 「游戏库」的编排层：把**本地 SQLite 缓存**和**DLsite 联网抓取**接在一起。
//
// 分工：
//   - 抓取/解析      internal/service/getMeta（不知道数据库的存在）
//   - 持久化         internal/database/dlsite_repo.go（只搬数据）
//   - 缓存策略       ← 本文件（先查库，没有再联网，抓回来顺手入库）
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
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
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
	Source  string `json:"source"`  // 元数据来源，空 = dlsite
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

// Fetch 取一个作品：**缓存优先**。
//
//  1. 能确定编号时先查 library_works（按 source + work_id），命中就把 raw_json
//     还原成 map 返回（不打网络）；
//  2. 没命中（或 Refresh=true）才联网抓取，抓回来 upsert 入库；
//  3. 返回值里额外注入 cached / cover* 几个键，前端据此显示「缓存」标记与本地封面。
//
// 返回的 map 与 metaApi.FetchWork 的字段完全一致（就是同一份 DLsiteWork 的 json），
// 只是多了上面那几个键 —— 这样两个服务的返回类型对前端是兼容的。
func (s *LibraryService) Fetch(ctx context.Context, req FetchWorkRequest) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	keyword := strings.TrimSpace(req.Keyword)
	workID := strings.TrimSpace(req.WorkID)
	if workID == "" {
		if id, ok := getmeta.NormalizeDLsiteID(keyword); ok {
			workID = id
		}
	}
	source := req.Source
	if strings.TrimSpace(source) == "" {
		source = database.SourceDLsite
	}

	// ① 缓存
	if workID != "" && !req.Refresh {
		row, err := s.DB.FindLibraryWork(ctx, source, workID)
		// 查缓存失败不该让整个页面不可用：退化成"没缓存"，继续联网抓。
		if err == nil && row != nil {
			if work, ok := workFromRawJSON(row); ok {
				return decorate(work, row, true), nil
			}
			// raw_json 坏了（手改过库/旧数据）→ 当作没缓存，重新抓一份覆盖掉
		}
	}

	// ② 联网
	data, err := s.ensureMeta().FetchWork(ctx, getmeta.WorkRequest{Keyword: keyword, WorkID: workID})
	if err != nil {
		return nil, err
	}
	work, ok := data.(map[string]any)
	if !ok {
		return nil, errors.New("后端返回了无法识别的作品数据")
	}

	// ③ 入库（失败不影响这次展示：缓存是"下次更快"，不该挡住当前结果）
	row := rowFromWork(source, work)
	if err := s.DB.UpsertLibraryWork(ctx, &row); err != nil {
		fmt.Printf("写入游戏库失败（本次结果仍会返回）: %v\n", err)
	} else if saved, ferr := s.DB.FindLibraryWork(ctx, source, row.WorkID); ferr == nil && saved != nil {
		row = *saved // 取回库里那份，才能拿到已有的 cover_* 与时间戳
	}

	return decorate(work, &row, false), nil
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

// Remove 从游戏库删掉一条作品（磁盘上的图片保留）。
func (s *LibraryService) Remove(ctx context.Context, workID, source string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
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
	return s.DB.DeleteLibraryWork(ctx, source, workID)
}

// CoverDataURL 把游戏库里记录的本地封面读成 data URL，给 <img> 直接显示。
//
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

// ===== 内部实现 =====

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
