// src/stores/theme.ts
import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'system'

export const useThemeStore = defineStore('theme', () => {
  // ===== State =====
  const mode = ref<ThemeMode>('system') // 当前模式：light / dark / system
  
  // ===== Getters =====
  // 实际生效的主题（考虑系统偏好）
  const effectiveTheme = computed<'light' | 'dark'>(() => {
    if (mode.value === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'dark' 
        : 'light'
    }
    return mode.value
  })
  
  // 是否为深色模式
  const isDark = computed(() => effectiveTheme.value === 'dark')
  
  // ===== Actions =====
  // 设置主题模式
  const setMode = (newMode: ThemeMode) => {
    mode.value = newMode
    applyTheme()
    localStorage.setItem('theme-mode', newMode)
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
  
  // 应用主题到 DOM
  const applyTheme = () => {
    const theme = effectiveTheme.value
    if (theme === 'dark') {
      document.documentElement.setAttribute('data-theme', 'dark')
    } else {
      document.documentElement.removeAttribute('data-theme')
    }
  }
  
  // 恢复保存的主题
  const restoreState = () => {
    const saved = localStorage.getItem('theme-mode') as ThemeMode | null
    if (saved && ['light', 'dark', 'system'].includes(saved)) {
      mode.value = saved
    } else {
      mode.value = 'system'
    }
    applyTheme()
  }
  
  // ===== 监听系统主题变化 =====
  const setupSystemListener = () => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => {
      if (mode.value === 'system') {
        applyTheme()
      }
    }
    mediaQuery.addEventListener('change', handler)
    return () => mediaQuery.removeEventListener('change', handler)
  }
    
  return {
    // State
    mode,
    // Getters
    effectiveTheme,
    isDark,
    // Actions
    setMode,
    toggleTheme,
    restoreState,
    setupSystemListener,
    applyTheme,
  }
})