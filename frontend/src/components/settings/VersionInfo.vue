<template>
  <n-tag size="small" :bordered="true">{{ version }}</n-tag>
</template>

<script setup lang="ts">
import { NTag } from 'naive-ui'

withDefaults(defineProps<{
  version?: string
}>(), {
  version: 'beta'
})
</script>
