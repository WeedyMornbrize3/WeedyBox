// src/stores/sidebar.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useSidebarStore = defineStore('sidebar', () => {
  // ===== State =====
  const isCollapsed = ref<boolean>(false)
  
  // ===== Getters =====
  // 全局侧边栏宽度设置
  const barWidth: number = 72
  const barWidthCollapsed: number = 250

  const sidebarWidth = computed(() => isCollapsed.value ? barWidth : barWidthCollapsed)
  const mainPaddingLeft = computed(() => isCollapsed.value ? barWidth : (barWidthCollapsed-barWidth))
  const isExpanded = computed(() => !isCollapsed.value)
  
  // ===== Actions =====
  const toggleSidebar = () => {
    isCollapsed.value = !isCollapsed.value
    localStorage.setItem('sidebar-collapsed', String(isCollapsed.value))
  }
  
  const setCollapsed = (value: boolean) => {
    isCollapsed.value = value
    localStorage.setItem('sidebar-collapsed', String(isCollapsed.value))
  }
  
  const restoreState = () => {
    const saved = localStorage.getItem('sidebar-collapsed')
    if (saved !== null) {
      isCollapsed.value = saved === 'true'
    }
  }
  
  return {
    // State
    isCollapsed,
    // Getters
    sidebarWidth,
    mainPaddingLeft,
    isExpanded,
    // Actions
    toggleSidebar,
    setCollapsed,
    restoreState
  }
})