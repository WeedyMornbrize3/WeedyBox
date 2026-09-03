<template>
  <div class="flex items-center justify-between py-2.5 border-b border-theme-light last:border-none last:pb-0">
    <div class="flex-1 flex flex-col gap-0.5 min-w-0">
      <span class="text-sm font-medium text-primary">{{ label }}</span>
      <span class="text-xs text-tertiary">{{ description }}</span>
    </div>
    <div class="flex items-center gap-3 flex-shrink-0 ml-4">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  description?: string
}>()
</script>