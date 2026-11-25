<!-- frontend/src/components/todo/TodoList.vue -->
<!-- 排版取向（Minimalism & Swiss）：统一间距节奏、去掉多余装饰（无投影/渐变）、
     只保留必要元素；筛选默认收起，批量操作只在有选中时才出现（渐进披露）。

     批量操作模型：
       行的左侧选择框负责「选中哪些」，工具栏右侧的「删除所选」
       对选中集合执行删除，确认走 naive 的 n-popconfirm 气泡。
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
        <n-button
          quaternary
          size="tiny"
          :aria-expanded="isFilterOpen"
          aria-controls="todo-filter-panel"
          @click="toggleFilterPanel"
        >
          <template #icon>
            <span class="i-lucide-sliders-horizontal icon-xs" aria-hidden="true" />
          </template>
          {{ isFilterOpen ? '收起筛选' : '筛选' }}
          <span
            v-if="activeFilterCount"
            class="ml-1 px-1.5 rounded-full text-[10px] font-medium bg-brand-light text-brand tabular-nums"
          >
            {{ activeFilterCount }}
          </span>
        </n-button>

        <!-- 全选当前可见（键盘可达的替代路径：逐个点行内选择框亦可） -->
        <n-button
          v-if="visibleIds.length"
          quaternary
          size="tiny"
          :type="allVisibleSelected ? 'primary' : 'default'"
          :aria-pressed="allVisibleSelected"
          @click="toggleSelectAll"
        >
          <template #icon>
            <span :class="allVisibleSelected ? 'i-lucide-square-check-big icon-xs' : 'i-lucide-square icon-xs'" aria-hidden="true" />
          </template>
          {{ allVisibleSelected ? '取消全选' : '全选' }}
        </n-button>

        <!-- 批量删除：对「已选中」执行。没有选中时不渲染，避免一排禁用按钮占位 -->
        <div v-if="selectedIds.length" class="ml-auto flex items-center gap-1.5">
          <n-popconfirm
            positive-text="删除"
            negative-text="取消"
            :positive-button-props="{ type: 'error' }"
            @positive-click="deleteSelected"
          >
            <template #trigger>
              <n-button
                type="error"
                size="small"
                :loading="deleting"
                :disabled="deleting"
                :aria-label="`删除选中的 ${selectedIds.length} 项`"
              >
                <template #icon>
                  <span class="i-lucide-trash-2 icon-xs" aria-hidden="true" />
                </template>
                删除所选
                <span class="bulk-count">{{ selectedIds.length }}</span>
              </n-button>
            </template>
            删除选中的 {{ selectedIds.length }} 项？此操作不可撤销。
          </n-popconfirm>
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

    <!-- 操作反馈走全局浮层（styles/naive.ts 的 n-message，入口 utils/toast.ts）：不在本页占位，
         出现/消失都不会把下面的列表卡片往下推。 -->

    <!-- 列表卡片：flex-1 吃掉剩余高度并内部滚动；上下边缘按滚动位置渐隐（v-scroll-fade） -->
    <!-- 不要用 max-h-[calc(100vh-Npx)] 固定扣减：那个 N 算不准，且描述框一展开卡片又被下推，
         导致最后一条滚不到底。交给 flex 自适应即无需任何魔法数字。 -->
    <div
      v-scroll-fade
      class="scroll-fade card p-2 flex-1 min-h-[120px] overflow-y-auto scrollbar-theme flex flex-col gap-2"
    >
      <!-- 加载状态 -->
      <div v-if="todoStore.loading" class="flex-1 flex-col-center py-8 gap-2">
        <n-spin size="medium" description="加载中…" />
      </div>

      <!-- 错误 -->
      <div v-else-if="todoStore.error" class="flex-1 flex-col-center py-8 gap-2" role="alert">
        <n-alert type="error" :bordered="true" class="w-full max-w-[420px]">{{ todoStore.error }}</n-alert>
        <n-button size="small" quaternary @click="todoStore.loadTodos()">
          <template #icon>
            <span class="i-lucide-rotate-ccw icon-xs" aria-hidden="true" />
          </template>
          重新加载
        </n-button>
      </div>

      <!-- 空状态：本来就没有数据 -->
      <n-empty v-else-if="!hasTodos" class="flex-1 py-8" description="还没有 TODO">
        <template #extra>
          <p class="m-0 text-xs text-muted">在上方输入框写下第一件事，按 Enter 即可添加</p>
        </template>
      </n-empty>

      <!-- 空状态：被筛选条件挡住 -->
      <div v-else-if="todoStore.isFilteredEmpty" class="flex-1 flex-col-center py-8 gap-2">
        <n-empty description="当前筛选条件下没有匹配的 TODO" />
        <n-button size="small" quaternary @click="todoStore.resetFilters()">
          <template #icon>
            <span class="i-lucide-rotate-ccw icon-xs" aria-hidden="true" />
          </template>
          恢复默认筛选
        </n-button>
      </div>

      <!-- TODO 列表：用 TransitionGroup 做进出场，删除/新增不再生硬跳变 -->
      <!-- TODO 行之间保留 8px 间隔（用户 2026-11-25 要求）：行本身已是方形，
           但列表行需要可点的呼吸感，不做「只靠 1px 边框」的贴边堆叠。 -->
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
import { NAlert, NButton, NEmpty, NPopconfirm, NSpin } from 'naive-ui'
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
}

