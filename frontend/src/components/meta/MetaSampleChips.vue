<!-- frontend/src/components/meta/MetaSampleChips.vue
     快捷示例 chips：省得每次手打编号。只负责展示 + emit，填哪个字段由页面决定。 -->
<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <span class="text-xs text-tertiary">{{ label }}</span>
    <button v-for="sample in samples" :key="sample" type="button" class="chip" @click="emit('pick', sample)">
      {{ sample }}
    </button>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ samples: string[]; label?: string }>(), { label: '示例' })

const emit = defineEmits<{ pick: [sample: string] }>()
</script>
