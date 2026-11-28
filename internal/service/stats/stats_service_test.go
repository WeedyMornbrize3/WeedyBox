// internal/service/stats/stats_service_test.go
//
// 只测纯计算函数（不碰数据库）：这几条都是"算错了界面会明显不对劲"的地方 ——
// 跨小时切分、跨午夜、区间文本解析、连续日历补洞。
package stats

import (
	"testing"
	"time"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
)

func span(start, end time.Time) database.PlaySessionSpan {
	return database.PlaySessionSpan{StartedAt: start, EndedAt: end}
}

// 21:50 → 22:40（50 分钟）：21 点记 10 分钟、22 点记 40 分钟。
// 会话数只算在开始的 21 点。
func TestAccumulatePlayHours_SplitAcrossHours(t *testing.T) {
	start := time.Date(2026, 11, 24, 21, 50, 0, 0, time.Local)
	end := time.Date(2026, 11, 24, 22, 40, 0, 0, time.Local)

	buckets := accumulatePlayHours(makeEmptyHourBuckets(), []database.PlaySessionSpan{span(start, end)})
	if len(buckets) != 24 {
		t.Fatalf("应恒返回 24 个桶，实际 %d", len(buckets))
	}
	if got := buckets[21].Seconds; got != 10*60 {
		t.Errorf("21 点应记 600 秒，实际 %d", got)
	}
	if got := buckets[22].Seconds; got != 40*60 {
		t.Errorf("22 点应记 2400 秒，实际 %d", got)
	}
	if buckets[21].Sessions != 1 || buckets[22].Sessions != 0 {
		t.Errorf("会话数应只记在开始的 21 点: 21=%d 22=%d", buckets[21].Sessions, buckets[22].Sessions)
	}

	// 24 个桶求和必须等于会话总时长
	var sum int64
	for _, b := range buckets {
		sum += b.Seconds
	}
	if sum != int64(end.Sub(start).Seconds()) {
		t.Errorf("分桶求和 %d 应等于会话时长 %d", sum, int64(end.Sub(start).Seconds()))
	}
}

// 跨午夜：23:40 → 00:20（次日）也要正确落到 23 点与 0 点两个桶。
func TestAccumulatePlayHours_CrossMidnight(t *testing.T) {
	start := time.Date(2026, 11, 26, 23, 40, 0, 0, time.Local)
	end := time.Date(2026, 11, 27, 0, 20, 0, 0, time.Local)

	buckets := accumulatePlayHours(makeEmptyHourBuckets(), []database.PlaySessionSpan{span(start, end)})
	if buckets[23].Seconds != 20*60 {
		t.Errorf("23 点应记 1200 秒，实际 %d", buckets[23].Seconds)
	}
	if buckets[0].Seconds != 20*60 {
		t.Errorf("0 点应记 1200 秒，实际 %d", buckets[0].Seconds)
	}
	if buckets[23].Sessions != 1 || buckets[0].Sessions != 0 {
		t.Errorf("跨午夜会话只算 1 次、记在开始的 23 点: 23=%d 0=%d", buckets[23].Sessions, buckets[0].Sessions)
	}
}

// 零长度 / 倒置的区间不该产生任何数据（也不能死循环）。
func TestAccumulatePlayHours_IgnoresBadSpans(t *testing.T) {
	start := time.Date(2026, 11, 24, 21, 0, 0, 0, time.Local)
	buckets := accumulatePlayHours(makeEmptyHourBuckets(), []database.PlaySessionSpan{
		span(start, start),
		span(start.Add(time.Hour), start),
	})
	for _, b := range buckets {
		if b.Seconds != 0 || b.Sessions != 0 {
			t.Fatalf("异常区间不该产生数据: %+v", b)
		}
	}
}

func TestNormalizeDay(t *testing.T) {
	cases := map[string]string{
		"2026-11-27":                   "2026-11-27",
		"  2026-11-27  ":               "2026-11-27",
		"2026-11-27T16:00:00.000Z":     "2026-11-27",
		"2026-11-27 08:30:00":          "2026-11-27",
		"":                             "",
		"乱七八糟":                         "",
		"2026-13-99":                   "",
	}
	for input, want := range cases {
		if got := normalizeDay(input); got != want {
			t.Errorf("normalizeDay(%q) = %q，期望 %q", input, got, want)
		}
	}
}

func TestFillDayRange(t *testing.T) {
	sparse := []model.PlayDayTotal{
		{Day: "2026-11-20", Seconds: 1800},
		{Day: "2026-11-23", Seconds: 600},
	}
	filled := fillDayRange(sparse, "2026-11-20", "2026-11-23")
	if len(filled) != 4 {
		t.Fatalf("应补成 4 天，实际 %d: %+v", len(filled), filled)
	}
	wantDays := []string{"2026-11-20", "2026-11-21", "2026-11-22", "2026-11-23"}
	for i, want := range wantDays {
		if filled[i].Day != want {
			t.Errorf("第 %d 天应是 %s，实际 %s", i, want, filled[i].Day)
		}
	}
	if filled[0].Seconds != 1800 || filled[1].Seconds != 0 || filled[3].Seconds != 600 {
		t.Errorf("补出来的空天应是 0，有数据的天保留原值: %+v", filled)
	}

	// 参数不全会原样返回（没法补）
	if got := fillDayRange(sparse, "", "2026-11-23"); len(got) != len(sparse) {
		t.Errorf("缺 from 时应原样返回: %+v", got)
	}
	// 跨度太大不做补洞
	if got := fillDayRange(sparse, "1970-01-01", "2026-11-23"); len(got) != len(sparse) {
		t.Errorf("跨度超过上限时应原样返回: %+v", got)
	}
}

func TestDescribeRange(t *testing.T) {
	cases := []struct {
		req  RangeRequest
		want string
	}{
		{RangeRequest{}, "全部时间"},
		{RangeRequest{From: "2026-11-01", To: "2026-11-28"}, "2026-11-01 至 2026-11-28"},
		{RangeRequest{From: "2026-11-01"}, "2026-11-01 起"},
		{RangeRequest{To: "2026-11-28"}, "截至 2026-11-28"},
		{RangeRequest{From: "2026-11-28", To: "2026-11-28"}, "2026-11-28"},
	}
	for _, c := range cases {
		if got := DescribeRange(c.req); got != c.want {
			t.Errorf("DescribeRange(%+v) = %q，期望 %q", c.req, got, c.want)
		}
	}
}
