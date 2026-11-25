<!-- frontend/src/components/todo/TodoInput.vue -->
<!-- 新增 TODO 的输入区（2026-11-24 改为 naive 的 n-input-group）：
     标题输入 + 描述开关 + 添加按钮拼成一组；
     描述用 n-input type=textarea，展开/收起仍走原来的过渡。
     快捷键：Enter 添加；描述区里 Enter 也添加、Ctrl/⌘ + Enter 换行、Esc 收起。 -->
<template>
  <div class="flex flex-col">
    <!-- 标题行：输入框（内嵌描述开关）+ 添加按钮 -->
    <n-input-group>
      <n-input
        v-model:value="newTodo"
        :placeholder="showDesc ? '添加新的 TODO 标题…' : '添加新的 TODO…'"
        aria-label="TODO 标题"
        @keyup.enter="handleSubmit"
      >
        <template #prefix>
          <span class="i-lucide-plus icon-sm text-tertiary" aria-hidden="true" />
        </template>
        <template #suffix>
          <!-- 描述开关：图标按钮，带可访问名称与展开状态 -->
          <n-button
            quaternary
            circle
            size="tiny"
            :type="showDesc ? 'primary' : 'default'"
            :title="showDesc ? '收起描述' : '添加描述'"
            :aria-label="showDesc ? '收起描述' : '添加描述'"
            :aria-expanded="showDesc"
            aria-controls="todo-desc-field"
            @click="toggleDesc"
          >
            <template #icon>
              <span class="i-lucide-align-left icon-sm" aria-hidden="true" />
            </template>
          </n-button>
        </template>
      </n-input>

      <!-- 提交按钮：输入为空时禁用（避免"可点但没反应"） -->
      <n-button
        type="primary"
        class="shrink-0"
        :disabled="!newTodo.trim() || loading"
        :loading="loading"
        @click="handleSubmit"
      >
        <template #icon>
          <span class="i-lucide-plus icon-sm" aria-hidden="true" />
        </template>
        {{ loading ? '添加中…' : '添加' }}
      </n-button>
    </n-input-group>

    <!-- 描述区：点击图标后展开，带高度/透明度过渡 -->
    <Transition name="desc-slide">
      <div
        v-show="showDesc"
        id="todo-desc-field"
        class="desc-field flex flex-col"
        @keydown.enter.exact.prevent="handleSubmit"
        @keydown.ctrl.enter.prevent="insertLineBreak"
        @keydown.meta.enter.prevent="insertLineBreak"
        @keydown.esc="toggleDesc"
      >
        <n-input
          ref="descRef"
          v-model:value="desc"
          type="textarea"
          placeholder="添加描述（可选）…"
          aria-label="TODO 描述"
          :autosize="{ minRows: 2, maxRows: 6 }"
        />
        <!-- 快捷键提示：排在下方右对齐，不与输入文字重叠 -->
        <span class="desc-hint">Enter 添加 · Ctrl/⌘ + Enter 换行 · Esc 收起</span>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { NButton, NInput, NInputGroup, type InputInst } from 'naive-ui'
import { useTodoStore } from '../../stores/todoStore'

const todoStore = useTodoStore()
const newTodo = ref('')
const desc = ref('')
const showDesc = ref(false)
const loading = ref(false)
const descRef = ref<InputInst | null>(null)

// 说明：提交用 @keyup.enter（标题）/ @keydown.enter.exact.prevent（描述）。
// 为什么需要 insertLineBreak：Chromium/WebKit 的 textarea 在 Ctrl+Enter（及 Cmd+Enter）
//   下【不会】插入换行，带 Ctrl 的 Enter 被丢弃，所以必须自己插入 '\n'。
//   n-input 不暴露内部 textarea 的 ref，这里直接取事件冒泡上来的真实元素。
const insertLineBreak = (event: KeyboardEvent) => {
  const el = event.target as HTMLTextAreaElement | null
  if (!el || typeof el.selectionStart !== 'number') return
  const start = el.selectionStart
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
    descRef.value?.focus()
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
/* ===== 描述区容器 ===== */
.desc-field {
  margin-top: 0.5rem;
}

/* 快捷键提示：与输入框同级，排在下方右对齐，不会与 placeholder 重叠；
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
