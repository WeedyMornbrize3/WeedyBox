<!-- frontend/src/components/todo/TodoList.vue -->
<!-- 排版取向（Minimalism & Swiss）：统一间距节奏、去掉多余装饰（无投影/渐变）、
     只保留必要元素；筛选默认收起，批量操作只在有选中时才出现（渐进披露）。

     批量操作模型：
       行的左侧选择框负责「选中哪些」，工具栏右侧的「删除所选」
       对选中集合执行删除。
       按钮要求**连点两下**（第一下进入待确认态，第二下才执行），
       并且不会弹任何确认框或展开额外面板。
       （「完成全部 / 取消完成」已随完成功能一起移除，2026-11-23。） -->
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
        <TodoStats :todos="todoStore.filteredTodos" class="flex-1 min-w-0" />
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

        <!-- 批量删除：对「已选中」执行。没有选中时不渲染，避免一排禁用按钮占位 -->
        <div v-if="selectedIds.length" class="ml-auto flex items-center gap-1.5">
          <button
            type="button"
            class="bulk-btn bulk-delete"
            :class="{ 'is-armed': armed === 'delete' }"
            :style="{ '--bulk-accent': 'var(--color-danger)' }"
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
            {{ deleting ? '删除中…' : armed === 'delete' ? '再点一次确认' : '删除所选' }}
            <span class="bulk-count">{{ selectedIds.length }}</span>
          </button>
        </div>
      </div>

      <!-- 筛选面板（唯一的下拉面板） -->
      <Transition
        name="collapse"
        @enter="collapse.enter"
        @after-enter="collapse.afterEnter"
        @leave="collapse.leave"
        @after-leave="collapse.afterLeave"
      >
        <div v-show="isFilterOpen" id="todo-filter-panel">
          <FilterBar class="mt-2 flex-shrink-0 scrollbar-theme" />
        </div>
      </Transition>
    </div>

    <!-- 操作反馈走全局浮层（components/AppToaster.vue）：不在本页占位，
         出现/消失都不会把下面的列表卡片往下推。文案由 utils/toast.ts 统一管理。 -->

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
          @delete="handleDelete"
          @copy="handleCopy"
        />
      </TransitionGroup>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useTodoStore } from '../../stores/todoStore'
import { useAutoHeightStore } from '../../stores/autoHeightStore'
import { copyText } from '../../utils/clipboard'
import { toastError, toastSuccess } from '../../utils/toast'
import TodoInput from './TodoInput.vue'
import TodoItem from './TodoItem.vue'
import TodoStats from './TodoStats.vue'
import TodoPrioritySelector from './TodoPrioritySelector.vue'
import FilterBar from './FilterBar.vue'

const todoStore = useTodoStore()
// 展开/收起过渡钩子（原 composables/useAutoHeight，按项目约定搬进 stores）
const collapse = useAutoHeightStore()

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

// ===== 批量删除：连点两下 =====
// 「完成」相关的批量操作已随完成功能移除（2026-11-23），这里只剩删除。
const armed = ref<'delete' | null>(null)
const deleting = ref(false)

const disarm = () => {
  armed.value = null
}

// 选择集合一变就撤销待确认态：避免「看到的是 A，确认时动的是 B」
watch(selectedIds, disarm)

// 切换筛选时清空所有选择。
// 理由：筛选会换掉列表里可见的条目，留着旧选择会造成
// 「看不见的选中项被批量删除带上」，极易误操作。
// 同时撤销待确认态，避免按钮停在 armed 状态。
watch(
  () => [todoStore.filters.priority, todoStore.filters.time],
  () => {
    clearSelection()
    disarm()
  }
)

function handleDeleteClick() {
  if (deleting.value || !selectedIds.value.length) return
  if (armed.value !== 'delete') {
    armed.value = 'delete'
    return
  }
  armed.value = null
  void deleteSelected()
}

