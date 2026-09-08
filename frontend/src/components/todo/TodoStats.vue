<!-- frontend/src/components/todo/TodoStats.vue -->
<template>
  <div class="flex flex-wrap gap-4 px-4 py-2 rounded-lg bg-secondary/50">
    <span class="text-sm text-primary">
      🕶️ 总计: <span class="font-medium text-primary">{{ stats.total }}</span>
    </span>
    <span class="text-sm text-primary">
      ✅ 已完成: <span class="font-medium text-green-500">{{ stats.completed }}</span>
    </span>
    <span class="text-sm text-primary">
      ⏳ 待完成: <span class="font-medium text-yellow-500">{{ stats.pending }}</span>
    </span>
    <span class="text-sm text-primary" v-if="stats.total > 0">
      完成率: <span class="font-medium text-brand">{{ completionRate }}%</span>
    </span>
  </div>
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