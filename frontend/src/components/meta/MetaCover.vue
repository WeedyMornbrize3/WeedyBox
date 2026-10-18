<!-- frontend/src/components/meta/MetaCover.vue
     封面缩略图：固定尺寸盒子 + object-cover，加载失败或没有 URL 时退化成占位图标。
     尺寸只有两档（候选行 44px / 详情 140×105），都写成字面量表以配合 UnoCSS 扫描。 -->
<template>
  <div class="border-theme bg-secondary flex-center overflow-hidden shrink-0" :class="SIZE_CLASS[size]">
    <img
      v-if="src && !failed"
      :src="src"
      :alt="alt"
      class="w-full h-full object-cover"
      referrerpolicy="no-referrer"
      @error="failed = true"
    />
    <span v-else :class="FALLBACK_ICON[size]" class="text-muted" aria-hidden="true" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

type Size = 'sm' | 'md'

const props = withDefaults(defineProps<{ src?: string; alt?: string; size?: Size }>(), {
  src: '',
  alt: '',
  size: 'sm',
})

/** 图片加载失败标记：换作品（src 变化）时要复位，否则新封面永远显示不出来。 */
const failed = ref(false)
watch(
  () => props.src,
  () => {
    failed.value = false
  }
)

const SIZE_CLASS: Record<Size, string> = {
  sm: 'w-11 h-11 rounded-md',
  md: 'w-[140px] h-[105px] rounded-lg',
}

const FALLBACK_ICON: Record<Size, string> = {
  sm: 'i-lucide-image icon-sm',
  md: 'i-lucide-image-off icon-lg',
}
</script>
