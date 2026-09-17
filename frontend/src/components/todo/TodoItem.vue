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

      <p v-if="todo.description" class="todo-desc-block group-hover:border-brand-l">
        {{ todo.description }}
      </p>
    </div>

    <!-- 删除：图标按钮，带可访问名称。默认半透明以降低视觉噪音， need-change: 改为连点两下删除 不在弹窗
         悬停/键盘聚焦时完全不透明——不靠 hover 才可发现。 -->
    <button
      type="button"
      class="btn-delete opacity-40 group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100"
      :disabled="isDeleting"
      :aria-label="`删除「${todo.title}」`"
      title="删除"
      @click.stop="emit('delete', todo.id)"
    >
      <span :class="isDeleting ? 'i-lucide-loader-circle icon-sm spin' : 'i-lucide-trash-2 icon-sm'" aria-hidden="true" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
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
