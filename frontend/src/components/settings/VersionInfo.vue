<template>
  <span class="text-xs text-tertiary bg-secondary px-3 py-0.75 rounded-md border border-theme-light">
    {{ version }}
  </span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  version?: string
}>(), {
  version: 'beta'
})
</script>