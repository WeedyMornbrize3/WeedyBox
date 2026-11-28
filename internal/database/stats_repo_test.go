// internal/database/stats_repo_test.go
//
// 分析聚合查询的约定（2026-11-28 新增）：
//   - 日桶是秒数的权威来源，会话表只补充"次数/区间"；
//   - 老数据（只有日桶、没有会话）必须照常出现在时长里，Sessions=0；
//   - 作品被移出游戏库后，它的日桶仍然显示（名字回落成编号），不能凭空消失；
//   - 区间过滤按本地日期文本比较，两头都是闭区间。
package database

import (
	"context"
	"testing"
	"time"

	"WeedyBox/internal/model"
)

// seedSession 写一段会话（同时按会话的起止写日桶，模拟真实结算路径）。
func seedSession(t *testing.T, s *SQLiteService, source, workID string, start, end time.Time) {
	t.Helper()
	days := []model.PlayTimeDay{
		{Day: start.Format("2006-01-02"), Seconds: int64(end.Sub(start).Seconds())},
	}
	if err := s.RecordGameSessionSpan(context.Background(), source, workID, start, end,
		int64(end.Sub(start).Seconds()), days); err != nil {
		t.Fatalf("写入会话失败: %v", err)
	}
}

func TestStats_DayTotals(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if err := s.UpsertLibraryWork(ctx, ptrWork(sampleWork())); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	day := time.Date(2026, 11, 20, 20, 0, 0, 0, time.Local)
	seedSession(t, s, SourceDLsite, "RJ01721474", day, day.Add(30*time.Minute))
	seedSession(t, s, SourceDLsite, "RJ01721474", day.Add(2*time.Hour), day.Add(2*time.Hour).Add(15*time.Minute))

	totals, err := s.ListPlayDayTotals(ctx, StatsRange{From: "2026-11-20", To: "2026-11-20"})
	if err != nil {
		t.Fatalf("查询每日时长失败: %v", err)
	}
	if len(totals) != 1 {
		t.Fatalf("应只有一天: %+v", totals)
	}
	if totals[0].Seconds != 45*60 {
		t.Errorf("当天总时长应为 2700 秒，实际 %d", totals[0].Seconds)
	}
	if totals[0].Sessions != 2 {
		t.Errorf("当天应有 2 次会话，实际 %d", totals[0].Sessions)
	}

	// 区间外：一天都查不到
	out, err := s.ListPlayDayTotals(ctx, StatsRange{From: "2026-11-21", To: "2026-11-30"})
	if err != nil {
		t.Fatalf("查询每日时长失败: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("区间外不该有数据: %+v", out)
	}
}

// 老数据：只有日桶、没有会话行 → 时长照常统计，次数为 0。
func TestStats_LegacyDayBucketsStillCount(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if err := s.UpsertLibraryWork(ctx, ptrWork(sampleWork())); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	// 不带 startedAt 的旧口径结算（只写账本）
	if err := s.RecordGameSession(ctx, SourceDLsite, "RJ01721474", time.Now(), 1800, []model.PlayTimeDay{
		{Day: "2026-11-01", Seconds: 1800},
	}); err != nil {
		t.Fatalf("旧口径结算失败: %v", err)
	}

	totals, err := s.ListPlayDayTotals(ctx, StatsRange{})
	if err != nil {
		t.Fatalf("查询每日时长失败: %v", err)
	}
	if len(totals) != 1 || totals[0].Seconds != 1800 || totals[0].Sessions != 0 {
		t.Fatalf("老数据应只有时长没有次数: %+v", totals)
	}

	overview, err := s.GetPlayOverview(ctx, StatsRange{})
	if err != nil {
		t.Fatalf("读取总览失败: %v", err)
	}
	if overview.TotalSeconds != 1800 {
		t.Errorf("总时长应为 1800，实际 %d", overview.TotalSeconds)
	}
	if overview.SessionCount != 0 || overview.SessionsTotal != 0 {
		t.Errorf("老数据不该有会话记录: %+v", overview)
	}
}

func TestStats_WorkRanks(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	workA := sampleWork()
	workB := sampleWork()
	workB.WorkID = "RJ09999999"
	workB.Name = "第二部作品"
	for _, w := range []model.LibraryWork{workA, workB} {
		if err := s.UpsertLibraryWork(ctx, ptrWork(w)); err != nil {
			t.Fatalf("入库失败: %v", err)
		}
	}

	day := time.Date(2026, 11, 22, 21, 0, 0, 0, time.Local)
	seedSession(t, s, SourceDLsite, "RJ01721474", day, day.Add(2*time.Hour))
	seedSession(t, s, SourceDLsite, "RJ09999999", day.Add(3*time.Hour), day.Add(3*time.Hour).Add(time.Hour))

	ranks, err := s.ListPlayWorkRanks(ctx, StatsRange{}, 10)
	if err != nil {
		t.Fatalf("查询名次失败: %v", err)
	}
	if len(ranks) != 2 {
		t.Fatalf("应有两个作品: %+v", ranks)
	}
	// 排序：RJ01721474 玩了 2 小时 > RJ09999999 1 小时
	if ranks[0].WorkID != "RJ01721474" || ranks[0].Seconds != 7200 {
		t.Errorf("第一名应是 RJ01721474（7200 秒）: %+v", ranks[0])
	}
	if ranks[0].Name == "" {
		t.Error("名次应带上作品名")
	}
	if ranks[0].Sessions != 1 || ranks[0].LastPlayedAt == "" {
		t.Errorf("名次应带次数与最后一次时间: %+v", ranks[0])
	}

	// limit 生效
	top1, err := s.ListPlayWorkRanks(ctx, StatsRange{}, 1)
	if err != nil {
		t.Fatalf("查询名次失败: %v", err)
	}
	if len(top1) != 1 {
		t.Fatalf("limit=1 应只返回一条: %+v", top1)
	}
}

// 作品被移出游戏库：日桶还在（独立表），名字回落成编号。
func TestStats_RanksKeepOrphanWorks(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	work := sampleWork()
	if err := s.UpsertLibraryWork(ctx, ptrWork(work)); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	day := time.Date(2026, 11, 23, 19, 0, 0, 0, time.Local)
	seedSession(t, s, SourceDLsite, "RJ01721474", day, day.Add(time.Hour))

	if err := s.DeleteLibraryWork(ctx, SourceDLsite, "RJ01721474"); err != nil {
		t.Fatalf("删除作品失败: %v", err)
	}

	ranks, err := s.ListPlayWorkRanks(ctx, StatsRange{}, 10)
	if err != nil {
		t.Fatalf("查询名次失败: %v", err)
	}
	if len(ranks) != 1 {
		t.Fatalf("删掉的作品仍应保留游玩记录: %+v", ranks)
	}
	if ranks[0].Name != "RJ01721474" {
		t.Errorf("没名字时应回落成编号，实际 %q", ranks[0].Name)
	}
}

func TestStats_SessionSpans(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	if err := s.UpsertLibraryWork(ctx, ptrWork(sampleWork())); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	start := time.Date(2026, 11, 24, 21, 50, 0, 0, time.Local)
	seedSession(t, s, SourceDLsite, "RJ01721474", start, start.Add(50*time.Minute))

	spans, err := s.ListPlaySessionSpans(ctx, StatsRange{})
	if err != nil {
		t.Fatalf("读取区间失败: %v", err)
	}
	if len(spans) != 1 {
		t.Fatalf("应有一段会话: %+v", spans)
	}
	if !spans[0].StartedAt.Equal(start) {
		t.Errorf("起点不对: %+v", spans[0])
	}
	if got := int(spans[0].EndedAt.Sub(spans[0].StartedAt).Minutes()); got != 50 {
		t.Errorf("时长应为 50 分钟，实际 %d", got)
	}
}

// 来源过滤：只看 steam 时不该出现 dlsite 的记录。
func TestStats_SourceFilter(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	dlsite := sampleWork()
	steam := sampleWork()
	steam.Source = SourceSteam
	steam.WorkID = "4122860"
	for _, w := range []model.LibraryWork{dlsite, steam} {
		if err := s.UpsertLibraryWork(ctx, ptrWork(w)); err != nil {
			t.Fatalf("入库失败: %v", err)
		}
	}

	day := time.Date(2026, 11, 25, 20, 0, 0, 0, time.Local)
	seedSession(t, s, SourceDLsite, "RJ01721474", day, day.Add(time.Hour))
	seedSession(t, s, SourceSteam, "4122860", day.Add(2*time.Hour), day.Add(2*time.Hour).Add(2*time.Hour))

	all, err := s.ListPlayWorkRanks(ctx, StatsRange{}, 10)
	if err != nil {
		t.Fatalf("查询名次失败: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("不过滤应有两个作品: %+v", all)
	}

	steamOnly, err := s.ListPlayWorkRanks(ctx, StatsRange{Source: SourceSteam}, 10)
	if err != nil {
		t.Fatalf("查询名次失败: %v", err)
	}
	if len(steamOnly) != 1 || steamOnly[0].Source != SourceSteam {
		t.Fatalf("来源过滤没生效: %+v", steamOnly)
	}
}

// ptrWork 取地址（测试里写 UpsertLibraryWork(ctx, &work) 需要它）。
func ptrWork(work model.LibraryWork) *model.LibraryWork {
	return &work
}
