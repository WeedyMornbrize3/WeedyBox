<!-- frontend/src/components/meta/MetaDownloadButton.vue
     入库按钮：只负责「长相 + 四态（可点 / 禁用 / 入库中 / 已入库）」，真正的请求在 store 里。
     文案用「入库」而不是「下载封面」：这一步确实会把封面下载到 preImg/ 并写进游戏库
     （元数据在抓取时就已入库），对用户来说语义就是"收进库里"。
     「已入库」（stored）= 库里已经有这个作品的封面文件：文字变「已入库」、图标变勾、按钮禁用。
     禁用的两个原因是分开表达的：没有封面 URL（disabled）与已入库（stored）；
     「正在入库」由 loading 单独管，三者叠加也不会让文案说错话。
     已入库后仍可点是有害的：下载实现刻意不覆盖同名文件（重名加 -2/-3 后缀），
     再点一次只会多出一份重复的图。 -->
<template>
  <button
    type="button"
    class="btn-brand flex items-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
    :disabled="disabled || stored || loading"
    @click="emit('click')"
  >
    <span :class="iconClass" class="icon-sm" aria-hidden="true" />
    {{ text }}
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    label?: string
    loadingText?: string
    loading?: boolean
    disabled?: boolean
    stored?: boolean
  }>(),
  { label: '入库', loadingText: '入库中', loading: false, disabled: false, stored: false }
)

const emit = defineEmits<{ click: [] }>()

/** 已入库优先：入库成功的那一轮 loading 也已落回 false，两个条件不会同时为真 */
const text = computed(() => (props.stored ? '已入库' : props.loading ? props.loadingText : props.label))
const iconClass = computed(() =>
  props.stored ? 'i-lucide-check' : props.loading ? 'i-lucide-loader-circle spin' : 'i-lucide-download'
)
</script>
