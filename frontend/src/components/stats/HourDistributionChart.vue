<!-- frontend/src/components/stats/HourDistributionChart.vue
     「时间运行分析」的时段分布（2026-11-28 用户要求「时间运行分析」）。

     呈现方式：0-23 点的面积折线（横向 24 个刻度）。
     为什么用面积折线而不是柱状：24 根柱子在这个宽度下会挤成栅栏，而时段分布的形状
     （"晚上 8 点起翘、凌晨 2 点后回落的曲线")本身就是连续的，曲线更容易看出规律。

     数据口径：只能来自 work_play_sessions（会话区间），按真实重叠时长切进小时 ——
     老数据（只有日桶）在这里看不见。**调用方**负责在 sessionsTotal == 0 时
     用文字空态替换本图（见 StatsPage），本组件不做那个判断。

     高度：填满 split pane 的剩余高度（外层 .stats-fill-card 是 flex 列，
     本组件根 flex-1 min-h-0），横向 split 拖动改左栏高度时图跟着伸缩，
     不再用写死的 220px —— 否则会留白或被裁。
-->
<template>
  <StatsChart :option="option" :loading="loading" class="flex-1 min-h-0" />
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
  verticalAreaGradient,
} from '../../utils/statsChart'
import { useThemeStore } from '../../stores/Theme'
import type { PlayHourBucket } from '../../services/statsService'

const props = withDefaults(
  defineProps<{
    hours: PlayHourBucket[]
    loading?: boolean
  }>(),
  { loading: false }
)

const theme = useThemeStore()

/** 0 → "00:00"，23 → "23:00"：横轴用整点标签，比 "0" "23" 更明确 */
function hourLabel(hour: number): string {
  return `${String(hour).padStart(2, '0')}:00`
}

const option = computed<StatsChartOption>(() => {
  // 读一次签名：换四色主题 / 明暗时都要重算配色（与 DayDurationChart 同一原因）
  void theme.themeSignature
  const palette = chartPalette()
  // 后端固定给 24 项；这里兜一层，避免半截数据把 x 轴缩短
  const items = Array.from({ length: 24 }, (_, hour) => {
    return props.hours.find((item) => item.hour === hour) ?? { hour, seconds: 0, sessions: 0 }
  })

  return {
    tooltip: {
      ...chartTooltip(),
      trigger: 'axis',
      axisPointer: { type: 'line', lineStyle: { color: palette.axis } },
      formatter: (params: unknown) => {
        const list = Array.isArray(params) ? params : [params]
        const first = list[0] as { dataIndex?: number } | undefined
        const item = items[first?.dataIndex ?? 0]
        if (!item) return ''
        const sessions = item.sessions > 0 ? `<br/>这个点开始玩了 ${item.sessions} 次` : ''
        return `${hourLabel(item.hour)} – ${hourLabel((item.hour + 1) % 24)}<br/><b>${formatDuration(item.seconds)}</b>${sessions}`
      },
    },
    grid: { left: 8, right: 12, top: 16, bottom: 4, containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: items.map((item) => hourLabel(item.hour)),
      axisLabel: {
        color: palette.textTertiary,
        fontSize: 11,
        // 24 个标签太密：每 3 小时显示一个（00 / 03 / 06 …）
        interval: 2,
        hideOverlap: true,
      },
      axisLine: { lineStyle: { color: palette.axis } },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
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
        type: 'line',
        data: items.map((item) => item.seconds),
        smooth: 0.35,
        showSymbol: false,
        lineStyle: { color: palette.brand, width: 2 },
        itemStyle: { color: palette.brand },
        // 面积渐变：从品牌色淡淡铺到透明，强调"一天里的形状"
        areaStyle: { color: verticalAreaGradient(palette.areaFrom, palette.areaTo) },
      },
    ],
  }
})
</script>
