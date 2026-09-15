<!-- frontend/src/components/todo/TodoList.vue -->
<template>
  <div class="todo-container select-none h-full min-h-0 flex flex-col">
    <!-- 页头 -->
    <header class="flex items-center gap-2 flex-shrink-0">
      <span class="i-lucide-list-todo icon-lg text-brand" aria-hidden="true" />
      <h2 class="text-xl font-semibold text-primary m-0">TODO 列表</h2>
    </header>

    <!-- 统计 + 新建优先级 -->
    <div class="flex items-center gap-3 mt-3 flex-shrink-0">
      <TodoStats :stats="todoStore.stats" class="flex-shrink-0" />
      <TodoPrioritySelector class="flex-shrink-0" />
    </div>

    <!-- 输入框（描述框展开时高度自增，下方卡片自动让位） -->
    <TodoInput class="mt-3 flex-shrink-0" />

    <!-- 筛选栏（横向滚动，不换行） -->
    <FilterBar class="mt-3 flex-shrink-0 scrollbar-theme" />

    <!-- 操作反馈：短暂显示后自动消失，role=status 让读屏软件播报 -->
    <Transition name="toast">
      <div
        v-if="toast"
        class="mt-2 flex items-center gap-2 px-3 py-2 rounded-lg text-xs border flex-shrink-0"
        :class="toast.kind === 'error'
          ? 'text-danger border-theme-light bg-secondary-soft'
          : 'text-success border-theme-light bg-secondary-soft'"
        role="status"
        aria-live="polite"
      >
        <span :class="toast.kind === 'error' ? 'i-lucide-circle-alert icon-sm' : 'i-lucide-circle-check-big icon-sm'" aria-hidden="true" />
        {{ toast.text }}
      </div>
    </Transition>

    <!-- 列表卡片：flex-1 吃掉剩余高度并内部滚动 -->
    <!-- 不要用 max-h-[calc(100vh-Npx)] 固定扣减：那个 N 算不准，且描述框一展开卡片又被下推，
         导致最后一条滚不到底。交给 flex 自适应即无需任何魔法数字。 -->
    <div class="card mt-3 mb-2 p-2 flex-1 min-h-[120px] overflow-y-auto scrollbar-theme flex flex-col gap-2">
      <!-- 加载状态 -->
      <div v-if="todoStore.loading" class="flex-1 flex-col-center py-8 text-tertiary text-sm gap-2">
        <span class="i-lucide-loader-circle icon-lg spin" aria-hidden="true" />
        加载中…
      </div>

      <!-- 错误 -->
      <div v-else-if="todoStore.error" class="flex-1 flex-col-center py-8 text-danger text-sm gap-2" role="alert">
        <span class="i-lucide-circle-alert icon-lg" aria-hidden="true" />
        <span>{{ todoStore.error }}</span>
        <button type="button" class="chip mt-1" @click="todoStore.loadTodos()">
          <span class="i-lucide-rotate-ccw icon-xs" aria-hidden="true" />
          重新加载
        </button>
      </div>

      <!-- 空状态：本来就没有数据 -->
      <div v-else-if="todoStore.todos.length === 0" class="flex-1 flex-col-center py-8 text-tertiary gap-2">
        <span class="i-lucide-inbox icon-xl text-muted" aria-hidden="true" />
        <p class="m-0 text-sm">还没有 TODO</p>
        <p class="m-0 text-xs text-muted">在上方输入框写下第一件事，按 Enter 即可添加</p>
      </div>

      <!-- 空状态：被筛选条件挡住 -->
      <div v-else-if="todoStore.isFilteredEmpty" class="flex-1 flex-col-center py-8 text-tertiary gap-2">
        <span class="i-lucide-search-x icon-xl text-muted" aria-hidden="true" />
        <p class="m-0 text-sm">当前筛选条件下没有匹配的 TODO</p>
        <button type="button" class="chip mt-1" @click="todoStore.resetFilters()">
          <span class="i-lucide-rotate-ccw icon-xs" aria-hidden="true" />
          恢复默认筛选
        </button>
      </div>

      <!-- TODO 列表 -->
      <template v-else>
        <TodoItem
          v-for="todo in todoStore.filteredTodos"
          :key="todo.id"
          :todo="todo"
          :is-updating="busyIds.has(todo.id)"
          :is-deleting="busyIds.has(todo.id)"
          @toggle="handleToggle"
          @delete="handleDelete"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue'
import { useTodoStore } from '../../stores/todoStore'
import TodoInput from './TodoInput.vue'
import TodoItem from './TodoItem.vue'
import TodoStats from './TodoStats.vue'
import TodoPrioritySelector from './TodoPrioritySelector.vue'
import FilterBar from './FilterBar.vue'

const todoStore = useTodoStore()

// 正在处理中的条目 id（用于禁用该行的勾选框/删除按钮，避免重复提交）
// ⚠️ 每次变更都整体换一个新 Set：ref 内部的 Set 原地 add/delete 不会触发模板更新。
const busyIds = ref<Set<number>>(new Set())

function setBusy(id: number, busy: boolean) {
  const next = new Set(busyIds.value)
  if (busy) next.add(id)
  else next.delete(id)
  busyIds.value = next
}

// 轻量反馈条：只显示「已完成/已删除」这类一次性结果，
// 长时间停留的错误统一由 store.error 渲染在列表区。
const toast = ref<{ kind: 'error' | 'success'; text: string } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | undefined

function showToast(kind: 'error' | 'success', text: string) {
  toast.value = { kind, text }
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = null), 2400)
}

async function runWithBusy(id: number, action: () => Promise<unknown>, okText: string) {
  if (busyIds.value.has(id)) return
  setBusy(id, true)
  try {
    await action()
    showToast('success', okText)
  } catch (e) {
    showToast('error', e instanceof Error ? e.message : '操作失败')
  } finally {
    setBusy(id, false)
  }
}

const handleToggle = (id: number) =>
  runWithBusy(id, () => todoStore.toggleComplete(id), '已更新完成状态')

const handleDelete = (id: number) => {
  if (confirm('确定要删除这个 TODO 吗？')) {
    runWithBusy(id, () => todoStore.deleteTodo(id), '已删除')
  }
}

onMounted(() => {
  todoStore.init()
})

onBeforeUnmount(() => {
  clearTimeout(toastTimer)
})
</script>

<style scoped>
/* 反馈条进出场：只做透明度 + 轻微位移，不改变布局高度之外的任何东西 */
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
