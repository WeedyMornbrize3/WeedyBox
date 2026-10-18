<!-- frontend/src/components/meta/MetaTextBlock.vue
     一段可能很长的文本（作品简介）：小标题 + 保留换行的正文，默认限高 220px 自己滚。
     whitespace-pre-wrap 是关键 —— DLsite 的 summary 里换行有意义，压成一行就读不清了。 -->
<template>
  <MetaSubsection :icon="icon">
    <template #title>{{ title }}</template>
    <p
      class="m-0 text-xs leading-relaxed text-secondary whitespace-pre-wrap break-words"
      :class="scrollable ? 'max-h-[220px] overflow-y-auto scrollbar-theme pr-2' : ''"
    >
      {{ text }}
    </p>
  </MetaSubsection>
</template>

<script setup lang="ts">
import MetaSubsection from './MetaSubsection.vue'

withDefaults(defineProps<{ text: string; title: string; icon?: string; scrollable?: boolean }>(), {
  icon: '',
  scrollable: false,
})
</script>
