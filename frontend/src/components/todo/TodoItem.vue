<!-- frontend/src/components/todo/TodoItem.vue -->
<!-- 单条 TODO。交互分工（与旧版的区别）：
       1. 左侧勾选框从「完成」改为「选择」——勾选只表达选中，不再改完成状态，
          因此它对应的是批量操作（完成全部 / 删除选中的目标）。
       2. 「完成」是行右侧的独立按钮，位于删除按钮左侧，两种状态互换图标与语义。
       3. 根元素是 <div> 而不是 <label>：旧版把删除按钮放在 <label> 内部，
          点删除会先触发 label 的默认行为，行为不可靠；现在控件之间是同级 flex item。
       4. 选择框是原生 input（键盘可达、有可见焦点环）。 -->
<template>
  <div
    class="group flex items-start gap-3 px-3 py-2.5 rounded-lg bg-card border-theme transition-colors duration-200"
    :class="[
      todo.completed ? 'opacity-60' : '',
      selected ? 'border-brand bg-brand-light' : 'hover:border-brand',
    ]"
  >
    <!-- 选择框：只表示「已选中」，用来做批量操作；不改变完成状态 -->
    <input
      type="checkbox"
      class="todo-checkbox flex-shrink-0 w-[18px] h-[18px] mt-[2px] rounded-[5px] cursor-pointer appearance-none flex-center text-transparent transition-colors duration-200 border-2 border-theme-dark bg-transparent hover:border-brand checked:border-brand focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand disabled:cursor-not-allowed disabled:opacity-50"
      :checked="selected"
      :disabled="isUpdating"
      :aria-label="selected ? `取消选择「${todo.title}」` : `选择「${todo.title}」`"
      @change="emit('select', todo.id)"
    />

    <!-- 主内容区：标题主行 + 描述块 -->
    <div class="flex-1 min-w-0 flex flex-col gap-0.5">
      <div class="flex items-center gap-2 w-full">
        <!-- 优先级：图标 + 文本标签（不靠颜色单独表意，色盲用户也能区分） -->
        <span
          class="flex items-center gap-1 flex-shrink-0 text-[11px] font-medium whitespace-nowrap"
          :class="priorityClass"
          :title="`优先级：${priorityLabel}`"
        >
          <span :class="[priorityIcon, 'icon-xs']" aria-hidden="true" />
          {{ priorityLabel }}
        </span>

        <span
          class="flex-1 min-w-0 truncate text-sm font-medium"
          :class="todo.completed ? 'line-through text-tertiary' : 'text-primary'"
          :title="todo.title"
        >
          {{ todo.title }}
        </span>

        <time
          class="text-xs text-muted flex-shrink-0 whitespace-nowrap"
          :datetime="todo.createdAt"
          :title="fullDate"
        >
          {{ formatDate(todo.createdAt) }}
        </time>
      </div>

      <!-- 描述块：默认只有左侧一条竖线，hover 时只有这条竖线换成品牌色。
           竖线本身（宽度/样式/基础色）定义在下方 scoped 样式里，
           这里只负责「悬停时换成品牌色」。
           为什么这样拆：UnoCSS 对同一属性（border-left）只保留一条规则，
           两条都改 border-left 的类同时写在模板上会丢掉其中一个（实测）；
           而整框的 border-brand 会给四边都加 1px，变成方框。 -->
      <p v-if="todo.description" class="todo-desc-block group-hover:border-l-brand">
        {{ todo.description }}
      </p>
    </div>

    <!-- 行内操作：完成在左、删除在右。
         颜色类全部放进动态 :class，静态 class 里不留任何颜色工具类——
         UnoCSS 按字母序输出 rules，同权重时排后面的会压掉排前面的
         （`.text-tertiary` 会压掉 `.text-success`），同一属性只能由一处决定。
         ⚠️ 静息态用 opacity-75 而不是更弱的 40%：实测 40% 时图标对比度仅 1.78:1，
         达不到「有意义的图标需 ≥3:1」；0.75 时两套主题最低 3.18:1，正好达标。 -->
    <div class="flex items-center gap-1 flex-shrink-0">
      <!-- 完成 / 取消完成：与删除按钮同一种描边图标按钮。
           两种状态都保持「描边 + 图标」的同构外观，只用颜色区分：
             未完成 → 中性灰描边；已完成 → 成功绿实心（表示当前就是这个状态）。
           点击即切换，不要求连点——它不是破坏性操作。 -->
      <button
        type="button"
        class="btn-row opacity-75 group-hover:opacity-100 group-focus-within:opacity-100 hover:opacity-100 focus-visible:opacity-100"
        :class="todo.completed ? 'row-btn-done' : 'row-btn-todo'"
        :disabled="isUpdating"
        :aria-label="todo.completed ? `将「${todo.title}」标记为未完成` : `将「${todo.title}」标记为已完成`"
        :aria-pressed="todo.completed"
        :title="todo.completed ? '标记为未完成' : '标记为已完成'"
        @click.stop="emit('toggle', todo.id)"
      >
        <span :class="todo.completed ? 'i-lucide-circle-check-big icon-sm' : 'i-lucide-circle icon-sm'" aria-hidden="true" />
      </button>

      <!-- 删除：连点两下删除，不弹确认框。
           第一下进入待确认态（图标变红 + 底色变红，提示改为「再点一次删除」），
           第二下才真正删除；移开鼠标或失焦即自动撤销。 -->
      <button
        v-if="!isDeleting"
        type="button"
        class="btn-row opacity-75 group-hover:opacity-100 group-focus-within:opacity-100 hover:opacity-100 focus-visible:opacity-100"
        :class="isArmed ? 'row-btn-delete-armed opacity-100 scale-110' : 'row-btn-delete'"
        :aria-label="isArmed ? `再次点击确认删除「${todo.title}」` : `删除「${todo.title}」（需连点两下）`"
        :title="isArmed ? '再点一次删除' : '删除（连点两下）'"
        @click.stop="handleDeleteClick"
        @mouseleave="cancelDelete"
        @blur="cancelDelete"
      >
        <span :class="isArmed ? 'i-lucide-check icon-sm' : 'i-lucide-trash-2 icon-sm'" aria-hidden="true" />
      </button>

      <!-- 提交中：短暂态，避免删除瞬间元素直接消失造成的视觉跳跃 -->
      <span v-else class="btn-row row-btn-done cursor-default opacity-100" aria-hidden="true">
        <span class="i-lucide-loader-circle icon-sm spin" />
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onBeforeUnmount, watch } from 'vue'
import type { Todo } from '../../../bindings/WeedyBox/internal/model'

