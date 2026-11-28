// frontend/src/stores/statsStore.ts
// 数据分析页的状态层（2026-11-28 用户要求「数据分析需求 … 名次统计 / 单日运行时长」）。
//
// 为什么单独一个 store，不塞进 libraryStore：
//   libraryStore 描述「库里现在有什么」，生命周期跟着游戏库列表走；
//   这里描述「一段区间内的统计结果」，换区间就要整批重算，与列表无关。
//
// 四个区块各自独立（overview / days / ranks / hours）：
// 用户改区间时四块一起刷，但某一块失败不该把其它块清空 —— 所以各自管自己的 error，
// 页面按块显示"这块没取到"，而不是整页变成错误态。
//
// 时间区间的默认值 = 近 30 天（含今天）：这是最常用的视角，
// 也刚好够日历热力图铺满一行多一点。
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { emptyOverview, statsApi } from '../services/statsService'
import type {
  PlayDayTotal,
  PlayHourBucket,
  PlayOverview,
  PlayWorkRank,
  StatsRange,
  StatsSource,
} from '../services/statsService'
import { toMessage } from '../utils/metaFormat'

/** 默认看近 30 天（含今天） */
const DEFAULT_RANGE_DAYS = 30

/** 名次榜一屏能看全的条数 */
export const RANK_LIMIT = 10

/** 把 Date 写成后端要的本地日期 YYYY-MM-DD（**不能用 toISOString**：那是 UTC，会差一天） */
export function toDayString(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

/** 今天往前 days-1 天 → 今天（闭区间） */
function recentRange(days: number): { from: string; to: string } {
  const today = new Date()
  const start = new Date(today)
  start.setDate(start.getDate() - (days - 1))
  return { from: toDayString(start), to: toDayString(today) }
}

export const useStatsStore = defineStore('stats', () => {
  // ===== State =====
  const { from: defaultFrom, to: defaultTo } = recentRange(DEFAULT_RANGE_DAYS)

  /** 当前区间（本地日期文本，闭区间） */
  const from = ref(defaultFrom)
  const to = ref(defaultTo)
  /** 来源过滤：空串 = 所有来源 */
  const source = ref<StatsSource>('')

  const overview = ref<PlayOverview>(emptyOverview())
  const days = ref<PlayDayTotal[]>([])
  const ranks = ref<PlayWorkRank[]>([])
  const hours = ref<PlayHourBucket[]>([])

  const loading = ref(false)
  /** 至少成功加载过一次（骨架屏用：第一次才铺骨架，之后换区间保持旧数据可见） */
  const loaded = ref(false)
  /** 整页级错误（四块全挂时才有意义；单块失败走各自的 xxxError） */
  const error = ref('')
  const overviewError = ref('')
  const daysError = ref('')
  const ranksError = ref('')
  const hoursError = ref('')

  // ===== Getters =====
  /** 当前区间条件（模板与 action 共用，避免各处手拼） */
  const range = computed<StatsRange>(() => ({ from: from.value, to: to.value, source: source.value }))

  /** 整段区间一秒都没玩：页面显示空状态而不是一片 0 */
  const isEmpty = computed(
    () => loaded.value && !loading.value && overview.value.totalSeconds === 0 && overview.value.sessionCount === 0,
  )

  /**
   * 时段分布能不能用：会话表里一条记录都没有时，24 个小时全是 0，
   * 那不是"你只在 0 点玩"，而是"这个维度还没有数据"。
   */
  const hasSessionData = computed(() => overview.value.sessionsTotal > 0)

  /** 有日桶但没会话的那些天（老口径），界面上用来解释"为什么这里没有次数" */
  const legacyDayCount = computed(() => days.value.filter((d) => d.seconds > 0 && d.sessions === 0).length)

  // ===== Actions =====

  /** 一次拉齐四块（换区间 / 进页面 / 点刷新都走它） */
  async function load(): Promise<void> {
    if (loading.value) return
    loading.value = true
    error.value = ''
    overviewError.value = ''
    daysError.value = ''
    ranksError.value = ''
    hoursError.value = ''
    const current = range.value

    // 四块互不依赖，并发拉；单块失败只影响自己那块
    const [overviewResult, daysResult, ranksResult, hoursResult] = await Promise.allSettled([
      statsApi.overview(current),
      statsApi.dayTotals(current),
      statsApi.workRanks(current, RANK_LIMIT),
      statsApi.hourBuckets(current),
    ])

    if (overviewResult.status === 'fulfilled') {
      overview.value = overviewResult.value
    } else {
      overviewError.value = `总览读取失败：${toMessage(overviewResult.reason)}`
    }

    if (daysResult.status === 'fulfilled') {
      days.value = daysResult.value
    } else {
      daysError.value = `单日时长读取失败：${toMessage(daysResult.reason)}`
    }

    if (ranksResult.status === 'fulfilled') {
      ranks.value = ranksResult.value
    } else {
      ranksError.value = `名次读取失败：${toMessage(ranksResult.reason)}`
    }

    if (hoursResult.status === 'fulfilled') {
      hours.value = hoursResult.value
    } else {
      hoursError.value = `时段分布读取失败：${toMessage(hoursResult.reason)}`
    }

    // 四块全挂才算整页失败（其余情况各块自己提示，页面其它部分照常可用）
    const allFailed = [overviewResult, daysResult, ranksResult, hoursResult].every((r) => r.status === 'rejected')
    if (allFailed) {
      error.value = '统计数据读取失败，请稍后重试'
    }

    loaded.value = true
    loading.value = false
  }

  /**
   * 换区间并重新加载。
   * 传进来的两个值必须是 YYYY-MM-DD（n-date-picker 的 value-format 已经保证了），
   * 空串 = 不限那一侧（后端与描述文案都支持）。
   */
  async function setRange(nextFrom: string, nextTo: string): Promise<void> {
    from.value = nextFrom || ''
    to.value = nextTo || ''
    await load()
  }

  /** 切换来源过滤（'' = 全部） */
  async function setSource(next: StatsSource): Promise<void> {
    source.value = next
    await load()
  }

  /** 快捷区间：近 N 天（含今天） */
  async function useRecentDays(daysCount: number): Promise<void> {
    const next = recentRange(daysCount)
    await setRange(next.from, next.to)
  }

  /** 恢复默认（近 30 天 + 全部来源） */
  async function resetRange(): Promise<void> {
    const next = recentRange(DEFAULT_RANGE_DAYS)
    source.value = ''
    await setRange(next.from, next.to)
  }

  return {
    // State
    from,
    to,
    source,
    overview,
    days,
    ranks,
    hours,
    loading,
    loaded,
    error,
    overviewError,
    daysError,
    ranksError,
    hoursError,
    // Getters
    range,
    isEmpty,
    hasSessionData,
    legacyDayCount,
    // Actions
    load,
    setRange,
    setSource,
    useRecentDays,
    resetRange,
  }
})
