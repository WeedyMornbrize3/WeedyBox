// src/stores/Theme.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ThemePalette } from '../styles/themePresets'
import {
  CUSTOM_THEME_ID,
  DEFAULT_THEME_ID,
  buildCustomPreset,
  getThemePreset,
  type CustomThemePalette,
} from '../styles/themePresets'
import { applyThemeVars } from '../styles/theme'
import { normalizeHex } from '../utils/color'

export type ThemeMode = 'light' | 'dark' | 'system'
/** 四色的四个槽位（与 ThemePalette 的字段一一对应） */
export type PaletteKey = keyof ThemePalette

const MEDIA_QUERY = '(prefers-color-scheme: dark)'
/** localStorage 键：主题模式（light/dark/system） */
const MODE_KEY = 'theme-mode'
/** localStorage 键：四色主题 id（内置主题 id 或 'custom'） */
const PRESET_KEY = 'theme-id'
/** localStorage 键：自定义主题的四色（明暗各一套） */
const CUSTOM_KEY = 'theme-custom'

/**
 * 把任意存下来的东西收敛成一套合法的四色。
 * 用户可能在 DevTools 里改坏了 localStorage，也可能来自旧版本 —— 缺哪个槽位就用
 * fallback 里同名的那个补上，绝不让半套颜色进到 theme.ts 的对比度计算里。
 */
function sanitizePalette(raw: unknown, fallback: ThemePalette): ThemePalette {
  const source = (raw ?? {}) as Partial<Record<PaletteKey, unknown>>
  const pick = (key: PaletteKey): string =>
    normalizeHex(typeof source[key] === 'string' ? (source[key] as string) : null) ?? fallback[key]
  return {
    bgPrimary: pick('bgPrimary'),
    bgSecondary: pick('bgSecondary'),
    bgTertiary: pick('bgTertiary'),
    brand: pick('brand'),
  }
}

/** 读一份自定义四色；缺项用内置默认主题的四色补（保证任何情况下都能渲染） */
function readCustomPalette(): CustomThemePalette {
  const base = getThemePreset(DEFAULT_THEME_ID)
  let raw: Partial<Record<'light' | 'dark', unknown>> = {}
  try {
    const saved = localStorage.getItem(CUSTOM_KEY)
    if (saved) raw = JSON.parse(saved) as Partial<Record<'light' | 'dark', unknown>>
  } catch {
    raw = {}
  }
  return {
    light: sanitizePalette(raw.light, base.light),
    dark: sanitizePalette(raw.dark, base.dark),
  }
}

