// internal/service/game/play_time.go
//
// 游戏游玩时长的计时与结算（2026-11-27 用户要求）。
//
// 用户原话：
//
//	「model 添加 quitGameTimeStamp，初始值为添加时间，复用 TODO 时间判定逻辑，
//	  根据 quitGameTimeStamp 记录时间判定上次启动时间；并在启动时临时记录 quitGameTimeStamp，
//	  在退出游戏时和新的 quitGameTimeStamp 进行差值运算，记录本次游玩时间；
//	  当天的游玩时间由初始 quitGameTimeStamp 判定，如果新的 quitGameTimeStamp 不再是当天，
//	  则把超出当日的游玩时间记录到明天。注意 quitGameTimeStamp 目前只用来在启动所在卡片
//	  显示上一次启动时间和运行总时长。」
//
// 落地口径（与 database.GamePlayStats 一一对应）：
//   - quit_game_time_stamp：计时起点。启动时写成启动时间（"启动时临时记录"），
//     退出时写成真正的退出时间；两个值相减就是本次游玩时长；
//   - last_launch_at：上一次启动时间（启动时写、退出后保留，动作栏显示它）；
//   - play_time_total_seconds：累计运行总时长；
//   - work_play_time：按天分桶。跨午夜时按**本地日历日**切开，超出当天的部分记到第二天。
//
// 会话怎么判定结束：复用「运行状态」那套进程检测（exe 路径 / PID）。
// 启动后起一个 5 秒一次的 watcher，进程消失了就结算 —— 不依赖前端页面还开着。
// Steam 方式且没记录本地 exe 时检测不到进程，会话留到应用退出或下次启动时结算。
package game

import (
	"context"
	"log"
	"strings"
	"time"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	// playPollInterval 是会话进程检测间隔（与前端状态轮询同频就够用）。
	playPollInterval = 5 * time.Second
	// playStartGrace 是「启动后还没看到进程」的宽限：Steam 拉起游戏、首次解包、
	// 更新补丁都可能要一阵子，宽限内没看到进程不算"已经退出"。
	playStartGrace = 15 * time.Minute
	// playSessionCap 是单次会话计入的上限：游戏挂着过夜不该无限累计。
	playSessionCap = 24 * time.Hour
	// playWriteTimeout 是结算/记录时间戳时给数据库的超时。
	playWriteTimeout = 5 * time.Second
)

// playSession 是一次正在进行的游戏会话的计时状态（只存在内存里）。
//
// 应用正常退出时由 ServiceShutdown 结算；应用被直接杀掉时，库里会留下
// quit_game_time_stamp == last_launch_at 的特征，下次 Status 看到游戏还在跑就按
// 那个时间恢复计时（见 recoverPlaySession）。
type playSession struct {
	key     string
	source  string
	workID  string
	started time.Time // = 启动时写进 quit_game_time_stamp 的值，本次时长的起算点
	path    string    // 用于进程检测的 exe 绝对路径（可能为空）
	pid     uint32    // exe 方式启动时已知的 PID（0 = 靠扫描找）
	seen    bool      // 是否已经观察到进程在跑（Steam 拉起慢，没见到之前不算退出）
}

// playSessionKey 归一化成「来源|编号」，与库里的定位口径一致。
func playSessionKey(source, workID string) string {
	return source + "|" + workID
}

// formatStatusTime 把时间写成给前端的 RFC3339 字符串；零值写空串（"从没启动过"）。
func formatStatusTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ===== 对外入口（由 game_service.go 的 Launch / Status / 关闭钩子调用）=====

