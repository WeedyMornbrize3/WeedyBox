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

    <!-- 工具行：筛选 / 删除，同级别；各自的下拉面板默认收起（渐进披露） -->
    <div v-if="hasTodos" class="flex flex-col flex-shrink-0">
      <div class="flex items-center gap-1">
        <button
          type="button"
          class="panel-toggle flex items-center gap-1.5 px-2 py-1 rounded-md text-xs cursor-pointer border-none bg-transparent hover:bg-hover transition-colors duration-200"
          :class="!isFilterOpen && 'text-tertiary'"
          :aria-expanded="isFilterOpen"
          aria-controls="todo-filter-panel"
          @click="toggleFilterPanel"
        >
          <span class="i-lucide-sliders-horizontal icon-xs" aria-hidden="true" />
          {{ isFilterOpen ? '收起筛选' : '筛选' }}
          <span
            v-if="activeFilterCount"
            class="ml-0.5 px-1.5 rounded-full text-[10px] font-medium bg-brand-light text-brand tabular-nums"
          >
            {{ activeFilterCount }}
          </span>
        </button>

        <button
          type="button"
          class="panel-toggle flex items-center gap-1.5 px-2 py-1 rounded-md text-xs cursor-pointer border-none bg-transparent hover:bg-hover transition-colors duration-200"
          :class="isDeleteOpen ? 'text-danger' : 'text-tertiary'"
          :aria-expanded="isDeleteOpen"
          aria-controls="todo-delete-panel"
          @click="toggleDeletePanel"
        >
          <span class="i-lucide-trash-2 icon-xs" aria-hidden="true" />
          {{ isDeleteOpen ? '收起删除' : '删除' }}
          <span
            v-if="selectedIds.length"
            class="ml-0.5 px-1.5 rounded-full text-[10px] font-medium bg-brand-light text-danger tabular-nums"
          >
            {{ selectedIds.length }}
          </span>
        </button>
      </div>

      <!-- ===== 筛选面板 ===== -->
      <Transition
        name="collapse"
        @enter="collapseEnter"
        @after-enter="collapseAfterEnter"
        @leave="collapseLeave"
        @after-leave="collapseAfterLeave"
      >
        <div v-show="isFilterOpen" id="todo-filter-panel">
          <FilterBar class="mt-2 flex-shrink-0 scrollbar-theme" />
        </div>
      </Transition>

      <!-- ===== 删除面板：先选范围（可再手动勾选目标），确认后一次删除 ===== -->
      <Transition
        name="collapse"
        @enter="collapseEnter"
        @after-enter="collapseAfterEnter"
        @leave="collapseLeave"
        @after-leave="collapseAfterLeave"
      >
        <div v-show="isDeleteOpen" id="todo-delete-panel" class="mt-2 rounded-lg bg-secondary-soft border-theme-light p-3 flex flex-col gap-2.5">
          <!-- 范围预设：点一下即选中该范围内的全部条目 -->
          <div class="flex flex-wrap items-center gap-1.5">
            <span class="text-xs text-tertiary mr-0.5">范围</span>
            <button
              v-for="scope in DELETE_SCOPES"
              :key="scope.value"
              type="button"
              class="chip"
              :class="{ 'chip-active': deleteScope === scope.value }"
              :aria-pressed="deleteScope === scope.value"
              @click="applyScope(scope.value)"
            >
              {{ scope.label }}
            </button>
          </div>

          <!-- 目标清单：在所选范围内自由勾选 -->
          <div class="flex flex-col gap-1">
            <div class="flex items-center justify-between gap-2">
              <span class="text-xs text-tertiary">
                {{ scopeLabel }} · 共 {{ scopedTodos.length }} 项
              </span>
              <div v-if="scopedTodos.length" class="flex items-center gap-2">
                <button type="button" class="link-btn" @click="selectAllScoped">全选</button>
                <button type="button" class="link-btn" @click="clearSelection">清空</button>
              </div>
            </div>

            <p v-if="!scopedTodos.length" class="m-0 text-xs text-muted py-1">
              该范围内没有 TODO
            </p>

            <!-- 条目多时内部滚动，面板高度保持稳定 -->
            <ul v-else class="delete-list m-0 p-0 list-none flex flex-col gap-0.5">
              <li v-for="todo in scopedTodos" :key="todo.id">
                <label class="flex items-center gap-2 px-2 py-1 rounded-md cursor-pointer hover:bg-hover transition-colors duration-200">
                  <input
                    type="checkbox"
                    class="delete-check"
                    :checked="selectedIds.includes(todo.id)"
                    @change="toggleSelected(todo.id)"
                  />
                  <span class="flex-1 min-w-0 truncate text-xs" :class="todo.completed ? 'text-tertiary line-through' : 'text-secondary'">
                    {{ todo.title }}
                  </span>
                  <span class="flex-shrink-0 text-[10px] text-muted">{{ priorityText(todo.priority) }}</span>
                </label>
              </li>
            </ul>
          </div>

          <!-- 确认区 -->
          <div class="flex items-center gap-2 pt-1 border-t border-theme-light">
            <span class="text-xs text-tertiary">
              待删除 <span class="font-semibold text-danger tabular-nums">{{ selectedIds.length }}</span> 项
            </span>
            <button
              type="button"
              class="ml-auto btn-danger px-3 py-1.5 rounded-md text-xs font-medium cursor-pointer border-none transition-colors duration-200 disabled:opacity-40 disabled:cursor-not-allowed"
              :disabled="!selectedIds.length || deleting"
              @click="confirmDelete"
            >
              {{ deleting ? '删除中…' : confirmLabel }}
            </button>
          </div>
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

