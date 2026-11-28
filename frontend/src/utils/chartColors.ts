// frontend/src/utils/chartColors.ts
//
// 数据可视化（饼图 / 名次榜占比条）专用的 5 色配色（2026-11-28 用户要求：
// 「为数据表格颜色使用新建5色显示 深色 浅色两种配色 不使用全局的四色配置」）。
//
// 关键约束：**独立于全局四色主题** —— 不走 styles/themePresets.ts 的四色，
// 也不从 theme.ts 推导出的 --color-* 里取色。所以无论用户切到 nightfall 还是
// ember、还是自定义四色，这 5 个切片颜色都不变；只有明暗模式（light/dark）
// 决定用哪一套 —— 深色背景配亮色、浅色背景配深色，保证对比度。
//
// 为什么是 5 个：图表里切片超过 5 个就分不清谁是谁了。所以约定「最多 5 个切片」，
// 多出来的最小项合并成「其他」（见 mergeToSlices）。5 色正好对应 5 个槽位。

/** 浅色主题（深色文字环境）：5 个色相、饱和度偏深，保证在浅底上可读 */
const LIGHT_COLORS = ['#2563eb', '#0d9488', '#16a34a', '#d97706', '#9333ea'] as const

/** 深色主题（浅色文字环境）：同 5 个色相、整体调亮，保证在深底上可读 */
const DARK_COLORS = ['#60a5fa', '#2dd4bf', '#4ade80', '#fbbf24', '#a78bfa'] as const

/** 合并出的「其他」切片固定用第 5 色（最后一个槽位），与前三名无关 */
export const CHART_SLICE_MAX = 5

/** 按明暗返回一套 5 色（新数组，避免调用方误改常量） */
export function chartSeriesColors(isDark: boolean): string[] {
  return [...(isDark ? DARK_COLORS : LIGHT_COLORS)]
}

/** 单个数据切片（饼图 / 占比条共用的最小结构） */
export interface ChartSlice {
  name: string
  value: number
}

/**
 * 把一组按值降序的条目收敛成「最多 maxSlices 个切片」：
 *   - 条目数 ≤ maxSlices：原样返回（不排序也不合并）；
 *   - 条目数 >  maxSlices：前 (maxSlices-1) 个保留，其余（最小的那些）的
 *     value 求和合并成一个 name='其他' 的切片。
 *
 * 用户原话（2026-11-28）：「如果是五种以上的最小数值的总和则显示其他」。
 * 所以 maxSlices 默认 5：前 4 名各占一色，第 5 名及之后合并成「其他」占第 5 色。
 *
 * ⚠️ 只合并、不改原数组；输入应按 value 已降序（名次数据天然如此），
 *    但这里仍做一次防御性排序，避免调用方传乱序时「其他」误吞大值。
 */
export function mergeToSlices(entries: ChartSlice[], maxSlices = CHART_SLICE_MAX): ChartSlice[] {
  const filtered = entries.filter((item) => item.value > 0)
  if (filtered.length <= maxSlices) return filtered

  const sorted = [...filtered].sort((a, b) => b.value - a.value)
  const head = sorted.slice(0, maxSlices - 1)
  const rest = sorted.slice(maxSlices - 1)
  const othersValue = rest.reduce((sum, item) => sum + item.value, 0)
  return [...head, { name: '其他', value: othersValue }]
}
