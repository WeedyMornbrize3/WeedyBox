<!-- frontend/src/components/meta/MetaWorkDetails.vue
     作品明细：简介（summary）+ 信息表（outline）。
     两块都可能很长，所以各自限高自己滚；哪个没有就不渲染，不留空壳。
     showSummary / showOutline 供页面按需裁剪（作品库想收着看时可以只留简介）。 -->
<template>
  <div class="flex flex-col gap-5">
    <MetaTextBlock
      v-if="showSummary && work.summary"
      icon="i-lucide-align-left"
      :title="`作品内容（summary · ${work.summary.length} 字符）`"
      :text="work.summary"
      scrollable
    />

    <MetaSubsection v-if="showOutline && work.outline?.length" icon="i-lucide-table">
      <template #title>Outline（{{ work.outline.length }} 行，保留原始日文标签）</template>
      <MetaKeyValueList :rows="work.outline" dense scrollable />
    </MetaSubsection>
  </div>
</template>

<script setup lang="ts">
import type { WorkMeta } from '../../services/metaService'
import MetaKeyValueList from './MetaKeyValueList.vue'
import MetaSubsection from './MetaSubsection.vue'
import MetaTextBlock from './MetaTextBlock.vue'

withDefaults(
  defineProps<{ work: WorkMeta; showSummary?: boolean; showOutline?: boolean }>(),
  { showSummary: true, showOutline: true }
)
</script>
