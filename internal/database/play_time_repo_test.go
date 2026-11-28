// internal/database/play_time_repo_test.go
//
// 游玩统计三列（quit_game_time_stamp / last_launch_at / play_time_total_seconds）
// 与日桶表 work_play_time 的读写约定：
//   - 入库时 quit 初始化为添加时间；
//   - 启动写两格、退出结算总时长与日桶；
//   - 刷新元数据（Upsert）不会把它们冲掉。
package database

import (
	"context"
	"testing"
	"time"

	"WeedyBox/internal/model"
)

func TestPlayStats_InitialQuitIsAddTime(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()

	work := sampleWork()
	if err := s.UpsertLibraryWork(ctx, &work); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	stats, err := s.GetGamePlayStats(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("读取游玩统计失败: %v", err)
	}
	// 用户要求：quitGameTimeStamp 初始值 = 添加时间
	if stats.QuitGameTimeStamp.IsZero() {
		t.Fatal("入库后 quit_game_time_stamp 应有初始值（添加时间）")
	}
	if stats.LastLaunchAt.IsZero() == false || stats.PlayTimeTotalSeconds != 0 {
		t.Fatalf("新入库的作品不该有启动时间/游玩时长: %+v", stats)
	}
}

func TestPlayStats_LaunchThenSettle(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	work := sampleWork()
	if err := s.UpsertLibraryWork(ctx, &work); err != nil {
		t.Fatalf("入库失败: %v", err)
	}

	// 启动：quit 与 lastLaunchAt 都写成启动时间（"启动时临时记录"）
	start := time.Now().Add(-90 * time.Minute)
	if err := s.MarkGameLaunched(ctx, SourceDLsite, "RJ01721474", start); err != nil {
		t.Fatalf("记录启动时间失败: %v", err)
	}
	stats, err := s.GetGamePlayStats(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("读取游玩统计失败: %v", err)
	}
	if stats.LastLaunchAt.Unix() != start.Unix() || stats.QuitGameTimeStamp.Unix() != start.Unix() {
		t.Fatalf("启动后 quit 与 lastLaunchAt 都应是启动时间: %+v", stats)
	}

	// 退出：新的 quit = 退出时间，总时长累加，日桶按天写
	quit := start.Add(90 * time.Minute)
	if err := s.RecordGameSession(ctx, SourceDLsite, "RJ01721474", quit, 5400, []model.PlayTimeDay{
		{Day: start.Format("2006-01-02"), Seconds: 5400},
	}); err != nil {
		t.Fatalf("结算会话失败: %v", err)
	}

	stats, err = s.GetGamePlayStats(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("读取游玩统计失败: %v", err)
	}
	if stats.QuitGameTimeStamp.Unix() != quit.Unix() {
		t.Fatalf("退出后 quit 应是退出时间: %+v", stats)
	}
	if stats.LastLaunchAt.Unix() != start.Unix() {
		t.Fatalf("退出后 lastLaunchAt 应保留启动时间: %+v", stats)
	}
	if stats.PlayTimeTotalSeconds != 5400 {
		t.Fatalf("总时长应为 5400 秒，实际 %d", stats.PlayTimeTotalSeconds)
	}

	days, err := s.ListPlayTimeDays(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("读取日桶失败: %v", err)
	}
	if len(days) != 1 || days[0].Day != start.Format("2006-01-02") || days[0].Seconds != 5400 {
		t.Fatalf("日桶不对: %+v", days)
	}

	// 同一天再玩一段：日桶与总时长都累加，不新增行
	if err := s.RecordGameSession(ctx, SourceDLsite, "RJ01721474", quit.Add(time.Hour), 600, []model.PlayTimeDay{
		{Day: start.Format("2006-01-02"), Seconds: 600},
	}); err != nil {
		t.Fatalf("第二次结算失败: %v", err)
	}
	days, err = s.ListPlayTimeDays(ctx, SourceDLsite, "RJ01721474")
	if err != nil {
		t.Fatalf("读取日桶失败: %v", err)
	}
	if len(days) != 1 || days[0].Seconds != 6000 {
		t.Fatalf("同一天的日桶应累加到 6000 秒: %+v", days)
	}
	stats, _ = s.GetGamePlayStats(ctx, SourceDLsite, "RJ01721474")
	if stats.PlayTimeTotalSeconds != 6000 {
		t.Fatalf("总时长应累加到 6000 秒，实际 %d", stats.PlayTimeTotalSeconds)
	}

	// 刷新元数据（重新抓取/编辑保存都走 Upsert）不能把统计冲掉
	refreshed := sampleWork()
	refreshed.Name = "刷新后的名字"
	if err := s.UpsertLibraryWork(ctx, &refreshed); err != nil {
		t.Fatalf("刷新元数据失败: %v", err)
	}
	stats, _ = s.GetGamePlayStats(ctx, SourceDLsite, "RJ01721474")
	if stats.PlayTimeTotalSeconds != 6000 || stats.LastLaunchAt.IsZero() {
		t.Fatalf("刷新元数据把游玩统计冲掉了: %+v", stats)
	}
}

// RecordGameSession 传入负数/空日桶时不能把总时长减回去，也不能报错。
func TestPlayStats_ToleratesBadInput(t *testing.T) {
	s := newTestService(t)
	ctx := context.Background()
	work := sampleWork()
	if err := s.UpsertLibraryWork(ctx, &work); err != nil {
		t.Fatalf("入库失败: %v", err)
	}
	if err := s.RecordGameSession(ctx, SourceDLsite, "RJ01721474", time.Now(), -100, []model.PlayTimeDay{
		{Day: "", Seconds: 10},
		{Day: "2026-11-27", Seconds: -5},
	}); err != nil {
		t.Fatalf("异常输入不该报错: %v", err)
	}
	stats, _ := s.GetGamePlayStats(ctx, SourceDLsite, "RJ01721474")
	if stats.PlayTimeTotalSeconds != 0 {
		t.Fatalf("负数时长应被夹到 0，实际 %d", stats.PlayTimeTotalSeconds)
	}
}
