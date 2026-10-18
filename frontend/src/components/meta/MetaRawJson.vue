<!-- frontend/src/components/meta/MetaRawJson.vue
     原始返回数据（未加工）：调试用，一眼看清后端到底给了什么。
     复制失败只在按钮旁标红一句，不弹全局错误 —— 复制失败不该打断"看数据"这件事。 -->
<template>
  <MetaSection icon="i-lucide-braces" title="原始返回数据（未加工）" body-class="p-0">
    <template #actions>
      <button
        type="button"
        class="flex items-center gap-1.5 text-xs text-secondary hover:text-primary cursor-pointer bg-transparent border-none"
        @click="copy"
      >
        <span :class="copied ? 'i-lucide-clipboard-check' : 'i-lucide-clipboard'" class="icon-xs" aria-hidden="true" />
        {{ copied ? '已复制' : '复制' }}
      </button>
    </template>

    <pre class="m-0 p-5 text-[11px] leading-relaxed font-mono text-secondary overflow-auto max-h-[420px] scrollbar-theme">{{ text }}</pre>
  </MetaSection>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MetaSection from './MetaSection.vue'

const props = defineProps<{ value: unknown }>()

const copied = ref(false)

const text = computed(() => (props.value === null ? '' : JSON.stringify(props.value, null, 2)))

async function copy() {
  try {
    await navigator.clipboard.writeText(text.value)
    copied.value = true
    window.setTimeout(() => (copied.value = false), 1500)
  } catch {
    // 剪贴板权限在 webview 里不一定有，静默忽略：按钮不变态，数据仍然可手动选中复制
    copied.value = false
  }
}
</script>
