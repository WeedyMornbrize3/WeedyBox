<template>
  <aside 
    class="sidebar"
    :class="sidebarClasses"
    :style="{ width: sidebarStore.sidebarWidth + 'px' }"
  >
    <!-- Logo/标题区域 -->
    <div class="sidebar-header" @click="sidebarStore.toggleSidebar">
      <div class="logo-icon">
        <!-- 手绘 svg → 统一图标族（lucide 线性 2px），与全站图标风格一致 -->
        <span class="i-lucide-package icon-lg" aria-hidden="true" />
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
        <span class="nav-icon icon-md" :class="item.icon" aria-hidden="true" />
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
        <span class="nav-icon icon-md" :class="item.icon" aria-hidden="true" />
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
// icon 存 UnoCSS 图标类名（presetIcons 按需内联），不再使用 emoji——
// emoji 依赖字体、跨平台渲染不一致，且无法用主题变量控色。
const menuItemsUp = ref<MenuItem[]>([
  { path: '/', label: '首页', icon: 'i-lucide-house' },
  { path: '/test', label: '测试', icon: 'i-lucide-flask-conical' },
  { path: '/todo', label: 'TODO', icon: 'i-lucide-list-todo' }
])

// 下导航菜单项（从下到上排列）
const menuItemsDown = ref<MenuItem[]>([
  { path: '/settings', label: '设置', icon: 'i-lucide-settings' }
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