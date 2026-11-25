<script setup lang="ts">
import { watch } from 'vue'
import { NConfigProvider } from 'naive-ui'
import './styles/global.css'
import { naiveConfig, naiveDateLocale, naiveLocale, syncNaiveTheme } from './styles/naive'
import { useThemeStore } from './stores/Theme'

const themeStore = useThemeStore()

// 主题切换（明暗模式 或 四色主题）→ 重新读 token → 更新 naive 主题。
// flush: 'post' 保证 store 的 applyTheme() 已经把新的 --color-* 写到 <html> 上，
// 否则会读到上一套主题的变量（换肤后控件颜色不跟着变）。
watch(
  () => `${themeStore.effectiveTheme}:${themeStore.themeId}`,
  () => syncNaiveTheme(themeStore.isDark),
  { immediate: true, flush: 'post' },
)
</script>

<template>
  <!-- naive-ui 的全局入口：主题、主题覆盖、中文文案都在这里注入 -->
  <n-config-provider
    :theme="naiveConfig.theme"
    :theme-overrides="naiveConfig.themeOverrides"
    :locale="naiveLocale"
    :date-locale="naiveDateLocale"
    class="h-full"
  >
    <!-- ⚠️ 这里必须是 router-view，不能直接写 <MainLayout />：
         MainLayout 已经是路由 '/' 的布局组件（见 router/index.ts），直接挂一份会让
         router-view 再按 '/' 匹配渲染第二份 MainLayout —— DOM 里就会出现两套
         Sidebar / TitleBar（完全重合看不出来），而页面还被套了两层 padding。 -->
    <router-view />
  </n-config-provider>
</template>

<style>
/* 全局设置文字不能选中 */
* {
  user-select: none;
  -webkit-user-select: none;
}

input,
textarea,
[contenteditable="true"],
.selectable,
.selectable * {
  user-select: text !important;
  -webkit-user-select: text !important;
}
</style>
