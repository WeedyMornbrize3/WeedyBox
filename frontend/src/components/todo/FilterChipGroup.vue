<!-- frontend/src/components/todo/FilterChipGroup.vue -->
<!-- 可复用筛选组：一行标签 + 单选药丸按钮，v-model 绑定当前值 -->
<template>
  <div class="filter-group">
    <span class="filter-label">{{ label }}</span>
    <div class="filter-chips">
      <button
        v-for="option in options"
        :key="String(option.value)"
        type="button"
        class="chip"
        :class="{ 'chip-active': modelValue === option.value }"
        :title="option.label"
        @click="$emit('update:modelValue', option.value)"
      >
        {{ option.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends string | number | null">
defineProps<{
  label: string
  options: { value: T; label: string }[]
  modelValue: T
}>()

defineEmits<{
  'update:modelValue': [value: T]
}>()
</script>
