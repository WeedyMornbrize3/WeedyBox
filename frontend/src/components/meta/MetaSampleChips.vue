<!-- frontend/src/components/meta/MetaSampleChips.vue
     快捷示例 chips：省得每次手打编号。只负责展示 + emit，填哪个字段由页面决定。
     用 n-tag 的 checkable 形态（悬停/点击的反馈由 naive 提供），但不持有选中态。 -->
<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <span class="text-xs text-tertiary">{{ label }}</span>
    <n-tag
      v-for="sample in samples"
      :key="sample"
      size="small"
      checkable
      :checked="false"
      @click="emit('pick', sample)"
    >
      {{ sample }}
    </n-tag>
  </div>
</template>

<script setup lang="ts">
import { NTag } from 'naive-ui'

withDefaults(defineProps<{ samples: string[]; label?: string }>(), { label: '示例' })

const emit = defineEmits<{ pick: [sample: string] }>()
</script>