async function deleteSelected() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  deleting.value = true
  try {
    const n = await todoStore.deleteMany(ids)
    toastSuccess(`已删除 ${n} 项`)
    clearSelection()
  } catch (e) {
    toastError('批量删除失败', {
      description: e instanceof Error ? e.message : '请稍后再试',
    })
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

// ===== 一次性反馈 =====
// 只显示「已删除」「已复制」这类一次性结果，长时间停留的错误统一由 store.error 渲染在列表区。
// 反馈条本身在 components/AppToaster.vue（fixed 浮层），这里只负责说什么。

/** 行内动作的通用壳：置忙 → 执行 → 成功/失败提示 → 解除置忙 */
async function runWithBusy(
  id: number,
  action: () => Promise<unknown>,
  ok: { message: string; description?: string },
) {
  if (busyIds.value.has(id)) return
  setBusy(id, true)
  try {
    await action()
    toastSuccess(ok.message, { description: ok.description })
  } catch (e) {
    toastError('操作失败', {
      description: e instanceof Error ? e.message : '请稍后再试',
    })
  } finally {
    setBusy(id, false)
  }
}

// 单条删除：确认动作由 TodoItem 的「连点两下」承担，这里不弹确认框。
// 删除后条目就不在列表里了，所以要在删除前把标题取出来当提示的第二行。
const handleDelete = (id: number) => {
  const title = todoStore.todos.find((todo) => todo.id === id)?.title?.trim() ?? ''
  runWithBusy(id, () => todoStore.deleteTodo(id), { message: '已删除', description: title })
}

// 点击卡片复制「标题 + 描述」。
// 复制能力与提示文案集中在这里：写剪贴板失败时用同一条 toast 报错，不弹系统对话框。
async function handleCopy(text: string) {
  try {
    await copyText(text)
    toastSuccess('已复制到剪贴板', { description: text })
  } catch {
    toastError('复制失败', { description: '剪贴板不可用' })
  }
}

onMounted(() => {
  todoStore.init()
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

/* ===== 一次性反馈 =====
   已改为全局浮层（components/AppToaster.vue → vue-sonner），本文件不再需要反馈条样式：
   旧实现是列表容器里的一个 flex 子项，弹出/消失会把下面的列表卡片上下推动，
   而且只有一种字号、没有图标层次。 */

/* ===== 批量操作按钮（删除所选）=====
   现在只剩这一个按钮（完成相关的两个已随完成功能移除）：
     默认态：透明底 + accent 描边 + accent 文字
     hover  ：填 accent + 反色文字
     待确认 ：同 hover + 填 accent（.is-armed），不再额外加外描边
   把颜色放在 scoped 里而不是模板工具类，是为了避免再踩
   「UnoCSS 出序/同权重覆盖」那类静默失效——这里只有一处决定颜色。
   反色文字必须用随主题翻转的语义变量：深色主题下 danger 是亮色，
   白字压上去只有 1.7~2.8:1，不达标。 */
.bulk-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.625rem;
  border-radius: 0.375rem;
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  border: 1px solid var(--bulk-accent);
  background-color: transparent;
  color: var(--bulk-accent);
  transition: background-color var(--motion-fast) var(--ease-enter),
              border-color var(--motion-fast) var(--ease-enter),
              color var(--motion-fast) var(--ease-enter);
}

.bulk-btn:hover:not(:disabled) {
  background-color: var(--bulk-accent);
  color: var(--bulk-on-accent);
}

.bulk-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 待确认态：填 accent + 外描边强调，与默认态明确区分 (人为修改  将外描边删除  当agent读到这一段时不要修改为成为外描边 且将备注修改)  且记住这一次修改 以后不额外添加外描边强调*/
.bulk-btn.is-armed {
  background-color: var(--bulk-accent);
  color: var(--bulk-on-accent);
  /* box-shadow: 0 0 0 2px var(--color-bg-primary), 0 0 0 4px var(--bulk-accent);  这一段删去*/
}

/* 数量徽标：用 currentColor 派生，随按钮文字色自动反转 */
.bulk-count {
  padding: 0 0.375rem;
  border-radius: 9999px;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  background-color: color-mix(in srgb, currentColor 22%, transparent);
}

/* 各动作的反色文字：默认（浅色主题）为白字，深色主题在下方覆盖 */
.bulk-delete {
  --bulk-on-accent: #ffffff;
}

[data-theme="dark"] .bulk-delete {
  --bulk-on-accent: #2a0a0a;
}
</style>
