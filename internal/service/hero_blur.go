// internal/service/hero_blur.go
//
// 预生成「模糊 hero 图」并缓存到 covers/。
//
// 背景（2026-11-25 用户要求）：
//
//	「只用一张模糊hero图片对所有内容进行遮罩 … 卡片模糊效果可以使用外部库处理
//	  缓存到cover文件夹后直接使用模糊后的图片 避免渲染算力浪费」
//
// 为什么放后端做：
//
//	/works 的遮罩层原来每层都在 CSS 里做 filter: blur(18~22px)，滚动时 WebView2 要反复
//	重新计算大面积高斯模糊，是上下滑卡顿的主因。这里一次算好、落盘，前端只把它当普通图片
//	铺到各层（同一张 bitmap，不再有任何 filter），滚动只剩合成。
//
// 实现选择：不引第三方图像库（项目准则：能不加依赖就不加）——
//
//	先缩到 480px 宽（模糊图不需要高分辨率），再用两遍可分离盒式模糊近似高斯，
//	JPEG 80 落盘。480px 的图 base64 后约 20~40KB，传给 WebView 可接受。
//	源图解码只依赖标准库的 jpeg/png/gif；webp 源解码失败时返回错误，前端自动不做遮罩。
package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	// blurredHeroMaxWidth 预模糊图的宽度上限：遮罩本来就是糊的，480px 足够，
	// 再大只会增加 base64 传输与解码成本。
	blurredHeroMaxWidth = 480
	// blurredHeroRadius 在缩小后的图上做盒式模糊的半径（两遍）。
	blurredHeroRadius = 10
	// blurredHeroQuality JPEG 质量：模糊图不需要高保真。
	blurredHeroQuality = 80
)

// 文件名里只保留字母/数字/下划线/连字符，避免编号里出现路径分隔符。
var unsafeFileToken = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// HeroBlur 返回某个作品 hero 图的「预模糊版本」（data URL），并缓存到 covers/。
//
// rawURL 是前端当前用的 hero 图（Steam 源是本地图的 data URL，DLsite 源是预览图直链）；
// 缓存文件名带 URL 的哈希，换图/重新抓取后会自动生成新文件，不会拿到旧图的模糊版本。
// 缓存命中直接读盘，不联网、不重新计算。
func (s *LibraryService) HeroBlur(ctx context.Context, workID, source, rawURL string) (string, error) {
	workID = strings.TrimSpace(workID)
	rawURL = strings.TrimSpace(rawURL)
	if workID == "" || rawURL == "" {
		return "", nil
	}

	dir := s.ensureMeta().CoversDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("创建封面目录失败: %w", err)
	}
	name := fmt.Sprintf(
		"hero_blur_v3_%s_%s_%08x.jpg",
		normalizeBlurSource(source),
		safeFileToken(workID),
		hashURL(rawURL),
	)
	path := filepath.Join(dir, name)

	// ① 缓存命中：直接读回
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		return dataURLJPEG(data), nil
	}

	// ② 取原图字节（data URL 直接解 base64；http(s) 复用 getMeta 的下载器，带 UA/Referer）
	data, err := s.loadHeroImageBytes(ctx, rawURL)
	if err != nil {
		return "", err
	}

	// ③ 解码（webp 不支持 —— 解码失败就返回错误，前端退化为不做遮罩而不是崩页面）
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("hero 图解码失败（可能是 webp）: %w", err)
	}

	// ④ 缩小 + 两遍盒式模糊 → JPEG
	small := downscaleNRGBA(img, blurredHeroMaxWidth)
	blurred := boxBlur(small, blurredHeroRadius)
	blurred = boxBlur(blurred, blurredHeroRadius)
	// 用户要求：模糊后的图上下镜像一次再用作遮罩
	blurred = flipVertical(blurred)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, blurred, &jpeg.Options{Quality: blurredHeroQuality}); err != nil {
		return "", fmt.Errorf("生成模糊 hero 图失败: %w", err)
	}

	// 先写临时文件再改名，避免读到半张图
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err == nil {
		if err := os.Rename(tmp, path); err != nil {
			_ = os.Remove(path)
			_ = os.Rename(tmp, path)
		}
	}
	// 同一作品只保留这一张：hero 图可能先给远程预览、后切成 Steam 本地 hero，
	// 两次生成会得到两个哈希文件；写完新的就把旧版本清掉（用户反馈「生成两张图片」）。
	s.pruneHeroBlurCache(workID, source, path)
	return dataURLJPEG(buf.Bytes()), nil
}

