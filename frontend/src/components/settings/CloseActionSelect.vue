<template>
  <!-- 关闭按钮行为：与 ThemeModeSelector 同构的分段单选（选项文字短，比下拉更快也更直观） -->
  <n-radio-group
    :value="modelValue"
    size="small"
    name="close-action"
    aria-label="点击关闭时的行为"
    @update:value="onChange"
  >
    <n-radio-button v-for="option in CLOSE_ACTION_OPTIONS" :key="option.value" :value="option.value">
      {{ option.label }}
    </n-radio-button>
  </n-radio-group>
</template>

<script setup lang="ts">
import { NRadioButton, NRadioGroup } from 'naive-ui'
import { CLOSE_ACTION_OPTIONS, type CloseAction } from '../../stores/App'

defineProps<{
  modelValue: CloseAction
}>()

const emit = defineEmits<{
  'update:modelValue': [value: CloseAction]
}>()

function onChange(value: string | number | boolean) {
  emit('update:modelValue', value as CloseAction)
}
</script>
