// internal/service/stats/stats_service.go
//
// 数据分析服务：把游玩记录聚合成前端分析页要的几块数据。
//
// 用户原话（2026-11-28）：
//
//	「数据分析需求 需要建立如游戏 时间运行分析 名次统计 单日运行时长 处理数据服务
//	  前端图标需要使用 先确定前端使用的图标 决定函数 后端通用函数 前端特化
//	  新建router界面」
//
// 对应关系：
//   - 「处理数据服务」= 本服务（独立子包 stats，与 game / scan 一致的组织方式）；
//   - 「后端通用函数」= 区间解析、按小时切分、补零日历这些**不带业务偏好**的纯函数，
//     放在本文件下半部分，前端拿到的是已经算好的数组，不用自己二次加工；
//   - 「前端特化」= 每个区块一个方法（Overview / DayTotals / WorkRanks / HourBuckets），
//     各自只返回那一块要的东西，前端一个区块一次调用、可以独立加载与重试。
//
// 为什么不做成"一个 GetReport 返回所有东西"：
// 分析页的三个区块刷新节奏不同（单日表格可以只翻页、名次可以只换区间），
// 一个大对象会让改区间等于重新拉全部。分开四个方法，前端各取所需。
//
// 数据口径（很关键，报告里的数字要能自圆其说）：
//   - 时长 / 天：work_play_time（日桶），**老数据也在这里**，是秒数的权威来源；
//   - 次数 / 时段 / 最长一次：work_play_sessions（2026-11-28 起才有数据），
//     老日期只有时长没有次数，Overview.SessionsSince 会告诉前端断层从哪天开始。
package stats

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"WeedyBox/internal/database"
	"WeedyBox/internal/model"
)

// StatsService 是暴露给前端的分析服务。
type StatsService struct {
	DB *database.SQLiteService
}

// ===== 入参 =====

// RangeRequest 是分析页的时间区间（分析页顶部那个日期范围选择器）。
//
// 两个字段都是**本地日期**文本 YYYY-MM-DD，闭区间；空串 = 不限那一侧。
// 可选参数在 Wails 绑定里不好表达，所以统一走结构体（与 FetchWorkRequest 一致）。
type RangeRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Source string `json:"source"` // 空 = 所有来源（dlsite + steam 一起看）
}

// RankRequest 是名次统计的入参：区间 + 要几个名次。
//
// Limit <= 0 时取 defaultRankLimit：界面上一屏画不出更多，传 0 就按默认。
type RankRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Source string `json:"source"`
	Limit  int    `json:"limit"`
}

// 分析页的默认值：名次榜默认 10 条（一屏能看全），单日表默认近 30 天。
const (
	defaultRankLimit = 10
	maxRankLimit     = 100
)

// ===== 对外方法（每块数据一个，前端按区块各取所需）=====

// Overview 返回顶部那排总览数字（总时长 / 次数 / 活跃天数 / 作品数 / 最长一次）。
//
// 区间为空时统计全量 —— 刚装好软件还没有任何记录也只是返回一堆 0，不报错。
func (s *StatsService) Overview(ctx context.Context, req RangeRequest) (model.PlayOverview, error) {
	if s == nil || s.DB == nil {
		return model.PlayOverview{}, nil
	}
	return s.DB.GetPlayOverview(ctx, toRange(req))
}

// DayTotals 返回区间内每天的游玩时长（日期升序）。
//
// 区间两头都给了时会**补成连续日历**（没玩的那天也有一项、时长 0）：
// 折线图与日历热力图都需要连续 x 轴，与其每个图表组件各写一遍补洞，
// 不如后端一次给全（这就是用户说的「后端通用函数，前端特化」）。
// 只给一头（或都不给）时返回稀疏数据 —— 跨度未知，没法补。
func (s *StatsService) DayTotals(ctx context.Context, req RangeRequest) ([]model.PlayDayTotal, error) {
	if s == nil || s.DB == nil {
		return []model.PlayDayTotal{}, nil
	}
	days, err := s.DB.ListPlayDayTotals(ctx, toRange(req))
	if err != nil {
		return nil, err
	}
	return fillDayRange(days, req.From, req.To), nil
}

// WorkRanks 返回区间内的作品名次（按时长降序，最多 Limit 条）。
func (s *StatsService) WorkRanks(ctx context.Context, req RankRequest) ([]model.PlayWorkRank, error) {
	if s == nil || s.DB == nil {
		return []model.PlayWorkRank{}, nil
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultRankLimit
	}
	if limit > maxRankLimit {
		limit = maxRankLimit
	}
	return s.DB.ListPlayWorkRanks(ctx, toRange(RangeRequest{
		From:   req.From,
		To:     req.To,
		Source: req.Source,
	}), limit)
}

// HourBuckets 返回 0-23 点每个小时的累计游玩时长与起始会话数。
//
// 固定返回 24 项（没玩过的小时给 0）：前端直接喂给图表，不用自己补洞。
// **本块只能来自会话表**，老数据（只有日桶）在这一块是看不见的 ——
// 所以前端会在 SessionsTotal == 0 时把整块换成"还没有时段数据"的说明。
func (s *StatsService) HourBuckets(ctx context.Context, req RangeRequest) ([]model.PlayHourBucket, error) {
	buckets := makeEmptyHourBuckets()
	if s == nil || s.DB == nil {
		return buckets, nil
	}
	spans, err := s.DB.ListPlaySessionSpans(ctx, toRange(req))
	if err != nil {
		return nil, err
	}
	return accumulatePlayHours(buckets, spans), nil
}

