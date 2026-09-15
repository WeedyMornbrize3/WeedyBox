<!-- frontend/src/components/todo/TodoInput.vue -->
<template>
  <div class="flex flex-col">
    <!-- 标题行：输入框（内嵌描述开关）+ 添加按钮 -->
    <div class="flex gap-2">
      <!--
        输入框 + 内嵌图标按钮用同级 flex item 排布（不用 absolute + 手调 padding）：
        两者永不重叠，且图标按钮随输入框高度居中。
        min-w-0 让输入框可收缩，否则长内容会把添加按钮挤出容器。
      -->
      <div
        class="flex items-center gap-1 flex-1 min-w-0 pl-3 pr-1 rounded-[10px] bg-secondary border-theme transition-colors duration-200 focus-within:border-brand"
      >
        <span class="i-lucide-plus icon-sm text-tertiary flex-shrink-0" aria-hidden="true" />

        <input
          v-model="newTodo"
          type="text"
          :placeholder="showDesc ? '添加新的 TODO 标题…' : '添加新的 TODO…'"
          aria-label="TODO 标题"
          class="flex-1 min-w-0 bg-transparent border-none outline-none py-2.5 text-sm text-primary placeholder-tertiary"
          @keyup.enter="handleSubmit"
        />

        <!-- 描述开关：图标按钮，带可访问名称与展开状态 -->
        <button
          type="button"
          class="desc-toggle"
          :class="{ 'desc-toggle-active': showDesc }"
          :title="showDesc ? '收起描述' : '添加描述'"
          :aria-label="showDesc ? '收起描述' : '添加描述'"
          :aria-expanded="showDesc"
          aria-controls="todo-desc-field"
          @click="toggleDesc"
        >
          <span class="i-lucide-align-left icon-sm" aria-hidden="true" />
        </button>
      </div>

      <button
        type="button"
        :disabled="!newTodo.trim() || loading"
        class="flex-shrink-0 flex items-center gap-1.5 px-4 py-2 rounded-[10px] text-sm font-medium transition-colors duration-200 bg-brand text-inverse hover:bg-brand-hover border-none cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        @click="handleSubmit"
      >
        <span :class="loading ? 'i-lucide-loader-circle icon-sm spin' : 'i-lucide-plus icon-sm'" aria-hidden="true" />
        {{ loading ? '添加中…' : '添加' }}
      </button>
    </div>

    <!-- 描述区：点击图标后展开，带高度/透明度过渡 -->
    <Transition name="desc-slide">
      <div v-show="showDesc" id="todo-desc-field" class="desc-field flex flex-col">
        <textarea
          ref="descInput"
          v-model="desc"
          placeholder="添加描述（可选）…"
          aria-label="TODO 描述"
          class="w-full h-[4.5rem] leading-relaxed resize-none outline-none rounded-[10px] px-3 py-2.5 text-xs bg-secondary text-secondary placeholder-tertiary border-theme transition-colors duration-200 focus:border-brand"
          @keydown.enter.exact.prevent="handleSubmit"
          @keydown.ctrl.enter.prevent="insertLineBreak"
          @keydown.meta.enter.prevent="insertLineBreak"
          @keydown.esc="toggleDesc"
        />
        <!-- 快捷键提示：作为 textarea 的同级元素排在下方右对齐，不与输入文字重叠 -->
        <span class="desc-hint">Enter 添加 · Ctrl/⌘ + Enter 换行 · Esc 收起</span>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useTodoStore } from '../../stores/todoStore'

const todoStore = useTodoStore()
const newTodo = ref('')
const desc = ref('')
const showDesc = ref(false)
const loading = ref(false)
const descInput = ref<HTMLTextAreaElement | null>(null)

// 说明：提交用 @keydown.enter.exact.prevent
//   .exact 让 Ctrl+Enter / Shift+Enter 不匹配本监听器。
// 为什么需要 insertLineBreak：Chromium/WebKit 的 textarea 在 Ctrl+Enter（及 Cmd+Enter）
//   下【不会】插入换行——原生只有 Enter 和 Shift+Enter 会换行，带 Ctrl 的 Enter 被丢弃。
//   所以 Ctrl+Enter 必须自己插入 '\n'，不能指望原生行为。
// 标题框仍是单行 input，Enter 直接添加。

// 在光标处插入换行，并同步 v-model（textarea 自己改 value 不会触发 input 事件）
const insertLineBreak = () => {
  const el = descInput.value
  if (!el) return
  const start = el.selectionStart ?? el.value.length
  const end = el.selectionEnd ?? start
  desc.value = el.value.slice(0, start) + '\n' + el.value.slice(end)
  nextTick(() => {
    el.selectionStart = el.selectionEnd = start + 1
  })
}

const toggleDesc = async () => {
  showDesc.value = !showDesc.value
  if (showDesc.value) {
    await nextTick()
    descInput.value?.focus()
  } else {
    desc.value = ''
  }
}

const handleSubmit = async () => {
  const title = newTodo.value.trim()
  if (!title || loading.value) return

  loading.value = true
  try {
    await todoStore.addTodo(title, desc.value.trim())
    newTodo.value = ''
    desc.value = ''
  } catch {
    // 错误已写入 store.error，由 TodoList 统一提示
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* ===== 内嵌在输入框内的图标按钮 =====
   定位交给父级 flex（不用 absolute + 手调 padding：那种写法依赖手调值，
   内边距与按钮实际宽度一旦不一致就会重叠）。这里只负责外观与尺寸。 */
.desc-toggle {
  flex-shrink: 0;
  width: 1.75rem;
  height: 1.75rem;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  background: transparent;
  color: var(--color-text-tertiary);
  transition: color 0.2s ease, background-color 0.2s ease;
}

.desc-toggle:hover {
  color: var(--color-primary);
  background: var(--color-bg-hover);
}

.desc-toggle-active {
  color: var(--color-primary);
  background: var(--color-primary-light);
}

/* ===== 描述区容器 ===== */
.desc-field {
  margin-top: 0.5rem;
}

/* 快捷键提示：与 textarea 同级，排在下方右对齐，不会与 placeholder 重叠；
   平时低透明度，聚焦时提亮，避免常驻干扰 */
.desc-hint {
  align-self: flex-end;
  margin-top: 0.375rem;
  font-size: 10px;
  line-height: 1.2;
  color: var(--color-text-muted);
  opacity: 0.7;
  transition: opacity 0.2s ease;
}

.desc-field:focus-within .desc-hint {
  opacity: 1;
}

/* ===== 展开/收起过渡（配合 <Transition name="desc-slide">）=====
   用 max-height 做高度动画：height:auto 无法参与 transition */
.desc-slide-enter-active,
.desc-slide-leave-active {
  overflow: hidden;
  transition: max-height 0.28s cubic-bezier(0.4, 0, 0.2, 1),
              opacity 0.22s ease,
              margin-top 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.desc-slide-enter-from,
.desc-slide-leave-to {
  max-height: 0;
  opacity: 0;
  margin-top: 0 !important;
}

.desc-slide-enter-to,
.desc-slide-leave-from {
  max-height: 14rem;
  opacity: 1;
}
</style>
