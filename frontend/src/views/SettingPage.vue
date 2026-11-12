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

      <!-- 标签显示数量：只影响 DLsite 解析出来的标签显示多少，自定义标签始终全显示 -->
      <SettingsItem
        label="标签显示数量"
        description="作品详情里 DLsite 标签最多显示多少个（4–64）；自己添加的标签不受限制"
      >
        <TagLimitInput :model-value="appStore.tagLimit" @update:model-value="appStore.setTagLimit" />
      </SettingsItem>
      
      <SettingsItem label="版本" description="当前应用版本">
        <VersionInfo version="beta" />
      </SettingsItem>
    </SettingsSection>

    <!-- 存储设置 -->
    <SettingsSection icon="i-lucide-hard-drive" title="存储">
      <!-- 清理缓存：进入本页即扫描 cache/ 占用并显示在按钮左边（CacheCleanup 组件里 onMounted 做） -->
      <SettingsItem
        label="清理缓存"
        description="搜索结果的缩略图缓存在 cache/，随时可删；退出应用时也会自动清空（preImg/ 里的入库封面不受影响）"
      >
        <CacheCleanup />
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
import CacheCleanup from '../components/settings/CacheCleanup.vue'
import TagLimitInput from '../components/settings/TagLimitInput.vue'

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