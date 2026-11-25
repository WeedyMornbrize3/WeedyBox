<!-- frontend/src/views/WorksPage.vue
     作品库（/works）：只展示「一个作品的信息」。
     入口：游戏库（/library）里点封面（点谁就 loadById 谁，走缓存优先）；侧栏没有入口。
     搜索不在这页 —— 入口在游戏库顶栏的「添加游戏」（居中弹窗，见 LibraryPage.vue）。

     排版（2026-11-24 用户按 Steam 页面截图重排）：
       ┌ hero 大背景图（本机 Steam librarycache 的 hero 图 / DLsite 第二张预览图）
       │   左下：源·社团·发售日·形式·评分 + 作品名 + 标签；右下：竖版封面
       ├ 动作栏（MetaWorkTopBar）：入库/启动 + 启动方式 + 库状态/入库时间/启动方式 + 右侧图标
       ├ 下划线 tab（仿截图二的选中样式）：作品信息 / 游戏统计 / 启动配置 / 编辑 / 评价
       └ 各 tab 面板（简介+Outline+元数据 / Steam 动态+统计 / 启动方式与位置 / 元数据表单+标签 / 评分与评价入口）
     原来的左信息栏（MetaWorkSidebar）已删除：可展示的字段并进 hero 小字与各 tab。
     与调试页（views/test.vue → MetaApiTestCard.vue）共用 stores/metaStore.ts 与 components/meta/*。 -->
<template>
  <!-- 全站唯一一张模糊遮罩（用户 2026-11-25：「只使用一张模糊遮罩 … 不使用多张遮罩」）：
       --work-hero-image = 后端预模糊好的那张图，只铺在 .work-page-ambient 这一层；
       各内容容器用半透明底透出它（2026-11-25 用户要求取消视差，遮罩保持静态）。 -->
  <div ref="pageEl" class="works-page w-full max-w-6xl mx-auto h-full min-h-0 flex flex-col" :style="glassStyle">
    <div
      v-if="meta.heroBlurred"
      class="work-page-ambient"
      aria-hidden="true"
    />
    <!-- 自己开滚动（MainLayout 的 router-view 是 overflow-hidden）；上下边缘按滚动位置渐隐。
         overflow-x-hidden：遮罩层用负 inset / scale 外扩，曾把容器撑出 27px 横向溢出。 -->
    <div
      ref="scrollEl"
      v-scroll-fade
      class="scroll-fade flex-1 min-h-0 overflow-y-auto overflow-x-hidden scrollbar-theme pr-2 flex flex-col gap-2 pb-2 select-text"
      @scroll.passive="handleScroll"
    >
      <template v-if="meta.work">
        <!-- hero 与动作栏包在一层「模糊 hero 遮罩」里（2026-11-25 用户口径：
             hero 画面本身不模糊；用**后端预模糊好的一张图**给所有 UI 做遮罩，
             不做玻璃反光效果，也不在 CSS 里做 filter: blur）。
             · --work-hero-image = 那张预模糊图的 data URL（所有遮罩层共用同一张 bitmap）；
             · 动作栏内部盖一层同图遮罩（向右淡出），见 MetaWorkTopBar 的 ::after。 -->
        <div class="work-glass">

          <!-- 顶部大图：背景图 + 作品名 + 标签 + 竖版封面 -->
          <MetaWorkHero
            :work="meta.work"
            :background-src="meta.backgroundUrl"
            :cover-src="coverSrc"
            :tags="meta.tagList"
            :source-label="sourceLabel"
          />

          <!-- 动作栏：仿截图一（左动作 / 中状态 / 右图标） -->
          <MetaWorkTopBar
            :disabled="!meta.work.coverUrl"
            :downloading="meta.downloading"
            :stored="meta.stored"
            :game-path="meta.gamePath"
            :game-path-exists="meta.gamePathExists"
            :launch-mode="meta.effectiveLaunchMode"
            :steam-source="meta.source === 'steam'"
            :running="meta.gameRunning"
            :launching="meta.launching"
            :loading="meta.loading"
            :source-url="meta.work.sourceUrl ?? ''"
            :source-label="sourceLabel"
            :library-updated-at="meta.libraryUpdatedAt"
            @download="handleStoreIn"
            @launch="meta.launchGame"
            @change-path="meta.changeGamePath"
            @set-launch-mode="meta.setGameLaunchMode"
            @refetch="refetch"
          />
        </div>

        <!-- exe 被删除/移走、入库/启动失败、启动成功等一次性反馈统一走 toast
             （2026-11-25 用户要求：页面内的 MetaMessage 全部改成 toast） -->

        <!-- tab 区（下划线 tab + 选中后的内容卡片）也做同一套模糊 hero 遮罩：
             环境层垫在整块 tab 区后面，tab 内的内容卡片半透明 + 背景模糊，
             模糊 hero 会从卡片底下透上来（用户 2026-11-25：「tab 显示界面也做遮罩」）。 -->
        <div class="work-tabs-wrap">
          <n-tabs v-model:value="activeTab" type="line" class="work-tabs">
          <!-- ① 作品信息：简介 / Outline / 全部元数据 -->
          <n-tab-pane name="info" tab="作品信息">
            <div class="flex flex-col gap-2 pt-1">
              <!-- 卡片堆栈：卡片之间只留一条 1px 边框（global.css 的 .card-stack） -->
              <div class="card-stack">
                <n-card v-if="meta.work.summary" size="small">
                  <MetaTextBlock
                    icon="i-lucide-align-left"
                    :title="`作品内容（summary · ${meta.work.summary.length} 字符）`"
                    :text="meta.work.summary"
                  />
                </n-card>

                <n-card v-if="meta.work.outline?.length" size="small">
                  <MetaSubsection icon="i-lucide-table">
                    <template #title>Outline（{{ meta.work.outline.length }} 行，保留原始日文标签）</template>
                    <MetaKeyValueList :rows="meta.work.outline" dense />
                  </MetaSubsection>
                </n-card>

                <n-card size="small">
                  <MetaSubsection icon="i-lucide-list">
                    <template #title>元数据</template>
                    <MetaKeyValueList :rows="summaryRows" dense />
                  </MetaSubsection>

                  <div class="mt-3 flex items-center gap-3 flex-wrap text-[11px] text-tertiary">
                    <code class="font-mono">{{ meta.work.id }}</code>
                    <span v-if="savedAtText">入库时间：{{ savedAtText }}</span>
                    <n-tag v-if="meta.cached" size="small" :bordered="true">本地缓存</n-tag>
                    <n-button
                      v-if="meta.work.sourceUrl"
                      text
                      type="primary"
                      size="tiny"
                      tag="a"
                      :href="meta.work.sourceUrl"
                      target="_blank"
                      rel="noreferrer noopener"
                      class="ml-auto"
                    >
                      <template #icon>
                        <span class="i-lucide-external-link icon-xs" aria-hidden="true" />
                      </template>
                      原始页面
                    </n-button>
                  </div>
                </n-card>
              </div>

              <MetaEmptyState
                v-if="!meta.work.summary && !meta.work.outline?.length"
                icon="i-lucide-file-question"
                title="这个作品没有简介与 Outline"
                hint="元数据仍可在上面的表格里查看；点右上角图标可以重新抓取一次。"
              />
            </div>
          </n-tab-pane>

          <!-- ② 游戏统计：元数据里的数字 + Steam 动态（评分与评价在「评价」页） -->
          <n-tab-pane name="stats" tab="游戏统计">
            <div class="flex flex-col gap-2 pt-1">
              <div class="card-stack">
                <n-card size="small">
                  <div class="flex items-start gap-10 flex-wrap">
                    <n-statistic label="发售日" :value="meta.work.releaseDate || '—'" />
                    <n-statistic label="文件容量" :value="meta.work.fileSize || '—'" />
                    <n-statistic label="入库时间" :value="savedDateText || '未入库'" />
                    <n-statistic label="抓取时间" :value="fetchedDateText || '—'" />
                  </div>
                  <p class="m-0 mt-4 text-xs text-tertiary">
                    数字来自元数据抓取（入库 / 重新抓取时更新）；本地游戏时长与次数暂未统计。
                    评分与评价数据见「评价」页。
                  </p>
                </n-card>

                <!-- Steam 源的动态：评价汇总 / 当前在线 / 最近公告（DLsite 源没有这块） -->
                <MetaSteamStats v-if="meta.source === 'steam'" />
              </div>
              <p v-if="meta.source !== 'steam'" class="m-0 text-xs text-tertiary">
                DLsite 源没有公开的在线人数与公告数据，评分与评价见「评价」页。
              </p>
            </div>
          </n-tab-pane>

          <!-- ③ 启动配置：启动方式 / 程序位置 / 文件状态 -->
          <n-tab-pane name="launch" tab="启动配置">
            <div class="flex flex-col gap-2 pt-1">
              <n-card size="small">
                <template v-if="meta.stored">
                  <MetaSubsection icon="i-lucide-play">
                    <template #title>启动</template>

                    <n-descriptions :column="1" size="small" bordered label-placement="left">
                      <n-descriptions-item label="启动方式">{{ launchModeText }}</n-descriptions-item>
                      <n-descriptions-item v-if="meta.effectiveLaunchMode === 'steam'" label="启动地址">
                        <code class="font-mono text-[11px]">steam://rungameid/{{ meta.work.id }}</code>
                      </n-descriptions-item>
                      <n-descriptions-item v-else label="程序位置">
                        <span class="break-all">{{ meta.gamePath || '还没有记录（启动时再选）' }}</span>
                      </n-descriptions-item>
                      <n-descriptions-item v-if="meta.effectiveLaunchMode !== 'steam'" label="文件状态">
                        <span :class="fileStateClass">{{ fileStateText }}</span>
                      </n-descriptions-item>
                      <n-descriptions-item label="运行状态">
                        {{ meta.gameRunning ? '运行中' : '未运行' }}
                      </n-descriptions-item>
                    </n-descriptions>

                    <div v-if="meta.source === 'steam'" class="mt-4 flex flex-col gap-2">
                      <p class="m-0 text-xs text-tertiary">切换启动方式（只改「下次怎么启动」，不会立刻启动游戏）</p>
                      <n-radio-group
                        :value="meta.effectiveLaunchMode"
                        size="small"
                        @update:value="onLaunchModeChange"
                      >
                        <n-radio-button value="steam">通过 Steam</n-radio-button>
                        <n-radio-button value="exe">本地程序</n-radio-button>
                      </n-radio-group>
                    </div>

                    <n-button
                      v-if="meta.effectiveLaunchMode !== 'steam'"
                      size="small"
                      :disabled="meta.launching"
                      class="mt-4"
                      @click="meta.changeGamePath"
                    >
                      <template #icon>
                        <span class="i-lucide-folder-search icon-xs" aria-hidden="true" />
                      </template>
                      更换游戏程序
                    </n-button>
                  </MetaSubsection>
                </template>

                <p v-else class="m-0 text-xs text-tertiary">
                  这件作品还没入库 —— 先在动作栏点「入库」，之后才能记录程序位置、切换启动方式。
                </p>
              </n-card>
            </div>
          </n-tab-pane>

          <!-- ④ 编辑：元数据表单 + 标签（都先改草稿，点各自的「保存」才写库） -->
          <n-tab-pane name="edit" tab="编辑">
            <div class="flex flex-col gap-2 pt-1">
              <div class="card-stack">
                <n-card size="small">
                  <MetaWorkEditorForm />
                </n-card>

                <n-card size="small">
                  <MetaTagEditor />
                </n-card>
              </div>

              <p class="m-0 text-xs text-tertiary">
                元数据与标签都是「改完点保存」才写库：元数据走打补丁（只提交改过的字段），
                标签整份覆盖；清空某项输入框保存后，raw_json 与库里对应的列会一起清掉。
                想重新抓取原站数据用动作栏右侧的刷新图标（抓取只读，不会覆盖这里的修改）。
              </p>
            </div>
          </n-tab-pane>

          <!-- ⑤ 评价：评分 / 评价数 / 去商店页看全部评价 -->
          <n-tab-pane name="review" tab="评价">
            <div class="flex flex-col gap-2 pt-1">
              <n-card size="small">
                <div class="flex items-start gap-10 flex-wrap">
                  <n-statistic label="评分" :value="ratingText" />
                  <n-statistic label="评价数" :value="ratingCountText" />
                  <n-statistic v-if="steamReviewText" label="Steam 评价" :value="steamReviewText" />
                </div>

                <p class="m-0 mt-4 text-xs text-tertiary">
                  这里的数字只是元数据里抓到的汇总；逐条评价仍在商店页面里，点下面的按钮打开。
                </p>

                <div class="mt-4 flex items-center gap-3 flex-wrap">
                  <n-button
                    v-if="meta.work.sourceUrl"
                    size="small"
                    tag="a"
                    :href="meta.work.sourceUrl"
                    target="_blank"
                    rel="noreferrer noopener"
                  >
                    <template #icon>
                      <span class="i-lucide-external-link icon-xs" aria-hidden="true" />
                    </template>
                    打开{{ sourceLabel }}作品页
                  </n-button>
                  <n-button
                    v-if="meta.source === 'steam'"
                    size="small"
                    tag="a"
                    :href="steamReviewUrl"
                    target="_blank"
                    rel="noreferrer noopener"
                  >
                    <template #icon>
                      <span class="i-lucide-message-square-text icon-xs" aria-hidden="true" />
                    </template>
                    Steam 社区评价
                  </n-button>
                </div>
              </n-card>
            </div>
          </n-tab-pane>
        </n-tabs>
        </div>
      </template>

      <MetaEmptyState
        v-else
        icon="i-lucide-inbox"
        title="还没有选中作品"
        hint="去「游戏库」点一张封面进来，或在那里用「添加游戏」添加新作品。"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import {
  NButton,
  NCard,
  NDescriptions,
  NDescriptionsItem,
  NRadioButton,
  NRadioGroup,
  NStatistic,
  NTabPane,
  NTabs,
  NTag,
} from 'naive-ui'
import { useMetaStore } from '../stores/metaStore'
import { useLibraryStore } from '../stores/libraryStore'
import { buildSummaryRows } from '../utils/metaFormat'
import { toastSuccess } from '../utils/toast'
import MetaEmptyState from '../components/meta/MetaEmptyState.vue'
import MetaKeyValueList from '../components/meta/MetaKeyValueList.vue'
import MetaSteamStats from '../components/meta/MetaSteamStats.vue'
import MetaSubsection from '../components/meta/MetaSubsection.vue'
import MetaTagEditor from '../components/meta/MetaTagEditor.vue'
import MetaTextBlock from '../components/meta/MetaTextBlock.vue'
import MetaWorkEditorForm from '../components/meta/MetaWorkEditorForm.vue'
import MetaWorkHero from '../components/meta/MetaWorkHero.vue'
import MetaWorkTopBar from '../components/meta/MetaWorkTopBar.vue'

// 状态与请求全在 store 里；本页面只做排布。
const meta = useMetaStore()
const library = useLibraryStore()

/**
 * 当前选中的 tab（仿截图二的下划线切换）。
 * 默认落在「作品信息」而不是「游戏统计」：从游戏库点封面进来时，第一眼想看的是简介与元数据。
 */
