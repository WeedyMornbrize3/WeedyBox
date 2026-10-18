<!-- frontend/src/components/meta/MetaCoverFigure.vue
     封面 + 下方小字标注（原始 URL 或"未解析到封面"）。
     标注用 break-all 而不是 truncate：URL 很长，换行比截断更有用（要能核对抓的是哪张图）。 -->
<template>
  <figure class="m-0 flex flex-col gap-2 w-[140px] shrink-0">
    <MetaCover :src="src" :alt="alt" size="md" />
    <figcaption class="text-[11px] text-tertiary break-all leading-snug">
      {{ src || fallback }}
    </figcaption>
  </figure>
</template>

<script setup lang="ts">
import MetaCover from './MetaCover.vue'

withDefaults(defineProps<{ src?: string; alt?: string; fallback?: string }>(), {
  src: '',
  alt: '',
  fallback: '（该作品未解析到封面）',
})
</script>
