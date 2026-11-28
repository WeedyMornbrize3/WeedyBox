// internal/database/stats_repo.go
//
// 游玩数据分析的聚合查询层（2026-11-28 用户要求「数据分析需求 … 时间运行分析 /
// 名次统计 / 单日运行时长 / 处理数据服务」）。
//
// 为什么聚合放在 database 层而不是 service 层：
// 这里全部是「同一批表、换个 GROUP BY」的 SQL，没有业务规则。放这里的好处是
// 时间区间、来源过滤、空库回落这些口径只写一次，service 层只管组装和切小时。
// 真正的算法（跨小时切分、区间解析）在 internal/service/stats。
//
// 两张表的分工（重要，别看串）：
//   - work_play_time    秒数的**权威**来源，老数据全在这里；按 day 文本聚合。
//   - work_play_sessions 区间来源（一次启动到退出一行），2026-11-28 起才有数据；
//     次数 / 时段 / 最长一次只能用它。
// 所以同一份报表里 Seconds 与 Sessions 可能对不上（老日期的 Sessions 是 0），
// 这是设计如此，不是 bug —— service 层会用 Overview.SessionsSince 告诉前端断层在哪。
package database

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"WeedyBox/internal/model"
)

// statsQueryTimeout 给聚合查询的上限：报表是整表扫描，但库很小（几千行量级），
// 3 秒足够；真卡住说明库损坏，早报错比挂着好。
const statsQueryTimeout = 5 * time.Second

// StatsRange 是一次分析请求的时间与来源条件。
//
// From / To 都是**本地日期**（YYYY-MM-DD），闭区间（两边都含）：
// 用户看到的是「11-24 到 11-30」这样的日期，用日期比较最不容易出歧义。
// 空 From / To = 不限那一侧。
type StatsRange struct {
	From   string
	To     string
	Source string // 空 = 所有来源
}

// dayFilter 生成 work_play_time 的 WHERE 子句与参数（day 是 YYYY-MM-DD 文本，可直接比较）。
func (r StatsRange) dayFilter(alias string) (string, []interface{}) {
	clauses := make([]string, 0, 3)
	args := make([]interface{}, 0, 3)
	if day := strings.TrimSpace(r.From); day != "" {
		clauses = append(clauses, alias+"day >= ?")
		args = append(args, day)
	}
	if day := strings.TrimSpace(r.To); day != "" {
		clauses = append(clauses, alias+"day <= ?")
		args = append(args, day)
	}
	if src := strings.ToLower(strings.TrimSpace(r.Source)); src != "" {
		clauses = append(clauses, alias+"source = ?")
		args = append(args, src)
	}
	return joinWhere(clauses), args
}

// sessionFilter 生成 work_play_sessions 的 WHERE 子句与参数。
//
// 区间用的是**起点的日期**（started_at 的前 10 个字符）：
// 一段从 23:40 玩到次日 00:20 的会话算在开始那天，和大多数人"这算哪天玩的"直觉一致；
// 而日桶那边是被切成两天的。两者口径不同但各自自洽，报表里不混着看就行。
func (r StatsRange) sessionFilter() (string, []interface{}) {
	clauses := make([]string, 0, 3)
	args := make([]interface{}, 0, 3)
	if day := strings.TrimSpace(r.From); day != "" {
		clauses = append(clauses, "substr(started_at, 1, 10) >= ?")
		args = append(args, day)
	}
	if day := strings.TrimSpace(r.To); day != "" {
		clauses = append(clauses, "substr(started_at, 1, 10) <= ?")
		args = append(args, day)
	}
	if src := strings.ToLower(strings.TrimSpace(r.Source)); src != "" {
		clauses = append(clauses, "source = ?")
		args = append(args, src)
	}
	return joinWhere(clauses), args
}

// joinWhere 把子句拼成 " WHERE a AND b"（没有子句时空串）。
func joinWhere(clauses []string) string {
	if len(clauses) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(clauses, " AND ")
}

// ===== 单日运行时长（按天总量）=====

