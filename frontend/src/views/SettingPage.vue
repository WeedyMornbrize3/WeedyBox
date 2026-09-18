<!-- src/views/Settings.vue -->
<template>
  <SettingsPage>
    <!-- 外观设置 -->
    <SettingsSection icon="i-lucide-palette" title="外观">
      <!-- 深色模式切换 -->
      <SettingsItem :label="modeDes" description="点击切换 深色/浅色 模式">
        <ThemeToggle />
      </SettingsItem>

      <!-- 主题模式选择 -->
      <SettingsItem label="主题模式" description="选择您偏好的主题模式">
        <ThemeModeSelector v-model="themeMode" />
      </SettingsItem>
    </SettingsSection>

    <!-- 通用设置 -->
    <SettingsSection icon="i-lucide-sliders-horizontal" title="通用">
      <!-- 关闭按钮行为：与 TitleBar 的 X 联动 -->
      <SettingsItem label="点击关闭时" :description="closeActionDescription">
        <CloseActionSelect v-model="closeAction" />
      </SettingsItem>

      <SettingsItem label="语言" description="选择界面语言">
        <LanguageSelect v-model="language" />
      </SettingsItem>
      
      <SettingsItem label="版本" description="当前应用版本">
        <VersionInfo version="beta" />
      </SettingsItem>
    </SettingsSection>
  </SettingsPage>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useThemeStore } from '../stores/Theme'
import { useAppStore, CLOSE_ACTION_OPTIONS } from '../stores/App'
import SettingsPage from '../components/settings/SettingsPage.vue'
import SettingsSection from '../components/settings/SettingsSection.vue'
import SettingsItem from '../components/settings/SettingsItem.vue'
import ThemeToggle from '../components/settings/ThemeToggle.vue'
import ThemeModeSelector from '../components/settings/ThemeModeSelector.vue'
import CloseActionSelect from '../components/settings/CloseActionSelect.vue'
import LanguageSelect from '../components/settings/LanguageSelect.vue'
import VersionInfo from '../components/settings/VersionInfo.vue'

const themeStore = useThemeStore()
const themeMode = computed({
  get: () => themeStore.mode,
  set: (value: 'light' | 'dark' | 'system') => themeStore.setMode(value)
})

const modeDes = computed(() => 
  themeStore.isDark ? '深色模式' : '浅色模式'
)

// 标题栏关闭按钮行为
const appStore = useAppStore()
const closeAction = computed({
  get: () => appStore.closeAction,
  set: (value) => appStore.setCloseAction(value),
})
const closeActionDescription = computed(
  () => CLOSE_ACTION_OPTIONS.find((o) => o.value === appStore.closeAction)?.description ?? ''
)

const language = ref('zh-CN')
</script>

<style scoped>
/* 响应式样式保留 */
@media (max-width: 600px) {
  .settings-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }
  
  .item-control {
    width: 100%;
    margin-left: 0;
  }
  
  .theme-toggle-btn {
    width: 100%;
    height: 44px;
  }
  
  .theme-options {
    width: 100%;
    justify-content: stretch;
  }
  
  .theme-option {
    flex: 1;
    justify-content: center;
  }
  
  .settings-select {
    width: 100%;
    min-width: unset;
  }
}
</style>