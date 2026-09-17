<!-- frontend/src/components/todo/TodoItem.vue -->
<!-- 单条 TODO。
     结构要点（与旧版的区别）：
       1. 根元素是 <div> 而不是 <label>——旧版把删除按钮放在 <label> 内部，
          点删除会先触发 label 的默认行为（切换勾选框），再靠 @click 冒泡语义兜底，
          行为不可靠；现在勾选框、优先级、标题、删除按钮是同级 flex item。
       2. 勾选框是可以真正聚焦的原生 input（旧版 w-0 h-0 被完全隐藏，键盘无法操作）。
       3. 删除按钮用 type="button" + aria-label，点击不冒泡到行本身。 -->
<template>
  <div
    class="group flex items-start gap-3 px-3 py-2.5 rounded-lg bg-card border-theme transition-colors duration-200 hover:border-brand"
    :class="{ 'opacity-60': todo.completed }"
  >
    <!-- 勾选框：原生 input 提供语义、键盘与无障碍状态，外观由 .todo-checkbox 绘制。
         这些 utility 刻意不封装成 shortcut——串联过长的 shortcut 会被 UnoCSS 截断。 -->
    <input
      type="checkbox"
      class="todo-checkbox flex-shrink-0 w-[18px] h-[18px] mt-[2px] rounded-[5px] cursor-pointer appearance-none flex-center text-transparent transition-colors duration-200 border-2 border-theme-dark bg-transparent hover:border-brand checked:border-brand focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand disabled:cursor-not-allowed disabled:opacity-50"
      :checked="todo.completed"
      :disabled="isUpdating"
      :aria-label="todo.completed ? `标记「${todo.title}」为未完成` : `标记「${todo.title}」为已完成`"
      @change="emit('toggle', todo.id)"
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

    <!-- 删除：连点两下删除，不弹确认框。
         第一下只是「进入待确认态」（图标变红 + 加粗边框 + 底色变红，鼠标提示变为「再点一次删除」），
         第二下才真正删除；移开鼠标或失焦即自动撤销，因此不需要模态确认也不会误删。 -->
    <!-- 颜色类全部放进下面的动态 :class，静态 class 里不留任何颜色工具类。
         原因：UnoCSS 按字母序输出 rules，`.text-danger` 排在 `.text-white` 之后
         （权重相同，0,1,0），会把待确认态的白字压掉；`.bg-transparent` 同理压 `.bg-danger`。
         所以同一属性只能由一处决定。 -->
    <button
      v-if="!isDeleting"
      type="button"
      class="btn-delete group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100"
      :class="isArmed
        ? 'opacity-100 bg-danger border-danger text-white scale-110'
        : 'opacity-40 bg-transparent border-danger text-danger hover:opacity-100 hover:bg-danger hover:border-danger hover:text-white'"
      :aria-label="isArmed ? `再次点击确认删除「${todo.title}」` : `删除「${todo.title}」（需连点两下）`"
      :title="isArmed ? '再点一次删除' : '删除（连点两下）'"
      @click.stop="handleDeleteClick"
      @mouseleave="cancelDelete"
      @blur="cancelDelete"
    >
      <span :class="isArmed ? 'i-lucide-check icon-sm' : 'i-lucide-trash-2 icon-sm'" aria-hidden="true" />
    </button>

    <!-- 提交中：短暂态，避免删除瞬间元素直接消失造成的视觉跳跃 -->
    <span v-else class="btn-delete opacity-100 text-danger cursor-default" aria-hidden="true">
      <span class="i-lucide-loader-circle icon-sm spin" />
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onBeforeUnmount, watch } from 'vue'
import type { Todo } from '../../../bindings/WeedyBox/internal/model'

const props = defineProps<{
  todo: Todo
  isUpdating?: boolean
  isDeleting?: boolean
}>()

const emit = defineEmits<{
  toggle: [id: number]
  delete: [id: number]
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
/* 描述块的左侧竖线：宽度 / 样式 / 基础色都在这里定义，
   UnoCSS 只负责 group-hover 时的颜色切换（见模板的 group-hover:border-l-brand）。
   这样同一属性只有一个来源，不依赖 UnoCSS 的规则输出顺序。 */
.todo-desc-block {
  border-left: 2px solid var(--color-border-light);
}
</style>