// ListPlayDayTotals 返回区间内每一天的总时长 + 会话数（日期升序）。
//
// 只返回**有数据**的日期：前端要连续日历（补 0）自己补，后端不编造空行 ——
// 报表页的日历热力图需要"哪天没玩"这个信息，那由前端按区间铺满即可。
func (s *SQLiteService) ListPlayDayTotals(ctx context.Context, r StatsRange) ([]model.PlayDayTotal, error) {
	if s == nil || s.DB == nil {
		return []model.PlayDayTotal{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, statsQueryTimeout)
	defer cancel()
	if err := s.ensureLibraryTables(ctx); err != nil {
		return nil, err
	}

	dayWhere, dayArgs := r.dayFilter("")
	rows, err := s.Query(ctx,
		`SELECT day, SUM(seconds) AS seconds, COUNT(DISTINCT work_id) AS works
		 FROM work_play_time`+dayWhere+`
		 GROUP BY day ORDER BY day ASC`, dayArgs...)
	if err != nil {
		return nil, fmt.Errorf("统计每日游玩时长失败: %w", err)
	}

	out := make([]model.PlayDayTotal, 0, len(rows))
	for _, row := range rows {
		out = append(out, model.PlayDayTotal{
			Day:     textOf(row["day"]),
			Seconds: intOf(row["seconds"]),
		})
	}
	if len(out) == 0 {
		return out, nil
	}

	// 会话数单独查一遍：它来自另一张表，直接 JOIN 会把"只有日桶的老日期"整个丢掉。
	sessWhere, sessArgs := r.sessionFilter()
	sessRows, err := s.Query(ctx,
		`SELECT substr(started_at, 1, 10) AS day, COUNT(*) AS sessions
		 FROM work_play_sessions`+sessWhere+`
		 GROUP BY day`, sessArgs...)
	if err != nil {
		return nil, fmt.Errorf("统计每日游玩次数失败: %w", err)
	}
	sessions := make(map[string]int64, len(sessRows))
	for _, row := range sessRows {
		sessions[textOf(row["day"])] = intOf(row["sessions"])
	}
	for i := range out {
		out[i].Sessions = sessions[out[i].Day]
	}
	return out, nil
}

// ===== 名次统计（按作品）=====

// ListPlayWorkRanks 返回区间内每个作品的时长与次数，按**秒数降序**（前端取前 N 画名次）。
//
// limit <= 0 表示不限。时长来自日桶，次数来自会话表；两者在 SQL 里没法一次算准
// （不同表、不同粒度），所以分两步查再按「来源|编号」合并 —— 合并结果是并集：
// 只出现在日桶里的老作品也保留（Sessions=0），只是排得靠后。
func (s *SQLiteService) ListPlayWorkRanks(ctx context.Context, r StatsRange, limit int) ([]model.PlayWorkRank, error) {
	if s == nil || s.DB == nil {
		return []model.PlayWorkRank{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, statsQueryTimeout)
	defer cancel()
	if err := s.ensureLibraryTables(ctx); err != nil {
		return nil, err
	}

	dayWhere, dayArgs := r.dayFilter("")
	rows, err := s.Query(ctx,
		`SELECT source, work_id, SUM(seconds) AS seconds
		 FROM work_play_time`+dayWhere+`
		 GROUP BY source, work_id ORDER BY seconds DESC`, dayArgs...)
	if err != nil {
		return nil, fmt.Errorf("统计作品游玩时长失败: %w", err)
	}

	out := make([]model.PlayWorkRank, 0, len(rows))
	index := make(map[string]int, len(rows))
	for _, row := range rows {
		key := sessionKey(textOf(row["source"]), textOf(row["work_id"]))
		index[key] = len(out)
		out = append(out, model.PlayWorkRank{
			Source:  textOf(row["source"]),
			WorkID:  textOf(row["work_id"]),
			Seconds: intOf(row["seconds"]),
		})
	}

	// 会话：次数 + 最后一次退出时间，按同一 key 合并进上一步的结果
	sessWhere, sessArgs := r.sessionFilter()
	sessRows, err := s.Query(ctx,
		`SELECT source, work_id, COUNT(*) AS sessions, MAX(ended_at) AS last_at
		 FROM work_play_sessions`+sessWhere+`
		 GROUP BY source, work_id`, sessArgs...)
	if err != nil {
		return nil, fmt.Errorf("统计作品游玩次数失败: %w", err)
	}
	for _, row := range sessRows {
		source := textOf(row["source"])
		workID := textOf(row["work_id"])
		key := sessionKey(source, workID)
		pos, ok := index[key]
		if !ok {
			// 只有会话没有日桶：理论上不该出现（同一事务里两张表一起写），
			// 但历史上手工改库/半截写入可能留下这种记录，宁可显示出来也别丢。
			pos = len(out)
			index[key] = pos
			out = append(out, model.PlayWorkRank{Source: source, WorkID: workID})
		}
		out[pos].Sessions = intOf(row["sessions"])
		out[pos].LastPlayedAt = formatTimeOf(row["last_at"])
	}

	// 补齐作品名（库里有就显示名字，被移出游戏库的回落成编号）
	if err := s.fillWorkNames(ctx, out); err != nil {
		return nil, err
	}

	// 合并后顺序可能被"只有会话"的新行打乱，这里重新按秒数排一次（稳定：同秒比名字）
	sortRanks(out)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// fillWorkNames 给排名行补上作品名（一条 SQL 拿全表，不逐条查）。
//
// 查全表而不是 IN (...)：作品名是展示用的，库规模是"一个人玩过的游戏"，
// 一次全取比拼可变长度的 IN 更简单，也避免 SQL 参数上限的问题。
func (s *SQLiteService) fillWorkNames(ctx context.Context, ranks []model.PlayWorkRank) error {
	if len(ranks) == 0 {
		return nil
	}
	rows, err := s.Query(ctx, `SELECT source, work_id, name FROM library_works`)
	if err != nil {
		return fmt.Errorf("读取作品名称失败: %w", err)
	}
	names := make(map[string]string, len(rows))
	for _, row := range rows {
		names[sessionKey(textOf(row["source"]), textOf(row["work_id"]))] = textOf(row["name"])
	}
	for i := range ranks {
		if name := names[sessionKey(ranks[i].Source, ranks[i].WorkID)]; name != "" {
			ranks[i].Name = name
		}
		if ranks[i].Name == "" {
			// 作品已经被移出游戏库：仍然有游玩记录，用编号兜底而不是留空白
			ranks[i].Name = ranks[i].WorkID
		}
	}
	return nil
}

// sortRanks 按时长降序、同时长按名字升序（结果稳定，两次请求顺序一致）。
func sortRanks(ranks []model.PlayWorkRank) {
	sort.SliceStable(ranks, func(i, j int) bool {
		if ranks[i].Seconds != ranks[j].Seconds {
			return ranks[i].Seconds > ranks[j].Seconds
		}
		return ranks[i].Name < ranks[j].Name
	})
}

// sessionKey 归一化成「来源|编号」，与 game 服务的 playSessionKey 同一口径。
// 各处拼 key 都走这里，免得大小写/空格写法不一致导致同样的作品被算成两条。
func sessionKey(source, workID string) string {
	return normalizeSource(source) + "|" + normalizeWorkID(workID)
}

// formatTimeOf 把行里的时间转成 RFC3339 本地时区字符串；零值写空串
// （前端看到空串就显示「—」，比 0001-01-01 好判断）。
func formatTimeOf(v interface{}) string {
	return formatTimeStamp(timeOf(v))
}

// ===== 时段分布（按小时）=====

// PlaySessionSpan 是一段会话的起止（时段分布要按真实重叠小时切分）。
type PlaySessionSpan struct {
	Source    string
	WorkID    string
	StartedAt time.Time
	EndedAt   time.Time
	Seconds   int64
}

// ListPlaySessionSpans 返回区间内所有会话的起止时间（按起点升序）。
//
// 为什么不直接在 SQL 里 GROUP BY strftime('%H', started_at)：
// 那样只会把整段时长记到"开始的那个小时"，21:50 玩到 22:40 的会话会全算给 21 点。
// 按小时切分需要同时看两个时间点，交给 Go 做更清楚（见 stats 服务的 splitPlayHours）。
// 代价是把区间内的会话行全取回来 —— 一个人玩游戏的会话量是几百到几千行，可接受。
func (s *SQLiteService) ListPlaySessionSpans(ctx context.Context, r StatsRange) ([]PlaySessionSpan, error) {
	if s == nil || s.DB == nil {
		return []PlaySessionSpan{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, statsQueryTimeout)
	defer cancel()
	if err := s.ensureLibraryTables(ctx); err != nil {
		return nil, err
	}

	where, args := r.sessionFilter()
	rows, err := s.Query(ctx,
		`SELECT source, work_id, started_at, ended_at, seconds
		 FROM work_play_sessions`+where+`
		 ORDER BY started_at ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("读取游玩区间失败: %w", err)
	}
	out := make([]PlaySessionSpan, 0, len(rows))
	for _, row := range rows {
		start := timeOf(row["started_at"])
		end := timeOf(row["ended_at"])
		if start.IsZero() || end.IsZero() || !end.After(start) {
			continue
		}
		out = append(out, PlaySessionSpan{
			Source:    textOf(row["source"]),
			WorkID:    textOf(row["work_id"]),
			StartedAt: start,
			EndedAt:   end,
			Seconds:   intOf(row["seconds"]),
		})
	}
	return out, nil
}

// ===== 总览 =====

// GetPlayOverview 汇总区间内的总时长 / 会话数 / 活跃天数 / 作品数 / 最长一次。
//
// 前三项来自日桶与会话表各自的聚合，最后两项（最长一次、首末时间）只能来自会话表。
func (s *SQLiteService) GetPlayOverview(ctx context.Context, r StatsRange) (model.PlayOverview, error) {
	var out model.PlayOverview
	if s == nil || s.DB == nil {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, statsQueryTimeout)
	defer cancel()
	if err := s.ensureLibraryTables(ctx); err != nil {
		return out, err
	}

	dayWhere, dayArgs := r.dayFilter("")
	dayRows, err := s.Query(ctx,
		`SELECT COALESCE(SUM(seconds), 0) AS seconds,
		        COUNT(DISTINCT day) AS days,
		        COUNT(DISTINCT source || '|' || work_id) AS works
		 FROM work_play_time`+dayWhere, dayArgs...)
	if err != nil {
		return out, fmt.Errorf("统计总时长失败: %w", err)
	}
	if len(dayRows) > 0 {
		out.TotalSeconds = intOf(dayRows[0]["seconds"])
		out.ActiveDays = intOf(dayRows[0]["days"])
		out.WorkCount = intOf(dayRows[0]["works"])
	}

	sessWhere, sessArgs := r.sessionFilter()
	sessRows, err := s.Query(ctx,
		`SELECT COUNT(*) AS sessions,
		        COALESCE(MAX(seconds), 0) AS longest,
		        MIN(started_at) AS first_at,
		        MAX(ended_at) AS last_at
		 FROM work_play_sessions`+sessWhere, sessArgs...)
	if err != nil {
		return out, fmt.Errorf("统计游玩次数失败: %w", err)
	}
	if len(sessRows) > 0 {
		out.SessionCount = intOf(sessRows[0]["sessions"])
		out.LongestSessionSeconds = intOf(sessRows[0]["longest"])
		out.FirstPlayedAt = formatTimeOf(sessRows[0]["first_at"])
		out.LastPlayedAt = formatTimeOf(sessRows[0]["last_at"])
	}

	// 不受区间限制的会话表元信息：前端据此提示"哪些日子只有时长没有次数"
	allRows, err := s.Query(ctx,
		`SELECT COUNT(*) AS sessions, MIN(started_at) AS first_at FROM work_play_sessions`)
	if err != nil {
		return out, fmt.Errorf("读取游玩区间总览失败: %w", err)
	}
	if len(allRows) > 0 {
		out.SessionsTotal = intOf(allRows[0]["sessions"])
		out.SessionsSince = formatTimeOf(allRows[0]["first_at"])
	}
	return out, nil
}
