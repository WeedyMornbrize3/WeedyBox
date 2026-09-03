<!-- frontend/src/components/todo/TodoItem.vue -->
<template>
  <div
    class="flex items-center gap-3 px-4 py-3 rounded-lg bg-card border-theme hover:border-brand-light transition-colors"
    :class="{ 'opacity-60': todo.completed }"
  >
    <!-- 复选框 -->
    <input
      type="checkbox"
      :checked="todo.completed"
      class="w-4 h-4 accent-brand cursor-pointer"
      @change="$emit('toggle', todo.id)"
    />

    <!-- 优先级标签 -->
    <span
      class="w-2 h-2 rounded-full flex-shrink-0"
      :class="priorityColor"
      :title="priorityLabel"
    />

    <!-- 标题 -->
    <span
      class="flex-1 text-sm text-primary"
      :class="{ 'line-through text-tertiary': todo.completed }"
    >
      {{ todo.title }}
    </span>

    <!-- 描述 -->
    <span
      v-if="todo.description"
      class="text-xs text-tertiary truncate max-w-[200px] hidden sm:block"
    >
      {{ todo.description }}
    </span>

    <!-- 时间 -->
    <span class="text-xs text-tertiary flex-shrink-0 hidden sm:block">
      {{ formatDate(todo.createdAt) }}
    </span>

    <!-- 删除按钮 -->
    <button
      class="text-tertiary hover:text-danger transition-colors flex-shrink-0 p-1"
      @click="$emit('delete', todo.id)"
      title="删除"
    >
      ✕
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Todo } from '../../../bindings/WeedyBox/internal/model'

const props = defineProps<{
  todo: Todo
}>()

const emit = defineEmits<{
  toggle: [id: number]
  delete: [id: number]
}>()

const priorityColor = computed(() => {
  const colors: Record<number, string> = {
    0: 'bg-green-500',
    1: 'bg-yellow-500',
    2: 'bg-red-500',
  }
  return colors[props.todo.priority] || 'bg-gray-500'
})

const priorityLabel = computed(() => {
  const labels: Record<number, string> = {
    0: '低优先级',
    1: '中优先级',
    2: '高优先级',
  }
  return labels[props.todo.priority] || '未知'
})

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  if (diff < 24 * 60 * 60 * 1000) {
    return '今天'
  }
  if (diff < 48 * 60 * 60 * 1000) {
    return '昨天'
  }
  if (diff < 7 * 24 * 60 * 60 * 1000) {
    return `${Math.floor(diff / (24 * 60 * 60 * 1000))}天前`
  }
  return date.toLocaleDateString('zh-CN')
}
</script>