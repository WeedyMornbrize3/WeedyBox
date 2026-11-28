<!-- frontend/src/components/stats/DayHeatmapChart.vue
     「单日运行时长」的热力图视图（2026-11-28 用户要求：
     「单日运行时长界面 通过selector(在单日运行时长界面卡片内)可切换柱状图还是热力图
     Heatmap naive ui」）。

     与 DayDurationChart（柱状图）是同一份数据（stats.days）的两种视图：
       · 柱状图看「哪天玩得久」，一眼比高低；
       · 热力图看「这一段时间哪几天有玩、强度如何」，像 GitHub 贡献图，适合看连续性和空白。

     为什么用 naive 的 n-heatmap 而不是 echarts 的热力图：
       · 用户明确点名「Heatmap naive ui」—— 用 naive 内置组件，与全站控件观感一致，
         滚动条、tooltip、配色都跟着 naive 主题走，不另起一套；
       · n-heatmap 就是日历热力图（按天铺格），正好匹配「单日」这个粒度。

     数据格式：n-heatmap 要 { timestamp: number, value?: number }（timestamp 是毫秒）；
     后端给的是 { day: 'YYYY-MM-DD', seconds }。这里把 day 转成本地 0 点的 Date。
     只有「有时长」的天有 value，没玩的天空缺（不铺格子），热力图自然留白。

     配色：热力图表示「强度」，用品牌色阶（浅→深），随四色主题切换；
     与饼图/名次榜那套独立的 5 色**不是一回事**（那是分类色，这是顺序色）。
-->
<template>
  <div class="day-heatmap">
    <n-heatmap
      :data="heatmapData"
      :active-colors="colors"
      :show-week-labels="true"
      :show-month-labels="true"
      size="medium"
      :tooltip="true"
    >
      <template #tooltip="{ value, timestamp }">
        <span>{{ tooltipText(value, timestamp) }}</span>
      </template>
    </n-heatmap>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NHeatmap } from 'naive-ui'
import { useThemeStore } from '../../stores/Theme'
import { chartPalette, formatDuration } from '../../utils/statsChart'
import type { PlayDayTotal } from '../../services/statsService'

const props = withDefaults(
  defineProps<{
    days: PlayDayTotal[]
  }>(),
  {}
)

const theme = useThemeStore()

/** day 文本 → 本地 0 点的毫秒时间戳（n-heatmap 认毫秒） */
function dayToTimestamp(day: string): number {
  const [y, m, d] = day.split('-').map(Number)
  return new Date(y, (m ?? 1) - 1, d ?? 1).getTime()
}

/**
 * n-heatmap 的 data：只有有时长的天才有值，没玩的天空缺。
 * value 用秒（原值），tooltip 里再格式化成"x 小时 x 分"。
 */
const heatmapData = computed(() =>
  props.days.filter((item) => item.seconds > 0).map((item) => ({
    timestamp: dayToTimestamp(item.day),
    value: item.seconds,
  })),
)

/**
 * 品牌色阶（浅→深），随四色主题 / 明暗切换。
 * 用 chartPalette 拿到的品牌色做 4 档：掺白 60% → 品牌色 → 掺黑 30%。
 * 这样"没玩"是底色、玩得多越接近品牌深色，与全站「品牌色=强调」的语义一致。
 */
const colors = computed(() => {
  // 读一次签名，让换主题时重算（chartPalette 读 getComputedStyle，非响应式）
  void theme.themeSignature
  const p = chartPalette()
  const brand = p.brand
  return [
    mix(brand, '#ffffff', 0.55),
    mix(brand, '#ffffff', 0.2),
    brand,
    mix(brand, '#000000', 0.28),
  ]
})

/** #rrggbb 与目标色按比例混（ratio 是目标色的占比） */
function mix(hex: string, target: string, ratio: number): string {
  const parse = (c: string) => {
    const h = c.trim()
    if (!h.startsWith('#') || h.length < 7) return null
    return [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16))
  }
  const a = parse(hex)
  const b = parse(target)
  if (!a || !b) return hex
  const out = a.map((v, i) => Math.round(v + (b[i] - v) * ratio))
  return `#${out.map((v) => v.toString(16).padStart(2, '0')).join('')}`
}

/** tooltip：日期 + 时长 */
function tooltipText(value: number | null | undefined, timestamp: number): string {
  const date = new Date(timestamp)
  const day = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  const duration = value != null && value > 0 ? formatDuration(value) : '未游玩'
  return `${day} · ${duration}`
}
</script>

<style scoped>
.day-heatmap {
  width: 100%;
  min-width: 0;
}
</style>
