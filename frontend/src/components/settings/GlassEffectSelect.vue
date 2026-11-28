<!-- frontend/src/components/settings/GlassEffectSelect.vue
     设置页「玻璃效果」：**按明暗模式分别勾选**（用户 2026-11-28）：
       「浅色模式默认不使用 blur hero 模糊遮罩，设置里面可以调整，
        选择在什么模式下使用玻璃效果，通过勾选决定」
     · 勾上 = 作品详情页铺那张预模糊 hero 遮罩、卡片与动作栏半透明；
     · 不勾 = 完全不渲染遮罩层，卡片保持实色底（浅色默认不勾）。
     状态在 stores/App.ts（glassLight / glassDark），这里只做两行勾选。 -->
<template>
  <div class="flex items-center gap-4">
    <n-checkbox
      :checked="appStore.glassLight"
      size="small"
      @update:checked="(value: boolean) => appStore.setGlass('light', value)"
    >
      <span class="flex items-center gap-1 whitespace-nowrap">
        <span class="i-lucide-sun icon-xs" aria-hidden="true" />
        浅色模式
      </span>
    </n-checkbox>

    <n-checkbox
      :checked="appStore.glassDark"
      size="small"
      @update:checked="(value: boolean) => appStore.setGlass('dark', value)"
    >
      <span class="flex items-center gap-1 whitespace-nowrap">
        <span class="i-lucide-moon icon-xs" aria-hidden="true" />
        深色模式
      </span>
    </n-checkbox>
  </div>
</template>

<script setup lang="ts">
import { NCheckbox } from 'naive-ui'
import { useAppStore } from '../../stores/App'

const appStore = useAppStore()
</script>