// ===== 渐进披露：两个面板默认收起 =====
const TOOL_PANEL_KEY = 'todo-tool-panel'
const isFilterOpen = ref(false)
const isDeleteOpen = ref(false)

onMounted(() => {
  const saved = localStorage.getItem(TOOL_PANEL_KEY)
  isFilterOpen.value = saved === 'filter'
  isDeleteOpen.value = saved === 'delete'
})

function persistPanelState() {
  localStorage.setItem(TOOL_PANEL_KEY, isDeleteOpen.value ? 'delete' : isFilterOpen.value ? 'filter' : '')
}

// 同一行上的两个面板互斥展开：避免两段内容同时撑开、把列表挤得很短
function toggleFilterPanel() {
  isFilterOpen.value = !isFilterOpen.value
  if (isFilterOpen.value) {
    isDeleteOpen.value = false
    resetDeletePanel()
  }
  persistPanelState()
}

function toggleDeletePanel() {
  isDeleteOpen.value = !isDeleteOpen.value
  if (isDeleteOpen.value) {
    isFilterOpen.value = false
    // 打开时默认选中「已完成」——批量删除里最常用的诉求
    applyScope('completed')
  } else {
    resetDeletePanel()
  }
  persistPanelState()
}

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

// ===== 删除指定范围 =====
// 「自由选择」的落点：范围只决定候选集，候选集里的每一条都可再手动勾选。
const DELETE_SCOPES = [
  { value: 'completed', label: '已完成' },
  { value: 'pending', label: '未完成' },
  { value: 'priority-high', label: '高优先级' },
  { value: 'priority-medium', label: '中优先级' },
  { value: 'priority-low', label: '低优先级' },
  { value: 'all', label: '全部' },
] as const

type DeleteScope = (typeof DELETE_SCOPES)[number]['value']

const deleteScope = ref<DeleteScope>('completed')
const selectedIds = ref<number[]>([])
const deleting = ref(false)

const scopeLabel = computed(
  () => DELETE_SCOPES.find((s) => s.value === deleteScope.value)?.label ?? ''
)

