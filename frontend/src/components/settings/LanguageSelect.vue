<template>
  <!-- 语言选择：仍是占位实现（前后端均未接入 i18n），用禁用的 n-select 如实告知「还没支持」 -->
  <n-select
    :value="modelValue ?? 'zh-CN'"
    :options="options"
    disabled
    size="small"
    aria-label="界面语言（暂未支持切换）"
    style="width: 132px"
  />
</template>

<script setup lang="ts">
import { NSelect } from 'naive-ui'

defineProps<{
  modelValue?: string
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()

const options = [{ label: '简体中文', value: 'zh-CN' }]
</script>
