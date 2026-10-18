<!-- frontend/src/components/meta/MetaPreImgPath.vue
     封面落地目录：相对目录名固定 preImg/，绝对路径要靠后端问（工作目录可能变）。
     读取失败时绝对路径留空，只显示相对目录 —— 不报错，因为这只影响"路径长什么样"，
     不影响搜索与下载本身。 -->
<template>
  <div class="flex items-start gap-2 text-xs text-tertiary">
    <span class="i-lucide-folder-open icon-xs shrink-0 mt-[1px]" aria-hidden="true" />
    <span class="min-w-0 break-all">
      封面保存目录（相对工作目录）：<code class="font-mono text-secondary">preImg/</code>
      <template v-if="path">
        <br />绝对路径：<code class="font-mono text-secondary">{{ path }}</code>
      </template>
    </span>
    <button
      type="button"
      class="ml-auto shrink-0 text-xs text-brand hover:underline cursor-pointer bg-transparent border-none"
      @click="emit('refresh')"
    >
      刷新
    </button>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ path?: string }>(), { path: '' })

const emit = defineEmits<{ refresh: [] }>()
</script>
