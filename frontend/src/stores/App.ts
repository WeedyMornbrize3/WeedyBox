// frontend/src/stores/App.ts
// 应用级偏好（非主题、非侧边栏）：窗口与壳层行为。
// 目前有「标题栏关闭按钮的行为」「标签显示数量」「玻璃效果（明暗各自开关）」，
// 未来同类偏好也放这里。
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

/**
 * 玻璃效果（作品页的模糊 hero 遮罩）在哪种明暗模式下启用。
 *
 * 用户 2026-11-28：「浅色模式默认不使用 blur hero 模糊遮罩，设置里面可以调整，
 * 选择在什么模式下使用玻璃效果，通过勾选决定」——所以是**两套明暗各自的开关**，
 * 不是全局一个：浅色默认关、深色默认开（深色下那张氛围遮罩是现有观感的一部分）。
 */
export type GlassScheme = 'light' | 'dark'

const GLASS_KEY = 'glass-effect'
const GLASS_DEFAULT: Record<GlassScheme, boolean> = { light: false, dark: true }

/**
 * 作品页（/works）左右两侧的留白宽度（px）。
 *
 * 用户 2026-11-28：「为 work 界面的左右两侧添加 面板分割 Split 供使用者可以自行调整
 * work 卡片显示范围  Split 条调为 1px」——两侧各一条 1px 的竖条，拖动改留白，
 * 中间的作品卡片区跟着变窄/变宽。0 = 不留白（默认，与改动前完全一致）。
 * 上限由页面按窗口当前宽度自己钳（见 WorksPage.vue 的 splitMaxPx），
 * 这里只做「别把脏数据从 localStorage 读回来」的基本保护。
 */
export type WorksSplitSide = 'left' | 'right'

const WORKS_SPLIT_KEY = 'works-split'
const WORKS_SPLIT_LIMIT = 720

function clampWorksSplit(value: unknown): number {
  const n = Number(value)
  if (!Number.isFinite(n) || n <= 0) return 0
  return Math.min(WORKS_SPLIT_LIMIT, Math.round(n))
}

export const useAppStore = defineStore('app', () => {
  const closeAction = ref<CloseAction>('ask')
  const tagLimit = ref(TAG_LIMIT_DEFAULT)
  /** 浅色模式是否启用玻璃效果（默认关） */
  const glassLight = ref(GLASS_DEFAULT.light)
  /** 深色模式是否启用玻璃效果（默认开） */
  const glassDark = ref(GLASS_DEFAULT.dark)
  /** 作品页左侧留白（px，0 = 不留白） */
  const worksSplitLeft = ref(0)
  /** 作品页右侧留白（px，0 = 不留白） */
  const worksSplitRight = ref(0)

  function setCloseAction(value: CloseAction) {
    closeAction.value = value
    localStorage.setItem(STORAGE_KEY, value)
  }

  function setTagLimit(value: number) {
    tagLimit.value = clampTagLimit(value)
  }

  /** 当前明暗模式下是否启用玻璃效果（页面/组件只读这一个判断） */
  function glassEnabled(scheme: GlassScheme): boolean {
    return scheme === 'dark' ? glassDark.value : glassLight.value
  }

  /** 勾选/取消某一明暗模式下的玻璃效果 */
  function setGlass(scheme: GlassScheme, value: boolean) {
    if (scheme === 'dark') glassDark.value = value
    else glassLight.value = value
  }

  /** 设置作品页某一侧的留白宽度（px；越小越好，别吃满整页） */
  function setWorksSplit(side: WorksSplitSide, value: number) {
    if (side === 'left') worksSplitLeft.value = clampWorksSplit(value)
    else worksSplitRight.value = clampWorksSplit(value)
  }

  /** 双侧一起写（恢复默认时用） */
  function resetWorksSplit() {
    worksSplitLeft.value = 0
    worksSplitRight.value = 0
  }

  /** 从 localStorage 恢复；值非法时回到默认（ask / 浅色关、深色开） */
  function restoreState() {
    const saved = localStorage.getItem(STORAGE_KEY) as CloseAction | null
    closeAction.value = saved && VALID.includes(saved) ? saved : 'ask'

    const savedTagLimit = Number(localStorage.getItem(TAG_LIMIT_KEY))
    tagLimit.value = clampTagLimit(savedTagLimit || TAG_LIMIT_DEFAULT)

    let glass: Partial<Record<GlassScheme, unknown>> = {}
    try {
      const raw = localStorage.getItem(GLASS_KEY)
      if (raw) glass = JSON.parse(raw) as Partial<Record<GlassScheme, unknown>>
    } catch {
      glass = {}
    }
    glassLight.value = typeof glass.light === 'boolean' ? glass.light : GLASS_DEFAULT.light
    glassDark.value = typeof glass.dark === 'boolean' ? glass.dark : GLASS_DEFAULT.dark

    let split: { left?: unknown; right?: unknown } = {}
    try {
      const raw = localStorage.getItem(WORKS_SPLIT_KEY)
      if (raw) split = JSON.parse(raw) as { left?: unknown; right?: unknown }
    } catch {
      split = {}
    }
    worksSplitLeft.value = clampWorksSplit(split.left)
    worksSplitRight.value = clampWorksSplit(split.right)
  }

  // 兜底：任何路径改了它都落盘，避免只在 setter 里持久化被绕过
  watch(closeAction, (v) => localStorage.setItem(STORAGE_KEY, v))
  watch(tagLimit, (v) => localStorage.setItem(TAG_LIMIT_KEY, String(v)))
  watch([glassLight, glassDark], () =>
    localStorage.setItem(GLASS_KEY, JSON.stringify({ light: glassLight.value, dark: glassDark.value })),
  )
  watch([worksSplitLeft, worksSplitRight], () =>
    localStorage.setItem(
      WORKS_SPLIT_KEY,
      JSON.stringify({ left: worksSplitLeft.value, right: worksSplitRight.value }),
    ),
  )

  return {
    closeAction,
    setCloseAction,
    tagLimit,
    setTagLimit,
    glassLight,
    glassDark,
    glassEnabled,
    setGlass,
    // 作品页左右留白（Split 条拖出来的）
    worksSplitLeft,
    worksSplitRight,
    setWorksSplit,
    resetWorksSplit,
    restoreState,
  }
})
