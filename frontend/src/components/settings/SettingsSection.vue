<template>
  <div class="settings-section">
    <div class="section-header">
      <span class="section-icon icon-md" :class="icon" aria-hidden="true" />
      <h2 class="section-title">{{ title }}</h2>
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  // UnoCSS 图标类名，如 i-lucide-palette（不再传 emoji）
  icon: string
  title: string
}>()
</script>