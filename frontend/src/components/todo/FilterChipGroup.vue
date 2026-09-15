<!-- frontend/src/components/todo/FilterChipGroup.vue -->
<!-- 可复用筛选组：一行标签 + 单选药丸按钮，v-model 绑定当前值。
     无障碍：每组用 role="group" + aria-labelledby 关联可见标签；
     每个药丸是 <button aria-pressed>，选中态同时体现在边框/底色/字重上，
     不依赖颜色单独表意，键盘 Tab 可达。 -->
<template>
  <div class="filter-group" role="group" :aria-labelledby="labelId">
    <span :id="labelId" class="filter-label">{{ label }}</span>
    <div class="filter-chips">
      <button
        v-for="option in options"
        :key="String(option.value)"
        type="button"
        class="chip"
        :class="{ 'chip-active': modelValue === option.value }"
        :aria-pressed="modelValue === option.value"
        :title="option.label"
        @click="$emit('update:modelValue', option.value)"
      >
        {{ option.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends string | number | null">
import { useId } from 'vue'

defineProps<{
  label: string
  options: { value: T; label: string }[]
  modelValue: T
}>()

defineEmits<{
  'update:modelValue': [value: T]
}>()

// useId 生成稳定的唯一 id，避免同一维度出现多组时 id 冲突
const labelId = useId()
</script>
