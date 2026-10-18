<!-- frontend/src/components/meta/MetaSavedNotice.vue
     封面下载成功的回执：容量 + 相对路径 + 一键打开本地文件。
     用 file:// 直开是 Wails webview 的便利：不需要额外的"打开文件夹"后端接口。 -->
<template>
  <div class="flex flex-col gap-1">
    <p class="m-0 text-xs text-success flex items-start gap-1.5">
      <span class="i-lucide-circle-check-big icon-xs shrink-0 mt-[1px]" aria-hidden="true" />
      <span class="min-w-0 break-all">
        已保存 {{ formatBytes(saved.bytes) }} → <code class="font-mono">{{ saved.path }}</code>
      </span>
    </p>
    <a
      :href="fileUrl(saved.absPath)"
      target="_blank"
      rel="noreferrer noopener"
      class="text-xs text-brand hover:underline w-fit"
    >
      打开已保存的图片
    </a>
  </div>
</template>

<script setup lang="ts">
import type { DownloadResult } from '../../services/metaService'
import { fileUrl, formatBytes } from '../../utils/metaFormat'

defineProps<{ saved: DownloadResult }>()
</script>
