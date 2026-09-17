<!-- frontend/src/components/todo/TodoList.vue -->
<!-- 排版取向（Minimalism & Swiss）：统一间距节奏、去掉多余装饰（无投影/渐变）、
     只保留必要元素；统计与筛选默认收起，需要时再展开（渐进披露）。 -->
<template>
  <div class="todo-container select-none h-full min-h-0 flex flex-col gap-3">
    <!-- 页头 -->
    <header class="flex items-center gap-2 flex-shrink-0">
      <span class="i-lucide-list-todo icon-lg text-brand" aria-hidden="true" />
      <h2 class="text-xl font-semibold text-primary m-0">TODO 列表</h2>
    </header>

    <!-- 统计 + 新建优先级（有数据时才出现；无数据时页头直接把注意力导向输入框） -->
    <!-- ⚠️ 这里不能让 TodoStats 不可收缩（曾写 flex-shrink-0）：
         统计栏固有宽度约 271px，优先级选择器约 172px，两者都不可收缩时
         在容器 <455px（实测 380~452px）下合计超出容器宽度，
         统计栏内部的 flex-wrap 永远得不到触发，选择器被顶出卡片右边框。
         改为 flex-1 min-w-0：统计栏吃掉剩余空间、可收缩，空间不足时在内部换行。 -->
    <Transition name="fade-rise">
      <div v-if="hasTodos" class="flex items-center gap-3 flex-shrink-0">
        <TodoStats :stats="todoStore.stats" class="flex-1 min-w-0" />
        <TodoPrioritySelector class="flex-shrink-0" />
      </div>
    </Transition>

    <!-- 输入框：常驻，不藏（藏起来反而多一次点击） -->
    <TodoInput class="flex-shrink-0" />

    <!-- 高级筛选：默认收起，点「筛选」才展开 -->
    <div v-if="hasTodos" class="flex flex-col flex-shrink-0">
      <button
        type="button"
        class="filter-toggle self-start flex items-center gap-1.5 px-2 py-1 -ml-2 rounded-md text-xs cursor-pointer border-none bg-transparent hover:bg-hover transition-colors duration-200"
        :class="!isFilterActive && 'text-tertiary'"
        :aria-expanded="isFilterActive"
        aria-controls="todo-filter-panel"
        @click="isFilterActive = !isFilterActive"
      >
        <span class="i-lucide-sliders-horizontal icon-xs" aria-hidden="true" />
        {{ isFilterActive ? '收起筛选' : '筛选' }}
        <span
          v-if="activeFilterCount"
          class="ml-0.5 px-1.5 rounded-full text-[10px] font-medium bg-brand-light text-brand tabular-nums"
        >
          {{ activeFilterCount }}
        </span>
      </button>

      <Transition
        name="collapse"
        @enter="collapseEnter"
        @after-enter="collapseAfterEnter"
        @leave="collapseLeave"
        @after-leave="collapseAfterLeave"
      >
        <div v-show="isFilterActive" id="todo-filter-panel">
          <FilterBar class="mt-2 flex-shrink-0 scrollbar-theme" />
        </div>
      </Transition>
    </div>

    <!-- 操作反馈：短暂显示后自动消失，role=status 让读屏软件播报 -->
    <Transition name="toast">
      <div
        v-if="toast"
        class="flex items-center gap-2 px-3 py-2 rounded-lg text-xs border-theme-light bg-secondary-soft flex-shrink-0"
        :class="toast.kind === 'error' ? 'text-danger' : 'text-success'"
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
    <div class="card p-2 flex-1 min-h-[120px] overflow-y-auto scrollbar-theme flex flex-col gap-2">
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
      <div v-else-if="!hasTodos" class="flex-1 flex-col-center py-8 text-tertiary gap-2">
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

      <!-- TODO 列表：用 TransitionGroup 做进出场，删除/新增不再生硬跳变 -->
      <TransitionGroup v-else name="list" tag="div" class="relative flex flex-col gap-2">
        <TodoItem
          v-for="todo in todoStore.filteredTodos"
          :key="todo.id"
          :todo="todo"
          :is-updating="busyIds.has(todo.id)"
          :is-deleting="busyIds.has(todo.id)"
          @toggle="handleToggle"
          @delete="handleDelete"
        />
      </TransitionGroup>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useTodoStore } from '../../stores/todoStore'
