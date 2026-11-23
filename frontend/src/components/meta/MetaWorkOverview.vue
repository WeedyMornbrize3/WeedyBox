<!-- frontend/src/components/meta/MetaWorkOverview.vue
     作品概览块：封面 + 名称 + 关键字段 + 标签 + 入库/启动按钮 + 入库回执。
     「详情卡的上半部分」—— 与 MetaWorkDetails（summary / outline）拼起来就是完整详情，
     拆开是为了调试页（/testing）能把概览放宽、把明细折叠。
     动作区（入库 / 游戏启动 + 启动方式）已抽成 MetaWorkActions：/works 的新排版
     把它放在右侧主区顶部，调试页仍放在这里，两处共用同一份。
     按钮的语义由 MetaDownloadButton 自己判断：未入库是「入库」，已入库替换成「游戏启动」
     （位置就是 library_works 的 game_path 列，由 stores/metaStore.ts 读取/启动）。 -->
<template>
  <div class="flex gap-5 items-start flex-wrap">
    <MetaCoverFigure :src="work.coverUrl" :alt="`${work.name} 封面`" />

    <div class="flex-1 min-w-[260px] flex flex-col gap-3">
      <h3 class="text-base font-medium text-primary m-0 leading-snug break-words">{{ work.name }}</h3>

      <MetaKeyValueList :rows="buildSummaryRows(work)" />
      <!-- 标签云：作品库由 MetaTagEditor 统一管（可编辑 + 保存），这里只在调试页显示 -->
      <MetaTags v-if="showTags" :tags="work.tags" :limit="app.tagLimit" />

      <MetaWorkActions
        :disabled="!work.coverUrl"
        :downloading="downloading"
        :stored="stored"
        :game-path="gamePath"
        :game-path-exists="gamePathExists"
        :launch-mode="launchMode"
        :steam-source="steamSource"
        :running="running"
        :launching="launching"
        @download="emit('download')"
        @launch="emit('launch')"
        @change-path="emit('changePath')"
        @set-launch-mode="emit('setLaunchMode', $event)"
      />

      <MetaSavedNotice v-if="saved" :saved="saved" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DownloadResult, WorkMeta } from '../../services/metaService'
import { useAppStore } from '../../stores/App'
import { buildSummaryRows } from '../../utils/metaFormat'
import MetaCoverFigure from './MetaCoverFigure.vue'
import MetaKeyValueList from './MetaKeyValueList.vue'
import MetaSavedNotice from './MetaSavedNotice.vue'
import MetaTags from './MetaTags.vue'
import MetaWorkActions from './MetaWorkActions.vue'

defineProps<{
  work: WorkMeta
  downloading?: boolean
  saved?: DownloadResult | null
  stored?: boolean
  /** 库里记录的游戏程序路径（空 = 还没设置） */
  gamePath?: string
  /** 记录的 exe 是否还存在（false = 被删除/移走） */
  gamePathExists?: boolean
  /** 生效的启动方式（Steam 源才有 steam 这个选项） */
  launchMode?: 'steam' | 'exe'
  /** 当前作品是否来自 Steam 源：只有它能选「通过 Steam 启动 / 本地程序启动」 */
  steamSource?: boolean
  /** 游戏是否正在运行 */
  running?: boolean
  /** 正在启动游戏 */
  launching?: boolean
  /** 是否显示抓取来的标签云（作品库传 false：标签统一交给 MetaTagEditor，避免同页出现两份） */
  showTags?: boolean
}>()

// 标签显示数量是应用级设置（设置页可调），这里只读不写
const app = useAppStore()

const emit = defineEmits<{
  download: []
  launch: []
  changePath: []
  setLaunchMode: [mode: 'steam' | 'exe']
}>()
</script>