export const useThemeStore = defineStore('theme', () => {
  // ===== State =====
  const mode = ref<ThemeMode>('system') // 当前模式：light / dark / system
  /** 当前四色主题 id：内置主题 id 或 CUSTOM_THEME_ID */
  const themeId = ref(getThemePreset(null).id)
  /** 自定义主题的四色（明暗各一套；用户在设置页现调，与内置主题互不影响） */
  const customPalette = ref<CustomThemePalette>(readCustomPalette())
  /**
   * 系统偏好是否深色。
   * 必须放进 ref 而不是在 computed 里直接读 matchMedia：
   * computed 只跟踪响应式依赖，matchMedia.matches 不是响应式的，
   * 系统切换后 effectiveTheme 不会重算 —— naive-ui 的主题也就不会跟着换。
   */
  const systemDark = ref(typeof window !== 'undefined' && window.matchMedia(MEDIA_QUERY).matches)

  // ===== Getters =====
  // 实际生效的主题（考虑系统偏好）
  const effectiveTheme = computed<'light' | 'dark'>(() => {
    if (mode.value === 'system') return systemDark.value ? 'dark' : 'light'
    return mode.value
  })

  // 是否为深色模式
  const isDark = computed(() => effectiveTheme.value === 'dark')

  /** 是不是正在用自定义四色 */
  const isCustom = computed(() => themeId.value === CUSTOM_THEME_ID)

  // 当前四色主题（内置 / 自定义拼成同构对象；设置页与 App 都读它）
  const activePreset = computed(() =>
    themeId.value === CUSTOM_THEME_ID
      ? buildCustomPreset(customPalette.value)
      : getThemePreset(themeId.value),
  )

  /** 当前明暗下正在生效的那四个颜色（设置页四个圆盘直接绑它） */
  const activePalette = computed<ThemePalette>(() => activePreset.value[effectiveTheme.value])

  /**
   * 主题签名：明暗 + 主题 id + 当前四色。
   *
   * 为什么不能只看 id：自定义主题的 id 一直是 'custom'，用户拖动色盘时 id 不变，
   * 只监听 `effectiveTheme:themeId` 的话 naive 主题与图表都不会重算（换肤只生效一半）。
   * 把当前四色捏进签名里，任何一次取色都会让依赖它的 watch/computed 重跑。
   */
  const themeSignature = computed(() => {
    const p = activePalette.value
    return `${effectiveTheme.value}:${themeId.value}:${p.bgPrimary}${p.bgSecondary}${p.bgTertiary}${p.brand}`
  })

  // ===== Actions =====
  /**
   * 把「当前四色主题 + 当前明暗」写进 <html> 的 CSS 变量。
   * naive-ui 的 themeOverrides 随后从这些变量里读色值，所以必须在本函数之后同步。
   */
  const applyTheme = () => {
    applyThemeVars(activePreset.value, effectiveTheme.value)
  }

  // 设置主题模式
  const setMode = (newMode: ThemeMode) => {
    mode.value = newMode
    applyTheme()
    localStorage.setItem(MODE_KEY, newMode)
  }

  const persistCustom = () => {
    localStorage.setItem(CUSTOM_KEY, JSON.stringify(customPalette.value))
  }

  /** 换一套四色主题（设置页的「主题配色」；可以是内置 id，也可以是 CUSTOM_THEME_ID） */
  const setThemeId = (id: string) => {
    const next = id === CUSTOM_THEME_ID ? CUSTOM_THEME_ID : getThemePreset(id).id
    if (next === themeId.value) return
    themeId.value = next
    applyTheme()
    localStorage.setItem(PRESET_KEY, next)
  }

  /**
   * 改自定义四色里的某一个槽位（当前明暗那一套）。
   *
   * 用户 2026-11-28：「主题配色由四色选择改为四色圆盘显示四种颜色，并提供自定义 rgb 选色」。
   * 所以点开任一圆盘调色即进入自定义主题：先把当前生效的四色**整份抄成底板**
   * （否则一调色就会从内置主题跳到一份过期/默认的自定义配色，观感很突兀），再改那一个槽位。
   * 颜色非法（null / 半截输入）直接忽略，保持原值。
   */
  const setCustomColor = (key: PaletteKey, value: string | null) => {
    const normalized = normalizeHex(value)
    if (!normalized) return
    const scheme = effectiveTheme.value

    // 首次进入自定义（或从内置主题转过来）：以"当前看到的四色"为底板
    if (themeId.value !== CUSTOM_THEME_ID) {
      const seed = getThemePreset(themeId.value)
      customPalette.value = { light: { ...seed.light }, dark: { ...seed.dark } }
      themeId.value = CUSTOM_THEME_ID
      localStorage.setItem(PRESET_KEY, CUSTOM_THEME_ID)
    }

    const current = customPalette.value[scheme]
    if (current[key] === normalized) return
    customPalette.value = {
      ...customPalette.value,
      [scheme]: { ...current, [key]: normalized },
    }
    persistCustom()
    applyTheme()
  }

  /** 「恢复默认」：把当前明暗下的自定义四色退回内置默认主题的那一套 */
  const resetCustomColors = () => {
    const base = getThemePreset(DEFAULT_THEME_ID)
    const scheme = effectiveTheme.value
    customPalette.value = {
      ...customPalette.value,
      [scheme]: { ...base[scheme] },
    }
    persistCustom()
    if (themeId.value === CUSTOM_THEME_ID) applyTheme()
  }

  // 切换深色/浅色（在 light 和 dark 之间切换，忽略 system）
  const toggleTheme = () => {
    if (mode.value === 'dark') {
      setMode('light')
    } else if (mode.value === 'light') {
      setMode('dark')
    } else {
      // 如果在 system 模式，根据当前生效的主题切换
      setMode(effectiveTheme.value === 'dark' ? 'light' : 'dark')
    }
  }

  // 恢复保存的主题模式与四色主题
  const restoreState = () => {
    const savedMode = localStorage.getItem(MODE_KEY) as ThemeMode | null
    if (savedMode && ['light', 'dark', 'system'].includes(savedMode)) {
      mode.value = savedMode
    } else {
      mode.value = 'system'
    }
    customPalette.value = readCustomPalette()
    const savedId = localStorage.getItem(PRESET_KEY)
    themeId.value = savedId === CUSTOM_THEME_ID ? CUSTOM_THEME_ID : getThemePreset(savedId).id
    systemDark.value = window.matchMedia(MEDIA_QUERY).matches
    applyTheme()
  }

  // ===== 监听系统主题变化 =====
  const setupSystemListener = () => {
    const mediaQuery = window.matchMedia(MEDIA_QUERY)
    const handler = (event: MediaQueryListEvent) => {
      systemDark.value = event.matches
      if (mode.value === 'system') applyTheme()
    }
    mediaQuery.addEventListener('change', handler)
    return () => mediaQuery.removeEventListener('change', handler)
  }

  return {
    // State
    mode,
    themeId,
    customPalette,
    // Getters
    effectiveTheme,
    isDark,
    isCustom,
    activePreset,
    activePalette,
    themeSignature,
    // Actions
    setMode,
    setThemeId,
    setCustomColor,
    resetCustomColors,
    toggleTheme,
    restoreState,
    setupSystemListener,
    applyTheme,
  }
})