const toggleSelectAll = () => {
  if (allVisibleSelected.value) {
    // 只取消当前可见的部分，保留其它范围内的选择
    const visible = new Set(visibleIds.value)
    selectedIds.value = selectedIds.value.filter((id) => !visible.has(id))
  } else {
    selectedIds.value = [...new Set([...selectedIds.value, ...visibleIds.value])]
  }
}

const clearSelection = () => {
  selectedIds.value = []
}

// ===== 批量删除 =====
// 确认交给 n-popconfirm（2026-11-24 从「连点两下 + 移开撤销」换过来），
// 因此不再需要 armed / disarm 这套本地状态。
const deleting = ref(false)

// 切换筛选时清空所有选择。
// 理由：筛选会换掉列表里可见的条目，留着旧选择会造成
// 「看不见的选中项被批量删除带上」，极易误操作。
watch(
  () => [todoStore.filters.priority, todoStore.filters.time],
  () => {
    clearSelection()
  }
)

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
// 反馈条本身是 utils/toast.ts → naive 的 n-message（fixed 浮层），这里只负责说什么。

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

// 单条删除：确认由 TodoItem 的 n-popconfirm 承担，这里不弹确认框。
// 删除后条目就不在列表里了，所以要在删除前把标题取出来当提示的第二行。
const handleDelete = (id: number) => {
  const title = todoStore.todos.find((todo) => todo.id === id)?.title?.trim() ?? ''
  runWithBusy(id, () => todoStore.deleteTodo(id), { message: '已删除', description: title })
}

// 点击卡片复制「标题 + 描述」。
// 2026-11-25 用户明确要求「TODO 复制成功后添加土司提醒」——
// 这是四类保留提示之外的一个显式例外；失败仍报错。
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
   已改为全局浮层（utils/toast.ts → naive 的 n-message），本文件不再需要反馈条样式：
   旧实现是列表容器里的一个 flex 子项，弹出/消失会把下面的列表卡片上下推动，
   而且只有一种字号、没有图标层次。 */

/* ===== 批量删除的数量徽标 =====
   按钮本体是 naive 的 n-button（type=error，实心危险色），确认交给 n-popconfirm 气泡；
   确认态**不额外加外描边强调**（用户明确要求，勿改）。
   徽标只用 currentColor 派生底色，随按钮文字色自动反转。 */
.bulk-count {
  padding: 0 0.375rem;
  border-radius: 9999px;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  background-color: color-mix(in srgb, currentColor 22%, transparent);
}

/* 数量徽标：用 currentColor 派生，随按钮文字色自动反转（按钮本体是 n-button） */
</style>
