<template>
  <aside 
    class="sidebar"
    :class="sidebarClasses"
    :style="{ width: sidebarStore.sidebarWidth + 'px' }"
  >
    <!-- Logo/标题区域 -->
    <div class="sidebar-header" @click="sidebarStore.toggleSidebar">
      <div class="logo-icon">
        <svg viewBox="0 0 24 24" width="28" height="28">
          <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" 
                stroke="currentColor" fill="none" stroke-width="1.5"/>
        </svg>
      </div>
      <span 
        v-show="!sidebarStore.isCollapsed" 
        class="logo-text"
        :class="{ 'opacity-0 w-0': sidebarStore.isCollapsed }"
      >WeedyBox</span>
    </div>

    <!-- 上导航菜单 -->
    <nav class="sidebar-nav-top">
      <div 
        v-for="item in menuItemsUp" 
        :key="item.path"
        class="nav-item"
        :class="{ 'nav-item-active': currentRoute === item.path }"
        @click="navigateTo(item.path)"
      >
        <span class="nav-icon">{{ item.icon }}</span>
        <span 
          v-show="!sidebarStore.isCollapsed" 
          class="nav-text"
          :class="{ 
            'opacity-0 w-0': sidebarStore.isCollapsed,
            'nav-text-active': currentRoute === item.path
          }"
        >{{ item.label }}</span>
        <div v-if="!sidebarStore.isCollapsed && item.badge" class="nav-badge">
          {{ item.badge }}
        </div>
      </div>
    </nav>

    <!-- 下导航菜单 -->
    <nav class="sidebar-nav-bottom">
      <div 
        v-for="item in menuItemsDown" 
        :key="item.path"
        class="nav-item"
        :class="{ 'nav-item-active': currentRoute === item.path }"
        @click="navigateTo(item.path)"
      >
        <span class="nav-icon">{{ item.icon }}</span>
        <span 
          v-show="!sidebarStore.isCollapsed" 
          class="nav-text"
          :class="{ 
            'opacity-0 w-0': sidebarStore.isCollapsed,
            'nav-text-active': currentRoute === item.path
          }"
        >{{ item.label }}</span>
        <div v-if="!sidebarStore.isCollapsed && item.badge" class="nav-badge">
          {{ item.badge }}
        </div>
      </div>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useSidebarStore } from '../stores/Sidebar'

const sidebarStore = useSidebarStore()

interface MenuItem {
  path: string
  label: string
  icon: string
  badge?: number
}

// 上导航菜单项（从上到下排列）
const menuItemsUp = ref<MenuItem[]>([
  { path: '/', label: '首页', icon: '🏠' },
  { path: '/test', label: '测试', icon: 'ℹ️' },
  { path: '/todo', label: 'TODO', icon: '📒' }
])

// 下导航菜单项（从下到上排列）
const menuItemsDown = ref<MenuItem[]>([
  { path: '/settings', label: '设置', icon: '⚙️' }
])

const router = useRouter()
const route = useRoute()

const currentRoute = computed(() => route.path)

const sidebarClasses = computed(() => ({
  'sidebar-collapsed': sidebarStore.isCollapsed,
  'sidebar-expanded': !sidebarStore.isCollapsed
}))

const navigateTo = (path: string) => {
  router.push(path)
}
</script>