// beginPlaySession 开始给一次刚启动的游戏计时：
//   - 把 quit_game_time_stamp 与 last_launch_at 都写成启动时间（用户说的"启动时临时记录"）；
//   - 在内存里记下本次会话；
//   - 有可检测的程序时起 watcher，进程消失就自动结算。
func (s *GameService) beginPlaySession(source, workID string, start time.Time, path string, pid uint32) {
	if s.DB == nil {
		return
	}
	src := sourceOrDefault(source)
	workID = strings.ToUpper(strings.TrimSpace(workID))
	if workID == "" {
		return
	}
	key := playSessionKey(src, workID)

	// 同一作品又启动了一次（上一段还没结算）：先把上一段按"这次启动的时间"结算掉，
	// 避免两段叠在一起；重复点击 / Steam 无进程检测时都可能走到这里。
	if prev, ok := s.takePlaySession(key); ok {
		s.settlePlaySession(prev, start)
	}

	ctx, cancel := context.WithTimeout(context.Background(), playWriteTimeout)
	defer cancel()
	if err := s.DB.MarkGameLaunched(ctx, src, workID, start); err != nil {
		log.Printf("记录游戏启动时间失败: %v", err)
	}

	s.mu.Lock()
	if s.playSessions == nil {
		s.playSessions = make(map[string]*playSession)
	}
	s.playSessions[key] = &playSession{
		key:     key,
		source:  src,
		workID:  workID,
		started: start,
		path:    path,
		pid:     pid,
		// exe 方式下进程是我们亲手拉起来的，直接算"已见到"；
		// Steam 方式要等扫描看到游戏进程再开始算，避免把 Steam 启动失败算成游玩。
		seen: pid != 0,
	}
	s.mu.Unlock()

	if strings.TrimSpace(path) != "" {
		go s.watchPlaySession(key)
	}
}

// recoverPlaySession 处理「应用重启时游戏还在跑」：库里 quit 还停在启动时间
// （与 lastLaunchAt 相同）说明上一次会话没来得及结算，就用那个时间接着计时。
// 只对确实检测到在运行的游戏调用。
func (s *GameService) recoverPlaySession(source, workID string, stats database.GamePlayStats, path string, pid uint32) {
	if s.DB == nil || stats.LastLaunchAt.IsZero() || stats.QuitGameTimeStamp.IsZero() {
		return
	}
	// 正常结算过的记录 quit > lastLaunch（退出时间晚于启动时间）；
	// 完全相等 = 启动后就没改过这一格，即"上次没结算"。
	if !stats.QuitGameTimeStamp.Equal(stats.LastLaunchAt) {
		return
	}
	src := sourceOrDefault(source)
	workID = strings.ToUpper(strings.TrimSpace(workID))
	if workID == "" {
		return
	}
	key := playSessionKey(src, workID)

	s.mu.Lock()
	if s.playSessions == nil {
		s.playSessions = make(map[string]*playSession)
	}
	if _, exists := s.playSessions[key]; exists {
		s.mu.Unlock()
		return
	}
	s.playSessions[key] = &playSession{
		key:     key,
		source:  src,
		workID:  workID,
		started: stats.LastLaunchAt,
		path:    path,
		pid:     pid,
		// 已经确认在运行（调用方刚检测过），可以直接按"见到进程"处理
		seen: true,
	}
	s.mu.Unlock()

	if strings.TrimSpace(path) != "" {
		go s.watchPlaySession(key)
	}
}

// activePlaySeconds 返回当前会话已经跑了多少秒（Status 里把实时进度加到总时长上）。
func (s *GameService) activePlaySeconds(source, workID string) int64 {
	sess, ok := s.peekPlaySession(playSessionKey(sourceOrDefault(source), strings.ToUpper(strings.TrimSpace(workID))))
	if !ok {
		return 0
	}
	elapsed := time.Since(sess.started)
	if elapsed <= 0 {
		return 0
	}
	if elapsed > playSessionCap {
		elapsed = playSessionCap
	}
	return int64(elapsed.Seconds())
}

// ServiceShutdown 让 Wails 在应用退出时结算所有还没结束的会话。
// 不这么做的话，退出前玩的这一段会丢（下次只能看到启动时间）。
func (s *GameService) ServiceShutdown(ctx context.Context, options application.ServiceOptions) error {
	now := time.Now()
	s.mu.Lock()
	keys := make([]string, 0, len(s.playSessions))
	for key := range s.playSessions {
		keys = append(keys, key)
	}
	s.mu.Unlock()

	for _, key := range keys {
		if sess, ok := s.takePlaySession(key); ok {
			s.settlePlaySession(sess, now)
		}
	}
	return nil
}

// ===== 内部实现 =====

// peekPlaySession 返回会话快照（拷贝）：调用方在锁外读字段是安全的。
func (s *GameService) peekPlaySession(key string) (playSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.playSessions[key]
	if !ok {
		return playSession{}, false
	}
	return *sess, true
}

// takePlaySession 把会话从内存里取走（只有取到的那一方能结算它）。
func (s *GameService) takePlaySession(key string) (*playSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.playSessions[key]
	if ok {
		delete(s.playSessions, key)
	}
	return sess, ok
}

