<!-- frontend/src/components/meta/MetaWorkOverview.vue
     作品概览块：封面 + 名称 + 关键字段 + 标签 + 入库/启动按钮 + 入库回执。
     「详情卡的上半部分」—— 与 MetaWorkDetails（summary / outline）拼起来就是完整详情，
     拆开是为了作品库能把概览放宽、把明细折叠，调试页则两块都摊开。
     按钮的语义由 MetaDownloadButton 自己判断：未入库是「入库」，已入库替换成「游戏启动」
     （位置就是 library_works 的 game_path 列，由 stores/metaStore.ts 读取/启动）。 -->
<template>
  <div class="flex gap-5 items-start flex-wrap">
    <MetaCoverFigure :src="work.coverUrl" :alt="`${work.name} 封面`" />

    <div class="flex-1 min-w-[260px] flex flex-col gap-3">
      <h3 class="text-base font-medium text-primary m-0 leading-snug break-words">{{ work.name }}</h3>

      <MetaKeyValueList :rows="buildSummaryRows(work)" />
      <MetaTags :tags="work.tags" :limit="app.tagLimit" />

      <div class="flex items-center gap-3 flex-wrap pt-1">
        <MetaDownloadButton
          :disabled="!work.coverUrl"
          :loading="downloading"
          :stored="stored"
          :game-path="gamePath"
          :game-path-exists="gamePathExists"
          :running="running"
          :launching="launching"
          @click="emit('download')"
          @launch="emit('launch')"
          @change-path="emit('changePath')"
        />
      </div>

      <MetaSavedNotice v-if="saved" :saved="saved" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DownloadResult, WorkMeta } from '../../services/metaService'
import { useAppStore } from '../../stores/App'
import { buildSummaryRows } from '../../utils/metaFormat'
import MetaCoverFigure from './MetaCoverFigure.vue'
import MetaDownloadButton from './MetaDownloadButton.vue'
import MetaKeyValueList from './MetaKeyValueList.vue'
import MetaSavedNotice from './MetaSavedNotice.vue'
import MetaTags from './MetaTags.vue'

defineProps<{
  work: WorkMeta
  downloading?: boolean
  saved?: DownloadResult | null
  stored?: boolean
  /** 库里记录的游戏程序路径（空 = 还没设置） */
  gamePath?: string
  /** 记录的 exe 是否还存在（false = 被删除/移走） */
  gamePathExists?: boolean
  /** 游戏是否正在运行 */
  running?: boolean
  /** 正在启动游戏 */
  launching?: boolean
}>()

// 标签显示数量是应用级设置（设置页可调），这里只读不写
const app = useAppStore()

const emit = defineEmits<{ download: []; launch: []; changePath: [] }>()
</script>
