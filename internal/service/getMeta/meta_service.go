// 本文件把 package getmeta 的抓取能力包装成一个 Wails 服务，
// 供前端「测试」页的接口测试卡片调用。
//
// 它是**薄包装**：解析逻辑全在 getFromDlsite.go，这里只做
// 「搜索 / 抓取 → 落盘图片 → 原样交给前端」这三件事。
//
// 数据刻意**不做二次加工**：FetchWork 返回的就是 DLsiteWork 的 json 编码，
// 前端直接展示原始字段，方便对着页面核对解析结果。
package getmeta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// preImgDir 是封面图片的落地目录。用**相对路径**，因此它的绝对位置
// 取决于进程的工作目录（实测我们会从仓库根目录跑，于是落在 WeedyBox/preImg/）。
// 返回值里同时给出这个相对路径和解析后的绝对路径，避免"到底存哪了"的猜测。
// 如果没有该文件夹就新建  -needcheck
const preImgDir = "preImg"

// MetaApiService 是暴露给前端的服务。
type MetaApiService struct {
	meta *MetaService
}

// NewMetaApiService 创建服务。
// 注意：不用 Wails 的默认构造逻辑，因为在 main.go 里直接 &MetaApiService{} 会漏掉
// meta 字段（Go 不会调用 NewMetaApiService），所以额外用 ensure() 兜底。
func NewMetaApiService() *MetaApiService {
	return &MetaApiService{meta: NewMetaService()}
}

func (s *MetaApiService) ensure() *MetaService {
	if s.meta == nil {
		s.meta = NewMetaService()
	}
	return s.meta
}

// ===== 返回给前端的结构（unexported：只用于生成 JSON，不需要生成 TS 类型）=====

type searchResponse struct {
	Query   string             `json:"query"`
	Kind    string             `json:"kind"` // "id"（按编号精确）或 "keyword"（按名字模糊）
	Matched bool               `json:"matched"`
	Message string             `json:"message,omitempty"`
	Work    any                `json:"work,omitempty"`
	Results []DLsiteSearchItem `json:"results,omitempty"`
}

// WorkRequest 是前端的入参。测试页同时支持「按编号」和「按名字模糊」，
// 但 Wails 绑定不擅长可选参数，所以两个字段都给，服务端自己决定用哪个。
type WorkRequest struct {
	Keyword string `json:"keyword"`
	WorkID  string `json:"workId"`
}

// ===== 对外方法 =====

// FetchWork 抓取一个作品：给 workId 就按编号精确抓，否则拿 keyword 模糊搜索后取最匹配的一个。
// 返回值是 `any`，实际是 map[string]any —— 即 DLsiteWork 的原样 json 编码。
func (s *MetaApiService) FetchWork(ctx context.Context, req WorkRequest) (any, error) {
	svc := s.ensure()
	keyword := strings.TrimSpace(req.Keyword)
	if workID := strings.TrimSpace(req.WorkID); workID != "" {
		keyword = workID
	}
	if keyword == "" {
		return nil, errors.New("请输入作品名或 RJ 编号")
	}

	// 关键词里带编号（"RJ01721474" 或 "RJ01721474 作品名"）就按编号精确抓，否则模糊搜索。
	if id, ok := NormalizeDLsiteID(keyword); ok {
		work, err := svc.GetWorkByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return toRawMap(work)
	}

	items, err := svc.SearchWorkByName(ctx, keyword)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("没有搜到与 %q 匹配的作品", keyword)
	}
	work, err := svc.GetWorkByID(ctx, items[0].ID)
	if err != nil {
		return nil, err
	}
	return toRawMap(work)
}

// SearchWorks 搜索作品：
//   - 关键词里含 RJ/RE/VJ 编号 → 按编号精确抓详情（kind = "id"），并附带该关键词的搜索结果列表
//   - 否则 → 模糊搜索，返回候选列表（kind = "keyword"），不自动抓详情
//
// 这样测试页能一次看到「搜索返回了什么」和「详情解析出了什么」。
func (s *MetaApiService) SearchWorks(ctx context.Context, keyword string) (*searchResponse, error) {
	svc := s.ensure()
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, errors.New("请输入作品名或 RJ 编号")
	}

	resp := &searchResponse{Query: keyword}

	// 列表页：失败不算致命（例如按编号搜列表页可能直接没有结果），只记录原因。
	items, searchErr := svc.SearchWorkByName(ctx, keyword)
	resp.Results = items

	if id, ok := NormalizeDLsiteID(keyword); ok {
		resp.Kind = "id"
		work, err := svc.GetWorkByID(ctx, id)
		if err != nil {
			if searchErr != nil {
				return nil, fmt.Errorf("抓取 %s 失败: %w", id, err)
			}
			return nil, err
		}
		raw, err := toRawMap(work)
		if err != nil {
			return nil, err
		}
		resp.Matched = true
		resp.Work = raw
		resp.Message = fmt.Sprintf("按编号 %s 精确抓取成功", id)
		return resp, nil
	}

	resp.Kind = "keyword"
	if searchErr != nil {
		return nil, searchErr
	}
	if len(items) == 0 {
		resp.Message = fmt.Sprintf("没有搜到与 %q 匹配的作品", keyword)
		return resp, nil
	}
	// 名称搜索不自动抓详情：候选可能有多个，由用户决定看哪个。
	resp.Message = fmt.Sprintf("按名字模糊搜索到 %d 个候选（搜索接口只返回 id/name/cover）", len(items))
	return resp, nil
}

