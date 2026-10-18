<!-- frontend/src/components/meta/MetaDownloadButton.vue
     下载封面按钮：只负责「长相 + 三态（可点 / 禁用 / 下载中）」，真正的请求在组合式函数里。
     禁用条件是「没有封面 URL」（后端没解析到图）而不是「正在下载」——
     后者由 loading 单独管，两个原因分开表达，按钮文案才能准确。 -->
<template>
  <button
    type="button"
    class="btn-brand flex items-center gap-1.5"
    :disabled="disabled || loading"
    @click="emit('click')"
  >
    <span :class="loading ? 'i-lucide-loader-circle spin' : 'i-lucide-download'" class="icon-sm" aria-hidden="true" />
    {{ loading ? loadingText : label }}
  </button>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{ label?: string; loadingText?: string; loading?: boolean; disabled?: boolean }>(),
  { label: '下载封面', loadingText: '下载中', loading: false, disabled: false }
)

const emit = defineEmits<{ click: [] }>()
</script>
