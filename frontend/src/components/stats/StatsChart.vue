<!-- frontend/src/components/stats/StatsChart.vue
     分析页的**唯一图表出口**：包装 vue-echarts 的 <VChart>，负责三件重复的事 ——

       1. 主题跟随：换四色主题或明暗模式时，--color-* 会被重写，图表必须跟着换色。
          做法是 watch 主题 store 的 effectiveTheme + themeId，重新算一份 option。
          （单个图表组件不各自监听，否则 4 个图表 = 4 个 watcher 做同一件事。）
       2. 高度与自适应：外层给固定高度的盒子，<VChart> 靠 auto-resize 填满；
          Wails 窗口缩放、侧栏折叠都会改变宽度，图表要自己跟着重排。
       3. 骨架与空态：数据还没来时铺 n-skeleton（全站统一的加载态），
          **不要在图表里画"无数据"字样** —— 那是页面的事（用户要求空状态加载完成后才出现）。

     ⚠️ 不做成"通用图表组件 + 一堆 props"：分析页只有柱状和折线两种，
        把它们分别做成语义化组件（DayDurationChart / HourDistributionChart）更好读，
        这里只留"装 ECharts 的盒子"这一层机械劳动。
-->
<template>
  <div class="stats-chart" :style="{ height: `${height}px` }">
    <n-skeleton v-if="loading" height="100%" :sharp="true" />
    <VChart v-else class="stats-chart-canvas" :option="option" autoresize />
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { NSkeleton } from 'naive-ui'
import VChart from 'vue-echarts'
import type { StatsChartOption } from '../../utils/echartsSetup'
import { useThemeStore } from '../../stores/Theme'
import { baseChartOption, chartTooltip } from '../../utils/statsChart'

const props = withDefaults(
  defineProps<{
    /** 图表配置（由父组件按数据算好；本组件只负责加上主题底座） */
    option: StatsChartOption
    /** 图表高度（px）：分析页用固定高度，避免 0 高度导致 ECharts 画不出来 */
    height?: number
    loading?: boolean
  }>(),
  { height: 220, loading: false }
)

const theme = useThemeStore()

/**
 * 主题版本号：换主题时（含自定义主题调色）重算 option。
 *
 * 为什么需要它：父组件传进来的 option 是纯数据配置（不含颜色），
 * 颜色是从 token 现读的。如果只在父组件数据变化时重算，换主题时图表会留在旧配色。
 * 这里把主题信号一并纳入依赖，就能在换肤时重新跑一遍 computed。
 * 用 store 的 themeSignature（明暗 + id + 四色），自定义主题拖动色盘时也能跟上。
 */
const themeVersion = computed(() => theme.themeSignature)

/**
 * 最终 option = 父组件的数据配置 + 主题底座 + tooltip 外观。
 * 主题相关的东西（背景/字体/tooltip 配色）统一在这里合并，
 * 让各图表组件只关心"画什么形状"。
 */
const themedOption = computed<StatsChartOption>(() => {
  // 显式读一次，让 computed 依赖 themeVersion（换主题时重算）
  void themeVersion.value
  const base = baseChartOption()
  const tooltip = chartTooltip()
  const incoming = props.option as Record<string, unknown>
  const incomingTooltip = (incoming.tooltip ?? {}) as Record<string, unknown>
  return {
    ...base,
    ...incoming,
    textStyle: { ...base.textStyle, ...(incoming.textStyle as object | undefined) },
    tooltip: { ...tooltip, ...incomingTooltip },
  } as StatsChartOption
})

/** option 变化时不需要手动 setOption：<VChart> 自己监听 option 做增量更新 */
const option = computed(() => themedOption.value)

watch(themeVersion, () => {
  // 主题变了：computed 会自然重算，这里只是把"依赖它"这件事写明白
})
</script>

<style scoped>
/* 图表盒子：撑满父容器宽度，高度由 :style 传进来（必须是具体像素，见 props.height 说明） */
.stats-chart {
  width: 100%;
  min-width: 0;
}

.stats-chart-canvas {
  width: 100%;
  height: 100%;
}
</style>
