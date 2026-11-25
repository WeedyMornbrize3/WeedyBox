// src/stores/Theme.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getThemePreset } from '../styles/themePresets'
import { applyThemeVars } from '../styles/theme'

export type ThemeMode = 'light' | 'dark' | 'system'

const MEDIA_QUERY = '(prefers-color-scheme: dark)'
/** localStorage 键：主题模式（light/dark/system） */
const MODE_KEY = 'theme-mode'
/** localStorage 键：四色主题 id（见 styles/themePresets.ts） */
const PRESET_KEY = 'theme-id'

export const useThemeStore = defineStore('theme', () => {
  // ===== State =====
  const mode = ref<ThemeMode>('system') // 当前模式：light / dark / system
  /** 当前四色主题 id（未知值回落到默认主题） */
  const themeId = ref(getThemePreset(null).id)
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

  // 当前四色主题（设置页与 App 都读它）
  const activePreset = computed(() => getThemePreset(themeId.value))

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

  /** 换一套四色主题（设置页的「主题配色」） */
  const setThemeId = (id: string) => {
    const next = getThemePreset(id).id
    if (next === themeId.value) return
    themeId.value = next
    applyTheme()
    localStorage.setItem(PRESET_KEY, next)
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
    themeId.value = getThemePreset(localStorage.getItem(PRESET_KEY)).id
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
    // Getters
    effectiveTheme,
    isDark,
    activePreset,
    // Actions
    setMode,
    setThemeId,
    toggleTheme,
    restoreState,
    setupSystemListener,
    applyTheme,
  }
})
