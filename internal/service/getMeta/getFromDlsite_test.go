package getmeta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// fixture 是真实抓下来的作品页（2026-10 抓取），放在 testdata/ 下。
// 用它做回归：一旦解析结果变化，测试立刻失败，而不是等到接口返回空值。
func loadFixture(t *testing.T, name string) *goquery.Document {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("打开 fixture 失败: %v", err)
	}
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatalf("解析 fixture 失败: %v", err)
	}
	return doc
}

// ===== NormalizeDLsiteID =====

func TestNormalizeDLsiteID(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		valid bool
	}{
		// 前导零必须保留：12 位新编号丢掉 0 会 404
		{"RJ01721474", "RJ01721474", true},
		{"RJ01682332", "RJ01682332", true},
		// 小写要转大写
		{"rj01721474", "RJ01721474", true},
		{"  RJ01721474  ", "RJ01721474", true},
		// 嵌在 URL / 文案里也要能抽出来
		{"https://www.dlsite.com/maniax/work/=/product_id/RJ01123456.html", "RJ01123456", true},
		{"【RJ189758】作品名", "RJ189758", true},
		{"foo RJ1234 RJ4321 bar", "RJ1234", true},
		// RE / VJ 前缀
		{"re123456", "RE123456", true},
		{"VJ123456", "VJ123456", true},
		// 无效输入
		{"", "", false},
		{"foobar", "", false},
		{"RJ", "", false},
		{"RJ123", "", false}, // 至少 4 位数字
		{"XX123456", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizeDLsiteID(c.in)
		if ok != c.valid || got != c.want {
			t.Errorf("NormalizeDLsiteID(%q) = (%q, %v)，期望 (%q, %v)", c.in, got, ok, c.want, c.valid)
		}
	}
}

// ===== 真实页面解析 =====

func TestParseWorkRJ01721474(t *testing.T) {
	svc := NewMetaService()
	doc := loadFixture(t, "work_RJ01721474.html")

	work, err := svc.parseWork(doc, "RJ01721474")
	if err != nil {
		t.Fatalf("parseWork 失败: %v", err)
	}

	// 第三方库正是在「作品名」上失败的（它只找 #work_name 里的 <a>，而现在的页面没有 <a>）
	if work.Name == "" {
		t.Error("Name 为空：说明又退回了「只找 #work_name 里的 <a>」的写法")
	}
	if !strings.Contains(work.Name, "ELF PARADISE") {
		t.Errorf("Name = %q，期望包含 ELF PARADISE", work.Name)
	}
	if work.Maker != "青春×フェティシズム" {
		t.Errorf("Maker = %q", work.Maker)
	}
	if work.MakerID != "RG51931" {
		t.Errorf("MakerID = %q，期望 RG51931", work.MakerID)
	}
	if work.Series != "ELF PARADISE" {
		t.Errorf("Series = %q", work.Series)
	}
	if work.WorkType != "ボイス・ASMR" {
		t.Errorf("WorkType = %q", work.WorkType)
	}
	if work.AgeRating != "R18" {
		t.Errorf("AgeRating = %q", work.AgeRating)
	}
	if work.FileSize != "16.83GB" {
		t.Errorf("FileSize = %q", work.FileSize)
	}
	// 2026年09月26日 → 2026-09-26
	if work.ReleaseDate != "2026-09-26" {
		t.Errorf("ReleaseDate = %q，期望 2026-09-26", work.ReleaseDate)
	}
	if work.Summary == "" || len(work.Summary) < 500 {
		t.Errorf("Summary 过短（%d 字符），简介可能没取到", len(work.Summary))
	}
	if work.SourceURL != "https://www.dlsite.com/maniax/work/=/product_id/RJ01721474.html" {
		t.Errorf("SourceURL = %q", work.SourceURL)
	}
	if work.CoverURL != "" && !strings.HasPrefix(work.CoverURL, "https://") {
		t.Errorf("CoverURL 不是绝对地址: %q", work.CoverURL)
	}
	if !strings.Contains(work.CoverURL, "_img_main") {
		t.Errorf("CoverURL = %q，期望命中 _img_main 大图", work.CoverURL)
	}
	if len(work.Outline) < 9 {
		t.Errorf("Outline 只有 %d 行，页面应该有 14 行左右", len(work.Outline))
	}
	// 价格 / 评分 / 币种都是 <meta itemprop="…" content="…">，值在 content 上。
	// 曾经写成取 .Text() 得到空串，故这里逐项断言。
	if work.Price != "1540" {
		t.Errorf("Price = %q，期望 1540（注意取值要用 content 属性）", work.Price)
	}
	if work.Currency != "JPY" {
		t.Errorf("Currency = %q，期望 JPY", work.Currency)
	}
	if work.Rating != "5" {
		t.Errorf("Rating = %q，期望 5", work.Rating)
	}
	if work.RatingCount != "16" {
		t.Errorf("RatingCount = %q，期望 16", work.RatingCount)
	}
	if work.SeriesID != "SRI0000043774" {
		t.Errorf("SeriesID = %q，期望 SRI0000043774", work.SeriesID)
	}
	if work.AlternateID == "" || strings.Contains(work.AlternateID, " ") {
		t.Errorf("AlternateID = %q，期望非空 slug", work.AlternateID)
	}
	if work.PriceUntil == "" {
		t.Error("PriceUntil 为空，页面里应有 priceValidUntil")
	}
	// 「声優」「作者」这类行在 outline 里，不进 Tags
	if len(work.Tags) < 5 {
		t.Errorf("Tags 只有 %d 个: %v", len(work.Tags), work.Tags)
	}

	t.Logf("Name=%s", work.Name)
	t.Logf("Maker=%s(%s) Series=%s", work.Maker, work.MakerID, work.Series)
	t.Logf("WorkType=%s AgeRating=%s FileSize=%s Release=%s", work.WorkType, work.AgeRating, work.FileSize, work.ReleaseDate)
	t.Logf("FileFormats=%v", work.FileFormats)
	t.Logf("Tags=%v", work.Tags)
	t.Logf("Summary %d 字符, Outline %d 行", len(work.Summary), len(work.Outline))
}