const activeTab = ref('info')

/** 预模糊 hero 图作为 CSS 变量传给唯一那张遮罩：不再做任何 filter: blur */
const glassStyle = computed<Record<string, string>>(() => {
  const style: Record<string, string> = {}
  const url = meta.heroBlurred
  if (url) style['--work-hero-image'] = `url("${url}")`
  return style
})

/** 来源名称：hero 的小字与「打开 xxx 作品页」按钮都用它 */
const sourceLabel = computed(() => (meta.source === 'steam' ? 'Steam' : 'DLsite'))

/**
 * hero 右侧的竖版小图：**优先用游戏库卡片那张本地封面**（用户要求），
 * 作品还没入库 / 库里没封面时才退回抓来的远程封面 URL。
 */
const coverSrc = computed(() => {
  const work = meta.work
  if (!work) return ''
  return library.coverFor({ source: meta.source, workId: work.id }) || work.coverUrl || ''
})

/** 元数据表：与调试页共用同一套字段口径（buildSummaryRows） */
const summaryRows = computed(() => (meta.work ? buildSummaryRows(meta.work) : []))

/** 评分与评价数：元数据里没有时显示占位符，而不是把空字符串塞进 n-statistic */
const ratingText = computed(() => meta.work?.rating?.trim() || '—')
const ratingCountText = computed(() => {
  const count = meta.work?.ratingCount?.trim()
  return count ? `${count} 条` : '—'
})