const props = defineProps<{
  todo: Todo
  isUpdating?: boolean
  isDeleting?: boolean
  /** 是否已被选中（选中只用于批量操作，与完成状态无关） */
  selected?: boolean
}>()

const emit = defineEmits<{
  toggle: [id: number]
  delete: [id: number]
  select: [id: number]
}>()

// ===== 连点两下删除 =====
// armed = 已点第一下、正在等待第二下确认；移开鼠标或失焦会自动撤销。
const armed = ref(false)
// 提交中：显示转圈并阻止再次点击
const submitting = ref(false)
// 提交态兜底复位：父组件无论成功或失败都会清掉 busy，
// 但若父组件根本没进入 busy（异常路径），这里保证按钮不会永久卡住。
const STUCK_TIMEOUT_MS = 4000
let stuckTimer: ReturnType<typeof setTimeout> | undefined
let timer: ReturnType<typeof setTimeout> | undefined

const isArmed = computed(() => armed.value && !props.isDeleting)
const isDeleting = computed(() => props.isDeleting || submitting.value)

const cancelDelete = () => {
  armed.value = false
}

function clearStuckTimer() {
  if (stuckTimer) {
    clearTimeout(stuckTimer)
    stuckTimer = undefined
  }
}

function handleDeleteClick() {
  if (!armed.value) {
    // 第一下：只进入待确认态，不发出删除事件
    armed.value = true
    return
  }
  armed.value = false
  submitting.value = true
  // 立即 emit（不延迟）：删除动作与这次点击在同一批更新里完成，
  // 因此按钮不会在指针仍按下、或状态尚未闭合时被移除。
  emit('delete', props.todo.id)
  clearStuckTimer()
  stuckTimer = setTimeout(() => {
    submitting.value = false
  }, STUCK_TIMEOUT_MS)
}

// 删除失败时父组件的 isDeleting 会由 true 落回 false（该行仍在列表里），
// 此时立即撤销本地的「提交中」，不必等兜底定时器。
watch(
  () => props.isDeleting,
  (now, prev) => {
    if (prev === true && now !== true) {
      submitting.value = false
      clearStuckTimer()
    }
  }
)