func TestParseWorkRJ01682332(t *testing.T) {
	svc := NewMetaService()
	doc := loadFixture(t, "work_RJ01682332.html")

	work, err := svc.parseWork(doc, "RJ01682332")
	if err != nil {
		t.Fatalf("parseWork 失败: %v", err)
	}

	if work.Name == "" {
		t.Error("Name 为空")
	}
	if !strings.Contains(work.Name, "乙女の花園") {
		t.Errorf("Name = %q，期望包含 乙女の花園", work.Name)
	}
	if work.Maker != "RaRo" {
		t.Errorf("Maker = %q", work.Maker)
	}
	if work.AgeRating != "全年齢" {
		t.Errorf("AgeRating = %q，期望 全年齢", work.AgeRating)
	}
	// 2026年10月07日 0時 → 2026-10-07（带「0時」也要能解析）
	if work.ReleaseDate != "2026-10-07" {
		t.Errorf("ReleaseDate = %q，期望 2026-10-07", work.ReleaseDate)
	}
	// 这个作品没有「シリーズ名」，应为空而不是解析失败
	if work.Series != "" {
		t.Errorf("Series = %q，期望为空", work.Series)
	}
	if work.Price != "1650" {
		t.Errorf("Price = %q，期望 1650", work.Price)
	}
	if work.Rating == "" || work.RatingCount == "" {
		t.Errorf("Rating/RatingCount 不应为空，得到 %q / %q", work.Rating, work.RatingCount)
	}
	if len(work.Outline) < 8 {
		t.Errorf("Outline 只有 %d 行，页面应该有 9 行左右", len(work.Outline))
	}
	if len(work.Tags) < 5 {
		t.Errorf("Tags 只有 %d 个: %v", len(work.Tags), work.Tags)
	}

	t.Logf("Name=%s", work.Name)
	t.Logf("Maker=%s AgeRating=%s Release=%s", work.Maker, work.AgeRating, work.ReleaseDate)
	t.Logf("Tags=%v", work.Tags)
}

// 页面结构变化时 parseWork 必须报错，而不是静默返回一个空壳。
func TestParseWorkRejectsEmptyPage(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<html><body><p>該当作品がありません</p></body></html>"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewMetaService().parseWork(doc, "RJ00000000"); err == nil {
		t.Error("空页面应返回错误，而不是一个字段全空的结构体")
	}
}