/** 入库时间（完整日期 + 时间）与只显示日期的版本（统计卡片里用短的） */
const savedAtText = computed(() => {
  if (!meta.libraryUpdatedAt) return ''
  const time = new Date(meta.libraryUpdatedAt)
  return Number.isNaN(time.getTime()) ? '' : time.toLocaleString('zh-CN')
})
const savedDateText = computed(() => {
  if (!meta.libraryUpdatedAt) return ''
  const time = new Date(meta.libraryUpdatedAt)
  return Number.isNaN(time.getTime()) ? '' : time.toLocaleDateString('zh-CN')
})

/** 元数据抓取时间（raw_json 里的 fetchedAt；「游戏统计」页用） */
const fetchedDateText = computed(() => {
  const fetchedAt = meta.work?.fetchedAt
  if (!fetchedAt) return ''
  const time = new Date(fetchedAt)
  return Number.isNaN(time.getTime()) ? '' : time.toLocaleDateString('zh-CN')
})

/** 启动方式的说法（Steam 源默认走 URL，其他来源只能跑本地程序） */
const launchModeText = computed(() =>
  meta.effectiveLaunchMode === 'steam' ? '通过 Steam（steam://rungameid/…）' : '本地程序',
)

/** 「启动配置」里的文件状态：没记录 / 存在 / 被删除或移动 */
const fileStateText = computed(() => {
  if (!meta.gamePath) return '未记录（启动时再选）'
  return meta.gamePathExists ? '存在' : '已被删除或移动'
})
const fileStateClass = computed(() =>
  meta.gamePath && !meta.gamePathExists ? 'text-danger' : 'text-secondary',
)

