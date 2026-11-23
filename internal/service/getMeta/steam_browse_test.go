// internal/service/getMeta/steam_browse_test.go
//
// GetItems / 标签字典 / storesearch 的解析与「文件夹名 → appid」匹配。
// 全部离线：解析用真实返回裁出来的 fixture，匹配用纯函数，不发网络请求。
package getmeta

import (
	"testing"
)

// getItemsFixture 是 GetItems 真实返回裁出来的片段（schinese，含 tags 权重与 reviews）。
const getItemsFixture = `{
  "response": {
    "store_items": [
      {
        "item_type": 0,
        "id": 1229490,
        "success": 1,
        "visible": true,
        "name": "ULTRAKILL",
        "appid": 1229490,
        "tagids": [1734, 4345, 1663, 19],
        "tags": [
          {"tagid": 1734, "weight": 829},
          {"tagid": 4345, "weight": 814},
          {"tagid": 1663, "weight": 783},
          {"tagid": 19, "weight": 362}
        ],
        "reviews": {
          "summary_filtered": {
            "percent_positive": 97,
            "review_count": 218393,
            "review_score": 9,
            "review_score_label": "好评如潮"
          }
        }
      }
    ]
  }
}`

const tagNamesFixture = `{
  "response": {
    "tags": [
      {"tagid": 1734, "name": "动作"},
      {"tagid": 4345, "name": "复古射击"},
      {"tagid": 1663, "name": "第一人称射击"},
      {"tagid": 19, "name": "动作"},
      {"tagid": 492, "name": "独立"}
    ]
  }
}`

const searchFixture = `{
  "total": 2,
  "items": [
    {"type": "app", "name": "Hades II", "id": 1145350,
     "price": {"currency": "CNY", "initial": 10800, "final": 10800},
     "tiny_image": "https://cdn/1145350.jpg"},
    {"type": "app", "name": "Hades", "id": 1145360,
     "price": {"currency": "CNY", "initial": 9200, "final": 9200},
     "tiny_image": "https://cdn/1145360.jpg"}
  ]
}`

// TestParseSteamBrowseItems 锁住 GetItems 的字段口径（标签权重排序 + 评价汇总）。
func TestParseSteamBrowseItems(t *testing.T) {
	items, err := parseSteamBrowseItems([]byte(getItemsFixture))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("store_items 数量 = %d，期望 1", len(items))
	}
	item := items[0]
	if item.AppID != 1229490 || item.Name != "ULTRAKILL" {
		t.Errorf("appid/名字不对: %d / %q", item.AppID, item.Name)
	}
	if len(item.Tags) != 4 || len(item.TagIDs) != 4 {
		t.Errorf("标签数量不对: tags=%d tagids=%d", len(item.Tags), len(item.TagIDs))
	}
	if item.Reviews.SummaryFiltered.ReviewCount != 218393 ||
		item.Reviews.SummaryFiltered.PercentPositive != 97 ||
		item.Reviews.SummaryFiltered.ReviewScoreLabel != "好评如潮" {
		t.Errorf("评价汇总解析不对: %+v", item.Reviews.SummaryFiltered)
	}
}

// TestSteamUserTags 把 tagid 换成中文名、按权重降序、去重。
func TestSteamUserTags(t *testing.T) {
	items, err := parseSteamBrowseItems([]byte(getItemsFixture))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	names, err := parseSteamTagNames([]byte(tagNamesFixture))
	if err != nil {
		t.Fatalf("解析标签字典失败: %v", err)
	}
	tags := steamUserTags(items[0], names)
	// 1734(829)「动作」> 4345(814)「复古射击」> 1663(783)「第一人称射击」> 19(362)「动作」重复被去重
	want := []string{"动作", "复古射击", "第一人称射击"}
	if len(tags) != len(want) {
		t.Fatalf("tags = %v，期望 %v", tags, want)
	}
	for i := range want {
		if tags[i] != want[i] {
			t.Errorf("第 %d 个标签 = %q，期望 %q（按权重降序）", i, tags[i], want[i])
		}
	}
}

// TestMergeSteamTags 玩家标签在前、原有标签在后，去重且截断到 steamTagLimit。
func TestMergeSteamTags(t *testing.T) {
	got := mergeSteamTags([]string{"动作", "射击"}, []string{"动作", "单人", "New Blood"})
	want := []string{"动作", "射击", "单人", "New Blood"}
	if len(got) != len(want) {
		t.Fatalf("合并结果 = %v，期望 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 个 = %q，期望 %q", i, got[i], want[i])
		}
	}

	many := make([]string, 0, steamTagLimit+10)
	for i := 0; i < steamTagLimit+10; i++ {
		many = append(many, string(rune('a'+i%26))+string(rune('0'+i%10)))
	}
	if merged := mergeSteamTags(many, nil); len(merged) > steamTagLimit {
		t.Errorf("合并后标签数 = %d，应截断到 %d", len(merged), steamTagLimit)
	}
}

// TestParseSteamSearchItems 店铺搜索的字段映射（id 转字符串、价格格式化）。
func TestParseSteamSearchItems(t *testing.T) {
	items, err := parseSteamSearchItems([]byte(searchFixture))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("候选数 = %d，期望 2", len(items))
	}
	if items[0].ID != "1145350" || items[0].Name != "Hades II" {
		t.Errorf("候选 0 = %+v", items[0])
	}
	if items[0].Price != "¥108.00" {
		t.Errorf("价格 = %q，期望 ¥108.00", items[0].Price)
	}
	if items[0].Cover == "" || items[0].Type != "app" {
		t.Errorf("封面/类型缺失: %+v", items[0])
	}
}

