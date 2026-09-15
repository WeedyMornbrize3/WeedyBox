<!-- frontend/src/components/todo/TodoPrioritySelector.vue -->
<!-- 新建 TODO 时选择优先级。用「信号强度」图标表达等级，
     三档强度形状递增，配合 tooltip/无障碍标签，不靠颜色单独表意。 -->
<template>
  <div
    class="flex items-center gap-1 bg-secondary p-[3px] rounded-[10px] border-theme"
    role="group"
    aria-labelledby="priority-selector-label"
  >
    <span id="priority-selector-label" class="sr-only">新建 TODO 的优先级</span>
    <button
      v-for="option in priorityOptions"
      :key="option.value"
      type="button"
      class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border-none cursor-pointer bg-transparent text-xs transition-colors duration-200 whitespace-nowrap hover:bg-hover"
      :class="
        todoStore.curPriority === option.value
          ? 'bg-card text-primary font-medium shadow-theme'
          : 'text-tertiary'
      "
      :aria-pressed="todoStore.curPriority === option.value"
      :title="`优先级：${option.label}`"
      @click="todoStore.curPriority = option.value"
    >
      <span :class="[option.icon, 'icon-xs']" aria-hidden="true" />
      <span>{{ option.label }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { useTodoStore } from '../../stores/todoStore'

const todoStore = useTodoStore()

// value 与后端 model.Todo.Priority 一致（0 低 / 1 中 / 2 高）
const priorityOptions = [
  { value: 0 as const, label: '低', icon: 'i-lucide-signal-low' },
  { value: 1 as const, label: '中', icon: 'i-lucide-signal-medium' },
  { value: 2 as const, label: '高', icon: 'i-lucide-signal-high' },
]
</script>
