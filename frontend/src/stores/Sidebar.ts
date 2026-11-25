// src/stores/sidebar.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useSidebarStore = defineStore('sidebar', () => {
  // ===== State =====
  const isCollapsed = ref<boolean>(false)
  
  // ===== Getters =====
  // 全局侧边栏宽度设置（折叠 72 / 展开 250）
  const COLLAPSED_WIDTH: number = 72
  const EXPANDED_WIDTH: number = 250

  const sidebarWidth = computed(() => isCollapsed.value ? COLLAPSED_WIDTH : EXPANDED_WIDTH)
  // 主内容让位用的就是侧栏宽度本身（aside 是 fixed，不让位会把内容压住）——
  // 旧的 mainPaddingLeft（= 展开时的 178 = 250−72）会让内容钻到侧栏底下，2026-11-25 已删。
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
    isExpanded,
    // Actions
    toggleSidebar,
    setCollapsed,
    restoreState
  }
})