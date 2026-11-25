<!-- frontend/src/components/settings/ThemePresetSelector.vue
     设置页「主题配色」：选项直接来自 styles/themePresets.ts 的四色主题清单。
     以后新增主题只往清单里加一项，这里与其它代码都不用改。 -->
<template>
  <n-select
    :value="modelValue"
    :options="options"
    size="small"
    class="w-[150px]"
    aria-label="主题配色"
    @update:value="onChange"
  />
</template>

<script setup lang="ts">
import { NSelect } from 'naive-ui'
import { THEME_PRESETS } from '../../styles/themePresets'

defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const options = THEME_PRESETS.map((preset) => ({
  label: preset.name,
  value: preset.id,
}))

function onChange(value: string) {
  emit('update:modelValue', value)
}
</script>
