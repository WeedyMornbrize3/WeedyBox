<!-- frontend/src/components/todo/TodoInput.vue -->
<template>
  <div class="flex flex-col gap-2 mb-4">
    <!-- 标题行：输入框 + 描述开关 + 添加按钮 -->
    <div class="flex gap-2">
      <input
        v-model="newTodo"
        type="text"
        :placeholder="showDesc ? '添加新的 TODO 标题...' : '添加新的 TODO...'"
        class="flex-1 px-4 py-2 rounded-[10px] text-sm outline-none transition-all duration-250 bg-secondary text-primary placeholder-tertiary border-theme hover:border-brand focus:border-brand focus:shadow-[0_0_0_1px_var(--color-primary-light)]"
        @keyup.enter="handleSubmit"
      />

      <!-- 描述开关：有内容时高亮 -->
      <button
        type="button"
        class="desc-toggle"
        :class="{ 'desc-toggle-active': showDesc || desc.trim() }"
        :title="showDesc ? '收起描述' : '添加描述'"
        @click="toggleDesc"
      >
        📝
      </button>

      <button
        :disabled="!newTodo.trim() || loading"
        class="px-5 py-2 rounded-[10px] text-sm font-medium cursor-pointer transition-all duration-250 bg-brand text-inverse hover:bg-brand-hover border-theme hover:border-brand-hover hover:scale-[1.03] active:scale-[0.97] disabled:opacity-50 disabled:cursor-not-allowed"
        @click="handleSubmit"
      >
        {{ loading ? '添加中...' : '添加' }}
      </button>
    </div>

    <!-- 描述输入框：点击 📝 后展开 -->
    <div v-show="showDesc" class="flex gap-2">
      <input
        ref="descInput"
        v-model="desc"
        type="text"
        placeholder="添加描述（可选）..."
        class="flex-1 px-4 py-2 rounded-[10px] text-xs outline-none transition-all duration-250 bg-secondary text-secondary placeholder-tertiary border-theme hover:border-brand focus:border-brand focus:shadow-[0_0_0_1px_var(--color-primary-light)]"
        @keyup.enter="handleSubmit"
        @keyup.esc="toggleDesc"
      />
      <!-- 占位：与上方「开关 + 添加」总宽度对齐 -->
      <div class="w-[8.25rem] flex-shrink-0" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useTodoStore } from '../../stores/todoStore'

const todoStore = useTodoStore()
const newTodo = ref('')
const desc = ref('')
const showDesc = ref(false)
const loading = ref(false)
const descInput = ref<HTMLInputElement | null>(null)

const toggleDesc = async () => {
  showDesc.value = !showDesc.value
  if (showDesc.value) {
    await nextTick()
    descInput.value?.focus()
  } else {
    desc.value = ''
  }
}

const handleSubmit = async () => {
  const title = newTodo.value.trim()
  if (!title) return

  loading.value = true
  try {
    await todoStore.addTodo(title, desc.value.trim())
    newTodo.value = ''
    desc.value = ''
  } catch (e) {
    // 错误已在 store 中处理
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.desc-toggle {
  flex-shrink: 0;
  width: 2.25rem;
  font-size: 0.875rem;
  line-height: 1;
  border-radius: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  color: var(--color-text-tertiary);
  transition: all 0.25s ease;
}

.desc-toggle:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  transform: scale(1.05);
}

.desc-toggle:active {
  transform: scale(0.95);
}

.desc-toggle-active {
  border-color: var(--color-primary);
  background: var(--color-primary-light);
  color: var(--color-primary);
}
</style>