// ===== 后端通用函数（纯计算，不碰数据库）=====
//
// 放在这里而不是各自散在方法里，是为了让「同一个算法只实现一次」：
// 前端不需要知道跨 midnight / 跨小时怎么切，也就不用在前端再写一遍。

// toRange 把入参收敛成内部查询条件：去空白、日期只认前 10 个字符。
//
// 为什么要截断：前端可能把 Date 直接 toISOString 成 "2026-11-27T16:00:00.000Z"，
// 那种字符串和库里的 "2026-11-27" 比较会全部落空（字典序比较，"2026-11-27T…" > "2026-11-27"）。
// 统一取前 10 位，前端怎么写都不会错位一天。
func toRange(req RangeRequest) database.StatsRange {
	return database.StatsRange{
		From:   normalizeDay(req.From),
		To:     normalizeDay(req.To),
		Source: strings.ToLower(strings.TrimSpace(req.Source)),
	}
}

// normalizeDay 只保留 YYYY-MM-DD；认不出来的直接当空串（= 不限）。
func normalizeDay(raw string) string {
	day := strings.TrimSpace(raw)
	if len(day) < 10 {
		return ""
	}
	day = day[:10]
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return ""
	}
	return day
}

// makeEmptyHourBuckets 生成 24 个空桶（0-23 点）。
func makeEmptyHourBuckets() []model.PlayHourBucket {
	buckets := make([]model.PlayHourBucket, 24)
	for hour := range buckets {
		buckets[hour].Hour = hour
	}
	return buckets
}

// accumulatePlayHours 把会话区间按**真实重叠时长**切进 24 个小时桶。
//
// 为什么不按 started_at 的小时整段归类：21:50 玩到 22:40 的会话，若整段算给 21 点，
// 「几点最常玩」就会被系统性推早；切分后 21 点记 10 分钟、22 点记 40 分钟，
// 和用户自己的体感一致。24 个桶的秒数求和 == 这些会话的总时长（可自检）。
//
// Sessions 只记在会话**开始**的那个小时里（不然一段跨 3 小时的会话会让 3 个小时都 +1，
// 乘出来就不是「玩了几次」了）。
func accumulatePlayHours(buckets []model.PlayHourBucket, spans []database.PlaySessionSpan) []model.PlayHourBucket {
	if len(buckets) != 24 {
		return buckets
	}
	for _, span := range spans {
		if span.StartedAt.IsZero() || !span.EndedAt.After(span.StartedAt) {
			continue
		}
		buckets[span.StartedAt.Hour()].Sessions++

		// 逐小时推进游标：每轮算到"本小时结束"与"会话结束"里更早的那个
		cursor := span.StartedAt
		for cursor.Before(span.EndedAt) {
			hourEnd := cursor.Truncate(time.Hour).Add(time.Hour)
			segmentEnd := span.EndedAt
			if hourEnd.Before(span.EndedAt) {
				segmentEnd = hourEnd
			}
			if seconds := int64(segmentEnd.Sub(cursor).Seconds()); seconds > 0 {
				buckets[cursor.Hour()].Seconds += seconds
			}
			cursor = segmentEnd
		}
	}
	return buckets
}

// fillDayRange 把「只有有数据的那些天」铺成连续日历（[from, to] 每天都有一项，缺的补 0）。
//
// 这是「后端通用函数」的一个样板：折线/日历需要一个连续的 x 轴，
// 补洞规则只在这里实现一次。from/to 必须是 YYYY-MM-DD 且两边都给；
// 跨度超过 maxFillDays 天时不补（防止误传 1970 年导致生成几十万行卡死），
// 此时原样返回稀疏数据。
const maxFillDays = 366 * 3

// fillDayRange 见上。返回的每一项都带 Day，前端可以直接当横轴标签。
func fillDayRange(days []model.PlayDayTotal, from, to string) []model.PlayDayTotal {
	start, err := time.ParseInLocation("2006-01-02", normalizeDay(from), time.Local)
	if err != nil {
		return days
	}
	end, err := time.ParseInLocation("2006-01-02", normalizeDay(to), time.Local)
	if err != nil || end.Before(start) {
		return days
	}
	total := int(end.Sub(start).Hours()/24) + 1
	if total > maxFillDays {
		return days
	}

	byDay := make(map[string]model.PlayDayTotal, len(days))
	for _, item := range days {
		byDay[item.Day] = item
	}

	out := make([]model.PlayDayTotal, 0, total)
	for cursor := start; !cursor.After(end); cursor = cursor.AddDate(0, 0, 1) {
		day := cursor.Format("2006-01-02")
		item, ok := byDay[day]
		if !ok {
			item = model.PlayDayTotal{Day: day}
		}
		out = append(out, item)
	}
	return out
}

// DescribeRange 把区间写成人话（"2026-11-01 至 2026-11-28" / "全部时间"）。
//
// 前端拿到的是已经格式化好的标题，不用在两个地方维护同一句文案；
// 出错了也不会因为文案问题影响数据。
func DescribeRange(req RangeRequest) string {
	from := normalizeDay(req.From)
	to := normalizeDay(req.To)
	switch {
	case from == "" && to == "":
		return "全部时间"
	case from == "":
		return "截至 " + to
	case to == "":
		return from + " 起"
	case from == to:
		return from
	default:
		return from + " 至 " + to
	}
}

// errNoDB 是"数据库没起来"的统一错误（目前各方法都返回空结果而非报错，
// 保留这个给以后需要区分"没数据"与"没库"的场景）。
var errNoDB = errors.New("数据库未就绪")

// ensure 是内部自检：服务被直接字面量构造（main.go 就是这么做的）时 DB 可能为空。
func (s *StatsService) ensure() error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("%w", errNoDB)
	}
	return nil
}
