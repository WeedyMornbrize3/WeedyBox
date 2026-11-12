// internal/service/library_service_test.go
//
// 入库前的完整性校验：扫描添加会一次入库几十部，这里锁住"空壳必须被挡住"与
// "rowFromWork 把作品信息完整带进库列"两条，防止以后改解析时悄悄退化。
package service

import "testing"

func TestWorkIsComplete(t *testing.T) {
	cases := []struct {
		name string
		work map[string]any
		want bool
	}{
		{name: "编号+名称+社团", work: map[string]any{"id": "RJ01721474", "name": "作品名", "maker": "社团"}, want: true},
		{name: "只有编号和名称（空壳）", work: map[string]any{"id": "RJ01721474", "name": "作品名"}, want: false},
		{name: "只有简介也算有详情", work: map[string]any{"id": "RJ01721474", "name": "作品名", "summary": "简介"}, want: true},
		{name: "缺名称", work: map[string]any{"id": "RJ01721474"}, want: false},
		{name: "缺编号", work: map[string]any{"name": "作品名"}, want: false},
		{name: "名称只有空白", work: map[string]any{"id": "RJ01721474", "name": "   "}, want: false},
		{name: "编号只有空白", work: map[string]any{"id": "\t", "name": "作品名"}, want: false},
		{name: "空壳", work: map[string]any{}, want: false},
		{name: "nil 值", work: map[string]any{"id": nil, "name": "作品名"}, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := workIsComplete(tc.work); got != tc.want {
				t.Errorf("workIsComplete(%v) = %v，期望 %v", tc.work, got, tc.want)
			}
		})
	}
}

// TestNormalizeUserTag 自定义标签的清洗：去首尾空白、中间连续空白压成一个空格。
func TestNormalizeUserTag(t *testing.T) {
	cases := map[string]string{
		"  想玩  ": "想玩",
		"已\t通关":  "已 通关",
		"a   b":  "a b",
		"":       "",
		"   ":    "",
	}
	for input, want := range cases {
		if got := normalizeUserTag(input); got != want {
			t.Errorf("normalizeUserTag(%q) = %q，期望 %q", input, got, want)
		}
	}
}

// TestRowFromWorkCarriesWorkInfo 入库时写进库的公共列必须来自抓回来的完整元数据，
// 而不是只有编号/名称（作品库详情、游戏库列表都靠这些列）。
func TestRowFromWorkCarriesWorkInfo(t *testing.T) {
	work := map[string]any{
		"id":          "rj01721474",
		"name":        "作品名",
		"maker":       "社团",
		"makerId":     "RG51931",
		"series":      "系列",
		"seriesId":    "SRI0000043774",
		"workType":    "ボイス・ASMR",
		"ageRating":   "R18",
		"releaseDate": "2026-09-26",
		"coverUrl":    "https://img.dlsite.jp/x.jpg",
		"sourceUrl":   "https://www.dlsite.com/maniax/work/=/product_id/RJ01721474.html",
		"fetchedAt":   "2026-11-12T00:00:00Z",
	}

	row := rowFromWork("dlsite", work)

	if row.Source != "dlsite" {
		t.Errorf("Source = %q，期望 dlsite", row.Source)
	}
	// 编号统一转大写，其余字段原样带进库
	if row.WorkID != "RJ01721474" {
		t.Errorf("WorkID = %q，期望 RJ01721474", row.WorkID)
	}
	if row.Name != "作品名" || row.Maker != "社团" || row.MakerID != "RG51931" {
		t.Errorf("名称/社团没带进库: %+v", row)
	}
	if row.Series != "系列" || row.SeriesID != "SRI0000043774" || row.WorkType != "ボイス・ASMR" {
		t.Errorf("系列/形式没带进库: %+v", row)
	}
	if row.AgeRating != "R18" || row.ReleaseDate != "2026-09-26" {
		t.Errorf("年龄/发售日没带进库: %+v", row)
	}
	if row.CoverURL == "" || row.SourceURL == "" || row.FetchedAt == "" {
		t.Errorf("封面/来源/抓取时间没带进库: %+v", row)
	}
	// 完整数据留在 raw_json（作品库详情、重新抓取都靠它）
	if row.RawJSON == "" || row.RawJSON == "{}" {
		t.Errorf("RawJSON 为空，完整元数据丢了")
	}
}
