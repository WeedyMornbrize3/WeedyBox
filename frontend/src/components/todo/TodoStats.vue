<!-- frontend/src/components/todo/TodoStats.vue
     统计栏：Minimalism 取向 —— 只保留必要元素。
     2026-11-23「完成」功能移除后，这里不再有已完成 / 待完成 / 完成率，
     改为按优先级统计（总计 + 高 / 中 / 低）：优先级本来就在每行显示，
     汇总一下能看出手头事情的分量，且不需要后端再多一个统计接口。
     刻意去掉每项前面的图标：标签文字已经表意，再配图标属于重复装饰。 -->
<template>
  <section
    class="flex flex-wrap items-center gap-x-4 gap-y-1.5 min-w-0 px-3.5 py-2 rounded-lg bg-secondary-soft border-theme-light"
    aria-label="TODO 统计"
  >
    <span class="flex items-baseline gap-1.5 text-xs text-tertiary">
      总计
      <span class="text-sm font-semibold text-primary tabular-nums">{{ total }}</span>
    </span>

    <span
      v-for="item in priorityStats"
      :key="item.value"
      class="flex items-baseline gap-1.5 text-xs text-tertiary"
    >
      {{ item.label }}
      <span class="text-sm font-semibold tabular-nums" :class="item.className">{{ item.count }}</span>
    </span>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Todo } from '../../../bindings/WeedyBox/internal/model'

const props = defineProps<{
  /** 当前列表（已按筛选条件过滤：统计跟着"看得见的"走） */
  todos: Todo[]
}>()

/** 优先级 0 低 / 1 中 / 2 高（与后端 model.Todo.Priority 一致） */
const PRIORITIES: { value: number; label: string; className: string }[] = [
  { value: 2, label: '高', className: 'text-priority-high' },
  { value: 1, label: '中', className: 'text-priority-medium' },
  { value: 0, label: '低', className: 'text-priority-low' },
]

const total = computed(() => props.todos.length)

const priorityStats = computed(() =>
  PRIORITIES.map((item) => ({
    ...item,
    count: props.todos.filter((todo) => todo.priority === item.value).length,
  })),
)
</script>
