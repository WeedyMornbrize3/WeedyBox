<!-- frontend/src/components/stats/WorkSharePieChart.vue
     「最长游戏时长排行」上方的饼图（2026-11-28 用户要求「最长游戏时长排行 上方用饼状统计图显示」）。

     它显示什么：所选区间内各作品游玩时长的**占比**（环图，第 1 名在正上方顺时针排）。
     为什么是环图（donut）而不是实心饼：
       · 中心可以放「总时长」这个总量，一眼同时读到"总量"和"构成"；
       · 实心饼中心是空白浪费，而排行榜最需要的恰恰是总量参照。

     配色与切片数（2026-11-28 用户后续要求，与 /works 的 split 同批）：
       · 「为数据表格颜色使用新建5色显示 深色 浅色两种配色 不使用全局的四色配置」
         → 切片用 utils/chartColors.ts 的 5 色（浅色一套 / 深色一套，**独立于四色主题**）；
       · 「如果是五种以上的最小数值的总和则显示其他」
         → 超过 5 个作品时，前 4 名各占一色，第 5 名及之后合并成一个「其他」切片。

     与下方 WorkRankList 的关系：
       列表仍然是完整的前 10 名（用户要的是名次榜），饼图是"构成比例"的浓缩视图；
       所以「其他」只在饼图里出现，列表不动 —— 两者视角不同，不强制一一对应。
-->
<template>
  <StatsChart :option="option" :loading="loading" class="flex-1 min-h-0" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import StatsChart from './StatsChart.vue'
import type { StatsChartOption } from '../../utils/echartsSetup'
import { chartTooltip, formatDuration } from '../../utils/statsChart'
import { chartSeriesColors, mergeToSlices } from '../../utils/chartColors'
import { useThemeStore } from '../../stores/Theme'
import type { PlayWorkRank } from '../../services/statsService'

const props = withDefaults(
  defineProps<{
    ranks: PlayWorkRank[]
    loading?: boolean
  }>(),
  { loading: false }
)

const theme = useThemeStore()

const option = computed<StatsChartOption>(() => {
  // 读一次签名：换明暗 / 换主题都要重算（5 色按明暗分套，深浅切换必须跟着换）
  const isDark = theme.isDark
  const colors = chartSeriesColors(isDark)

  // 有时长的作品才进饼图；超过 5 个时合并成「其他」（见 mergeToSlices）
  const slices = mergeToSlices(
    props.ranks
      .filter((item) => item.seconds > 0)
      .map((item) => ({ name: item.name, value: item.seconds })),
  )
  const totalSeconds = slices.reduce((sum, item) => sum + item.value, 0)

  return {
    tooltip: {
      ...chartTooltip(),
      trigger: 'item',
      formatter: (params: unknown) => {
        const p = params as { dataIndex?: number; percent?: number }
        const item = slices[p.dataIndex ?? 0]
        if (!item) return ''
        const percent = (p.percent ?? 0).toFixed(1)
        return `${item.name}<br/><b>${formatDuration(item.value)}</b>（${percent}%）`
      },
    },
    legend: {
      // 图例放右侧竖排：作品名多半很长，横排放不下会挤成两行把图压扁
      type: 'scroll',
      orient: 'vertical',
      right: 0,
      top: 'middle',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 8,
      textStyle: { color: theme.isDark ? '#d5d5d8' : '#6b6b70', fontSize: 11 },
      formatter: (name: string) => (name.length > 12 ? `${name.slice(0, 11)}…` : name),
      pageTextStyle: { color: theme.isDark ? '#d5d5d8' : '#6b6b70' },
      pageIconSize: 9,
    },
    // 中心留出放总量的空间，环的半径跟着高度走（小屏也不溢出）
    series: [
      {
        name: '游玩时长',
        type: 'pie',
        // 起始角度 90°：第 1 名从正上方开始，顺时针
        startAngle: 90,
        radius: ['52%', '76%'],
        center: ['36%', '50%'],
        avoidLabelOverlap: true,
        itemStyle: {
          color: (params: { dataIndex: number }) => colors[params.dataIndex % colors.length],
          borderColor: 'transparent',
          borderWidth: 0,
        },
        label: { show: false },
        labelLine: { show: false },
        // hover 时把切片微微外扩，强调"这块是谁"
        emphasis: { scale: true, scaleSize: 4 },
        data: slices.map((item) => ({ name: item.name, value: item.value })),
      },
    ],
    // 环心文字：总时长（用 graphic 而不是 title，位置好控且不占布局）
    graphic: [
      {
        type: 'text',
        left: '36%',
        top: '46%',
        style: {
          text: formatDuration(totalSeconds),
          fill: theme.isDark ? '#f5f5f6' : '#2f2e2f',
          fontSize: 15,
          fontWeight: 600,
          textAlign: 'center',
          textVerticalAlign: 'middle',
          fontFamily: 'inherit',
        },
      },
      {
        type: 'text',
        left: '36%',
        top: '58%',
        style: {
          text: '合计',
          fill: theme.isDark ? '#a8a8ad' : '#8a8a8f',
          fontSize: 11,
          textAlign: 'center',
          textVerticalAlign: 'middle',
          fontFamily: 'inherit',
        },
      },
    ],
  }
})
</script>