onBeforeUnmount(() => {
  clearTimeout(timer)
  clearStuckTimer()
})

// 优先级 0 低 / 1 中 / 2 高：颜色 + 图标 + 文字三重编码
const PRIORITY_MAP: Record<number, { label: string; icon: string; className: string }> = {
  0: { label: '低', icon: 'i-lucide-signal-low', className: 'text-priority-low' },
  1: { label: '中', icon: 'i-lucide-signal-medium', className: 'text-priority-medium' },
  2: { label: '高', icon: 'i-lucide-signal-high', className: 'text-priority-high' },
}

const meta = computed(() => PRIORITY_MAP[props.todo.priority] ?? PRIORITY_MAP[1])
const priorityLabel = computed(() => meta.value.label)
const priorityIcon = computed(() => meta.value.icon)
const priorityClass = computed(() => meta.value.className)

const fullDate = computed(() => new Date(props.todo.createdAt).toLocaleString('zh-CN'))

// 相对时间：只处理过去时间，未来时间（时钟偏差）按「刚刚」处理
function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  const time = date.getTime()
  if (Number.isNaN(time)) return ''

  const day = 24 * 60 * 60 * 1000
  const diff = Date.now() - time

  if (diff < 60 * 1000) return '刚刚'
  if (diff < 60 * 60 * 1000) return `${Math.floor(diff / (60 * 1000))} 分钟前`
  if (diff < day) return `${Math.floor(diff / (60 * 60 * 1000))} 小时前`
  if (diff < 2 * day) return '昨天'
  if (diff < 7 * day) return `${Math.floor(diff / day)} 天前`
  return `${date.getMonth() + 1} 月 ${date.getDate()} 日`
}
</script>

<style scoped>
/* 描述块：默认只有左侧一条竖线，hover 时只有这条竖线换成品牌色。
   竖线本身（宽度/样式/基础色）定义在这里，
   UnoCSS 只负责 group-hover 时的颜色切换（见模板的 group-hover:border-l-brand）。
   这样同一属性只有一个来源，不依赖 UnoCSS 的规则输出顺序。 */
.todo-desc-block {
  border-left: 2px solid var(--color-border-light);
}

/* ===== 行内操作按钮（完成 / 删除）=====
   两者共用同一套「描边图标按钮」外观，只在颜色与状态上区分，
   符合「同一层级只用一种描边/填充风格」的一致性要求。
   实心 hover 态的文字色必须用随主题翻转的语义变量：
   深色主题下 success / danger 都是亮色，白字压上去只有 1.7~2.8:1。

   静息不透明度由模板的 opacity-75 提供（实测 40% 时图标对比度仅 1.78:1，
   达不到「有意义的图标需 ≥3:1」；0.75 时两套主题最低 3.18:1）。
   提亮交给 group-hover / group-focus-within / hover 工具类，这里不重复声明。 */
.row-btn-todo,
.row-btn-done,
.row-btn-delete,
.row-btn-delete-armed {
  transition: background-color var(--motion-fast) var(--ease-enter),
              border-color var(--motion-fast) var(--ease-enter),
              color var(--motion-fast) var(--ease-enter),
              opacity var(--motion-fast) var(--ease-enter),
              transform var(--motion-fast) var(--ease-enter);
}

/* --- 完成（未完成态）：中性描边，hover 填品牌色 --- */
.row-btn-todo {
  background-color: transparent;
  border-color: var(--color-border-dark);
  color: var(--color-text-tertiary);
}

.row-btn-todo:hover:not(:disabled) {
  background-color: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--color-text-on-brand);
}

/* --- 完成（已完成态）：成功绿实心，表示「当前就是这个状态」 --- */
.row-btn-done {
  background-color: var(--color-success);
  border-color: var(--color-success);
  color: var(--color-text-on-success);
}

.row-btn-done:hover:not(:disabled) {
  background-color: transparent;
  color: var(--color-success);
}

/* --- 删除：默认只描边，hover 才填红 --- */
.row-btn-delete {
  background-color: transparent;
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.row-btn-delete:hover:not(:disabled) {
  background-color: var(--color-danger);
  border-color: var(--color-danger);
  color: var(--color-text-on-danger);
}

/* 待确认态：填红 + 轻微放大，明确区别于默认态 */
.row-btn-delete-armed {
  background-color: var(--color-danger);
  border-color: var(--color-danger);
  color: var(--color-text-on-danger);
}
</style>
