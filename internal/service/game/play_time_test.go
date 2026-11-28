// internal/service/game/play_time_test.go
//
// 游玩时长只测"不真的启动游戏"就能覆盖的规则：
//   - 跨午夜按本地日历日切成两个日桶；
//   - 启动写 quit=启动时间、退出结算总时长；
//   - 应用重启后（库里的 quit 仍等于 lastLaunchAt）能恢复计时。
package game

import (
	"context"
	"testing"
	"time"

	"WeedyBox/internal/database"
)

// 23:40 玩到次日 00:20：当天 20 分钟、明天 20 分钟（用户要求"超出当日的记到明天"）。
func TestSplitPlayDays_CrossMidnight(t *testing.T) {
	start := time.Date(2026, 11, 26, 23, 40, 0, 0, time.Local)
	end := time.Date(2026, 11, 27, 0, 20, 0, 0, time.Local)

	days := splitPlayDays(start, end)
	if len(days) != 2 {
		t.Fatalf("跨午夜应切成两个日桶，实际 %+v", days)
	}
	if days[0].Day != "2026-11-26" || days[0].Seconds != 20*60 {
		t.Errorf("当日部分不对: %+v", days[0])
	}
	if days[1].Day != "2026-11-27" || days[1].Seconds != 20*60 {
		t.Errorf("次日部分不对: %+v", days[1])
	}
}

// 同一天内：只有一个桶；起止相同/倒置：没有桶。
func TestSplitPlayDays_SameDayAndInvalid(t *testing.T) {
	start := time.Date(2026, 11, 27, 10, 0, 0, 0, time.Local)
	end := start.Add(90 * time.Minute)

	days := splitPlayDays(start, end)
	if len(days) != 1 || days[0].Day != "2026-11-27" || days[0].Seconds != 90*60 {
		t.Fatalf("同一天应只有一个桶: %+v", days)
	}
	if got := splitPlayDays(start, start); len(got) != 0 {
		t.Fatalf("零长度会话不该有日桶: %+v", got)
	}
	if got := splitPlayDays(end, start); len(got) != 0 {
		t.Fatalf("倒置的时间不该有日桶: %+v", got)
	}
}

// 启动 → 退出：总时长与日桶写进库，quit 变成退出时间，lastLaunchAt 保留启动时间。
func TestPlaySession_LaunchThenSettle(t *testing.T) {
	s := newTestGameService(t)
	seedLibraryWork(t, s, "RJ01234567")
	ctx := context.Background()
	src := database.SourceDLsite

	start := time.Date(2026, 11, 26, 23, 40, 0, 0, time.Local)
	quit := time.Date(2026, 11, 27, 0, 20, 0, 0, time.Local)

	// path 传空：不起 watcher（测试里不碰真实进程），手动结算
	s.beginPlaySession(src, "rj01234567", start, "", 0)

	stats, err := s.DB.GetGamePlayStats(ctx, src, "RJ01234567")
	if err != nil {
		t.Fatalf("读取统计失败: %v", err)
	}
	if stats.QuitGameTimeStamp.Unix() != start.Unix() || stats.LastLaunchAt.Unix() != start.Unix() {
		t.Fatalf("启动时应把 quit 与 lastLaunchAt 都写成启动时间: %+v", stats)
	}

	sess, ok := s.takePlaySession(playSessionKey(src, "RJ01234567"))
	if !ok {
		t.Fatal("beginPlaySession 后内存里应有会话")
	}
	s.settlePlaySession(sess, quit)

	stats, _ = s.DB.GetGamePlayStats(ctx, src, "RJ01234567")
	if stats.PlayTimeTotalSeconds != 40*60 {
		t.Fatalf("总时长应为 2400 秒，实际 %d", stats.PlayTimeTotalSeconds)
	}
	if stats.QuitGameTimeStamp.Unix() != quit.Unix() {
		t.Fatalf("退出后 quit 应是退出时间: %+v", stats)
	}
	if stats.LastLaunchAt.Unix() != start.Unix() {
		t.Fatalf("lastLaunchAt 应保留启动时间: %+v", stats)
	}

	days, err := s.DB.ListPlayTimeDays(ctx, src, "RJ01234567")
	if err != nil {
		t.Fatalf("读取日桶失败: %v", err)
	}
	if len(days) != 2 || days[0].Day != "2026-11-26" || days[1].Day != "2026-11-27" {
		t.Fatalf("跨午夜应记两个日桶: %+v", days)
	}
	if days[0].Seconds != 20*60 || days[1].Seconds != 20*60 {
		t.Fatalf("两个日桶各 20 分钟: %+v", days)
	}
}

// 应用重启时游戏还在跑：库里 quit == lastLaunchAt，recoverPlaySession 用这个时间接着计时。
func TestRecoverPlaySession(t *testing.T) {
	s := newTestGameService(t)
	seedLibraryWork(t, s, "RJ01234567")
	ctx := context.Background()
	src := database.SourceDLsite

	start := time.Now().Add(-30 * time.Minute).Truncate(time.Second)
	if err := s.DB.MarkGameLaunched(ctx, src, "RJ01234567", start); err != nil {
		t.Fatalf("记录启动时间失败: %v", err)
	}
	stats, err := s.DB.GetGamePlayStats(ctx, src, "RJ01234567")
	if err != nil {
		t.Fatalf("读取统计失败: %v", err)
	}

	s.recoverPlaySession(src, "RJ01234567", stats, "", 0)
	sess, ok := s.peekPlaySession(playSessionKey(src, "RJ01234567"))
	if !ok {
		t.Fatal("未结算的会话应被恢复")
	}
	if sess.started.Unix() != start.Unix() {
		t.Fatalf("恢复的起点应是库里的启动时间: %+v", sess)
	}

	// 已正常结算过的记录（quit > lastLaunchAt）不该再被恢复
	s2 := newTestGameService(t)
	seedLibraryWork(t, s2, "RJ01234567")
	if err := s2.DB.MarkGameLaunched(ctx, src, "RJ01234567", start); err != nil {
		t.Fatalf("记录启动时间失败: %v", err)
	}
	if err := s2.DB.RecordGameSession(ctx, src, "RJ01234567", start.Add(10*time.Minute), 600, nil); err != nil {
		t.Fatalf("结算失败: %v", err)
	}
	settled, _ := s2.DB.GetGamePlayStats(ctx, src, "RJ01234567")
	s2.recoverPlaySession(src, "RJ01234567", settled, "", 0)
	if _, ok := s2.peekPlaySession(playSessionKey(src, "RJ01234567")); ok {
		t.Fatal("已结算的会话不该被恢复")
	}
}

// 单次会话封顶 24 小时（挂机过夜不无限累计）。
func TestPlaySession_Capped(t *testing.T) {
	s := newTestGameService(t)
	seedLibraryWork(t, s, "RJ01234567")
	ctx := context.Background()
	src := database.SourceDLsite

	start := time.Now().Add(-48 * time.Hour)
	s.beginPlaySession(src, "RJ01234567", start, "", 0)
	sess, ok := s.takePlaySession(playSessionKey(src, "RJ01234567"))
	if !ok {
		t.Fatal("应有会话")
	}
	s.settlePlaySession(sess, time.Now())

	stats, _ := s.DB.GetGamePlayStats(ctx, src, "RJ01234567")
	if stats.PlayTimeTotalSeconds != int64(playSessionCap/time.Second) {
		t.Fatalf("单次会话应封顶 24 小时，实际 %d 秒", stats.PlayTimeTotalSeconds)
	}
}
