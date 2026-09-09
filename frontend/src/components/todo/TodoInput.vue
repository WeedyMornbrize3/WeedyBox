<!-- frontend/src/components/todo/TodoInput.vue -->
<template>
  <div class="flex gap-2 mb-4">
    <input
      v-model="newTodo"
      type="text"
      placeholder="添加新的 TODO..."
      class=" flex-1 px-4 py-2 rounded-[10px] text-sm outline-none transition-all duration-250 bg-secondary text-primary placeholder-tertiary border-theme hover:border-brand focus:border-brand focus:shadow-[0_0_0_1px_var(--color-primary-light)]"
      @keyup.enter="handleSubmit"
    />
    <button
      :disabled="!newTodo.trim() || loading"
      class="px-5 py-2 rounded-[10px] text-sm font-medium cursor-pointer transition-all duration-250 bg-brand text-inverse hover:bg-brand-hover border-theme hover:border-brand-hover hover:scale-[1.03] active:scale-[0.97] disabled:opacity-50 disabled:cursor-not-allowed"
      @click="handleSubmit"
    >
      {{ loading ? '添加中...' : '添加' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useTodoStore } from '../../stores/todoStore'

const todoStore = useTodoStore()
const newTodo = ref('')
const loading = ref(false)

const handleSubmit = async () => {
  const title = newTodo.value.trim()
  if (!title) return

  loading.value = true
  try {
    // todoStore.curPriority = 0  占位  可修改了
    await todoStore.addTodo(title,'')  //占位
    newTodo.value = ''
  } catch (e) {
    // 错误已在 store 中处理
  } finally {
    loading.value = false
  }
}
</script>