// DownloadCover 下载封面并**原样**存到工作目录的相对路径 preImg/ 下，返回落盘信息。
// 不改分辨率、不转换格式；webp 也原样存 .webp。
func (s *MetaApiService) DownloadCover(ctx context.Context, workID, coverURL string) (map[string]any, error) {
	coverURL = strings.TrimSpace(coverURL)
	if coverURL == "" {
		return nil, errors.New("coverUrl 为空：该作品没有解析到封面，或详情还没抓取")
	}
	id, ok := NormalizeDLsiteID(workID)
	if !ok {
		id = "unknown"
	}

	data, ext, err := s.fetchImage(ctx, coverURL)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(preImgDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目录 %s 失败: %w", preImgDir, err)
	}

	name := fmt.Sprintf("%s_cover%s", id, ext)
	relPath := filepath.Join(preImgDir, name)

	savedPath, err := writeFileUnique(relPath, data)
	if err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(savedPath)
	return map[string]any{
		"id":       id,
		"dir":      preImgDir,                   // 相对工作目录
		"path":     filepath.ToSlash(savedPath), // 相对路径，落在哪个盘都稳定
		"absPath":  absPath,                     // 实际绝对位置，方便对照
		"bytes":    len(data),
		"suffix":   ext,
		"coverUrl": coverURL,
	}, nil
}

// PreImgDir 返回图片落地目录的绝对路径，前端可以显示"存到哪了"。
func (s *MetaApiService) PreImgDir() string {
	abs, err := filepath.Abs(preImgDir)
	if err != nil {
		return preImgDir
	}
	return abs
}

// ===== 内部实现 =====

// toRawMap 把作品结构体转成 map —— 走 json 标签往返一次，
// 这样前端看到的字段名与 json tag 完全一致（而不是 Go 的导出字段名）。
func toRawMap(v any) (map[string]any, error) {
	blob, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("序列化作品数据失败: %w", err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(blob, &out); err != nil {
		return nil, fmt.Errorf("反序列化作品数据失败: %w", err)
	}
	return out, nil
}

// fetchImage 下载图片字节并给出安全的文件后缀，带一次重试。
func (s *MetaApiService) fetchImage(ctx context.Context, rawURL string) ([]byte, string, error) {
	client := s.ensure().client
	var lastErr error

	for attempt := 0; attempt < 2; attempt++ {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		data, ext, err := s.tryFetchImage(ctx, client, rawURL)
		if err == nil {
			return data, ext, nil
		}
		lastErr = err
	}

	return nil, "", fmt.Errorf("下载封面失败: %w", lastErr)
}

func (s *MetaApiService) tryFetchImage(ctx context.Context, client *http.Client, rawURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("构造图片请求失败: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Referer", dlsiteBaseURL+"/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("请求图片失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("图片返回 HTTP %d", resp.StatusCode)
	}

	// 超限直接报错，不静默截断成坏图
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxCoverBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("读取图片失败: %w", err)
	}
	if len(data) > maxCoverBytes {
		return nil, "", fmt.Errorf("图片超过 %d 字节上限，已放弃", maxCoverBytes)
	}
	if len(data) == 0 {
		return nil, "", errors.New("图片内容为空")
	}

	return data, imageExt(rawURL, resp.Header.Get("Content-Type"), data), nil
}

// maxCoverBytes 封面体积上限。最大的 _smp1 预览图约 808KB，留 16MB 余量。
const maxCoverBytes = 16 << 20

// imageExt 决定文件后缀：真实内容 > Content-Type > URL 后缀 > 默认 .jpg。
// 硬编码 jpg 是错的（存在 webp 封面，实测 RJ01721474 就有 webp 档位）。
func imageExt(rawURL, contentType string, data []byte) string {
	switch {
	case len(data) >= 12 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return ".webp"
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return ".jpg"
	case len(data) >= 8 && bytes.Equal(data[0:8], []byte("\x89PNG\r\n\x1a\n")):
		return ".png"
	case len(data) >= 6 && bytes.Equal(data[0:6], []byte("GIF89a")):
		return ".gif"
	case len(data) >= 6 && bytes.Equal(data[0:6], []byte("GIF87a")):
		return ".gif"
	}

	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch ct {
	case "image/webp":
		return ".webp"
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	}

	if ext := imageExtFromURL(rawURL); ext != "" {
		return ext
	}
	return ".jpg"
}

func imageExtFromURL(rawURL string) string {
	base := rawURL
	if i := strings.IndexAny(base, "?#"); i >= 0 {
		base = base[:i]
	}
	switch ext := strings.ToLower(filepath.Ext(base)); ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif", ".bmp":
		return ext
	}
	return ""
}

// writeFileUnique 写文件；重名时加 -2 / -3 后缀，**不覆盖已有文件**
// （测试页反复点「下载」不该把之前下过的图冲掉）。
func writeFileUnique(path string, data []byte) (string, error) {
	for i := 1; i <= 999; i++ {
		candidate := path
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d%s", strings.TrimSuffix(path, filepath.Ext(path)), i, filepath.Ext(path))
		}
		f, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			if _, werr := f.Write(data); werr != nil {
				f.Close()
				return "", fmt.Errorf("写入 %s 失败: %w", candidate, werr)
			}
			if cerr := f.Close(); cerr != nil {
				return "", fmt.Errorf("关闭 %s 失败: %w", candidate, cerr)
			}
			return candidate, nil
		}
		if !os.IsExist(err) {
			return "", fmt.Errorf("写入 %s 失败: %w", candidate, err)
		}
	}
	return "", fmt.Errorf("同名文件太多，放弃写入 %s", path)
}
