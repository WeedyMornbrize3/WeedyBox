<!-- frontend/src/components/meta/MetaSavedNotice.vue
     封面下载成功的回执：容量 + 相对路径 + 一键打开本地文件。
     用 file:// 直开是 Wails webview 的便利：不需要额外的"打开文件夹"后端接口。 -->
<template>
  <n-alert type="success" :bordered="true" size="small" :show-icon="true">
    <p class="m-0 text-xs text-success break-all">
      已保存 {{ formatBytes(saved.bytes) }} → <code class="font-mono">{{ saved.path }}</code>
    </p>
    <a
      :href="fileUrl(saved.absPath)"
      target="_blank"
      rel="noreferrer noopener"
      class="text-xs text-brand hover:underline w-fit mt-1 inline-block"
    >
      打开已保存的图片
    </a>
  </n-alert>
</template>

<script setup lang="ts">
import { NAlert } from 'naive-ui'
import type { DownloadResult } from '../../services/metaService'
import { fileUrl, formatBytes } from '../../utils/metaFormat'

defineProps<{ saved: DownloadResult }>()
</script>
