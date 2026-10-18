<!-- frontend/src/components/meta/MetaSubsection.vue
     卡片内部的小标题块：小图标 + 灰色标题 + 内容。
     用来把一张卡里的多条信息（summary / outline / 封面保存位置…）隔开，
     比每个小节都单独开一张 card 更紧凑。 -->
<template>
  <div class="flex flex-col gap-1.5 min-w-0">
    <h3 class="text-xs text-tertiary m-0 flex items-center gap-1.5">
      <span v-if="icon" :class="icon" class="icon-xs" aria-hidden="true" />
      <slot name="title">{{ title }}</slot>
    </h3>
    <slot />
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ icon?: string; title?: string }>(), { icon: '', title: '' })
</script>
