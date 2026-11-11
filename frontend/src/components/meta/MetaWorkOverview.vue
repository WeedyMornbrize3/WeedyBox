<!-- frontend/src/components/meta/MetaWorkOverview.vue
     作品概览块：封面 + 名称 + 关键字段 + 标签 + 入库按钮 + 入库回执。
     「详情卡的上半部分」—— 与 MetaWorkDetails（summary / outline）拼起来就是完整详情，
     拆开是为了作品库能把概览放宽、把明细折叠，调试页则两块都摊开。
     按钮文案是「入库」（原「下载封面」）：行为没变 —— 下载封面到 preImg/ 并写进游戏库；
     库里已有封面时传 stored，按钮显示「已入库」并禁用（判定在 stores/metaStore.ts）。 -->
<template>
  <div class="flex gap-5 items-start flex-wrap">
    <MetaCoverFigure :src="work.coverUrl" :alt="`${work.name} 封面`" />

    <div class="flex-1 min-w-[260px] flex flex-col gap-3">
      <h3 class="text-base font-medium text-primary m-0 leading-snug break-words">{{ work.name }}</h3>

      <MetaKeyValueList :rows="buildSummaryRows(work)" />
      <MetaTags :tags="work.tags" />

      <div class="flex items-center gap-3 flex-wrap pt-1">
        <MetaDownloadButton
          :disabled="!work.coverUrl"
          :loading="downloading"
          :stored="stored"
          @click="emit('download')"
        />
      </div>

      <MetaSavedNotice v-if="saved" :saved="saved" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DownloadResult, WorkMeta } from '../../services/metaService'
import { buildSummaryRows } from '../../utils/metaFormat'
import MetaCoverFigure from './MetaCoverFigure.vue'
import MetaDownloadButton from './MetaDownloadButton.vue'
import MetaKeyValueList from './MetaKeyValueList.vue'
import MetaSavedNotice from './MetaSavedNotice.vue'
import MetaTags from './MetaTags.vue'

defineProps<{ work: WorkMeta; downloading?: boolean; saved?: DownloadResult | null; stored?: boolean }>()

const emit = defineEmits<{ download: [] }>()
</script>
