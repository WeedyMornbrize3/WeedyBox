<template>
  <!-- 主题模式：naive 的分段单选（原来是手写的 chip 组） -->
  <n-radio-group
    :value="modelValue"
    size="small"
    name="theme-mode"
    aria-label="主题模式"
    @update:value="onChange"
  >
    <n-radio-button v-for="option in themeOptions" :key="option.value" :value="option.value">
      <span class="flex items-center gap-1 whitespace-nowrap">
        <span :class="option.icon" class="icon-xs" aria-hidden="true" />
        {{ option.label }}
      </span>
    </n-radio-button>
  </n-radio-group>
</template>

<script setup lang="ts">
import { NRadioButton, NRadioGroup } from 'naive-ui'
import type { ThemeMode } from '../../stores/Theme'

defineProps<{
  modelValue: ThemeMode
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ThemeMode]
}>()

const themeOptions: Array<{ value: ThemeMode; label: string; icon: string }> = [
  { value: 'light', label: '浅色', icon: 'i-lucide-sun' },
  { value: 'dark', label: '深色', icon: 'i-lucide-moon' },
  { value: 'system', label: '系统', icon: 'i-lucide-monitor' },
]

function onChange(value: string | number | boolean) {
  emit('update:modelValue', value as ThemeMode)
}
</script>