// markPlaySessionSeen 标记"已经看到进程在跑"（Steam 拉起慢，见到之前不判退出）。
func (s *GameService) markPlaySessionSeen(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.playSessions[key]; ok {
		sess.seen = true
	}
}

// watchPlaySession 是每次会话一个的进程看门狗：看到进程 → 记下；进程消失 → 结算。
// 没有可检测的程序（Steam 方式且没记本地 exe）直接退出，把结算留给退出/下次启动。
func (s *GameService) watchPlaySession(key string) {
	ticker := time.NewTicker(playPollInterval)
	defer ticker.Stop()

	for now := range ticker.C {
		sess, ok := s.peekPlaySession(key)
		if !ok {
			return
		}

		var running bool
		switch {
		case sess.pid != 0:
			running = processAlive(sess.pid, sess.path)
		case strings.TrimSpace(sess.path) != "":
			running, _ = s.isRunning(sess.path)
		default:
			return
		}

		if running {
			s.markPlaySessionSeen(key)
			continue
		}

		// 重新取一次：期间可能已经被 Status / 另一次启动取走并结算了
		sess, ok = s.peekPlaySession(key)
		if !ok {
			return
		}
		if sess.seen || now.Sub(sess.started) >= playStartGrace {
			s.finishPlaySession(key, now)
			return
		}
	}
}

// finishPlaySession 结束一个会话并按"现在"结算。
func (s *GameService) finishPlaySession(key string, quitAt time.Time) {
	sess, ok := s.takePlaySession(key)
	if !ok {
		return
	}
	s.settlePlaySession(sess, quitAt)
}

// settlePlaySession 把会话写进库：新的退出时间 + 本次秒数 + 按天日桶。
//
// 起止时间都用会话自己记的起点（不重新读库）：应用中途被杀、库里
// quit 被其它路径改写都不会把本次时长算错。
func (s *GameService) settlePlaySession(sess *playSession, quitAt time.Time) {
	if sess == nil || s.DB == nil {
		return
	}
	if quitAt.Before(sess.started) {
		quitAt = sess.started
	}
	// 单次会话封顶：挂着不退出也不该无限累计
	if quitAt.Sub(sess.started) > playSessionCap {
		quitAt = sess.started.Add(playSessionCap)
	}
	seconds := int64(quitAt.Sub(sess.started).Seconds())
	if seconds < 0 {
		seconds = 0
	}
	days := splitPlayDays(sess.started, quitAt)

	ctx, cancel := context.WithTimeout(context.Background(), playWriteTimeout)
	defer cancel()
	// 传 startedAt 是为了让数据库额外落一行 work_play_sessions（数据分析页要区间）：
	// 有起止时间才算得出「玩了几次 / 几点在玩」。日桶与总时长仍按 seconds 记。
	if err := s.DB.RecordGameSessionSpan(ctx, sess.source, sess.workID, sess.started, quitAt, seconds, days); err != nil {
		log.Printf("记录游玩时长失败（%s/%s）: %v", sess.source, sess.workID, err)
	}
}

// splitPlayDays 把一次会话按**本地日历日**切成日桶。//
// 用户要求「如果新的 quitGameTimeStamp 不再是当天，则把超出当日的游玩时间记录到明天」：
// 23:40 玩到次日 00:20 = 当天 20 分钟 + 明天 20 分钟。用日历日而不是"24 小时窗口"，
// 因为"第二天"在界面上就是一个日期，日桶必须能落到具体日期上。
func splitPlayDays(start, end time.Time) []model.PlayTimeDay {
	if !end.After(start) {
		return nil
	}
	days := make([]model.PlayTimeDay, 0, 2)
	cursor := start
	for cursor.Before(end) {
		midnight := time.Date(cursor.Year(), cursor.Month(), cursor.Day(), 0, 0, 0, 0, cursor.Location()).AddDate(0, 0, 1)
		segmentEnd := end
		if midnight.Before(end) {
			segmentEnd = midnight
		}
		seconds := int64(segmentEnd.Sub(cursor).Seconds())
		if seconds > 0 {
			days = append(days, model.PlayTimeDay{Day: cursor.Format("2006-01-02"), Seconds: seconds})
		}
		if !segmentEnd.After(cursor) {
			break
		}
		cursor = segmentEnd
	}
	return days
}
