<!-- frontend/src/components/todo/TodoItem.vue -->
<template>
  <label
    class="todo-item flex flex-col gap-1.5 px-4 py-3 rounded-lg bg-card border-theme hover:border-brand transition-colors cursor-pointer"
    :class="{ 'opacity-60': todo.completed }"
  >
    <!-- 主行：勾选 + 优先级 + 标题 + 时间 + 删除 -->
    <div class="flex items-center gap-3 w-full">
      <input
        type="checkbox"
        class="w-0 h-0"
        :checked="todo.completed"
        :disabled="isUpdating"
        @change="emit('toggle', props.todo.id)"
      />

      <!-- 优先级标签 -->
      <span
        class="w-3 h-3 rounded-full flex-shrink-0"
        :class="priorityColor"
        :title="priorityLabel"
      />

      <!-- 标题：主要信息 -->
      <span
        class="flex-1 min-w-0 truncate text-sm font-medium text-primary"
        :class="{ 'line-through text-tertiary': todo.completed }"
      >
        {{ todo.title }}
      </span>

      <!-- 时间 -->
      <span class="text-xs text-tertiary flex-shrink-0 hidden sm:block">
        {{ formatDate(todo.createdAt) }}
      </span>

      <!-- 删除按钮 - 需要阻止点击冒泡 -->
      <button
        class="btn-delete"
        style="border: 1px"
        @click="todoStore.deleteTodo(props.todo.id)"
        title="删除"
        :disabled="isDeleting"
      >
        <span v-if="isDeleting" class="inline-block animate-spin">⟳</span>
        <span v-else>✕</span>
      </button>
    </div>

    <!-- 描述：次要信息，缩进对齐标题并带左侧竖线 -->
    <p
      v-if="todo.description"
      class="todo-desc"
      :class="{ 'line-through': todo.completed }"
    >
      {{ todo.description }}
    </p>
  </label>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Todo } from '../../../bindings/WeedyBox/internal/model'
import { useTodoStore } from '../../stores/todoStore'

const props = defineProps<{ //只读属性
  todo: Todo
}>()

const emit = defineEmits<{
  toggle: [id: number]
  delete: [id: number]
  error: [error: Error]
}>()

const todoStore = useTodoStore()
const isUpdating = ref(false)
const isDeleting = ref(false)

// ===== Computed =====
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

// ===== Methods =====
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

<style scoped>
/* 描述：小字号 + 弱色 + 左侧竖线，与标题形成层级区分 */
.todo-desc {
  margin: 0;
  /* 24px = 勾选框(0) + gap3(12) + 圆点12 —— 与标题起始位置对齐 */
  padding-left: 24px;
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--color-text-tertiary);
  white-space: pre-wrap;
  word-break: break-word;
  border-left: 2px solid var(--color-border-light);
  transition: border-color 0.2s ease;
}

.todo-item:hover .todo-desc {
  border-left-color: var(--color-primary);
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.animate-spin {
  animation: spin 1s linear infinite;
}
</style>
