<!-- frontend/src/components/todo/TodoItem.vue -->
<!-- 单条 TODO。交互分工（2026-11-24 起控件换成 naive）：
       1. 左侧 n-checkbox 表示「选择」——勾选只表达选中，用于批量删除。
       2. 行右侧只剩删除（「完成」按钮与完成状态已按用户要求移除）；
          删除改用 n-popconfirm 气泡确认（原来是自己实现的「连点两下 + 移开撤销」）。
       3. 根元素是 <div> 而不是 <label>：控件之间是同级 flex item，点删除不会触发选择/复制。
       4. 点击卡片空白/文字区 = 复制「标题 + 描述」（emit copy，父组件弹 toast）；
          指针保持默认（不加 cursor-pointer），行内控件都 .stop 冒泡，避免误复制。 -->
<template>
  <div
    class="group todo-row flex items-start gap-3 px-3 py-2.5 rounded-none bg-card border-theme"
    :class="selected ? 'border-brand bg-brand-light' : 'hover:border-brand'"
    @click="handleCopy"
  >
    <!-- 选择框：只表示「已选中」，用来做批量删除 -->
    <div class="flex-shrink-0 mt-[2px]" @click.stop>
      <n-checkbox
        :checked="selected"
        :disabled="isUpdating"
        :aria-label="selected ? `取消选择「${todo.title}」` : `选择「${todo.title}」`"
        @update:checked="emit('select', todo.id)"
      />
    </div>

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

        <span class="flex-1 min-w-0 truncate text-sm font-medium text-primary" :title="todo.title">
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
           这里只负责「悬停时换成品牌色」——UnoCSS 对同一属性只保留一条规则。 -->
      <p v-if="todo.description" class="todo-desc-block group-hover:border-l-brand">
        {{ todo.description }}
      </p>
    </div>

    <!-- 行内操作：只剩删除（n-popconfirm 气泡确认） -->
    <div class="flex items-center gap-1 flex-shrink-0" @click.stop>
      <n-popconfirm
        v-if="!isDeleting"
        positive-text="删除"
        negative-text="取消"
        :positive-button-props="{ type: 'error' }"
        @positive-click="emit('delete', todo.id)"
      >
        <template #trigger>
          <n-button
            size="small"
            quaternary
            circle
            type="error"
            :disabled="isUpdating"
            :aria-label="`删除「${todo.title}」`"
            title="删除"
          >
            <template #icon>
              <span class="i-lucide-trash-2 icon-sm" aria-hidden="true" />
            </template>
          </n-button>
        </template>
        确定删除「{{ todo.title }}」吗？
      </n-popconfirm>

      <!-- 提交中：短暂态，避免删除瞬间元素直接消失造成的视觉跳跃 -->
      <n-button v-else size="small" quaternary circle loading disabled aria-hidden="true" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NCheckbox, NPopconfirm } from 'naive-ui'
import type { Todo } from '../../../bindings/WeedyBox/internal/model'

const props = defineProps<{
  todo: Todo
  isUpdating?: boolean
  isDeleting?: boolean
  /** 是否已被选中（选中只用于批量删除） */
  selected?: boolean
}>()

const emit = defineEmits<{
  delete: [id: number]
  select: [id: number]
  /** 点击卡片复制标题+描述：把拼好的文本交给父组件去写剪贴板并弹提示 */
  copy: [text: string]
}>()

/**
 * 点击卡片复制「标题 + 描述」。
 *
 * 设计取舍：
 *   - 卡片是整块点击区，但**刻意不加 cursor-pointer**：用户要求不把指针变成手型，
 *     免得整行看起来像按钮；控件区各自 `.stop` 掉冒泡，所以点它们不会误触发复制。
 *   - 真正的剪贴板写入与 toast 放在父组件（TodoList）里：错误提示与其它操作共用同一套 toast。
 */
function handleCopy() {
  const title = props.todo.title?.trim() ?? ''
  const desc = props.todo.description?.trim() ?? ''
  const text = [title, desc].filter(Boolean).join('\n')
  if (!text) return
  emit('copy', text)
}

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

const isDeleting = computed(() => props.isDeleting)

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
/* hover 特效统一（2026-11-28 用户「hover特效统一」）：
   原来模板上挂的是 UnoCSS 的 `transition-colors duration-200` —— 它一次展开
   color/background-color/border-color/text-decoration-color/fill/stroke/
   --un-gradient-* 共 8 个属性、200ms + (0.4,0,0.2,1)，与 design token（160ms +
   --ease-enter）不是一套。这里改成只过渡真正会变的两项：描边色与底色。 */
.todo-row {
  transition:
    border-color var(--motion-fast) var(--ease-enter),
    background-color var(--motion-fast) var(--ease-enter);
}

/* 描述块：默认只有左侧一条竖线，hover 时只有这条竖线换成品牌色。
   竖线本身（宽度/样式/基础色）定义在这里，
   UnoCSS 只负责 group-hover 时的颜色切换（见模板的 group-hover:border-l-brand）。
   竖线变色同样走 --motion-fast，与整行的 hover 节奏一致（否则会出现
   "行已经亮完了、竖线还在慢慢变色"的错位感）。 */
.todo-desc-block {
  border-left: 2px solid var(--color-border-light);
  transition: border-left-color var(--motion-fast) var(--ease-enter);
}
</style>
