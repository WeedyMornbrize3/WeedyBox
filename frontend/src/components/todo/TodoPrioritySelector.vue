<!-- frontend/src/components/todo/TodoPrioritySelector.vue -->
<!-- 新建 TODO 时选择优先级（2026-11-24 改用 naive 的分段单选）。
     用「信号强度」图标表达等级，三档强度形状递增，配合 tooltip/无障碍标签，
     不靠颜色单独表意。 -->
<template>
  <n-radio-group
    :value="todoStore.curPriority"
    size="small"
    name="todo-priority"
    aria-label="新建 TODO 的优先级"
    @update:value="onChange"
  >
    <n-radio-button
      v-for="option in priorityOptions"
      :key="option.value"
      :value="option.value"
      :title="`优先级：${option.label}`"
    >
      <span class="flex items-center gap-1 whitespace-nowrap">
        <span :class="[option.icon, 'icon-xs']" aria-hidden="true" />
        {{ option.label }}
      </span>
    </n-radio-button>
  </n-radio-group>
</template>

<script setup lang="ts">
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { useTodoStore } from '../../stores/todoStore'

const todoStore = useTodoStore()

// value 与后端 model.Todo.Priority 一致（0 低 / 1 中 / 2 高）
const priorityOptions = [
  { value: 0 as const, label: '低', icon: 'i-lucide-signal-low' },
  { value: 1 as const, label: '中', icon: 'i-lucide-signal-medium' },
  { value: 2 as const, label: '高', icon: 'i-lucide-signal-high' },
]

function onChange(value: string | number | boolean) {
  todoStore.curPriority = Number(value) as 0 | 1 | 2
}
</script>
