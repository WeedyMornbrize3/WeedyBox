<template>
  <div class="flex gap-1 bg-secondary p-[3px] rounded-[10px] border border-theme">
    <button
      v-for="option in themeOptions"
      :key="option.value"
      class="flex items-center gap-1 px-3 py-1.25 border-none rounded-lg cursor-pointer bg-transparent text-secondary text-xs transition-all duration-200 whitespace-nowrap hover:text-primary hover:bg-hover"
      :class="{ 
        '!bg-card !text-primary shadow-[0_1px_4px_var(--color-shadow)]': modelValue === option.value
      }"
      @click="$emit('update:modelValue', option.value)"
      :title="option.label"
    >
      <span class="text-sm">{{ option.icon }}</span>
      <span class="font-medium">{{ option.label }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: 'light' | 'dark' | 'system'
}>()

defineEmits<{
  'update:modelValue': [value: 'light' | 'dark' | 'system']
}>()

const themeOptions = [
  { value: 'light' as const, label: '浅色', icon: '☀️' },
  { value: 'dark' as const, label: '深色', icon: '🌙' },
  { value: 'system' as const, label: '系统', icon: '🔄' }
]
</script>