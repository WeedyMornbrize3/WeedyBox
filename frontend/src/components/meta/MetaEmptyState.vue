<!-- frontend/src/components/meta/MetaEmptyState.vue
     空态：还没搜、或搜了没结果时占位。
     一行标题 + 一句提示，不做插画 —— 桌面工具里空态是"告诉下一步做什么"，不是营销位。 -->
<template>
  <div class="card-flat flex flex-col items-center justify-center gap-2 py-12 text-center">
    <span :class="icon" class="icon-xl text-muted" aria-hidden="true" />
    <p class="m-0 text-sm text-secondary">{{ title }}</p>
    <p v-if="hint" class="m-0 text-xs text-tertiary max-w-[46ch]">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ title: string; hint?: string; icon?: string }>(), {
  hint: '',
  icon: 'i-lucide-inbox',
})
</script>
