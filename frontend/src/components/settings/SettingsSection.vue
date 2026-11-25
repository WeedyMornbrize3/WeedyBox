<template>
  <!-- 设置页的卡片外壳：改用 naive 的 n-card（底色/描边/圆角全部来自 styles/naiveTheme.ts） -->
  <n-card size="small" :bordered="true">
    <template #header>
      <div class="flex items-center gap-2.5">
        <span class="text-brand flex-center icon-md" :class="icon" aria-hidden="true" />
        <span class="text-[15px] font-semibold text-primary">{{ title }}</span>
      </div>
    </template>
    <slot />
  </n-card>
</template>

<script setup lang="ts">
import { NCard } from 'naive-ui'

defineProps<{
  // UnoCSS 图标类名，如 i-lucide-palette（不再传 emoji）
  icon: string
  title: string
}>()
</script>