/** Steam 评价汇总（只有 Steam 源才有）：好评如潮 97%（218,393 篇） */
const steamReviewText = computed(() => {
  const reviews = meta.steamStats?.reviews
  if (!reviews || !reviews.count) return ''
  return `${reviews.label?.trim() || '评价'} ${reviews.percentPositive}%`
})

/** Steam 社区评价页（逐条评价在社区里，商店页只有汇总） */
const steamReviewUrl = computed(() =>
  meta.work ? `https://steamcommunity.com/app/${meta.work.id}/reviews/` : '',
)

/** 切换启动方式：n-radio-group 给的是 string，收窄成 store 接受的两个值 */
function onLaunchModeChange(value: string | number) {
  const mode = value === 'steam' ? 'steam' : 'exe'
  void meta.setGameLaunchMode(mode)
}

/** 重新抓取元数据：跳过本地库缓存，强制联网再解析一次 */
async function refetch() {  const work = meta.work
  if (!work || meta.loading) return
  await meta.loadById(work.id, true)
  // 「刷新完成」是用户 2026-11-25 明确保留的四类成功提示之一；失败已在 store 里弹过错误
  if (!meta.error) {
    toastSuccess('刷新完成', { description: '元数据已重新抓取，只读不覆盖编辑过的内容' })
  }
}

