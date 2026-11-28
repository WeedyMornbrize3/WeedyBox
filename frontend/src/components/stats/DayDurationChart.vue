<!-- frontend/src/components/stats/DayDurationChart.vue
     「单日运行时长」的柱状图（2026-11-28 用户要求「单日运行时长」）。

     为什么用柱状而不是折线：单日时长是**离散的每日总量**，柱状能一眼看出"哪天玩得久"；
     折线暗示"天与天之间是连续的",而实际上没玩的那天就是个 0 柱子，柱子更诚实。

     数据来自 statsStore.days（后端已按区间补成连续日历，没玩的那天是 0），
     所以这里不再补洞、不再排序 —— 直接喂给 ECharts。
-->
<template>
  <StatsChart :option="option" :loading="loading" :height="240" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import StatsChart from './StatsChart.vue'
import type { StatsChartOption } from '../../utils/echartsSetup'
import {
  chartPalette,
  chartTooltip,
  formatAxisDuration,
  formatDuration,
  shortDay,
} from '../../utils/statsChart'
import { useThemeStore } from '../../stores/Theme'
import type { PlayDayTotal } from '../../services/statsService'

const props = withDefaults(
  defineProps<{
    days: PlayDayTotal[]
    loading?: boolean
  }>(),
  { loading: false }
)

const theme = useThemeStore()

const option = computed<StatsChartOption>(() => {
  // 读一次签名：换四色主题 / 明暗时都要重算配色（chartPalette 读的是 --color-*，
  // 本身不是响应式的，必须靠这里的依赖把"换肤"接进 computed）
  void theme.themeSignature
  const palette = chartPalette()
  const items = props.days
  const labels = items.map((item) => shortDay(item.day))
  const values = items.map((item) => item.seconds)

  return {
    tooltip: {
      ...chartTooltip(),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const list = Array.isArray(params) ? params : [params]
        const first = list[0] as { dataIndex?: number } | undefined
        const index = first?.dataIndex ?? 0
        const item = items[index]
        if (!item) return ''
        const sessions = item.sessions > 0 ? `<br/>启动 ${item.sessions} 次` : ''
        return `${item.day}<br/><b>${formatDuration(item.seconds)}</b>${sessions}`
      },
    },
    grid: { left: 8, right: 12, top: 16, bottom: 4, containLabel: true },
    xAxis: {
      type: 'category',
      data: labels,
      // 天数多时标签会挤成一团：间隔显示，且不旋转（旋转后中文字会歪着看）
      axisLabel: { color: palette.textTertiary, fontSize: 11, interval: 'auto', hideOverlap: true },
      axisLine: { lineStyle: { color: palette.axis } },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      // 纵轴用紧凑刻度（30m / 2h），完整中文留给 tooltip
      axisLabel: {
        color: palette.textTertiary,
        fontSize: 11,
        formatter: (value: number) => formatAxisDuration(value),
      },
      splitLine: { lineStyle: { color: palette.grid } },
    },
    series: [
      {
        name: '游玩时长',
        type: 'bar',
        data: values,
        // 柱子窄一点：单日格子多，太宽会粘连
        barMaxWidth: 22,
        itemStyle: { color: palette.brand, borderRadius: [2, 2, 0, 0] },
        emphasis: { itemStyle: { color: palette.brandText } },
      },
    ],
  }
})
</script>
