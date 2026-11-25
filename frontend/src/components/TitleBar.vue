<template>
  <header
    class="titlebar"
    style="--wails-draggable: drag; -webkit-app-region: drag;"
  >
    <div class="titlebar-left"></div>

    <div class="relative flex items-center gap-1" style="--wails-draggable: no-drag; -webkit-app-region: no-drag">
      <n-button quaternary circle size="small" title="最小化" aria-label="最小化窗口" @click="minimize">
        <template #icon>
          <span class="i-lucide-minus icon-sm" aria-hidden="true" />
        </template>
      </n-button>

      <n-button
        quaternary
        circle
        size="small"
        :title="isMaximised ? '还原' : '最大化'"
        :aria-label="isMaximised ? '还原窗口' : '最大化窗口'"
        @click="toggleMaximize"
      >
        <template #icon>
          <span :class="isMaximised ? 'i-lucide-copy' : 'i-lucide-square'" class="icon-sm" aria-hidden="true" />
        </template>
      </n-button>

      <!-- 关闭按钮：行为取决于设置。
           「每次询问」时点它不直接关窗，而是在下方浮出一个面板让用户选
           「隐藏到托盘 / 退出程序」——两个不同意图的动作，用模态对话框打断体验不值得。
           面板改用 naive 的 n-popover（trigger=manual），点外部关闭由它自己处理。 -->
      <n-popover
        :show="isPromptOpen"
        trigger="manual"
        placement="bottom-end"
        :show-arrow="false"
        :style="{ width: '224px' }"
        @update:show="(value: boolean) => (isPromptOpen = value)"
        @clickoutside="isPromptOpen = false"
      >
        <template #trigger>
          <n-button
            class="titlebar-close"
            :class="{ 'is-open': isPromptOpen }"
            quaternary
            circle
            size="small"
            title="关闭"
            aria-label="关闭窗口"
            :aria-expanded="isPromptOpen"
            @click="handleCloseClick"
          >
            <template #icon>
              <span class="i-lucide-x icon-sm" aria-hidden="true" />
            </template>
          </n-button>
        </template>

        <p class="m-0 px-1 pb-2 text-[11px] text-tertiary">
          隐藏到托盘，还是退出程序？
        </p>

        <n-button block quaternary size="small" class="close-opt" @click="choose('hide')">
          <template #icon>
            <span class="i-lucide-eye-off icon-sm" aria-hidden="true" />
          </template>
          隐藏到托盘
        </n-button>

        <n-button block quaternary size="small" type="error" class="close-opt" @click="choose('close')">
          <template #icon>
            <span class="i-lucide-power icon-sm" aria-hidden="true" />
          </template>
          退出程序
        </n-button>

        <n-checkbox v-model:checked="remember" size="small" class="mt-2 px-1">
          记住我的选择
        </n-checkbox>
      </n-popover>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { NButton, NCheckbox, NPopover } from 'naive-ui'
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
    case 'hide': return void hideToTray()
    case 'close': return void quit()
  }
}

/**
 * 点击 X：
 *   设置为「总是…」→ 直接执行，不再询问；
 *   设置为「每次询问」→ 第一次点开面板让用户选，不直接关窗。
 */
const handleCloseClick = () => {
  const action = appStore.closeAction
  if (action !== 'ask') {
    runAction(action)
    return
  }
  isPromptOpen.value = !isPromptOpen.value
}

/** 在面板里做出选择 */
const choose = (action: Exclude<CloseAction, 'ask'>) => {
  // 勾了「记住我的选择」就写进设置，之后 X 直接执行该动作
  if (remember.value) appStore.setCloseAction(action)
  runAction(action)
}

/** Esc 关闭面板（n-popover 只管点外部，键盘要自己接） */
const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape') isPromptOpen.value = false
}

onMounted(async () => {
  try {
    isMaximised.value = await IsWindowMaximised()
  } catch (e) {
    console.error('获取窗口状态失败', e)
  }
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
/* 关闭按钮要的是「破坏性」语义：hover / 展开时填红 + 反色图标。
   naive 的 quaternary 变体颜色全部由这几个 CSS 变量驱动，
   在同一个元素上覆盖比改主题更精准（只影响标题栏这一颗）。 */
.titlebar-close:hover,
.titlebar-close.is-open {
  --n-color-hover: var(--color-danger);
  --n-color-pressed: var(--color-danger);
  --n-text-color-hover: var(--color-text-on-danger);
  --n-text-color-pressed: var(--color-text-on-danger);
  --n-text-color: var(--color-text-on-danger);
  --n-color: var(--color-danger);
}

/* 面板里的两个选项：左对齐、占满宽度（n-button 默认居中） */
.close-opt {
  --n-content-margin: 0;
}

.close-opt :deep(.n-button__content) {
  justify-content: flex-start;
  width: 100%;
}
</style>