/**
 * 游戏运行状态轮询：进入详情先检测一次（store 的 applyWork 里已做），
 * 页面打开期间每 5 秒问一次，游戏启动/退出后按钮能在几秒内从
 * 「游戏启动」切成「运行中」再切回来；窗口重新获得焦点时立刻刷新一次。
 */
const GAME_STATUS_POLL_MS = 5000
let statusTimer: number | null = null

function pollGameStatus() {
  void meta.refreshGameStatus()
}

/**
 * 「入库」按钮 —— 这是**唯一**会把作品收进游戏库的动作：
 *   后端 `Store` 抓文本元数据写进库、把封面（优先 webp）落到 covers/ 并记账；
 *   搜索和打开详情都只读不写，所以游戏库里只会出现点过入库的作品。
 * 入库成功后重拉一次游戏库列表，卡片（含封面）立刻可见 —— 否则库里还停在"没有这件作品"。
 */
async function handleStoreIn() {
  if (meta.stored) return
  await meta.storeIn()
  // 成功/失败提示由 store 统一弹 toast（见 stores/metaStore.ts 的 done/fail）
  if (meta.stored) await library.load(true)
}

/** 页面根与滚动容器：遮罩盒子测量 / 滚动容器引用（2026-11-25 起遮罩静态，无视差） */
const pageEl = ref<HTMLElement | null>(null)
const scrollEl = ref<HTMLElement | null>(null)
let scrollFrame = 0

