<!-- frontend/src/components/meta/MetaMessage.vue
     一行提示（错误 / 状态 / 成功），统一图标 + 颜色 + 换行策略。
     toast 不适合这里：抓取结果需要留在屏幕上对照着看，不能 2 秒就消失。 -->
<template>
  <p class="m-0 flex items-start gap-2 text-xs" :class="TONE_CLASS[tone]">
    <span :class="TONE_ICON[tone]" class="icon-xs shrink-0 mt-[1px]" aria-hidden="true" />
    <span class="min-w-0 break-words">{{ text }}</span>
  </p>
</template>

<script setup lang="ts">
type Tone = 'info' | 'error' | 'success'

withDefaults(defineProps<{ text: string; tone?: Tone }>(), { tone: 'info' })

// 写成字面量表：UnoCSS 只扫源码里的完整类名，运行时拼出来的字符串不会被生成。
const TONE_CLASS: Record<Tone, string> = {
  info: 'text-secondary',
  error: 'text-danger',
  success: 'text-success',
}

const TONE_ICON: Record<Tone, string> = {
  info: 'i-lucide-info',
  error: 'i-lucide-triangle-alert',
  success: 'i-lucide-circle-check-big',
}
</script>
