<!-- frontend/src/components/meta/MetaSearchBar.vue
     搜索行：label + n-input + n-button（primary）。
     受控组件（v-model），请求本身不在这里发起 —— 只 emit('submit')，页面/组合式函数决定怎么搜。 -->
<template>
  <form class="flex items-end gap-2" @submit.prevent="emit('submit')">
    <label class="flex-1 min-w-0 flex flex-col gap-1.5">
      <span class="text-xs text-secondary">{{ label }}</span>
      <n-input
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="loading"
        autocomplete="off"
        spellcheck="false"
        @update:value="onInput"
      />
    </label>

    <n-button type="primary" attr-type="submit" :loading="loading" class="shrink-0">
      <template #icon>
        <span class="i-lucide-search icon-sm" aria-hidden="true" />
      </template>
      {{ loading ? loadingText : submitText }}
    </n-button>
  </form>
</template>

<script setup lang="ts">
import { NButton, NInput } from 'naive-ui'

withDefaults(
  defineProps<{
    modelValue: string
    label?: string
    placeholder?: string
    submitText?: string
    loadingText?: string
    loading?: boolean
  }>(),
  {
    label: '作品名或 RJ 编号',
    placeholder: '输入RJ号或名称搜索...',
    submitText: '搜索',
    loadingText: '请求中',
    loading: false,
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  submit: []
}>()

function onInput(value: string) {
  emit('update:modelValue', value)
}
</script>
