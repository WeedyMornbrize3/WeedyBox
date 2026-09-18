<template>
  <!-- 关闭按钮行为选择：与 ThemeModeSelector 同构的单选按钮组，
       选项文本短（4 字），用按钮组比下拉更快也更直观。 -->
  <div class="flex items-center gap-1 bg-secondary p-[3px] rounded-[10px] border-theme" role="group" aria-label="点击关闭时的行为">
    <button
      v-for="option in CLOSE_ACTION_OPTIONS"
      :key="option.value"
      type="button"
      class="theme-option"
      :class="{ 'theme-option-active': modelValue === option.value }"
      :aria-pressed="modelValue === option.value"
      :title="option.description"
      @click="$emit('update:modelValue', option.value)"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { CLOSE_ACTION_OPTIONS, type CloseAction } from '../../stores/App'

defineProps<{
  modelValue: CloseAction
}>()

defineEmits<{
  'update:modelValue': [value: CloseAction]
}>()
</script>
