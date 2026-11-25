<!-- frontend/src/components/settings/TagLimitInput.vue
     设置页里的紧凑数字控件（2026-11-24 改为 naive 的 n-input-number，带上下步进按钮）。
     设置页规范：约 28px 高、12px 字，不用 35.2px 的大按钮；
     边界（4–64）由 stores/App.ts 的常量统一约束，这里只负责交互与显示。 -->
<template>
  <n-input-number
    :value="modelValue"
    :min="TAG_LIMIT_MIN"
    :max="TAG_LIMIT_MAX"
    :step="1"
    size="small"
    button-placement="both"
    aria-label="作品标签显示数量"
    style="width: 104px"
    @update:value="onChange"
  />
</template>

<script setup lang="ts">
import { NInputNumber } from 'naive-ui'
import { TAG_LIMIT_MAX, TAG_LIMIT_MIN } from '../../stores/App'

const props = defineProps<{ modelValue: number }>()
const emit = defineEmits<{ 'update:modelValue': [value: number] }>()

function clamp(value: number): number {
  return Math.min(TAG_LIMIT_MAX, Math.max(TAG_LIMIT_MIN, Math.round(value)))
}

function onChange(value: number | null) {
  // 清空输入框时 value 是 null：保持原值，不把设置写成 0
  emit('update:modelValue', typeof value === 'number' ? clamp(value) : props.modelValue)
}
</script>