/**
 * 用户 2026-11-25：「遮罩随滚轮平移，滚轮向下多少、模糊图片向上多少，向下多少、向上多少（绝对位置不变）」。
 * 也就是 1:1 跟随内容：只把 transform 设成 -scrollTop，绝对位置（top/left/height 变量）完全不动，
 * 不做任何倍率视差；rAF 合并滚动事件，只写一个合成层 transform。
 */
function applyMaskScroll() {
  const mask = pageEl.value?.querySelector<HTMLElement>('.work-page-ambient')
  if (!mask) return
  const top = scrollEl.value?.scrollTop ?? 0
  mask.style.transform = `translate3d(0, ${-top}px, 0)`
}

function handleScroll() {
  if (scrollFrame !== 0) return
  scrollFrame = requestAnimationFrame(() => {
    scrollFrame = 0
    applyMaskScroll()
  })
}

/**
 * 用户 2026-11-25：「遮罩范围只在 hero 图下方」「和 hero 图大小一致」。
 * 把唯一遮罩量成 hero 的正下方、同宽同高的矩形（配合后端上下镜像 = 一张倒影）。
 * 用 ResizeObserver 跟 hero 尺寸（标题换行、标签行数、窗口缩放都会变）。
 */
let maskObserver: ResizeObserver | null = null
function measureMaskBox() {
  const root = pageEl.value
  const scroll = scrollEl.value
  const hero = root?.querySelector<HTMLElement>('.work-hero')
  if (!root || !scroll || !hero) return
  const rootRect = root.getBoundingClientRect()
  const scrollRect = scroll.getBoundingClientRect()
  const heroRect = hero.getBoundingClientRect()
  // hero 在滚动内容里的下沿（换算回未滚动坐标，遮罩不会跟着滚走）
  const heroBottomInScroll = heroRect.bottom - scrollRect.top + scroll.scrollTop
  root.style.setProperty('--mask-top', `${scrollRect.top - rootRect.top + heroBottomInScroll}px`)
  root.style.setProperty('--mask-left', `${Math.max(0, heroRect.left - rootRect.left)}px`)
  root.style.setProperty('--mask-right', `${Math.max(0, rootRect.right - heroRect.right)}px`)
  root.style.setProperty('--mask-height', `${heroRect.height}px`)
  applyMaskScroll()
}

function watchMaskBox() {
  measureMaskBox()
  const hero = pageEl.value?.querySelector<HTMLElement>('.work-hero')
  if (!maskObserver) maskObserver = new ResizeObserver(() => measureMaskBox())
  maskObserver.disconnect()
  if (hero) maskObserver.observe(hero)
  if (pageEl.value) maskObserver.observe(pageEl.value)
}

watch(
  () => meta.work?.id,
  () => void nextTick(watchMaskBox),
)

onMounted(() => {
  // 列表本身不在这页渲染；这里只是让游戏库 store 有数据（hero 的竖版封面、入库后新卡片都要它）
  void library.load()
  // 从别的页面进来时 store 里可能已经有作品（applyWork 已检测过），这里再对齐一次
  void meta.refreshGameStatus()
  statusTimer = window.setInterval(pollGameStatus, GAME_STATUS_POLL_MS)
  window.addEventListener('focus', pollGameStatus)
  window.addEventListener('resize', measureMaskBox)
  void nextTick(watchMaskBox)
})

onUnmounted(() => {
  if (statusTimer !== null) window.clearInterval(statusTimer)
  statusTimer = null
  window.removeEventListener('focus', pollGameStatus)
  window.removeEventListener('resize', measureMaskBox)
  maskObserver?.disconnect()
  maskObserver = null
  if (scrollFrame !== 0) cancelAnimationFrame(scrollFrame)
  scrollFrame = 0
})
</script>

<style scoped>
/* ===== 页面背景遮罩（2026-11-25 用户要求「背景也做遮罩」）=====
   同一张 hero 图的模糊拷贝铺满页面背景，径向 mask 只留在上方与两侧、往下淡出：
   blur 半径压到 22px —— 大半径 filter 在滚动时每帧重算，是上下滑卡顿的主因之一；
   tab 内容区保持干净，但空白处（卡片间隙、页面留白）能看到作品的氛围色。
   负 z-index + 根节点 isolation：垫在页面背景之上、所有内容之下。 */