import TodoInput from './TodoInput.vue'
import TodoItem from './TodoItem.vue'
import TodoStats from './TodoStats.vue'
import TodoPrioritySelector from './TodoPrioritySelector.vue'
import FilterBar from './FilterBar.vue'
import {
  collapseEnter,
  collapseAfterEnter,
  collapseLeave,
  collapseAfterLeave,
} from '../../composables/useAutoHeight'

const todoStore = useTodoStore()

const hasTodos = computed(() => todoStore.todos.length > 0)

// ===== 渐进披露：筛选面板默认收起 =====
const FILTER_PANEL_KEY = 'todo-filter-panel'
const isFilterActive = ref(false)

onMounted(() => {
  isFilterActive.value = localStorage.getItem(FILTER_PANEL_KEY) === 'true'
})
watch(isFilterActive, (v) => localStorage.setItem(FILTER_PANEL_KEY, String(v)))

// 处于非默认筛选时，即使面板收起也显示条件数量，
// 避免「看不见的筛选」让用户以为数据丢了。
const activeFilterCount = computed(() => {
  const f = todoStore.filters
  let n = 0
  if (f.completed !== 'pending') n++
  if (f.priority !== null) n++
  if (f.time !== 'all') n++
  return n
})

// ===== 处理中的条目 =====
// ⚠️ 每次变更都整体换一个新 Set：ref 内部的 Set 原地 add/delete 不会触发模板更新。
const busyIds = ref<Set<number>>(new Set())

function setBusy(id: number, busy: boolean) {
  const next = new Set(busyIds.value)
  if (busy) next.add(id)
  else next.delete(id)
  busyIds.value = next
}

// ===== 轻量反馈条 =====
// 只显示「已完成/已删除」这类一次性结果，长时间停留的错误统一由 store.error 渲染在列表区。
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

// 删除不再弹确认框：确认动作已由 TodoItem 的「连点两下」承担
// （第一下进入待确认态，第二下才 emit delete）。
const handleDelete = (id: number) => {
  runWithBusy(id, () => todoStore.deleteTodo(id), '已删除')
}

onMounted(() => {
  todoStore.init()
})

onBeforeUnmount(() => {
  clearTimeout(toastTimer)
})
</script>

<style scoped>
/* ===== 渐进披露：展开/收起 =====
   高度由 useAutoHeight 的 hooks 用 JS 测量（CSS 无法过渡到 height:auto），
   这里只补透明度，让显现更柔和。时长/缓动统一走 global.css 的动效 token。 */
.collapse-enter-from,
.collapse-leave-to {
  opacity: 0;
}

.collapse-enter-active,
.collapse-leave-active {
  transition: opacity var(--motion-slow) var(--ease-enter);
}

/* ===== 统计栏：有数据时才出现 ===== */
.fade-rise-enter-active,
.fade-rise-leave-active {
  transition: opacity var(--motion-base) var(--ease-enter),
              transform var(--motion-base) var(--ease-enter);
}

.fade-rise-enter-from,
.fade-rise-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* ===== 列表项进出场 ===== */
.list-enter-active {
  transition: opacity var(--motion-base) var(--ease-enter),
              transform var(--motion-base) var(--ease-enter);
}

.list-leave-active {
  transition: opacity var(--motion-fast) var(--ease-leave),
              transform var(--motion-fast) var(--ease-leave);
  /* 退场时脱离文档流，避免剩余项被撑住后突然回弹 */
  position: absolute;
  width: 100%;
}

.list-enter-from {
  opacity: 0;
  transform: translateY(-6px);
}

.list-leave-to {
  opacity: 0;
  transform: translateX(8px);
}

/* 位移中的项不接收点击 */
.list-leave-active,
.list-enter-active {
  pointer-events: none;
}

/* ===== 反馈条 ===== */
.toast-enter-active,
.toast-leave-active {
  transition: opacity var(--motion-base) var(--ease-enter),
              transform var(--motion-base) var(--ease-enter);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
