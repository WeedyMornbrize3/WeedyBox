<!-- frontend/src/components/todo/FilterChipGroup.vue -->
<!-- 可复用筛选组：一行标签 + 可点选的 n-tag，v-model 绑定当前值。
     无障碍：每组用 role="group" + aria-labelledby 关联可见标签；
     选中态由 n-tag 的 checkable 外观表达（底色 + 品牌色文字 + 字重），
     不依赖颜色单独表意，键盘可达。 -->
<template>
  <div class="filter-group" role="group" :aria-labelledby="labelId">
    <span :id="labelId" class="filter-label">{{ label }}</span>
    <div class="flex flex-nowrap gap-1">
      <n-tag
        v-for="option in options"
        :key="String(option.value)"
        size="small"
        checkable
        :checked="modelValue === option.value"
        :title="option.label"
        @click="$emit('update:modelValue', option.value)"
      >
        {{ option.label }}
      </n-tag>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends string | number | null">
import { useId } from 'vue'
import { NTag } from 'naive-ui'

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
