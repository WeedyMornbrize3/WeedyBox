<template>
  <header 
    class="titlebar"
    style="--wails-draggable: drag;
      -webkit-app-region: drag;
    "
  >
    <div class="titlebar-left"></div>

    <div class="relative flex items-center gap-1" style="--wails-draggable: no-drag; -webkit-app-region: no-drag">
      <button class="titlebar-btn" @click="minimize" title="最小化" aria-label="最小化窗口">
        <span class="i-lucide-minus icon-sm" aria-hidden="true" />
      </button>
      <button class="titlebar-btn" @click="toggleMaximize" :title="isMaximised ? '还原' : '最大化'" :aria-label="isMaximised ? '还原窗口' : '最大化窗口'">
        <span :class="isMaximised ? 'i-lucide-copy icon-sm' : 'i-lucide-square icon-sm'" aria-hidden="true" />
      </button>

      <!-- 关闭按钮：行为取决于设置。
           「每次询问」时点它不直接关窗，而是在下方弹出一个浮层让用户选
           「最小化 / 退出程序」——这是两个不同意图的动作，用模态对话框
           打断体验不值得，浮层更轻、也不阻塞其它操作。 -->
      <button
        class="titlebar-btn-close"
        :class="{ 'is-open': isPromptOpen }"
        @click="handleCloseClick"
        title="关闭"
        aria-label="关闭窗口"
        :aria-expanded="isPromptOpen"
        aria-controls="close-prompt"
      >
        <span class="i-lucide-x icon-sm" aria-hidden="true" />
      </button>

      <!-- 关闭浮层 -->
      <Transition name="close-pop">
        <div
          v-if="isPromptOpen"
          id="close-prompt"
          class="close-pop absolute top-[calc(100%+6px)] right-0 z-[1200] w-56 p-2 rounded-lg bg-card border-theme shadow-theme-heavy"
        >
          <p class="m-0 px-1.5 pb-1.5 text-[11px] text-tertiary">
            最小化、隐藏到托盘，还是退出程序？
          </p>

          <button type="button" class="close-opt" @click="choose('minimise')">
            <span class="i-lucide-minus icon-sm" aria-hidden="true" />
            <span class="flex-1 text-left">最小化到任务栏</span>
          </button>

          <button type="button" class="close-opt" @click="choose('hide')">
            <span class="i-lucide-eye-off icon-sm" aria-hidden="true" />
            <span class="flex-1 text-left">隐藏到托盘</span>
          </button>

          <button type="button" class="close-opt close-opt-danger" @click="choose('close')">
            <span class="i-lucide-power icon-sm" aria-hidden="true" />
            <span class="flex-1 text-left">退出程序</span>
          </button>

          <label class="flex items-center gap-2 mt-1.5 px-1.5 py-1 rounded-md text-[11px] text-tertiary cursor-pointer hover:bg-hover transition-colors duration-200">
            <input type="checkbox" class="close-remember" v-model="remember" />
            记住我的选择
          </label>
        </div>
      </Transition>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { 
  MinimizeWindow, 
  HideWindow,
  ToggleMaximizeWindow, 
  CloseWindow, 
  IsWindowMaximised 
} from '../../bindings/WeedyBox/internal/service/windowservice'
import { useAppStore, type CloseAction } from '../stores/App'

const appStore = useAppStore()

const isMaximised = ref(false)
const isPromptOpen = ref(false)
const remember = ref(false)

const minimize = async () => {
  try {
    await MinimizeWindow()
  } catch (e) {
    console.error('最小化失败', e)
  }
}

const toggleMaximize = async () => {
  try {
    await ToggleMaximizeWindow()
    isMaximised.value = await IsWindowMaximised()
  } catch (e) {
    console.error('切换最大化失败', e)
  }
}

const quit = async () => {
  try {
    await CloseWindow()
  } catch (e) {
    console.error('关闭失败', e)
  }
}

/** 隐藏到系统托盘：窗口完全不可见，靠托盘图标左键唤回 */
const hideToTray = async () => {
  try {
    await HideWindow()
  } catch (e) {
    console.error('隐藏失败', e)
  }
}

/** 执行某个关闭动作 */
const runAction = (action: Exclude<CloseAction, 'ask'>) => {
  isPromptOpen.value = false
  remember.value = false
  switch (action) {
    case 'minimise': return void minimize()
    case 'hide': return void hideToTray()
    case 'close': return void quit()
  }
}

/**
 * 点击 X：
 *   设置为「总是…」→ 直接执行，不再询问；
 *   设置为「每次询问」→ 第一次点开浮层让用户选，不直接关窗。
 */
const handleCloseClick = () => {
  const action = appStore.closeAction
  if (action !== 'ask') {
    runAction(action)
    return
  }
  isPromptOpen.value = !isPromptOpen.value
}

/** 在浮层里做出选择 */
const choose = (action: Exclude<CloseAction, 'ask'>) => {
  // 勾了「记住我的选择」就写进设置，之后 X 直接执行该动作
  if (remember.value) appStore.setCloseAction(action)
  runAction(action)
}

/** 点击浮层外部或按 Esc 关闭浮层 */
const onDocPointerDown = (e: PointerEvent) => {
  if (!isPromptOpen.value) return
  const target = e.target as HTMLElement | null
  if (target?.closest('#close-prompt') || target?.closest('.titlebar-btn-close')) return
  isPromptOpen.value = false
}
const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') isPromptOpen.value = false
}

onMounted(async () => {
  try {
    isMaximised.value = await IsWindowMaximised()
  } catch (e) {
    console.error('获取窗口状态失败', e)
  }
  document.addEventListener('pointerdown', onDocPointerDown)
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocPointerDown)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
/* ===== 关闭浮层 ===== */
.close-pop-enter-active,
.close-pop-leave-active {
  transition: opacity var(--motion-fast) var(--ease-enter),
              transform var(--motion-fast) var(--ease-enter);
}

.close-pop-enter-from,
.close-pop-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

/* 浮层内的选项：与批量按钮同构（透明底 + hover 填充） */
.close-opt {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.375rem 0.375rem;
  border: none;
  border-radius: 0.375rem;
  background-color: transparent;
  color: var(--color-text-secondary);
  font-size: 0.75rem;
  cursor: pointer;
  transition: background-color var(--motion-fast) var(--ease-enter),
              color var(--motion-fast) var(--ease-enter);
}

.close-opt:hover {
  background-color: var(--color-bg-hover);
  color: var(--color-text-primary);
}

/* 退出是破坏性动作：hover 用 danger 语义色 */
.close-opt-danger:hover {
  background-color: var(--color-danger);
  color: var(--color-text-on-danger);
}

/* 「记住我的选择」勾选框：与列表里的选择框同构 */
.close-remember {
  flex-shrink: 0;
  width: 14px;
  height: 14px;
  margin: 0;
  border: 1px solid var(--color-border-dark);
  border-radius: 4px;
  background-color: transparent;
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color var(--motion-fast) var(--ease-enter),
              border-color var(--motion-fast) var(--ease-enter);
}

.close-remember:checked {
  background-color: var(--color-bg-check);
  border-color: var(--color-bg-check);
}

.close-remember:checked::after {
  content: '';
  width: 8px;
  height: 8px;
  background-color: var(--color-check-mark);
  -webkit-mask: var(--icon-check) center / contain no-repeat;
  mask: var(--icon-check) center / contain no-repeat;
}

/* 浮层打开时，关闭按钮保持 danger 语义提示 */
.titlebar-btn-close.is-open {
  background-color: var(--color-danger);
  color: #ffffff;
}
</style>