const scopedTodos = computed(() => {
  const all = todoStore.todos
  switch (deleteScope.value) {
    case 'completed':
      return all.filter((t) => t.completed)
    case 'pending':
      return all.filter((t) => !t.completed)
    case 'priority-high':
      return all.filter((t) => t.priority === 2)
    case 'priority-medium':
      return all.filter((t) => t.priority === 1)
    case 'priority-low':
      return all.filter((t) => t.priority === 0)
    default:
      return all
  }
})

const confirmLabel = computed(() => {
  const n = selectedIds.value.length
  if (n === 0) return '确认删除'
  // 命中范围内全部条目时提示「全部」，否则提示数量，避免误以为只删一条
  return n === scopedTodos.value.length && n > 1 ? `删除全部 ${n} 项` : `删除 ${n} 项`
})

function applyScope(scope: DeleteScope) {
  deleteScope.value = scope
  selectedIds.value = scopedTodos.value.map((t) => t.id)
}

function toggleSelected(id: number) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter((x) => x !== id)
    : [...selectedIds.value, id]
}

const selectAllScoped = () => {
  selectedIds.value = scopedTodos.value.map((t) => t.id)
}
const clearSelection = () => {
  selectedIds.value = []
}

function resetDeletePanel() {
  selectedIds.value = []
  deleteScope.value = 'completed'
}

// 优先级中文标签（与 TodoItem 的三重编码一致：0 低 / 1 中 / 2 高）
const PRIORITY_TEXT: Record<number, string> = { 0: '低', 1: '中', 2: '高' }
const priorityText = (p: number) => PRIORITY_TEXT[p] ?? ''

async function confirmDelete() {
  const ids = [...selectedIds.value]
  if (!ids.length || deleting.value) return
  deleting.value = true
  try {
    const n = await todoStore.deleteMany(ids)
    showToast('success', `已删除 ${n} 项`)
    resetDeletePanel()
    // 删空之后整块工具行会消失，这里顺手收起面板状态
    if (todoStore.todos.length === 0) {
      isDeleteOpen.value = false
      persistPanelState()
    } else {
      // 范围内可能还有剩余条目，重新按当前范围选中，方便连续清理
      applyScope(deleteScope.value)
    }
  } catch (e) {
    showToast('error', e instanceof Error ? e.message : '批量删除失败')
  } finally {
    deleting.value = false
  }
}

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

/* ===== 删除面板 ===== */
/* 面板内文字按钮（全选/清空）：低调的链接式按钮 */
.link-btn {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--color-primary);
  font-size: 11px;
  cursor: pointer;
  transition: opacity var(--motion-fast) var(--ease-enter);
}

.link-btn:hover {
  opacity: 0.75;
}

/* 目标清单：条目多时内部滚动，避免面板把列表挤扁 */
.delete-list {
  max-height: 9.5rem;
  overflow-y: auto;
}

/* 勾选框：沿用与列表行一致的外观，但尺寸更小以匹配紧凑行高 */
.delete-check {
  flex-shrink: 0;
  width: 14px;
  height: 14px;
  margin: 0;
  border: 1.5px solid var(--color-border-dark);
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color var(--motion-fast) var(--ease-enter),
              border-color var(--motion-fast) var(--ease-enter);
}

.delete-check:checked {
  background-color: var(--color-bg-check);
  border-color: var(--color-bg-check);
}

.delete-check:checked::after {
  content: '';
  width: 8px;
  height: 8px;
  background-color: var(--color-check-mark);
  -webkit-mask: var(--icon-check) center / contain no-repeat;
  mask: var(--icon-check) center / contain no-repeat;
}

.delete-check:focus-visible {
  outline: 2px solid var(--color-ring);
  outline-offset: 2px;
  opacity: 1;
}

/* 确认删除按钮：唯一的实心强调色按钮，位置固定在面板右下 */
.btn-danger {
  background-color: var(--color-danger);
  color: #ffffff;
}

.btn-danger:hover:not(:disabled) {
  opacity: 0.88;
}

.btn-danger:disabled {
  cursor: not-allowed;
}
</style>
