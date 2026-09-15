<template>
  <div class="settings-item">
    <div class="item-info">
      <span class="item-label">{{ label }}</span>
      <span v-if="description" class="item-description">{{ description }}</span>
    </div>
    <div class="item-control">
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
