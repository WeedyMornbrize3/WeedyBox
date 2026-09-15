<!--
  语言选择：仍是占位实现（后端/前端均未接入 i18n），但不再是裸露的 <p>占位</p>，
  而是一个禁用态的下拉，视觉上与其它设置项一致，并如实告知尚未支持。
-->
<template>
  <div class="relative flex-shrink-0">
    <select
      :value="modelValue"
      disabled
      aria-label="界面语言（暂未支持切换）"
      class="appearance-none bg-secondary text-secondary text-xs rounded-md pl-3 pr-7 py-1.5 border-theme cursor-not-allowed opacity-70"
      @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
    >
      <option value="zh-CN">简体中文</option>
    </select>
    <span class="i-lucide-chevron-down icon-xs text-tertiary absolute right-2 top-1/2 -translate-y-1/2 pointer-events-none" aria-hidden="true" />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  modelValue?: string
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>
