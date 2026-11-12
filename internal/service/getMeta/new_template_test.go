// internal/service/getMeta/new_template_test.go
//
// 2026-11 起 DLsite 作品页换成了新模板（Vue + JSON-LD）：老的 #work_outline 表格、
// #work_maker、<meta itemprop="price/ratingValue"> 全都不在了。只认旧选择器时
// parseWork 会返回"只有名字+封面"的空壳 —— 扫描入库那批作品就是这么丢信息的。
// 这里用从真实页面摘出来的 fixture 锁住新模板的每个字段。
package getmeta

import (
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func loadFixture(t *testing.T, name string) *goquery.Document {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("读取 fixture 失败: %v", err)
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("解析 fixture HTML 失败: %v", err)
	}
	return doc
}

func TestParseWorkNewTemplate(t *testing.T) {
	service := &MetaService{}
	work, err := service.parseWork(loadFixture(t, "work_new_RJ01232863.html"), "RJ01232863")
	if err != nil {
		t.Fatalf("解析新模板页面失败: %v", err)
	}

	if work.Name != "ゴブリンの巣穴 戦国餓鬼道" {
		t.Errorf("Name = %q", work.Name)
	}
	if work.Maker != "ぺぺろんちーの" {
		t.Errorf("Maker = %q", work.Maker)
	}
	if work.MakerID != "RG24495" {
		t.Errorf("MakerID = %q", work.MakerID)
	}
	if work.Series != "ゴブリンの巣穴" {
		t.Errorf("Series = %q", work.Series)
	}
	if work.SeriesID != "SRI0000026263" {
		t.Errorf("SeriesID = %q", work.SeriesID)
	}
	if !strings.Contains(work.WorkType, "シミュレーション") {
		t.Errorf("WorkType = %q", work.WorkType)
	}
	if work.AgeRating != "R18" {
		t.Errorf("AgeRating = %q", work.AgeRating)
	}
	if work.FileSize != "9.05GB" {
		t.Errorf("FileSize = %q", work.FileSize)
	}
	if work.ReleaseDate != "2024-08-13" {
		t.Errorf("ReleaseDate = %q，期望归一化为 2024-08-13", work.ReleaseDate)
	}
	// 价格与评分来自 JSON-LD（旧页面是 meta itemprop，新页面已没有）
	if work.Price != "5500" || work.Currency != "JPY" {
		t.Errorf("Price/Currency = %q/%q", work.Price, work.Currency)
	}
	if work.Rating != "4.43" || work.RatingCount != "1484" {
		t.Errorf("Rating/RatingCount = %q/%q", work.Rating, work.RatingCount)
	}
	if work.AlternateID != "goburinnosuanasengokugakidou" {
		t.Errorf("AlternateID = %q", work.AlternateID)
	}
	if work.CoverURL != "https://img.dlsite.jp/modpub/images2/work/doujin/RJ01233000/RJ01232863_img_main.jpg" {
		t.Errorf("CoverURL = %q", work.CoverURL)
	}
	if !strings.Contains(work.Summary, "群雄割拠") {
		t.Errorf("Summary = %q", work.Summary)
	}
	if len(work.FileFormats) != 1 || work.FileFormats[0] != "アプリケーション" {
		t.Errorf("FileFormats = %v", work.FileFormats)
	}
	if work.SourceURL != "https://www.dlsite.com/maniax/work/=/product_id/RJ01232863.html" {
		t.Errorf("SourceURL = %q", work.SourceURL)
	}

	// 标签与 info 表（作品库详情页直接展示 outline）
	hasTag := func(want string) bool {
		for _, tag := range work.Tags {
			if tag == want {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"異種えっち", "R18"} {
		if !hasTag(want) {
			t.Errorf("Tags 缺少 %q：%v", want, work.Tags)
		}
	}
	if got := outlineValue(work.Outline, "ファイル容量"); got != "9.05GB" {
		t.Errorf("Outline 里没有文件容量: %v", work.Outline)
	}
	if got := outlineValue(work.Outline, "販売日"); got != "2024年8月13日" {
		t.Errorf("Outline 里没有販売日: %v", work.Outline)
	}
}

// TestParseWorkChineseTemplate 应用实际发的是 Cookie locale=zh-cn，页面标签是中文
// （社团名/发售日/系列名/年龄指定/文件容量/分类）。只认日文标签会在真实页面上全部取空，
// 这个 fixture 就是中文版页面，用来锁住「中日文标签都能认」。
func TestParseWorkChineseTemplate(t *testing.T) {
	service := &MetaService{}
	work, err := service.parseWork(loadFixture(t, "work_new_zh_RJ01232863.html"), "RJ01232863")
	if err != nil {
		t.Fatalf("解析中文模板页面失败: %v", err)
	}

	if work.Name != "【日文版】哥布林的巢穴 战国饿鬼道" {
		t.Errorf("Name = %q", work.Name)
	}
	if work.Maker != "ぺぺろんちーの" || work.MakerID != "RG24495" {
		t.Errorf("Maker/MakerID = %q/%q", work.Maker, work.MakerID)
	}
	if work.Series != "ゴブリンの巣穴" || work.SeriesID != "SRI0000026263" {
		t.Errorf("Series/SeriesID = %q/%q", work.Series, work.SeriesID)
	}
	if !strings.Contains(work.WorkType, "模拟") {
		t.Errorf("WorkType = %q", work.WorkType)
	}
	if work.AgeRating != "R18" {
		t.Errorf("AgeRating = %q", work.AgeRating)
	}
	if work.FileSize != "9.05GB" {
		t.Errorf("FileSize = %q（中文页标签是「文件容量」）", work.FileSize)
	}
	if work.ReleaseDate != "2024-08-13" {
		t.Errorf("ReleaseDate = %q（中文页标签是「发售日」）", work.ReleaseDate)
	}
	if work.Price != "5500" || work.Rating != "4.43" || work.RatingCount != "1484" {
		t.Errorf("价格/评分 = %q/%q/%q", work.Price, work.Rating, work.RatingCount)
	}
	if len(work.FileFormats) != 1 || work.FileFormats[0] != "软件" {
		t.Errorf("FileFormats = %v（中文页标签是「文件形式」）", work.FileFormats)
	}

	hasTag := func(want string) bool {
		for _, tag := range work.Tags {
			if tag == want {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"异种X", "R18"} {
		if !hasTag(want) {
			t.Errorf("Tags 缺少 %q：%v", want, work.Tags)
		}
	}
	if got := outlineValue(work.Outline, "发售日"); got != "2024年8月13日" {
		t.Errorf("Outline 里没有中文发售日: %v", work.Outline)
	}
}

// TestSplitTagText 用户要求的切分口径：空白字符 + "/"。
func TestSplitTagText(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"双声道立体声/人头麦", []string{"双声道立体声", "人头麦"}},
		{"みたかりん 野上菜月", []string{"みたかりん", "野上菜月"}},
		{"ASMR\t内射/中出  巨乳", []string{"ASMR", "内射", "中出", "巨乳"}},
		{"   ", nil},
		{"音声・ASMR", []string{"音声・ASMR"}},
		{"a/b/c", []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		got := splitTagText(tc.input)
		if strings.Join(got, "|") != strings.Join(tc.want, "|") {
			t.Errorf("splitTagText(%q) = %v，期望 %v", tc.input, got, tc.want)
		}
	}
}

// TestParseWorkTagsIncludeDetailFields 标签必须包含信息表里的
// 社团名/作者/剧情/插画/声优/分类，且多值字段要按空白与 "/" 切开，
// 不能把「双声道立体声/人头麦」「みたかりん 野上菜月」整段塞成一个标签。
func TestParseWorkTagsIncludeDetailFields(t *testing.T) {
	service := &MetaService{}
	work, err := service.parseWork(loadFixture(t, "work_new_zh_tags_RJ01721474.html"), "RJ01721474")
	if err != nil {
		t.Fatalf("解析标签 fixture 失败: %v", err)
	}

	want := []string{
		"青春×フェティシズム", // 社团名
		"遊助Ｄ",        // 作者
		"雛鳥めっせ",      // 剧情
		"そらなにいろ",     // 插画
		"みたかりん",      // 声优（空格切分）
		"野上菜月",
		"音声・ASMR", // 作品形式（按元素取）
		"有配音",     // 作品形式里的 span
		"有音乐",
		"ASMR",   // 分类链接
		"双声道立体声", // 分类里的 "/" 切分
		"人头麦",
		"内射",
		"中出",
		"R18", // 年龄指定
	}
	has := func(want string) bool {
		for _, tag := range work.Tags {
			if tag == want {
				return true
			}
		}
		return false
	}
	for _, tag := range want {
		if !has(tag) {
			t.Errorf("标签缺少 %q：%v", tag, work.Tags)
		}
	}
	for _, bad := range []string{"双声道立体声/人头麦", "みたかりん 野上菜月", "有配音有音乐"} {
		if has(bad) {
			t.Errorf("标签里仍有未切分的拼接值 %q：%v", bad, work.Tags)
		}
	}
}

// TestParseWorkRejectsPageWithoutName 页面结构再变化时宁可报错，也不能返回空壳。
func TestParseWorkRejectsPageWithoutName(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<html><body><p>nothing here</p></body></html>"))
	if err != nil {
		t.Fatalf("构造文档失败: %v", err)
	}
	if _, err := (&MetaService{}).parseWork(doc, "RJ00000000"); err == nil {
		t.Fatal("没有作品名的页面应当报错")
	}
}
