<!-- frontend/src/components/meta/MetaWorkTopBar.vue
     作品详情页 hero 下方的那条动作栏（2026-11-24 按用户给的第一张 Steam 页面截图仿制）：
       左：入库 / 游戏启动（+ 启动方式下拉）     ← MetaWorkActions，状态判断仍在 store
       中：库状态 / 入库时间 / 启动方式           ← 每组「小图标 + 灰标签 + 亮值」，模仿截图里的
                                                  云状态 / 最后运行日期 / 游戏时间
       右：两颗方形（圆角）图标按钮 —— 重新抓取、打开原始作品页（截图右侧的设置/信息图标位）
     只负责长相与事件转发；请求都在 stores/metaStore.ts。 -->
<template>
  <section class="work-topbar">
    <div class="flex items-center gap-x-5 gap-y-2 flex-wrap min-w-0 flex-1">
      <MetaWorkActions
        :disabled="disabled"
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

      <!-- 分隔竖线：让「按钮」与「状态」两块在视觉上分开（截图里是同一块，这里因为主题是浅色卡片，加一条细线更清楚） -->
      <span class="work-topbar-sep" aria-hidden="true" />

      <div class="work-topbar-stats">
        <div class="work-topbar-stat">
          <span
            class="icon-sm text-tertiary"
            :class="stored ? 'i-lucide-circle-check' : 'i-lucide-circle-dashed'"
            aria-hidden="true"
          />
          <div class="min-w-0">
            <p class="work-topbar-label">库状态</p>
            <p class="work-topbar-value">{{ stored ? '已入库' : '未入库' }}</p>
          </div>
        </div>

        <div v-if="stored && savedAtText" class="work-topbar-stat">
          <span class="icon-sm text-tertiary i-lucide-clock" aria-hidden="true" />
          <div class="min-w-0">
            <p class="work-topbar-label">入库时间</p>
            <p class="work-topbar-value">{{ savedAtText }}</p>
          </div>
        </div>

        <div v-if="stored" class="work-topbar-stat">
          <span
            class="icon-sm text-tertiary"
            :class="launchMode === 'steam' ? 'i-lucide-cloud' : 'i-lucide-monitor-play'"
            aria-hidden="true"
          />
          <div class="min-w-0">
            <p class="work-topbar-label">启动方式</p>
            <p class="work-topbar-value">{{ launchMode === 'steam' ? '通过 Steam' : '本地程序' }}</p>
          </div>
        </div>
      </div>
    </div>

    <div class="flex items-center gap-2 shrink-0">
      <n-tooltip trigger="hover">
        <template #trigger>
          <span class="work-topbar-icon">
            <n-button
              quaternary
              circle
              size="small"
              :loading="loading"
              :disabled="loading"
              aria-label="重新抓取元数据"
              @click="emit('refetch')"
            >
              <template #icon>
                <span class="i-lucide-refresh-cw icon-sm" aria-hidden="true" />
              </template>
            </n-button>
          </span>
        </template>
        重新抓取元数据
      </n-tooltip>

      <n-tooltip v-if="sourceUrl" trigger="hover">
        <template #trigger>
          <span class="work-topbar-icon">
            <n-button
              quaternary
              circle
              size="small"
              tag="a"
              :href="sourceUrl"
              target="_blank"
              rel="noreferrer noopener"
              :aria-label="sourceLinkLabel"
            >
              <template #icon>
                <span class="i-lucide-external-link icon-sm" aria-hidden="true" />
              </template>
            </n-button>
          </span>
        </template>
        {{ sourceLinkLabel }}
      </n-tooltip>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NTooltip } from 'naive-ui'
import MetaWorkActions from './MetaWorkActions.vue'

const props = withDefaults(
  defineProps<{
    /** 未入库时「入库」按钮是否禁用（没有封面 URL 时不能入库） */
    disabled?: boolean
    downloading?: boolean
    stored?: boolean
    gamePath?: string
    gamePathExists?: boolean
    launchMode?: 'steam' | 'exe'
    steamSource?: boolean
    running?: boolean
    launching?: boolean
    /** 正在重新抓取元数据（图标按钮转圈） */
    loading?: boolean
    /** 原始作品页链接（空 = 不显示那颗图标按钮） */
    sourceUrl?: string
    /** 来源名称（DLsite / Steam），只用在图标的提示文案里 */
    sourceLabel?: string
    /** 入库时间（ISO 字符串） */
    libraryUpdatedAt?: string
  }>(),
  {
    disabled: false,
    downloading: false,
    stored: false,
    gamePath: '',
    gamePathExists: true,
    launchMode: 'exe',
    steamSource: false,
    running: false,
    launching: false,
    loading: false,
    sourceUrl: '',
    sourceLabel: '',
    libraryUpdatedAt: '',
  },
)

const emit = defineEmits<{
  download: []
  launch: []
  changePath: []
  setLaunchMode: [mode: 'steam' | 'exe']
  refetch: []
}>()

/** 入库时间：只显示日期，时间栏才放得下（截图里「最后运行日期 6月18日」也是这个粒度） */
const savedAtText = computed(() => {
  if (!props.libraryUpdatedAt) return ''
  const time = new Date(props.libraryUpdatedAt)
  if (Number.isNaN(time.getTime())) return ''
  return time.toLocaleDateString('zh-CN')
})

const sourceLinkLabel = computed(() =>
  props.sourceLabel ? `打开 ${props.sourceLabel} 原始页面` : '打开原始页面',
)
</script>

<style scoped>
/* 动作栏本体（「启动所在卡片」）：65% 半透明底，让内容下方唯一那张遮罩透出来
   （80% 时遮罩那 12% 的透光几乎看不出来，用户反馈「没有透明效果」）。
   不铺自己的遮罩、不用 backdrop-filter；按钮等 UI 控件自身仍是不透明的。 */
.work-topbar {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 10px 14px;
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--color-border) 82%, transparent);
  border-radius: 0;
  background: color-mix(in srgb, var(--color-bg-card) 65%, transparent);
}

/* 按钮组与状态组之间的竖线 */
.work-topbar-sep {
  width: 1px;
  height: 26px;
  flex-shrink: 0;
  background: var(--color-border-light);
}

/* 右侧图标按钮：截图里是浅色圆角方块（不是圆形），所以外面套一层带底色的方框，
   内部按钮仍是 naive 的 quaternary（悬停才浮出底色），两种状态的层次不打架。 */
.work-topbar-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 1px solid var(--color-border-light);
  border-radius: 7px;
  background: var(--color-bg-secondary);
}

.work-topbar-stats {
  display: flex;
  align-items: center;
  gap: 22px;
  flex-wrap: wrap;
  margin: 0;
}

/* 一组状态：小图标 + 上下两行（灰标签 / 亮值），与截图的「云状态 / 已是最新」同构 */
.work-topbar-stat {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.work-topbar-label {
  margin: 0;
  font-size: 11px;
  line-height: 1.3;
  color: var(--color-text-tertiary);
  white-space: nowrap;
}

.work-topbar-value {
  margin: 0;
  font-size: 13px;
  line-height: 1.4;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
