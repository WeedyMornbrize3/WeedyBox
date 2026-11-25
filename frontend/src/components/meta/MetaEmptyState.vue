<!-- frontend/src/components/meta/MetaEmptyState.vue
     空态：还没搜、或搜了没结果时占位。
     用 naive 的 n-empty：一行标题 + 一句提示，不做插画
     —— 桌面工具里空态是"告诉下一步做什么"，不是营销位。 -->
<template>
  <n-empty :description="title" class="py-8">
    <template #icon>
      <span :class="icon" class="icon-xl text-muted" aria-hidden="true" />
    </template>
    <template v-if="hint" #extra>
      <p class="m-0 text-xs text-tertiary max-w-[46ch] text-center">{{ hint }}</p>
    </template>
  </n-empty>
</template>

<script setup lang="ts">
import { NEmpty } from 'naive-ui'

withDefaults(defineProps<{ title: string; hint?: string; icon?: string }>(), {
  hint: '',
  icon: 'i-lucide-inbox',
})
</script>