.works-page {
  position: relative;
  isolation: isolate;
}

.work-page-ambient {
  position: absolute;
  /* 只在 hero 下方、和 hero 图同宽同高（2026-11-25 用户要求）：
     top/left/right/height 由 script 量出，配合上下镜像就是一张「倒影」。 */
  top: var(--mask-top, 0px);
  left: var(--mask-left, 0px);
  right: var(--mask-right, 0px);
  height: var(--mask-height, 340px);
  will-change: transform;
  /* 遮罩在内容下方（用户 2026-11-25：只对卡片做半透明，UI 不做半透明）。
     卡片用 75% 底让它透出来；动作栏 / tab 条 / 按钮保持不透明，不盖遮罩。 */
  z-index: -1;
  background-image: var(--work-hero-image, none);
  background-position: center 18%;
  background-size: cover;
  /* 全站唯一一张模糊遮罩：垫在内容下方，靠卡片的半透明底透出来。
     图由后端预模糊并缓存（HeroBlur），无 filter: blur；视差也只移动这一张。 */
  opacity: 0.6;
  pointer-events: none;
  -webkit-mask-image: linear-gradient(to bottom, #000 0%, #000 70%, transparent 100%);
  mask-image: linear-gradient(to bottom, #000 0%, #000 70%, transparent 100%);
}

/* ===== hero + 动作栏（2026-11-25）=====
   hero 画面保持清晰（MetaWorkHero 里不加任何模糊）；唯一那张模糊遮罩是页面级
   .work-page-ambient，动作栏只靠半透明底色让它透上来。
   hero / 动作栏 / tab 区之间用 8px 间距（用户：「紧凑排布」）。 */
.work-glass {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.work-glass :deep(.work-hero) {
  position: relative;
  z-index: 4;
}

.work-glass :deep(.work-topbar) {
  position: relative;
  z-index: 1;
}

/* ===== tab 区：紧凑排布（2026-11-25）=====
   tab 条 / 内容卡片都不再各自铺遮罩，只把底色做半透明 —— 让页面底层唯一那张
   .work-page-ambient 透上来（用户：「不使用多张遮罩」）。
   · tab 条左右各留 12px —— 第一个 tab 标签不贴边（用户：「第一个 tab 标签有左间隔」）；
   · 紧凑排布：去掉 naive pane 默认的 12px 上内边距。 */
.work-tabs-wrap {
  position: relative;
  isolation: isolate;
}

.work-tabs {
  position: relative;
  z-index: 1;
}

/* tab 条：仿截图二 —— 整条底部一条极淡的分隔线，选中项下方一条 2px 品牌色下划线。
   控件外观（字色/字重/间距/下划线色）在 styles/naiveTheme.ts 的 Tabs 段，这里只补「整条底线」
   （naive 的 line 类型默认不画整条横线）。 */
.work-tabs :deep(.n-tabs-nav-scroll-content) {
  border-bottom: 1px solid var(--color-border-light);
}

.work-tabs :deep(.n-tab-pane) {
  padding-top: 4px;
}

/* tab 选择后的内容卡片：只做半透明底，让底层唯一那张遮罩透上来
   （不用 backdrop-filter —— 滚动时逐帧重采样背景会卡）。 */
/* 内容卡片同样 75% 半透明，让下面的唯一遮罩透出来（用户 2026-11-25：
   tab 与启动所在卡片也要能看出遮罩；按钮等 UI 控件本身保持不透明）。 */
.work-tabs :deep(.n-tab-pane .n-card) {
  background: color-mix(in srgb, var(--color-bg-card) 75%, transparent);
}

/* tab 条：卡片底同样半透明（让唯一遮罩透出来），左右 12px 内边距。
   标签文字、下划线等 UI 本身不受影响；外观仍由 naiveTheme 的 Tabs 段提供。 */
.work-tabs :deep(.n-tabs-nav) {
  padding: 2px 12px 0;
  background: color-mix(in srgb, var(--color-bg-card) 75%, transparent);
}

</style>
