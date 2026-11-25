<!-- frontend/src/components/meta/MetaCoversPath.vue
     封面落地目录：相对目录名固定 covers/，绝对路径要靠后端问（工作目录可能变）。
     读取失败时绝对路径留空，只显示相对目录 —— 不报错，因为这只影响"路径长什么样"，
     不影响搜索与下载本身。 -->
<template>
  <div class="flex items-start gap-2 text-xs text-tertiary">
    <span class="i-lucide-folder-open icon-xs shrink-0 mt-[1px]" aria-hidden="true" />
    <span class="min-w-0 break-all">
      封面保存目录（相对工作目录）：<code class="font-mono text-secondary">covers/</code>
      <template v-if="path">
        <br />绝对路径：<code class="font-mono text-secondary">{{ path }}</code>
      </template>
    </span>
    <n-button text size="tiny" type="primary" class="ml-auto shrink-0" @click="emit('refresh')">
      刷新
    </n-button>
  </div>
</template>

<script setup lang="ts">
import { NButton } from 'naive-ui'

withDefaults(defineProps<{ path?: string }>(), { path: '' })

const emit = defineEmits<{ refresh: [] }>()
</script>
