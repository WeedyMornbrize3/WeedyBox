// internal/service/stats/stats_e2e_test.go
//
// 端到端一致性：走「真实结算路径」（RecordGameSessionSpan）写出来的数据，
// 在四个聚合查询里口径必须自洽。
//
// 重点守的是这条账：**日桶秒数之和 == 会话秒数之和**。
// 2026-11-28 用 sqlite3 手工造测试数据时踩到过不一致（只写了会话、没写日桶 →
// 总时长显示 3 分、最长一次却显示 1 小时 15 分）。真实路径不会这样，
// 但这条断言能保证以后改写入逻辑时不会再次分裂。
package stats

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
	"github.com/wailsapp/wails/v3/pkg/services/sqlite"
)

func TestE2E_SessionMatchesDayBuckets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "e2e.db")
	raw := sqlite.NewWithConfig(&sqlite.Config{DBSource: path})
	if err := raw.Open(); err != nil {
		t.Fatalf("打开库失败: %v", err)
	}
	defer func() { _ = raw.Close() }()
	db := &database.SQLiteService{DB: raw}
	ctx := context.Background()

	work := model.LibraryWork{Source: database.SourceSteam, WorkID: "4122860", Name: "测试游戏", RawJSON: "{}"}
	if err := db.UpsertLibraryWork(ctx, &work); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	// 三段真实会话，含跨小时与跨午夜
	spans := []struct{ start, end time.Time }{
		{time.Date(2026, 11, 27, 21, 50, 0, 0, time.Local), time.Date(2026, 11, 27, 22, 40, 0, 0, time.Local)},
		{time.Date(2026, 11, 26, 14, 5, 0, 0, time.Local), time.Date(2026, 11, 26, 15, 20, 0, 0, time.Local)},
		{time.Date(2026, 11, 25, 23, 40, 0, 0, time.Local), time.Date(2026, 11, 26, 0, 20, 0, 0, time.Local)},
	}
	var wantSeconds int64
	for _, s := range spans {
		days := splitDayBuckets(s.start, s.end)
		sec := int64(s.end.Sub(s.start).Seconds())
		wantSeconds += sec
		if err := db.RecordGameSessionSpan(ctx, database.SourceSteam, "4122860", s.start, s.end, sec, days); err != nil {
			t.Fatalf("结算失败: %v", err)
		}
	}

	// 1) 日桶之和 == 会话之和
	days, err := db.ListPlayDayTotals(ctx, database.StatsRange{})
	if err != nil {
		t.Fatalf("读日桶失败: %v", err)
	}
	var daySum int64
	for _, d := range days {
		daySum += d.Seconds
	}
	if daySum != wantSeconds {
		t.Errorf("日桶之和 %d 应等于会话之和 %d", daySum, wantSeconds)
	}

	// 2) 总览：总时长与次数
	ov, err := db.GetPlayOverview(ctx, database.StatsRange{})
	if err != nil {
		t.Fatalf("读总览失败: %v", err)
	}
	if ov.TotalSeconds != wantSeconds {
		t.Errorf("总时长 %d 应等于 %d", ov.TotalSeconds, wantSeconds)
	}
	if ov.SessionCount != 3 {
		t.Errorf("应有 3 次会话，实际 %d", ov.SessionCount)
	}
	if ov.LongestSessionSeconds != 4500 {
		t.Errorf("最长一次应为 4500 秒（14:05-15:20），实际 %d", ov.LongestSessionSeconds)
	}

	// 3) 时段分桶之和 == 会话之和（跨小时切分不丢秒）
	spansRaw, err := db.ListPlaySessionSpans(ctx, database.StatsRange{})
	if err != nil {
		t.Fatalf("读区间失败: %v", err)
	}
	buckets := accumulatePlayHours(makeEmptyHourBuckets(), spansRaw)
	var hourSum int64
	for _, b := range buckets {
		hourSum += b.Seconds
	}
	if hourSum != wantSeconds {
		t.Errorf("时段分桶之和 %d 应等于会话之和 %d", hourSum, wantSeconds)
	}
	if buckets[21].Seconds != 600 || buckets[22].Seconds != 2400 {
		t.Errorf("21:50-22:40 应切成 21=600 / 22=2400，实际 21=%d 22=%d", buckets[21].Seconds, buckets[22].Seconds)
	}
	if buckets[23].Seconds != 1200 || buckets[0].Seconds != 1200 {
		t.Errorf("跨午夜应切成 23=1200 / 0=1200，实际 23=%d 0=%d", buckets[23].Seconds, buckets[0].Seconds)
	}

	// 4) 名次：单作品 3 次
	ranks, err := db.ListPlayWorkRanks(ctx, database.StatsRange{}, 10)
	if err != nil {
		t.Fatalf("读名次失败: %v", err)
	}
	if len(ranks) != 1 || ranks[0].Sessions != 3 || ranks[0].Seconds != wantSeconds {
		t.Fatalf("名次不对: %+v", ranks)
	}
	if ranks[0].Name != "测试游戏" {
		t.Errorf("作品名应补齐，实际 %q", ranks[0].Name)
	}
}

// splitDayBuckets 复刻 game 服务的 splitPlayDays 规则（按本地日历日切分），
// 用于本测试独立验证"日桶之和 == 会话之和"。
func splitDayBuckets(start, end time.Time) []model.PlayTimeDay {
	if !end.After(start) {
		return nil
	}
	var out []model.PlayTimeDay
	cursor := start
	for cursor.Before(end) {
		midnight := time.Date(cursor.Year(), cursor.Month(), cursor.Day(), 0, 0, 0, 0, cursor.Location()).AddDate(0, 0, 1)
		segEnd := end
		if midnight.Before(end) {
			segEnd = midnight
		}
		if sec := int64(segEnd.Sub(cursor).Seconds()); sec > 0 {
			out = append(out, model.PlayTimeDay{Day: cursor.Format("2006-01-02"), Seconds: sec})
		}
		if !segEnd.After(cursor) {
			break
		}
		cursor = segEnd
	}
	return out
}
