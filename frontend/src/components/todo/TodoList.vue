<!-- frontend/src/components/todo/TodoList.vue -->
<!-- 排版取向（Minimalism & Swiss）：统一间距节奏、去掉多余装饰（无投影/渐变）、
     只保留必要元素；筛选默认收起，批量操作只在有选中时才出现（渐进披露）。

     批量操作模型：
       行的左侧选择框负责「选中哪些」，工具栏右侧的
       「完成全部 / 删除全部」对选中集合执行动作。
       两个按钮都要求**连点两下**（第一下进入待确认态，第二下才执行），
       并且不会弹任何确认框或展开额外面板。 -->
<template>
  <div class="todo-container select-none h-full min-h-0 flex flex-col gap-3">
    <!-- 页头 -->
    <header class="flex items-center gap-2 flex-shrink-0">
      <span class="i-lucide-list-todo icon-lg text-brand" aria-hidden="true" />
      <h2 class="text-xl font-semibold text-primary m-0">TODO 列表</h2>
    </header>

    <!-- 统计 + 新建优先级（有数据时才出现） -->
    <Transition name="fade-rise">
      <div v-if="hasTodos" class="flex items-center gap-3 flex-shrink-0">
        <TodoStats :stats="todoStore.stats" class="flex-1 min-w-0" />
        <TodoPrioritySelector class="flex-shrink-0" />
      </div>
    </Transition>

    <!-- 输入框：常驻，不藏（藏起来反而多一次点击） -->
    <TodoInput class="flex-shrink-0" />

    <!-- 工具行：左「筛选」，右批量操作（有选中才出现） -->
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

        <!-- 全选当前可见（键盘可达的替代路径：逐个点行内选择框亦可） -->
        <button
          v-if="visibleIds.length"
          type="button"
          class="panel-toggle flex items-center gap-1.5 px-2 py-1 rounded-md text-xs cursor-pointer border-none bg-transparent hover:bg-hover transition-colors duration-200"
          :class="allVisibleSelected ? 'text-brand' : 'text-tertiary'"
          :aria-pressed="allVisibleSelected"
          @click="toggleSelectAll"
        >
          <span :class="allVisibleSelected ? 'i-lucide-square-check-big icon-xs' : 'i-lucide-square icon-xs'" aria-hidden="true" />
          {{ allVisibleSelected ? '取消全选' : '全选' }}
        </button>

        <!-- 批量操作：对「已选中」执行。没有选中时不渲染，避免一排禁用按钮占位 -->
        <div v-if="selectedIds.length" class="ml-auto flex items-center gap-1.5">
          <button
            type="button"
            class="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium cursor-pointer border border-solid transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
            :class="armed === 'complete' ? 'btn-success' : 'btn-complete'"
            :disabled="completing"
            :aria-label="completing
              ? (uncompleteMode ? '正在取消完成' : '正在完成')
              : armed === 'complete'
                ? `再次点击确认将这 ${selectedIds.length} 项${completeActionLabel}`
                : `将选中的 ${selectedIds.length} 项${completeActionLabel}（需连点两下确认）`"
            @click="handleCompleteClick"
            @mouseleave="disarm"
            @blur="disarm"
          >
            <span
              class="icon-xs"
              :class="completing
                ? 'i-lucide-loader-circle spin'
                : armed === 'complete'
                  ? 'i-lucide-check'
                  : uncompleteMode ? 'i-lucide-circle' : 'i-lucide-circle-check-big'"
              aria-hidden="true"
            />
            {{ completing ? (uncompleteMode ? '取消中…' : '完成中…') : armed === 'complete' ? '再点一次确认' : completeActionLabel }}
            <span class="px-1.5 rounded-full bg-white/25 text-[10px] tabular-nums">{{ selectedIds.length }}</span>
          </button>

          <button
            type="button"
            class="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium cursor-pointer border border-solid transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed"
            :class="armed === 'delete' ? 'btn-danger-armed' : 'btn-danger'"
            :disabled="deleting"
            :aria-label="deleting
              ? '正在删除'
              : armed === 'delete'
                ? `再次点击确认删除这 ${selectedIds.length} 项`
                : `删除选中的 ${selectedIds.length} 项（需连点两下确认）`"
            @click="handleDeleteClick"
            @mouseleave="disarm"
            @blur="disarm"
          >
            <span
              :class="deleting ? 'i-lucide-loader-circle icon-xs spin' : armed === 'delete' ? 'i-lucide-check icon-xs' : 'i-lucide-trash-2 icon-xs'"
              aria-hidden="true"
            />
            {{ deleting ? '删除中…' : armed === 'delete' ? '再点一次确认' : '删除全部' }}
            <span class="px-1.5 rounded-full bg-white/25 text-[10px] tabular-nums">{{ selectedIds.length }}</span>
          </button>
        </div>
      </div>

      <!-- 筛选面板（唯一的下拉面板） -->
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
          :selected="selectedIds.includes(todo.id)"
          :is-updating="busyIds.has(todo.id)"
          :is-deleting="busyIds.has(todo.id)"
          @select="toggleSelected"
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

// ===== 筛选面板（默认收起） =====
const TOOL_PANEL_KEY = 'todo-tool-panel'
const isFilterOpen = ref(false)