// loadHeroImageBytes 取得 hero 原图字节。
// data URL 走本地 base64 解码；http(s) 交给 getMeta 的下载函数（同一套 UA/Referer/重试）。
func (s *LibraryService) loadHeroImageBytes(ctx context.Context, rawURL string) ([]byte, error) {
	if strings.HasPrefix(rawURL, "data:") {
		comma := strings.Index(rawURL, ",")
		if comma < 0 || comma == len(rawURL)-1 {
			return nil, fmt.Errorf("hero data URL 格式不正确")
		}
		payload := rawURL[comma+1:]
		if !strings.Contains(rawURL[:comma], ";base64") {
			return nil, fmt.Errorf("hero data URL 不是 base64 编码")
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("hero data URL 解码失败: %w", err)
		}
		return data, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return s.ensureMeta().FetchImageBytes(ctx, rawURL)
}

// RemoveHeroBlurCache 删除某个作品所有版本的模糊 hero 图（作品被移除时调用）。
// 删不掉只忽略：它是纯缓存，留着也只占几十 KB。
func (s *LibraryService) removeHeroBlurCache(workID, source string) {
	workID = strings.TrimSpace(workID)
	if workID == "" {
		return
	}
	s.pruneHeroBlurCache(workID, source, "")
}

// pruneHeroBlurCache 删除同一作品旧版本的模糊图，keepPath 为空 = 全删。
// 名称里带 URL 哈希是为了换图后不复用旧图，但同一作品只在盘上留一张。
func (s *LibraryService) pruneHeroBlurCache(workID, source, keepPath string) {
	dir := s.ensureMeta().CoversDir()
	prefixes := []string{"hero_blur_v3", "hero_blur_v2", "hero_blur"}
	for _, prefix := range prefixes {
		pattern := filepath.Join(dir, fmt.Sprintf("%s_%s_%s_*.jpg", prefix, normalizeBlurSource(source), safeFileToken(workID)))
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, path := range matches {
			if keepPath != "" && filepath.Clean(path) == filepath.Clean(keepPath) {
				continue
			}
			_ = os.Remove(path)
		}
	}
}

func normalizeBlurSource(source string) string {
	src := strings.ToLower(strings.TrimSpace(source))
	if unsafeFileToken.MatchString(src) || src == "" {
		return "dlsite"
	}
	return src
}

func safeFileToken(value string) string {
	cleaned := unsafeFileToken.ReplaceAllString(value, "_")
	cleaned = strings.Trim(cleaned, "_")
	if cleaned == "" {
		return "work"
	}
	return cleaned
}

func hashURL(rawURL string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(rawURL))
	return h.Sum32()
}

func dataURLJPEG(data []byte) string {
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data)
}

// downscaleNRGBA 用面积平均缩到 maxWidth 宽（保持比例），质量比最近邻好、成本仍然很低。
func downscaleNRGBA(img image.Image, maxWidth int) *image.NRGBA {
	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, 1, 1))
	}
	if srcW <= maxWidth {
		out := image.NewNRGBA(image.Rect(0, 0, srcW, srcH))
		for y := 0; y < srcH; y++ {
			for x := 0; x < srcW; x++ {
				out.Set(x, y, img.At(bounds.Min.X+x, bounds.Min.Y+y))
			}
		}
		return out
	}

	dstW := maxWidth
	dstH := srcH * dstW / srcW
	if dstH < 1 {
		dstH = 1
	}
	out := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))
	// 关键性能点：不要在每个采样点上做 color.NRGBAModel.Convert(img.At(...))（接口 + 转换，
	// 1920x1080 → 480x270 约 200 万次，实测要几百毫秒）。按真实类型取色，JPEG 直接走 YCbCr。
	sample := newPixelSampler(img, bounds)
	for y := 0; y < dstH; y++ {
		sy0 := bounds.Min.Y + y*srcH/dstH
		sy1 := bounds.Min.Y + (y+1)*srcH/dstH
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < dstW; x++ {
			sx0 := bounds.Min.X + x*srcW/dstW
			sx1 := bounds.Min.X + (x+1)*srcW/dstW
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var r, g, b, a, n uint32
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					sr, sg, sb, sa := sample(sx, sy)
					r += sr
					g += sg
					b += sb
					a += sa
					n++
				}
			}
			if n == 0 {
				n = 1
			}
			out.SetNRGBA(x, y, color.NRGBA{uint8(r / n), uint8(g / n), uint8(b / n), uint8(a / n)})
		}
	}
	return out
}

