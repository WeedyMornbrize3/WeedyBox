<template>
  <aside
    class="sidebar"
    :style="{ width: sidebarStore.sidebarWidth + 'px' }"
  >
    <!-- Logo / 折叠开关：整块头部可点，行为与换 naive 之前一致 -->
    <div class="sidebar-header" @click="sidebarStore.toggleSidebar">
      <div class="logo-icon">
        <!-- 统一图标族（lucide 线性 2px），与全站图标风格一致 -->
        <span class="i-lucide-package icon-lg" aria-hidden="true" />
      </div>
      <span v-show="!sidebarStore.isCollapsed" class="logo-text">WeedyBox</span>
    </div>

    <!-- 上导航
         2026-11-25 起交给 naive 的 n-menu（用户要求「改用 naive ui 样式和逻辑」）：
         选中态 = 当前路由、折叠态只留图标并在 hover 时弹 tooltip、项间距/圆角/配色
         全部由 naive 的 Menu 主题驱动（见 styles/naiveTheme.ts 的 Menu 段）。 -->
    <nav class="sidebar-nav-top" aria-label="主导航">
      <n-menu v-bind="menuProps" :options="menuItemsUp" />
    </nav>

    <!-- 下导航（设置）：与上导航共用同一份 props，选中态同样跟着路由走 -->
    <nav class="sidebar-nav-bottom" aria-label="辅助导航">
      <n-menu v-bind="menuProps" :options="menuItemsDown" />
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NMenu, type MenuNodeProps, type MenuOption } from 'naive-ui'
import { useSidebarStore } from '../stores/Sidebar'

const sidebarStore = useSidebarStore()
const router = useRouter()
const route = useRoute()

interface MenuItem {
  /** 路由路径直接当菜单 key：选中态 = 当前路由，不需要额外的映射表 */
  path: string
  label: string
  /** UnoCSS 图标类名（presetIcons 构建期内联）；不用 emoji——跨平台渲染不一致，且没法跟主题色 */
  icon: string
}

/** 图标尺寸：同时喂给 n-menu 的 icon-size / collapsed-icon-size（折叠态靠它居中） */
const ICON_SIZE = 20

/**
 * 折叠态菜单宽度 = 侧栏 72px − 上下容器的 8px 内边距 ×2。
 * naive 折叠时用 `collapsedWidth / 2 - iconSize / 2` 当左内边距，宽度给不准图标就不会居中。
 */
const MENU_COLLAPSED_WIDTH = 56

/** 展开态项左内边距：8px 容器内边距 + 12px，与换之前自绘 item 的 px-4 对齐 */
const MENU_INDENT = 12

// 上导航菜单项（从上到下排列）
// 注意：**作品库（/works）刻意不出现在侧栏** —— 它只从游戏库点封面进入
// （见 views/LibraryPage.vue 的 openWork），免得导航里出现两个"库"让人分不清。
const menuItemsUp: MenuOption[] = [
  { path: '/', label: '首页', icon: 'i-lucide-house' },
  { path: '/library', label: '游戏库', icon: 'i-lucide-gamepad-2' },
  { path: '/todo', label: 'TODO', icon: 'i-lucide-list-todo' },
  { path: '/testing', label: '测试', icon: 'i-lucide-flask-conical' },
].map(toMenuOption)

// 下导航菜单项（沉在底部）
const menuItemsDown: MenuOption[] = [
  { path: '/settings', label: '设置', icon: 'i-lucide-settings' },
].map(toMenuOption)

/**
 * MenuItem → n-menu 的 option。
 * icon 用渲染函数而不是内置图标组件：项目图标是 UnoCSS 的 `i-lucide-*` 类（构建期内联成
 * data URI），图标盒子（宽高 / 字号 / 右外边距）由 naive 的 `.n-menu-item-content__icon` 负责。
 */
function toMenuOption(item: MenuItem): MenuOption {
  return {
    key: item.path,
    label: item.label,
    icon: () => h('span', { class: item.icon, 'aria-hidden': 'true' }),
  }
}

const navigateTo = (path: string) => {
  if (path !== route.path) void router.push(path)
}

/**
 * naive 的菜单项只给了 role="menuitem"，本身不可聚焦；这里用官方扩展点 `nodeProps`
 * 补 tabindex 与回车/空格，键盘也能切页（焦点环由 global.css 的 :focus-visible 兜底）。
 *
 * ⚠️ naive 把 nodeProps 的返回类型定成 `HTMLAttributes & Record<string, string | number | undefined>`
 *    ——函数类型的 onKeydown 与那个 Record 天然互斥（库自身的类型缺口），所以这里做一次显式转换。
 */
const menuNodeProps = ((option: MenuOption) => ({
  tabindex: 0,
  onKeydown: (event: KeyboardEvent) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      navigateTo(String(option.key))
    }
  },
})) as unknown as MenuNodeProps

/** 上下两个 n-menu 共用的 props（选中值、折叠配置、跳转回调） */
const menuProps = computed(() => ({
  value: route.path,
  collapsed: sidebarStore.isCollapsed,
  collapsedWidth: MENU_COLLAPSED_WIDTH,
  iconSize: ICON_SIZE,
  collapsedIconSize: ICON_SIZE,
  indent: MENU_INDENT,
  rootIndent: MENU_INDENT,
  nodeProps: menuNodeProps,
  'onUpdate:value': (key: string | number) => navigateTo(String(key)),
}))
</script>