// TestSteamFolderQuery 清洗文件夹名：去括号注释、去结尾点分版本号、下划线换空格。
func TestSteamFolderQuery(t *testing.T) {
	cases := map[string]string{
		"ULTRAKILL v1.0":    "ULTRAKILL",
		"黑神话悟空":             "黑神话悟空",
		"游戏名 [v1.2.3]":      "游戏名",
		"Stardew_Valley":    "Stardew Valley",
		"Hades 2":           "Hades 2", // 单位数不是版本号，不能削
		"深海迷航（中文版） v2.1":    "深海迷航",
		"[Group] Game_Name": "Game Name",
		"  空白  多的 名字 ":      "空白 多的 名字",
	}
	for input, want := range cases {
		if got := steamFolderQuery(input); got != want {
			t.Errorf("steamFolderQuery(%q) = %q，期望 %q", input, got, want)
		}
	}
}

// TestSteamQueryLadder 长词逐级截短（Steam 搜索按词匹配，整串常常 0 条）。
func TestSteamQueryLadder(t *testing.T) {
	ladder := steamQueryLadder("黑神话悟空")
	if len(ladder) < 2 || ladder[0] != "黑神话悟空" {
		t.Fatalf("ladder = %v，第一项应是原词", ladder)
	}
	if ladder[1] != "黑神话" {
		t.Errorf("第二项 = %q，期望 3 字前缀「黑神话」", ladder[1])
	}

	if short := steamQueryLadder("巫师3"); len(short) != 1 {
		t.Errorf("短词不该截断，实际 %v", short)
	}
}

// TestNormalizeSteamName 只留字母数字：标点/空格/全角符号都不参与比较。
func TestNormalizeSteamName(t *testing.T) {
	cases := map[string]string{
		"黑神话：悟空":    "黑神话悟空",
		"ULTRAKILL": "ultrakill",
		"R.E.P.O.":  "repo",
		"Hades II":  "hadesii",
		"赛博朋克 2077": "赛博朋克2077",
	}
	for input, want := range cases {
		if got := normalizeSteamName(input); got != want {
			t.Errorf("normalizeSteamName(%q) = %q，期望 %q", input, got, want)
		}
	}
}

// TestSteamNameScore 完全一致 2 分、差一两个字母按编辑距离给分、太少不像就给 0。
func TestSteamNameScore(t *testing.T) {
	if score := steamNameScore("hades", "hades"); score != 2 {
		t.Errorf("完全一致 = %v，期望 2", score)
	}
	if score := steamNameScore("黑神话悟空", "黑神话悟空"); score != 2 {
		t.Errorf("中文完全一致 = %v，期望 2（标点已在归一化时去掉）", score)
	}
	// 文件夹「黑神话悟空」vs 商店「黑神话：悟空」→ 归一化后完全一致
	if score := steamNameScore("黑神话悟空", "黑神话悟空"); score != 2 {
		t.Errorf("归一化后一致 = %v，期望 2", score)
	}
	// 差一个字母也应是同一个游戏（用户 2026-11-23 用「demon root」测出来的场景：
	// 文件夹没有 s，商店叫 Demons Roots）。旧的前缀占比法在这里给 0，猜不到 Steam 源。
	if score := steamNameScore("demonroot", "demonsroots"); score < steamMatchThreshold {
		t.Errorf("demon root vs Demons Roots = %v，期望 ≥ %v", score, steamMatchThreshold)
	}
	if score := steamNameScore("hades", "hadesii"); score <= 0 || score >= 2 {
		t.Errorf("差两个字母 = %v，期望 (0,2)，因为 hades 不是 hadesii", score)
	}
	if score := steamNameScore("ab", "abcdef"); score != 0 {
		t.Errorf("前缀太短 = %v，期望 0", score)
	}
	if score := steamNameScore("demonroot", "hades"); score != 0 {
		t.Errorf("完全不像 = %v，期望 0", score)
	}
}

// TestPickSteamCandidate 优先完全一致、其次更相似；原声/试玩这类陪跑要排后面。
func TestPickSteamCandidate(t *testing.T) {
	items := []steamSearchItem{
		{ID: "1145350", Name: "Hades II"},
		{ID: "1145360", Name: "Hades"},
		{ID: "1145370", Name: "Hades Original Soundtrack"},
	}
	best, score := pickSteamCandidate(normalizeSteamName("Hades"), items)
	if best == nil || best.ID != "1145360" {
		t.Fatalf("应选中 Hades(1145360)，实际 %+v（score=%v）", best, score)
	}
	if score != 2 {
		t.Errorf("完全一致得分 = %v，期望 2", score)
	}

	// 文件夹多带了后缀：仍应匹配到正片，而不是原声
	items2 := []steamSearchItem{
		{ID: "1", Name: "黑神话：悟空 游戏音乐精选集"},
		{ID: "2", Name: "黑神话：悟空"},
	}
	best2, _ := pickSteamCandidate(normalizeSteamName("黑神话悟空"), items2)
	if best2 == nil || best2.ID != "2" {
		t.Errorf("应选中正片(2)，实际 %+v", best2)
	}

	// 完全不像就不选
	if best3, _ := pickSteamCandidate(normalizeSteamName("Totally Different Game"), items); best3 != nil {
		t.Errorf("不像的候选不该被选中: %+v", best3)
	}
}

// TestSteamNoiseName 陪跑条目识别。
func TestSteamNoiseName(t *testing.T) {
	for _, name := range []string{"Hades Original Soundtrack", "ULTRAKILL Demo", "游戏 性能测试工具", "Artbook & 画集"} {
		if !steamNoiseName(name) {
			t.Errorf("%q 应判为陪跑条目", name)
		}
	}
	for _, name := range []string{"Hades", "ULTRAKILL", "黑神话：悟空"} {
		if steamNoiseName(name) {
			t.Errorf("%q 不该判为陪跑条目", name)
		}
	}
}
