<!-- frontend/src/components/meta/MetaSearchBar.vue
     搜索行：label + input-theme + btn-brand。
     受控组件（v-model），请求本身不在这里发起 —— 只 emit('submit')，页面/组合式函数决定怎么搜。 -->
<template>
  <form class="flex items-end gap-2" @submit.prevent="emit('submit')">
    <label class="flex-1 min-w-0 flex flex-col gap-1.5">
      <span class="text-xs text-secondary">{{ label }}</span>
      <input
        :value="modelValue"
        type="text"
        class="input-theme w-full"
        :placeholder="placeholder"
        autocomplete="off"
        spellcheck="false"
        @input="onInput"
      />
    </label>

    <button type="submit" class="btn-brand flex items-center gap-1.5 shrink-0" :disabled="loading">
      <span :class="loading ? 'i-lucide-loader-circle spin' : 'i-lucide-search'" class="icon-sm" aria-hidden="true" />
      {{ loading ? loadingText : submitText }}
    </button>
  </form>
</template>

<script setup lang="ts">
// input-theme 自带 control-h（2.2rem）固定高度，所以按钮也用 btn-brand 的同一高度来对齐底边。
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

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>
