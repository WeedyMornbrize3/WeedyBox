<template>
  <div class="bg-card rounded-xl border border-theme p-4 transition-colors duration-300">
    <div class="flex items-center gap-2.5 mb-3 pb-2.5 border-b border-theme-light">
      <span class="text-[18px]">{{ icon }}</span>
      <h2 class="text-[15px] font-semibold text-primary m-0">{{ title }}</h2>
    </div>
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  icon: string
  title: string
}>()
</script>