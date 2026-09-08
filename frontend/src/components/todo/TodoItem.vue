<!-- frontend/src/components/todo/TodoItem.vue -->
<template>
  <div
    class="flex items-center gap-3 px-4 py-3 rounded-lg bg-card border-theme hover:border-brand-light transition-colors"
    :class="{ 'opacity-60': todo.completed }"
  >
    <input
      type="checkbox"
      class="w-4 h-4 accent-brand cursor-pointer"
      :checked="todo.completed"
      :disabled="isUpdating"
      @change="emit('toggle', props.todo.id)"
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
      class="btn-brand"
      @click="handleDelete"
      title="删除"
      :disabled="isDeleting"
    >
      <span v-if="isDeleting" class="inline-block animate-spin">⟳</span>
      <span v-else>✕</span>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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

const handleDelete = async () => {
  if (isDeleting.value) return
  
  if (!confirm(`确定要删除 "${props.todo.title}" 吗？`)) {
    return
  }
  
  try {
    isDeleting.value = true
    await todoStore.deleteTodo(props.todo.id)
    emit('delete', props.todo.id)
  } catch (error) {
    console.error('删除失败:', error)
    emit('error', error as Error)
  } finally {
    isDeleting.value = false
  }
}

</script>

<style scoped>
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
.animate-spin {
  animation: spin 1s linear infinite;
}
</style>