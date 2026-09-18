// frontend/src/stores/App.ts
// 应用级偏好（非主题、非侧边栏）：窗口与壳层行为。
// 目前只有「标题栏关闭按钮的行为」，未来同类偏好也放这里。
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

/**
 * 关闭按钮（标题栏 X）的行为：
 *   ask       每次询问：弹出浮层选择「最小化 / 退出程序」
 *   minimise  总是最小化
 *   close     总是退出程序
 */
export type CloseAction = 'ask' | 'minimise' | 'close'

export const CLOSE_ACTION_OPTIONS: { value: CloseAction; label: string; description: string }[] = [
  { value: 'ask', label: '每次询问', description: '弹出选择：最小化到任务栏 / 退出程序' },
  { value: 'minimise', label: '总是最小化', description: '直接最小化到任务栏，不退出' },
  { value: 'close', label: '总是退出', description: '直接退出程序' },
]

const STORAGE_KEY = 'close-action'

export const useAppStore = defineStore('app', () => {
  const closeAction = ref<CloseAction>('ask')

  function setCloseAction(value: CloseAction) {
    closeAction.value = value
    localStorage.setItem(STORAGE_KEY, value)
  }

  /** 从 localStorage 恢复；值非法时回到 ask（最安全的默认） */
  function restoreState() {
    const saved = localStorage.getItem(STORAGE_KEY) as CloseAction | null
    closeAction.value = saved && ['ask', 'minimise', 'close'].includes(saved) ? saved : 'ask'
  }

  // 兜底：任何路径改了它都落盘，避免只在 setter 里持久化被绕过
  watch(closeAction, (v) => localStorage.setItem(STORAGE_KEY, v))

  return {
    closeAction,
    setCloseAction,
    restoreState,
  }
})
