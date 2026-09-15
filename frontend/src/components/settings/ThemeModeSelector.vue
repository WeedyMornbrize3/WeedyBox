<template>
  <div class="flex items-center gap-1 bg-secondary p-[3px] rounded-[10px] border-theme" role="group" aria-label="主题模式">
    <button
      v-for="option in themeOptions"
      :key="option.value"
      type="button"
      class="theme-option"
      :class="{ 'theme-option-active': modelValue === option.value }"
      :aria-pressed="modelValue === option.value"
      :title="option.label"
      @click="$emit('update:modelValue', option.value)"
    >
      <span :class="[option.icon, 'icon-xs']" aria-hidden="true" />
      <span>{{ option.label }}</span>
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
  { value: 'light' as const, label: '浅色', icon: 'i-lucide-sun' },
  { value: 'dark' as const, label: '深色', icon: 'i-lucide-moon' },
  { value: 'system' as const, label: '系统', icon: 'i-lucide-monitor' }
]
</script>