// newPixelSampler 给常见图像类型各返回一个无接口的取色函数。
// 遮罩生成是同步 IPC，缩图占了大头；这里把每次采样的开销压到一次数组访问。
func newPixelSampler(img image.Image, bounds image.Rectangle) func(x, y int) (uint32, uint32, uint32, uint32) {
	switch src := img.(type) {
	case *image.YCbCr:
		return func(x, y int) (uint32, uint32, uint32, uint32) {
			yi := src.YOffset(x, y)
			ci := src.COffset(x, y)
			r, g, b := color.YCbCrToRGB(src.Y[yi], src.Cb[ci], src.Cr[ci])
			return uint32(r), uint32(g), uint32(b), 255
		}
	case *image.RGBA:
		return func(x, y int) (uint32, uint32, uint32, uint32) {
			i := src.PixOffset(x, y)
			p := src.Pix[i : i+4]
			a := uint32(p[3])
			if a == 0 {
				return 0, 0, 0, 0
			}
			if a == 255 {
				return uint32(p[0]), uint32(p[1]), uint32(p[2]), 255
			}
			// RGBA 是预乘 alpha，摊回直通色
			return uint32(p[0]) * 255 / a, uint32(p[1]) * 255 / a, uint32(p[2]) * 255 / a, 255
		}
	case *image.NRGBA:
		return func(x, y int) (uint32, uint32, uint32, uint32) {
			i := src.PixOffset(x, y)
			p := src.Pix[i : i+4]
			return uint32(p[0]), uint32(p[1]), uint32(p[2]), uint32(p[3])
		}
	case *image.Gray:
		return func(x, y int) (uint32, uint32, uint32, uint32) {
			v := uint32(src.GrayAt(x, y).Y)
			return v, v, v, 255
		}
	default:
		return func(x, y int) (uint32, uint32, uint32, uint32) {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			return uint32(c.R), uint32(c.G), uint32(c.B), uint32(c.A)
		}
	}
}

// boxBlur 可分离盒式模糊（先横后纵），边缘按 clamp 处理。
// 两遍叠加后接近高斯，足够做背景遮罩，成本远低于逐像素高斯。
func boxBlur(src *image.NRGBA, radius int) *image.NRGBA {
	if radius <= 0 {
		return src
	}
	return blurVertical(blurHorizontal(src, radius), radius)
}

func blurHorizontal(src *image.NRGBA, radius int) *image.NRGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	window := 2*radius + 1
	for y := 0; y < h; y++ {
		var r, g, b int
		for i := -radius; i <= radius; i++ {
			x := clampInt(i, 0, w-1)
			c := src.NRGBAAt(x, y)
			r += int(c.R)
			g += int(c.G)
			b += int(c.B)
		}
		for x := 0; x < w; x++ {
			out.SetNRGBA(x, y, color.NRGBA{uint8(r / window), uint8(g / window), uint8(b / window), 255})
			addX := clampInt(x+radius+1, 0, w-1)
			subX := clampInt(x-radius, 0, w-1)
			add := src.NRGBAAt(addX, y)
			sub := src.NRGBAAt(subX, y)
			r += int(add.R) - int(sub.R)
			g += int(add.G) - int(sub.G)
			b += int(add.B) - int(sub.B)
		}
	}
	return out
}

func blurVertical(src *image.NRGBA, radius int) *image.NRGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	window := 2*radius + 1
	for x := 0; x < w; x++ {
		var r, g, b int
		for i := -radius; i <= radius; i++ {
			y := clampInt(i, 0, h-1)
			c := src.NRGBAAt(x, y)
			r += int(c.R)
			g += int(c.G)
			b += int(c.B)
		}
		for y := 0; y < h; y++ {
			out.SetNRGBA(x, y, color.NRGBA{uint8(r / window), uint8(g / window), uint8(b / window), 255})
			addY := clampInt(y+radius+1, 0, h-1)
			subY := clampInt(y-radius, 0, h-1)
			add := src.NRGBAAt(x, addY)
			sub := src.NRGBAAt(x, subY)
			r += int(add.R) - int(sub.R)
			g += int(add.G) - int(sub.G)
			b += int(add.B) - int(sub.B)
		}
	}
	return out
}

// flipVertical 上下镜像：用户要求「模糊后的图片要作上下镜像处理」。
// 对预模糊的小图做一次行倒序拷贝，成本可忽略；镜像后的图才会写进缓存。
func flipVertical(src *image.NRGBA) *image.NRGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		srcY := h - 1 - y
		for x := 0; x < w; x++ {
			out.SetNRGBA(x, y, src.NRGBAAt(x, srcY))
		}
	}
	return out
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
