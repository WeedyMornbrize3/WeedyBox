// internal/model/play_stats.go
//
// 游玩数据分析的聚合结果（2026-11-28 用户要求「数据分析需求 … 时间运行分析 / 名次统计 /
// 单日运行时长」）。
//
// 为什么单独一个文件而不是塞进 library.go：library.go 描述的是**一条库记录**长什么样，
// 这里描述的是**一组库记录聚合之后**长什么样（全站视角、按天/按小时/按作品切）。
// 前者是持久层的行，后者只存在于一次查询的结果里，不进任何表。
//
// 数据来源有两个，口径不同，别混：
//   - work_play_time（日桶，2026-11-27 起就有）：**秒数**的权威来源。
//     跨午夜的会话在这里已经被切成两天，所以按天求和永远等于真实时长；
//   - work_play_sessions（会话，2026-11-28 新增）：**次数 / 时段 / 起止**的来源。
//     一行 = 一次启动到退出。老数据没有这张表的内容，所以"次数"类字段会是 0。
package model

// PlayDayTotal 是「某一天全站一共玩了多久」（work_play_time 按 day 求和）。
//
// Seconds 来自日桶（含老数据），Sessions 只统计会话表里**当天启动**的那些；
// 所以老日期的 Seconds > 0 而 Sessions == 0 是正常现象，不是算错。
type PlayDayTotal struct {
	Day      string `json:"day"`      // 本地日期 YYYY-MM-DD
	Seconds  int64  `json:"seconds"`  // 当天总时长
	Sessions int64  `json:"sessions"` // 当天启动的会话数（老数据为 0）
}

// PlayWorkRank 是名次统计里的一行：一个作品在所选区间内的时长与次数。
//
// Name 取 library_works.name；作品已被移出游戏库时回落成编号（日桶是独立表，
// 不随 library_works 级联删除，所以这种"孤儿日桶"确实存在）。
type PlayWorkRank struct {
	Source   string `json:"source"`
	WorkID   string `json:"workId"`
	Name     string `json:"name"`
	Seconds  int64  `json:"seconds"`  // 区间内总时长（日桶口径）
	Sessions int64  `json:"sessions"` // 区间内启动次数（会话表口径，老数据为 0）
	// LastPlayedAt 是区间内最后一次**退出**时间（RFC3339 本地时区）；没有会话记录时为空。
	LastPlayedAt string `json:"lastPlayedAt"`
}

// PlayHourBucket 是「0-23 点每个小时各玩了多少」。
//
// 这个维度**只能来自会话表**：日桶只知道"某天一共多久"，不知道落在哪个小时。
// 跨小时的会话按真实重叠时长切进对应小时（见 stats 服务的 splitPlayHours），
// 所以 24 个小时的 Seconds 求和一定等于这些会话的时长之和。
type PlayHourBucket struct {
	Hour     int   `json:"hour"`     // 0-23，本地时区
	Seconds  int64 `json:"seconds"`  // 该小时内的累计游玩时长
	Sessions int64 `json:"sessions"` // 在该小时**开始**的会话数
}

// PlayOverview 是分析页顶部那排总览数字。
type PlayOverview struct {
	TotalSeconds          int64 `json:"totalSeconds"`          // 区间内总时长（日桶口径）
	SessionCount          int64 `json:"sessionCount"`          // 区间内会话数
	ActiveDays            int64 `json:"activeDays"`            // 有游玩记录的天数
	WorkCount             int64 `json:"workCount"`             // 玩过的作品数
	LongestSessionSeconds int64 `json:"longestSessionSeconds"` // 最长的一次会话
	// FirstPlayedAt / LastPlayedAt 是该区间内最早开始、最晚结束的会话时间
	// （RFC3339 本地时区；没有会话记录时为空）。
	FirstPlayedAt string `json:"firstPlayedAt"`
	LastPlayedAt  string `json:"lastPlayedAt"`
	// SessionsSince 是会话表里最早的一条启动时间（不受区间限制）。
	// 前端据此提示「X 日之前的时长只有日桶、没有时段/次数」。
	SessionsSince string `json:"sessionsSince"`
	// SessionsTotal 是会话表里的总行数（不受区间限制）：
	// 0 = 还没产生过任何会话记录，时段分布整块不可用。
	SessionsTotal int64 `json:"sessionsTotal"`
}
