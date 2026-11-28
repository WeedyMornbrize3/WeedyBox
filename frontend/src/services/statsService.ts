// frontend/src/services/statsService.ts
// 数据分析（游玩报表）的前端封装：对应后端 internal/service/stats（Wails 服务名 StatsService）。
//
// 与 libraryService / gameService 的分工：
//   libraryApi 回答「库里有什么」，gameApi 回答「怎么启动、在不在跑」，
//   statsApi   回答「这些东西一共玩了多少、哪天玩、哪个最多、几点玩」。
//
// 后端已经把三件事算好了（用户 2026-11-28 的「后端通用函数 前端特化」）：
//   · 区间补洞（DayTotals 两头给全时会返回连续日历）；
//   · 跨小时的会话按真实重叠时长切进 24 个小时桶；
//   · 名次按时长降序排好、作品名补齐、上限夹好。
// 所以这里只做「去掉 null / 兜底空数组」，不再做任何二次计算 ——
// 一旦前端也开始算，两边的口径就会分叉。
//
// 数据口径提醒（界面上要能自圆其说，别让用户觉得数字对不上）：
//   seconds / sessions 来自两张不同的表 —— 时长来自日桶（老数据也有），
//   次数来自 2026-11-28 才有的会话表；老日期「有时长没次数」是正常的。
//   overview.sessionsTotal == 0 表示从没有过会话记录，时段分布整块不可用。
//
// 注意：frontend/bindings/ 是 wails3 generate bindings 生成的（已被 .gitignore 忽略），
// 新克隆仓库后必须先跑一次生成命令，否则下面的 import 会解析失败。
import { StatsService } from '../../bindings/WeedyBox/internal/service/stats'
import type { RangeRequest, RankRequest } from '../../bindings/WeedyBox/internal/service/stats/models.js'
import type {
  PlayDayTotal,
  PlayHourBucket,
  PlayOverview,
  PlayWorkRank,
} from '../../bindings/WeedyBox/internal/model/models.js'

export type { PlayDayTotal, PlayHourBucket, PlayOverview, PlayWorkRank, RangeRequest, RankRequest }

/** 分析页来源过滤：空串 = 所有来源一起看 */
export type StatsSource = '' | 'dlsite' | 'steam'

/** 区间（本地日期 YYYY-MM-DD，闭区间；空串 = 不限那一侧） */
export interface StatsRange {
  from: string
  to: string
  source: StatsSource
}

/** 空总览：后端没数据 / 还没加载时的兜底，避免模板到处写 `?.` */
export function emptyOverview(): PlayOverview {
  return {
    totalSeconds: 0,
    sessionCount: 0,
    activeDays: 0,
    workCount: 0,
    longestSessionSeconds: 0,
    firstPlayedAt: '',
    lastPlayedAt: '',
    sessionsSince: '',
    sessionsTotal: 0,
  }
}

/** StatsRange → 后端入参（两个接口的字段几乎一样，统一在这里转一次） */
function toRangeRequest(range: StatsRange): RangeRequest {
  return { from: range.from, to: range.to, source: range.source }
}

export const statsApi = {
  /** 顶部总览（总时长 / 次数 / 活跃天数 / 作品数 / 最长一次） */
  async overview(range: StatsRange): Promise<PlayOverview> {
    return (await StatsService.Overview(toRangeRequest(range))) ?? emptyOverview()
  },

  /**
   * 单日运行时长（按天）。
   * 区间两头都给时后端会补成连续日历（没玩的天也有 0 值），可直接当图表横轴。
   */
  async dayTotals(range: StatsRange): Promise<PlayDayTotal[]> {
    return (await StatsService.DayTotals(toRangeRequest(range))) ?? []
  },

  /** 名次统计（按时长降序，只取前 limit 条；limit <= 0 时后端按 10 条给） */
  async workRanks(range: StatsRange, limit = 10): Promise<PlayWorkRank[]> {
    const req: RankRequest = { ...toRangeRequest(range), limit }
    return (await StatsService.WorkRanks(req)) ?? []
  },

  /**
   * 时段分布：固定 24 项（0-23 点）。
   * 只来自会话表 —— 老数据在这里看不见，调用方要用 overview.sessionsTotal 判断是否可用。
   */
  async hourBuckets(range: StatsRange): Promise<PlayHourBucket[]> {
    return (await StatsService.HourBuckets(toRangeRequest(range))) ?? []
  },
}
