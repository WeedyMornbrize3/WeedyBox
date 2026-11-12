// frontend/src/stores/App.ts
// 应用级偏好（非主题、非侧边栏）：窗口与壳层行为。
// 目前只有「标题栏关闭按钮的行为」，未来同类偏好也放这里。
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

/**
 * 关闭按钮（标题栏 X）的行为：
 *   ask    每次询问：弹出浮层让用户选
 *   hide   隐藏到系统托盘（窗口完全不可见，靠托盘图标唤回）
 *   close  退出程序
 *
 * 注：刻意不提供「最小化到任务栏」——标题栏左侧已有独立的最小化按钮，
 * 放在关闭行为里属于重复入口，且会让「关闭」这个决定变得含糊。
 */
export type CloseAction = 'ask' | 'hide' | 'close'

export const CLOSE_ACTION_OPTIONS: { value: CloseAction; label: string; description: string }[] = [
  { value: 'ask', label: '每次询问', description: '弹出选择：隐藏到托盘 / 退出程序' },
  { value: 'hide', label: '隐藏到托盘', description: '窗口完全隐藏，点托盘图标唤回' },
  { value: 'close', label: '退出程序', description: '直接退出程序' },
]

const STORAGE_KEY = 'close-action'
const VALID: CloseAction[] = ['ask', 'hide', 'close']

/**
 * 作品标签的显示数量上限（只影响 DLsite 解析出来的 tags 显示多少，
 * 用户自己加的标签始终全部显示）。值落在 [TAG_LIMIT_MIN, TAG_LIMIT_MAX]。
 */
export const TAG_LIMIT_MIN = 4
export const TAG_LIMIT_MAX = 64
export const TAG_LIMIT_DEFAULT = 24
const TAG_LIMIT_KEY = 'tag-limit'

function clampTagLimit(value: number): number {
  if (!Number.isFinite(value)) return TAG_LIMIT_DEFAULT
  return Math.min(TAG_LIMIT_MAX, Math.max(TAG_LIMIT_MIN, Math.round(value)))
}

export const useAppStore = defineStore('app', () => {
  const closeAction = ref<CloseAction>('ask')
  const tagLimit = ref(TAG_LIMIT_DEFAULT)

  function setCloseAction(value: CloseAction) {
    closeAction.value = value
    localStorage.setItem(STORAGE_KEY, value)
  }

  function setTagLimit(value: number) {
    tagLimit.value = clampTagLimit(value)
  }

  /** 从 localStorage 恢复；值非法时回到 ask（最安全的默认） */
  function restoreState() {
    const saved = localStorage.getItem(STORAGE_KEY) as CloseAction | null
    closeAction.value = saved && VALID.includes(saved) ? saved : 'ask'

    const savedTagLimit = Number(localStorage.getItem(TAG_LIMIT_KEY))
    tagLimit.value = clampTagLimit(savedTagLimit || TAG_LIMIT_DEFAULT)
  }

  // 兜底：任何路径改了它都落盘，避免只在 setter 里持久化被绕过
  watch(closeAction, (v) => localStorage.setItem(STORAGE_KEY, v))
  watch(tagLimit, (v) => localStorage.setItem(TAG_LIMIT_KEY, String(v)))

  return {
    closeAction,
    setCloseAction,
    tagLimit,
    setTagLimit,
    restoreState,
  }
})