onMounted(() => {
  isFilterOpen.value = localStorage.getItem(TOOL_PANEL_KEY) === 'filter'
})

function toggleFilterPanel() {
  isFilterOpen.value = !isFilterOpen.value
  localStorage.setItem(TOOL_PANEL_KEY, isFilterOpen.value ? 'filter' : '')
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

// ===== 选择集合 =====
const selectedIds = ref<number[]>([])

/** 当前列表里实际可见（经筛选后）的条目 id —— 用于「全选」 */
const visibleIds = computed(() => todoStore.filteredTodos.map((t) => t.id))

const allVisibleSelected = computed(
  () => visibleIds.value.length > 0 && visibleIds.value.every((id) => selectedIds.value.includes(id))
)

function toggleSelected(id: number) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter((x) => x !== id)
    : [...selectedIds.value, id]
  disarm()
}

const toggleSelectAll = () => {
  if (allVisibleSelected.value) {
    // 只取消当前可见的部分，保留其它范围内的选择
    const visible = new Set(visibleIds.value)
    selectedIds.value = selectedIds.value.filter((id) => !visible.has(id))
  } else {
    selectedIds.value = [...new Set([...selectedIds.value, ...visibleIds.value])]
  }
  disarm()
}

const clearSelection = () => {
  selectedIds.value = []
  disarm()
}

// ===== 批量操作：连点两下 =====
// 同一个时刻只允许一个按钮处于待确认态，避免「以为在确认完成、其实点的是删除」。
const armed = ref<'complete' | 'delete' | null>(null)
const completing = ref(false)
const deleting = ref(false)

// 在「已完成」筛选下，列表里看到的都是已完成项，
// 此时按钮语义反转为「取消完成」——把选中项改回未完成。
const uncompleteMode = computed(() => todoStore.filters.completed === 'completed')

const completeActionLabel = computed(() =>
  uncompleteMode.value ? '取消完成' : '标记为已完成'
)

const disarm = () => {
  armed.value = null
}

// 选择集合一变就撤销待确认态：避免「看到的是 A，确认时动的是 B」
watch(selectedIds, disarm)

function handleCompleteClick() {
  if (completing.value || !selectedIds.value.length) return
  if (armed.value !== 'complete') {
    armed.value = 'complete'
    return
  }
  armed.value = null
  void completeSelected()
}

function handleDeleteClick() {
  if (deleting.value || !selectedIds.value.length) return
  if (armed.value !== 'delete') {
    armed.value = 'delete'
    return
  }
  armed.value = null
  void deleteSelected()
}

async function completeSelected() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  // 在「已完成」筛选下反向操作：把选中项改回未完成
  const targetCompleted = !uncompleteMode.value
  completing.value = true
  try {
    const n = await todoStore.completeMany(ids, targetCompleted)
    showToast(
      'success',
      n > 0
        ? (targetCompleted ? `已完成 ${n} 项` : `已取消完成 ${n} 项`)
        : (targetCompleted ? '所选条目均已完成' : '所选条目均未完成')
    )
    clearSelection()
  } catch (e) {
    showToast('error', e instanceof Error ? e.message : '批量更新完成状态失败')
  } finally {
    completing.value = false
  }
}

async function deleteSelected() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  deleting.value = true
  try {
    const n = await todoStore.deleteMany(ids)
    showToast('success', `已删除 ${n} 项`)
    clearSelection()
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

// 单条完成 / 取消完成（行右侧按钮）
const handleToggle = (id: number) =>
  runWithBusy(id, () => todoStore.toggleComplete(id), '已更新完成状态')

// 单条删除：确认动作由 TodoItem 的「连点两下」承担，这里不弹确认框
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
/* ===== 筛选面板展开/收起 =====
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

/* 批量按钮的实心底色。文字色用语义变量（随主题翻转），
   因为深色主题下 success/danger 是亮色，白字会不达标。 */
.btn-danger {
  background-color: var(--color-danger);
  color: var(--color-text-on-danger);
}

.btn-danger:hover:not(:disabled) {
  opacity: 0.88;
}

/* 待确认态：比默认更醒目（提亮 + 描边），文字色同样走语义变量 */
.btn-danger-armed {
  background-color: var(--color-danger);
  border-color: var(--color-danger);
  color: var(--color-text-on-danger);
  box-shadow: 0 0 0 2px var(--color-bg-primary), 0 0 0 4px var(--color-danger);
}

/* 待确认态（完成）：同构的实心 + 描边 */
.btn-success {
  background-color: var(--color-success);
  border-color: var(--color-success);
  color: var(--color-text-on-success);
  box-shadow: 0 0 0 2px var(--color-bg-primary), 0 0 0 4px var(--color-success);
}

/* 完成按钮的默认态：与删除按钮同构——描边 + 中性文字，hover 才填成功绿。
   悬停/待确认变绿时文字必须同步换成 --color-text-on-success，
   否则深色主题下会变成「亮绿底 + 白字」(1.74:1，不达标)。 */
.btn-complete {
  background-color: transparent;
  border-color: var(--color-border-dark);
  color: var(--color-text-secondary);
}

.btn-complete:hover:not(:disabled) {
  background-color: var(--color-success);
  border-color: var(--color-success);
  color: var(--color-text-on-success);
}
</style>