// 页面没有 series_id 时不能 panic（FindStringSubmatch 无匹配返回 nil）。
func TestExtractSeriesIDWithoutMatch(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<html><body>no series here</body></html>"))
	if err != nil {
		t.Fatal(err)
	}
	if got := extractSeriesID(doc); got != "" {
		t.Errorf("extractSeriesID = %q，期望空串", got)
	}
}

// ===== 纯函数 =====

func TestNormalizeJapaneseDate(t *testing.T) {
	cases := map[string]string{
		"2026年09月26日":     "2026-09-26",
		"2026年10月07日 0時": "2026-10-07",
		"2026/09/26":       "2026-09-26",
		"2026.9.6":         "2026-09-06",
		"2026-09-26":       "2026-09-26",
		"":                 "",
		"未定":               "未定", // 解析不了就原样返回，不丢信息
	}
	for in, want := range cases {
		if got := normalizeJapaneseDate(in); got != want {
			t.Errorf("normalizeJapaneseDate(%q) = %q，期望 %q", in, got, want)
		}
	}
}

func TestSplitSlashList(t *testing.T) {
	got := splitSlashList("WAV / mp3 / mp4同梱")
	want := []string{"WAV", "mp3", "mp4同梱"}
	if len(got) != len(want) {
		t.Fatalf("splitSlashList = %v，期望 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 项 = %q，期望 %q", i, got[i], want[i])
		}
	}
	if splitSlashList("   ") != nil {
		t.Error("空白输入应返回 nil")
	}
}

func TestCleanText(t *testing.T) {
	if got := cleanText("  a \n\t b   c "); got != "a b c" {
		t.Errorf("cleanText = %q，期望 %q", got, "a b c")
	}
}

func TestNormalizeSourceURL(t *testing.T) {
	cases := map[string]string{
		"//img.dlsite.jp/x.jpg":  "https://img.dlsite.jp/x.jpg",
		"/img/x.jpg":             dlsiteBaseURL + "/img/x.jpg",
		"https://a.example/b":    "https://a.example/b",
		"http://a.example/b":     "http://a.example/b",
		"":                       "",
	}
	for in, want := range cases {
		if got := normalizeSourceURL(in); got != want {
			t.Errorf("normalizeSourceURL(%q) = %q，期望 %q", in, got, want)
		}
	}
}

func TestBuildWorkURL(t *testing.T) {
	// RJ/RE → maniax
	if got := buildWorkURL("RJ01721474"); !strings.Contains(got, "/maniax/work/") {
		t.Errorf("RJ 应走 maniax，得到 %q", got)
	}
	// VJ → pro
	if got := buildWorkURL("VJ123456"); !strings.Contains(got, "/pro/work/") {
		t.Errorf("VJ 应走 pro，得到 %q", got)
	}
	if !strings.HasPrefix(buildWorkURL("RJ01721474"), "https://") {
		t.Error("必须走 HTTPS：http:// 会被重定向或在部分网络下直接超时")
	}
}

func TestBuildSearchURL(t *testing.T) {
	got := buildSearchURL("ELF PARADISE")
	if strings.ContainsAny(got, " ") {
		t.Errorf("搜索词里的空格应转义成 +，得到 %q", got)
	}
	if !strings.Contains(got, "ELF+PARADISE") {
		t.Errorf("buildSearchURL = %q", got)
	}
}

// ===== 选项 =====

func TestNewMetaServiceOptions(t *testing.T) {
	svc := NewMetaService(WithTagLimit(2))
	if svc.tagLimit != 2 {
		t.Errorf("tagLimit = %d，期望 2", svc.tagLimit)
	}
	// 不传选项时应有默认超时，避免像第三方库那样无限等待
	if svc.client.Timeout <= 0 {
		t.Error("默认客户端必须有超时")
	}
	if time.Duration(svc.client.Timeout) != defaultTimeout {
		t.Errorf("默认超时 = %v，期望 %v", svc.client.Timeout, defaultTimeout)
	}

	tags := svc.extractTags(loadFixture(t, "work_RJ01721474.html"), parseOutline(loadFixture(t, "work_RJ01721474.html")))
	if len(tags) != 2 {
		t.Errorf("WithTagLimit(2) 后应只返回 2 个标签，得到 %d: %v", len(tags), tags)
	}
}
