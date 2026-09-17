<!-- frontend/src/components/todo/TodoStats.vue -->
<!-- 统计栏：Minimalism 取向 —— 只保留必要元素。
     刻意去掉每项前面的图标：标签文字「总计/已完成/待完成」已经表意，
     再配图标属于重复装饰；仅用颜色做轻量区分。
     完成率进度条保留，因为它是唯一无法从文字直接看出的信息。 -->
<template>
  <section
    class="flex flex-wrap items-center gap-x-4 gap-y-1.5 min-w-0 px-3.5 py-2 rounded-lg bg-secondary-soft border-theme-light"
    aria-label="TODO 统计"
  >
    <span class="flex items-baseline gap-1.5 text-xs text-tertiary">
      总计
      <span class="text-sm font-semibold text-primary tabular-nums">{{ stats.total }}</span>
    </span>

    <span class="flex items-baseline gap-1.5 text-xs text-tertiary">
      已完成
      <span class="text-sm font-semibold text-success tabular-nums">{{ stats.completed }}</span>
    </span>

    <span class="flex items-baseline gap-1.5 text-xs text-tertiary">
      待完成
      <span class="text-sm font-semibold text-primary tabular-nums">{{ stats.pending }}</span>
    </span>

    <!-- 完成率：数值 + 进度条（进度条只是强化，数值本身已表意，不靠颜色单独传达信息） -->
    <span v-if="stats.total > 0" class="flex items-center gap-2 ml-auto">
      <span
        class="w-14 h-1 rounded-full bg-tertiary overflow-hidden"
        role="progressbar"
        :aria-valuenow="completionRate"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-label="`完成率 ${completionRate}%`"
      >
        <span
          class="block h-full bg-brand rounded-full transition-[width] duration-500 ease-out"
          :style="{ width: completionRate + '%' }"
        />
      </span>
      <span class="text-xs font-semibold text-brand tabular-nums">{{ completionRate }}%</span>
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
