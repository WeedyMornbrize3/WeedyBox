<!-- frontend/src/components/todo/TodoStats.vue -->
<!-- 统计栏：Flat 卡片，图标全部为矢量图标（lucide），不再使用 emoji 作结构性图标 -->
<template>
  <section
    class="flex flex-wrap items-center gap-x-5 gap-y-2 min-w-0 px-4 py-2.5 rounded-lg bg-secondary-soft border-theme-light"
    aria-label="TODO 统计"
  >
    <span class="flex items-center gap-1.5 text-xs text-tertiary">
      <span class="i-lucide-layers icon-xs" aria-hidden="true" />
      总计
      <span class="text-sm font-semibold text-primary tabular-nums">{{ stats.total }}</span>
    </span>

    <span class="flex items-center gap-1.5 text-xs text-tertiary">
      <span class="i-lucide-circle-check-big icon-xs text-success" aria-hidden="true" />
      已完成
      <span class="text-sm font-semibold text-primary tabular-nums">{{ stats.completed }}</span>
    </span>

    <span class="flex items-center gap-1.5 text-xs text-tertiary">
      <span class="i-lucide-clock icon-xs text-warning" aria-hidden="true" />
      待完成
      <span class="text-sm font-semibold text-primary tabular-nums">{{ stats.pending }}</span>
    </span>

    <!-- 完成率：数字 + 进度条（进度条只是强化，数值本身已表意，不靠颜色单独传达信息） -->
    <span v-if="stats.total > 0" class="flex items-center gap-2 ml-auto">
      <span class="text-xs text-tertiary whitespace-nowrap">完成率</span>
      <span
        class="w-16 h-1.5 rounded-full bg-tertiary overflow-hidden"
        role="progressbar"
        :aria-valuenow="completionRate"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-label="`完成率 ${completionRate}%`"
      >
        <span class="block h-full bg-brand rounded-full transition-all duration-300" :style="{ width: completionRate + '%' }" />
      </span>
      <span class="text-sm font-semibold text-brand tabular-nums">{{ completionRate }}%</span>
    </span>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TodoStats } from '../../../bindings/WeedyBox/internal/model'

const props = defineProps<{
  stats: TodoStats
}>()

const completionRate = computed(() => {
  if (props.stats.total === 0) return 0
  return Math.round((props.stats.completed / props.stats.total) * 100)
})
</script>
