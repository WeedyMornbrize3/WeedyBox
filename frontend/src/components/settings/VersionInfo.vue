<template>
  <span class="version-text">{{ version }}</span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  version?: string
}>(), {
  version: 'beta'
})
</script>