<!-- frontend/src/views/StatsPage.vue
     数据分析页（/stats，2026-11-28 用户要求「数据分析需求 … 新建 router 界面」）。

     页面结构（全部来自本地 SQLite，不联网）：
       ① 顶部工具条  —— 区间选择 + 来源过滤 + 刷新（与其它 router 页同一套标题卡片）
       ② 总览数字    —— 总时长 / 游玩次数 / 活跃天数 / 玩过的作品 / 最长一次
       ③ 单日运行时长 —— 柱状图（后端已按区间补成连续日历）
       ④ 最长游戏时长排行 —— 上方环状饼图（各作品占比）+ 下方名次列表
       ⑤ 时间运行分析 —— 0-23 点的时段分布面积图

     布局（2026-11-28 用户第二轮要求：「参考work的split 样式 用来排布不同卡片之间的大小
     参考naive ui 示例 且添加滚条 方便上下滚轮移动」）：
       · 整页是一个滚动容器（自己开滚 + scroll-fade 边缘渐隐），滚轮上下移动所有卡片；
       · 下半区用**纵向 n-split** 切成「排行卡片 / 时段卡片」两块，中间一条 1px 可拖的分割条
         （实现照搬 /works 的 WorksSplitBar 手感：静息透明、hover 亮品牌色、
          上下各 4px 透明热区）；拖出来的高度存进 app store（localStorage，跨会话保留）；
       · 上半区的工具条与单日卡片不参与 split —— 它们是"看数据的固定前提"，
         跟着滚轮走即可，让用户把可调的注意力留给下面两块内容区。

     UI 口径（按 2026-11-28 用户的「与其他 router ui 一起排布 ui css 样式统一」）：
       · 根容器 = w-full max-w-* mx-auto h-full min-h-0 flex flex-col（内容区自己开滚动）；
       · 卡片外壳一律 MetaSection（header 图标 + 标题 + 右侧动作）；
       · 加载态 n-skeleton、空态在加载完成后才出现、出错显示本块文案；
       · 不手写图表 SVG —— 图表统一走 components/stats/StatsChart（封装 vue-echarts）。

     数据口径要在界面上说清楚（不然用户会觉得数字对不上）：
       · 时长来自日桶（含 2026-11-28 之前的老数据）；
       · 次数 / 时段分布来自 2026-11-28 起才记录的游戏会话（启动到退出为一段）；
       · 所以老日期"有时长没次数"是正常的 —— 工具条下方有一行小字说明断层从哪天开始。
-->
<template>
  <div class="w-full max-w-6xl mx-auto h-full min-h-0 flex flex-col select-text">
    <!-- 滚动容器：整页上下滚，用 naive 内置 n-scrollbar（用户 2026-11-28「检查滚动条
         和界面布局 内置滚动条」）—— 滚动条观感与 naive 控件统一，不手写 webkit 滚动条。 -->
    <n-scrollbar class="stats-scroll flex-1 min-h-0">
      <!-- min-h-full：内容不足一屏时也撑满，split 区的 flex-1 才能吃到剩余高度 -->
      <div class="stats-scroll-body flex flex-col gap-4 pr-2 pb-2 min-h-full">
        <!-- ① 工具条：标题 + 区间快捷 + 区间选择 + 来源过滤 + 刷新 -->
        <MetaSection icon="i-lucide-chart-column" title="游玩统计">
        <template #actions>
          <!-- 快捷区间：近 7 天 / 30 天 / 90 天 / 全部。选中态是品牌色文字。
               ⚠️ 这里**不显示区间文案**：右边的 n-date-picker 本身就显示着
               「2026-10-30 → 2026-11-28」，再写一遍是重复信息（截图核对时发现的）。 -->
          <n-button
            v-for="preset in quickRanges"
            :key="preset.days"
            text
            size="tiny"
            :type="isQuickActive(preset.days) ? 'primary' : 'default'"
            :disabled="stats.loading"
            @click="stats.useRecentDays(preset.days)"
          >
            {{ preset.label }}
          </n-button>

          <!-- 自定义区间：n-date-picker 的 range 模式（与 /works 编辑页同一套日历控件） -->
          <n-date-picker
            v-model:formatted-value="rangeValue"
            type="daterange"
            value-format="yyyy-MM-dd"
            size="small"
            clearable
            class="w-[220px]"
            :disabled="stats.loading"
            @update:formatted-value="handleRangeChange"
          />

          <!-- 来源过滤：全部 / DLsite / Steam -->
          <n-select
            :value="stats.source"
            :options="sourceOptions"
            size="small"
            class="w-[104px]"
            :disabled="stats.loading"
            @update:value="handleSourceChange"
          />

          <n-button text size="tiny" :disabled="stats.loading" @click="stats.load()">
            <template #icon>
              <span class="i-lucide-refresh-cw icon-xs" aria-hidden="true" />
            </template>
            {{ stats.loading ? '统计中…' : '刷新' }}
          </n-button>
        </template>

        <!-- 总览数字：五格，窄窗口自动折行（n-grid 的 responsive）
             数字用 naive-ui NNumberAnimation 做动画（用户 2026-11-28「游玩统计数字使用naive ui的数值动画」）：
             次数/天数/作品数 = 纯整数动画 + 后缀；总时长/最长一次 = 秒数动画 + formatDuration 实时格式化。 -->
        <n-grid :cols="'1 480:2 900:5'" :x-gap="12" :y-gap="12">
          <n-gi v-for="stat in overviewCards" :key="stat.key">
            <div class="stat-cell">
              <div class="flex items-center gap-1.5">
                <span :class="stat.icon" class="icon-xs text-tertiary" aria-hidden="true" />
                <span class="text-[11px] text-tertiary truncate">{{ stat.label }}</span>
              </div>
              <!-- 加载中：数字位铺骨架（与真实数字同高，避免加载完跳一下） -->
              <n-skeleton v-if="stats.loading && !stats.loaded" text style="width: 70%; height: 26px" />
              <!-- 无数据占位：次数/最长一次为 0 时显示「—」不动画 -->
              <span v-else-if="stat.isDash" class="stat-value text-primary">—</span>
              <!-- 数字动画：NNumberAnimation 从 0 跳到目标值，通过默认插槽自定义格式化输出 -->
              <n-number-animation
                v-else
                class="stat-value text-primary"
                :from="0"
                :to="stat.animationValue"
                :duration="800"
                :active="stats.loaded && !stats.loading"
              >
                <template #default="{ value }">
                  {{ stat.format(value) }}
                </template>
              </n-number-animation>
              <span v-if="stat.hint" class="text-[11px] text-tertiary">{{ stat.hint }}</span>
            </div>
          </n-gi>
        </n-grid>

        <!-- 口径说明：只有在"存在只有日桶的老数据"时才出现，不打扰新用户 -->
        <p v-if="stats.legacyDayCount > 0" class="m-0 mt-3 text-[11px] text-tertiary leading-relaxed">
          <span class="i-lucide-info icon-xs text-tertiary align-middle" aria-hidden="true" />
          有 {{ stats.legacyDayCount }} 天的时长是早期记录（只有总时长、没有次数与时段）。
          {{ sessionsSinceText }}
        </p>
      </MetaSection>

      <!-- ② 单日运行时长：柱状图 -->
      <MetaSection icon="i-lucide-calendar-days" title="单日运行时长">
        <template #actions>
          <span v-if="daysSummary" class="text-xs text-tertiary whitespace-nowrap">{{ daysSummary }}</span>
          <!-- 柱状图 / 热力图切换（用户 2026-11-28「通过selector(在单日运行时长界面卡片内)
               可切换柱状图还是热力图 Heatmap naive ui」） -->
          <n-select
            :value="dayView"
            :options="dayViewOptions"
            size="small"
            class="w-[104px]"
            @update:value="setDayView"
          />
        </template>
        <!-- 出错：显示这一块的错误文案（不是骨架屏——骨架屏表示"还在加载"，会让人一直等） -->
        <p v-if="stats.daysError" class="m-0 py-8 text-center text-xs text-tertiary">{{ stats.daysError }}</p>
        <!-- 空态：加载完成后才出现；此时不画空图表（会显得"图挂了"） -->
        <p v-else-if="stats.loaded && !stats.days.length" class="m-0 py-8 text-center text-xs text-tertiary">
          这段时间还没有游玩记录。
        </p>
        <DayDurationChart v-else-if="dayView === 'bar'" :days="stats.days" :loading="stats.loading && !stats.loaded" />
        <DayHeatmapChart v-else :days="stats.days" />
      </MetaSection>

      <!-- ③ 下半区（用户 2026-11-28 最终定稿：「左侧运行分析 右侧上方饼状图
           右侧下方最长游戏时间排名」）：
            横向 split 切「左栏（运行分析）| 右栏」；右栏内部纵向 split 切
            「饼图（上）| 最长游戏时长排名（下）」。
            于是下半区 = 左整列时段图 + 右上饼图 + 右下排名，两个方向都能拖。

            容器高度写死（用户 2026-11-28「外层卡片 容器大小写死」「split在容器里面」）：
            不再用 flex-1 吃剩余高度——那样首帧高度=0、量到后才弹开，
            n-split 的 pane 会闪一下。写死 480px 后 split 一进来就有确定尺寸，
            min/max 也确定，不会拉伸异常。 -->
      <div class="stats-split-host">
        <n-split
          direction="horizontal"
          class="stats-split"
          :size="chartsSplitSize"
          :min="chartsSplitMinStr"
          :max="chartsSplitMaxStr"
          :resize-trigger-size="1"
          @update:size="onChartsSplitSize"
        >
        <template #1>
          <div class="stats-pane flex flex-col min-h-0">
            <MetaSection icon="i-lucide-clock" title="时间运行分析" class="stats-fill-card" body-class="stats-body-fill">
              <template #actions>
                <span class="text-xs text-tertiary whitespace-nowrap">按小时 · 本地时间</span>
              </template>
              <p v-if="stats.hoursError" class="m-0 text-xs text-tertiary">{{ stats.hoursError }}</p>
              <!-- 时段分布只能来自会话表：一条会话都没有时，24 个 0 的图没有意义 -->
              <div v-else-if="!stats.hasSessionData" class="flex flex-col items-center justify-center gap-2 py-8">
                <span class="i-lucide-clock icon-lg text-tertiary" aria-hidden="true" />
                <p class="m-0 text-xs text-tertiary text-center leading-relaxed">
                  还没有时段数据。<br />
                  从这一版开始会记录每次启动的时间区间，玩过一次之后这里就有分布了。
                </p>
              </div>
              <HourDistributionChart v-else :hours="stats.hours" :loading="stats.loading && !stats.loaded" />
            </MetaSection>
          </div>
        </template>

        <template #resize-trigger>
          <StatsSplitBar direction="vertical" />
        </template>

        <template #2>
          <!-- 右栏：纵向 split 切「饼图（上）| 最长游戏时长排名（下）」。
               ⚠️ 这里用 h-full 而不是 flex-1：父容器是外层 split 的 pane2
               （overflow:hidden 普通块，不是 flex 容器），flex-1 会失效、高度塌成 0。 -->
          <n-split
            direction="vertical"
            class="stats-split-inner h-full min-h-0"
            :size="splitTopSize"
            :min="splitTopMinStr"
            :max="splitTopMaxStr"
            :resize-trigger-size="1"
            @update:size="onSplitSize"
          >
            <template #1>
              <div class="stats-pane flex flex-col min-h-0">
                <MetaSection icon="i-lucide-pie-chart" title="时长占比" class="stats-fill-card" body-class="stats-body-fill">
                  <template #actions>
                    <span class="text-xs text-tertiary whitespace-nowrap">前 {{ RANK_LIMIT }} 名</span>
                  </template>
                  <p v-if="stats.ranksError" class="m-0 text-xs text-tertiary">{{ stats.ranksError }}</p>
                  <!-- 饼图：各作品时长占比。没有数据时不画（空环图只会让人以为加载失败） -->
                  <WorkSharePieChart
                    v-else-if="stats.ranks.some((r) => r.seconds > 0)"
                    :ranks="stats.ranks"
                    :loading="stats.loading && !stats.loaded"
                  />
                  <p v-else class="m-0 py-8 text-center text-xs text-tertiary">这段时间还没有游玩记录。</p>
                </MetaSection>
              </div>
            </template>

            <template #resize-trigger>
              <StatsSplitBar />
            </template>

            <template #2>
              <!-- 名次列表：完整的按时长排序（饼图只截前 5 + 其他，这里仍是完整前 10） -->
              <div class="stats-pane flex flex-col min-h-0">
                <MetaSection icon="i-lucide-trophy" title="最长游戏时长排名" class="stats-fill-card" body-class="stats-body-fill">
                  <template #actions>
                    <span class="text-xs text-tertiary whitespace-nowrap">按时长 · 前 {{ RANK_LIMIT }} 名</span>
                  </template>
                  <p v-if="stats.ranksError" class="m-0 text-xs text-tertiary">{{ stats.ranksError }}</p>
                  <n-scrollbar v-else class="stats-ranks-scroll flex-1 min-h-0">
                    <WorkRankList :ranks="stats.ranks" :loading="stats.loading && !stats.loaded" />
                  </n-scrollbar>
                </MetaSection>
              </div>
            </template>
          </n-split>
        </template>
      </n-split>
      </div>
      </div>
    </n-scrollbar>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onMounted, ref, watch } from 'vue'
import { NButton, NDatePicker, NGi, NGrid, NNumberAnimation, NScrollbar, NSelect, NSkeleton, NSplit } from 'naive-ui'
import MetaSection from '../components/meta/MetaSection.vue'
import DayDurationChart from '../components/stats/DayDurationChart.vue'
import DayHeatmapChart from '../components/stats/DayHeatmapChart.vue'
import HourDistributionChart from '../components/stats/HourDistributionChart.vue'
import StatsSplitBar from '../components/stats/StatsSplitBar.vue'
import WorkRankList from '../components/stats/WorkRankList.vue'
import WorkSharePieChart from '../components/stats/WorkSharePieChart.vue'
import { RANK_LIMIT, useStatsStore } from '../stores/statsStore'
import type { StatsSource } from '../services/statsService'
import { formatDuration } from '../utils/statsChart'

const stats = useStatsStore()

// ===== 单日运行时长：柱状图 / 热力图切换 =====

type DayView = 'bar' | 'heatmap'
const dayView = ref<DayView>('bar')
const dayViewOptions = [
  { label: '柱状图', value: 'bar' },
  { label: '热力图', value: 'heatmap' },
]

function setDayView(value: string | number) {
  dayView.value = value === 'heatmap' ? 'heatmap' : 'bar'
}

// ===== 区间选择 =====

/** n-date-picker 的 daterange 值：[from, to]；null/[] 表示清空 */
const rangeValue = ref<[string, string] | null>(
  stats.from && stats.to ? [stats.from, stats.to] : null
)

/** store 里的区间被外部改动（快捷按钮、重置）时，把选择器同步过来 */
watch(
  () => [stats.from, stats.to] as const,
  ([from, to]) => {
    const next = from && to ? ([from, to] as [string, string]) : null
    const current = rangeValue.value
    // 值相同就不要写回去：n-date-picker 会因此重渲染并闪一下
    if (next && current && next[0] === current[0] && next[1] === current[1]) return
    if (!next && !current) return
    rangeValue.value = next
  }
)

/** 选完区间就重算；清空 = 不限时间（看全部） */
function handleRangeChange(value: [string, string] | null) {
  if (!value || value.length !== 2) {
    void stats.setRange('', '')
    return
  }
  void stats.setRange(value[0], value[1])
}

/**
 * 快捷区间：全部 = 两个空串（后端按"不限"处理）。
 * ⚠️ 不用「近 365 天」代替"全部"：那样更早的记录会被悄悄截掉，
 *    而用户点"全部"时想看的就是**所有**数据。
 */
const quickRanges = [
  { days: 7, label: '近 7 天' },
  { days: 30, label: '近 30 天' },
  { days: 90, label: '近 90 天' },
  { days: 0, label: '全部' },
] as const

/** 快捷按钮的选中态：日期区间正好等于「近 N 天」才算选中 */
function isQuickActive(days: number): boolean {
  if (days === 0) return !stats.from && !stats.to
  const today = new Date()
  const to = toLocalDay(today)
  const start = new Date(today)
  start.setDate(start.getDate() - (days - 1))
  return stats.from === toLocalDay(start) && stats.to === to
}

/** Date → YYYY-MM-DD（本地时区；toISOString 是 UTC，会差一天，绝对不能用） */
function toLocalDay(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// ===== 来源过滤 =====

const sourceOptions = [
  { label: '全部来源', value: '' },
  { label: 'DLsite', value: 'dlsite' },
  { label: 'Steam', value: 'steam' },
]

function handleSourceChange(value: string) {
  void stats.setSource(value as StatsSource)
}

// ===== 总览卡片 =====
//
// 五格数字统一在这里算好（模板里只 v-for），这样加一格只需要改这个数组。
// 数字用 NNumberAnimation 做动画（用户 2026-11-28「游玩统计数字使用naive ui的数值动画」）：
//   · animationValue = 动画终值（纯数字）
//   · format(value)  = 把动画过程中的中间值格式化成显示文本
//   · isDash         = 数据为 0 / 无会话时显示「—」不动画
//
// 时长单位（用户 2026-11-28「游戏统计时长显示单位」）：
// 总时长 / 最长一次 = 以小时为单位的数字动画 + 固定后缀「小时」，
// 仿 Steam「12.3 小时」——不用 formatDuration（它会 0分→不到1分→X分→X小时Y分 跳变，
// 动画过程中单位不断切换看起来很乱）。不足 1 小时的显示分钟 + 后缀「分钟」。

/** 秒 → 小时（保留一位小数）；<1 小时返回分钟数 */
function formatPlayHours(seconds: number): { value: number; unit: string } {
  const total = Math.max(0, Math.round(seconds || 0))
  if (total < 3600) {
    const minutes = Math.max(1, Math.round(total / 60))
    return { value: minutes, unit: '分钟' }
  }
  const hours = total / 3600
  return { value: hours >= 10 ? Math.round(hours) : Math.round(hours * 10) / 10, unit: '小时' }
}

const overviewCards = computed(() => {
  const o = stats.overview
  const totalH = formatPlayHours(o.totalSeconds)
  const longestH = formatPlayHours(o.longestSessionSeconds)
  return [
    {
      key: 'total',
      icon: 'i-lucide-hourglass',
      label: '总时长',
      animationValue: totalH.value,
      isDash: o.totalSeconds <= 0,
      format: (v: number) => `${Math.round(v * 10) / 10} ${totalH.unit}`,
      hint: o.activeDays > 0 ? `${o.activeDays} 天有记录` : '',
    },
    {
      key: 'sessions',
      icon: 'i-lucide-repeat-2',
      label: '游玩次数',
      animationValue: o.sessionCount,
      isDash: o.sessionCount <= 0,
      format: (v: number) => `${Math.round(v)} 次`,
      hint: o.sessionCount > 0 ? `平均 ${avgSessionText.value}` : '仅早期记录时无此项',
    },
    {
      key: 'activeDays',
      icon: 'i-lucide-calendar-days',
      label: '活跃天数',
      animationValue: o.activeDays,
      isDash: o.activeDays <= 0,
      format: (v: number) => `${Math.round(v)} 天`,
      hint: o.activeDays > 0 ? `平均每天 ${formatDuration(dailyAvg.value)}` : '',
    },
    {
      key: 'works',
      icon: 'i-lucide-gamepad-2',
      label: '玩过的作品',
      animationValue: o.workCount,
      isDash: o.workCount <= 0,
      format: (v: number) => `${Math.round(v)} 款`,
      hint: o.workCount > 0 ? '按区间内有时长的作品算' : '',
    },
    {
      key: 'longest',
      icon: 'i-lucide-timer',
      label: '最长一次',
      animationValue: longestH.value,
      isDash: o.longestSessionSeconds <= 0,
      format: (v: number) => `${Math.round(v * 10) / 10} ${longestH.unit}`,
      hint: o.longestSessionSeconds > 0 ? '' : '仅早期记录时无此项',
    },
  ]
})

/** 单次平均时长：总时长 ÷ 次数（只统计有会话的那些段） */
const avgSessionText = computed(() => {
  const { totalSeconds, sessionCount } = stats.overview
  if (sessionCount <= 0 || totalSeconds <= 0) return '—'
  return formatDuration(totalSeconds / sessionCount)
})

/** 活跃日平均：总时长 ÷ 活跃天数 */
const dailyAvg = computed(() => {
  const { totalSeconds, activeDays } = stats.overview
  if (activeDays <= 0) return 0
  return totalSeconds / activeDays
})

/** 单日卡片右上角：这段区间一共多久 / 覆盖多少天 */
const daysSummary = computed(() => {
  if (!stats.days.length) return ''
  const total = stats.days.reduce((sum, item) => sum + item.seconds, 0)
  return `共 ${formatDuration(total)} · ${stats.days.length} 天`
})

/**
 * 会话记录的起点说明：告诉用户"哪天之前的次数/时段是缺的"。
 * sessionsSince 是会话表里最早的一条启动时间（RFC3339）。
 */
const sessionsSinceText = computed(() => {
  const raw = stats.overview.sessionsSince
  if (!raw) return '从这一版开始记录后，这里会有次数与时段分布。'
  const since = new Date(raw)
  if (Number.isNaN(since.getTime())) return ''
  const day = toLocalDay(since)
  return `${day} 起才有"次数"与"时段分布"。`
})

// ===== 下半区的 split（用户 2026-11-28：「可以通过split调整排布
//          每一张卡片的间隔都可以用split调整布局」）=====
//
// 结构（模板里是嵌套两层 n-split，最终定稿「左侧运行分析 右侧上方饼状图
// 右侧下方最长游戏时间排名」）：
//   外层横向 split：左栏「时间运行分析」 / 右栏「饼图+排名」（拖中间竖线改左右宽度）；
//   内层纵向 split（右栏内部）：上「饼图」 / 下「最长游戏时长排名」（拖横线改上下高度）。
//
// 与 /works 那套同源：px 而不是百分比（拖出来的尺寸应该是固定的那个量，
// 窗口变大变小不留白自己长胖），分割条 1px + hover 才亮。
// 这里**不往 localStorage 存**：分析页每次进来都该是完整可见的默认布局，
// 用户临时调一下版式不必跨会话保留。

/** 上块高度（内层纵向 split 的 pane1 = 饼图）；0 = 还没拖过，用默认 */
const splitTopPx = ref(0)
/** 左栏宽度（外层横向 split 的 pane1 = 运行分析）；0 = 还没拖过，用默认 */
const chartsSplitPx = ref(0)

/** 饼图默认高度：够放下饼图（约 200 环 + 卡片头） */
const SPLIT_TOP_DEFAULT = 300
/** 饼图下限：再矮环就只剩一条线 */
const SPLIT_TOP_MIN = 180
/** 饼图上限：再高就挤掉排名列表（写死，不随宿主高度漂移） */
const SPLIT_TOP_MAX = 560
/** 运行分析左栏默认宽度（约占总宽的一半） */
const CHARTS_SPLIT_DEFAULT = 480
/** 运行分析左栏最小宽度 */
const CHARTS_SPLIT_MIN = 280
/** 运行分析左栏最大宽度（写死，不随宿主宽度漂移） */
const CHARTS_SPLIT_MAX = 820

/**
 * min / max 写死固定 px（用户 2026-11-28「下边界确定 写死 防止split拉伸异常」）：
 * 之前 max 是 computed（依赖 ResizeObserver 量到的宿主尺寸），首帧宿主高度=0
 * → max 塌成与 min 相同 → 默认尺寸被钳到下限 → 量到宿主后跳一下。
 * 写死后 n-split 的拖拽边界不再随窗口/侧栏变化而漂移，也不会首帧闪烁。
 */
const splitTopMinStr = `${SPLIT_TOP_MIN}px`
const splitTopMaxStr = `${SPLIT_TOP_MAX}px`
const chartsSplitMinStr = `${CHARTS_SPLIT_MIN}px`
const chartsSplitMaxStr = `${CHARTS_SPLIT_MAX}px`

const splitMeasured = ref(false)

const splitTopSize = computed(() => {
  if (!splitMeasured.value) return '100%'
  return `${clampedTopPx.value}px`
})

const clampedTopPx = computed(() => {
  const target = splitTopPx.value > 0 ? splitTopPx.value : SPLIT_TOP_DEFAULT
  return Math.max(SPLIT_TOP_MIN, Math.min(target, SPLIT_TOP_MAX))
})

/** 拖动纵向 split：上块高度 */
function onSplitSize(size: string | number) {
  const px = typeof size === 'number' ? size : Number.parseFloat(size)
  if (!Number.isFinite(px)) return
  splitTopPx.value = Math.round(px)
}

/**
 * 传给横向 n-split 的 size（时段图宽度）。
 * 首帧没量到宽度时用 '100%' 兜底（与纵向同一原因）。
 */
const chartsSplitSize = computed(() => {
  if (!splitMeasured.value) return '100%'
  return `${clampedChartsPx.value}px`
})

const clampedChartsPx = computed(() => {
  const target = chartsSplitPx.value > 0 ? chartsSplitPx.value : CHARTS_SPLIT_DEFAULT
  return Math.max(CHARTS_SPLIT_MIN, Math.min(target, CHARTS_SPLIT_MAX))
})

/** 拖动横向 split：时段图宽度 */
function onChartsSplitSize(size: string | number) {
  const px = typeof size === 'number' ? size : Number.parseFloat(size)
  if (!Number.isFinite(px)) return
  chartsSplitPx.value = Math.round(px)
}

// 每次进入页面都自动刷新（用户 2026-11-28「每次进入stats界面自动刷新」）：
// onMounted 覆盖首次进入，onActivated 覆盖 keep-alive 缓存页面被重新激活。
// 不判断 loaded —— 用户可能离开后回来玩了游戏，数据已变旧，每次都该拉新的。
onMounted(() => {
  void stats.load()
  void nextTick().then(() => { splitMeasured.value = true })
})

onActivated(() => {
  void stats.load()
})
</script>

<style scoped>
/* 总览小格：不加边框/底色（它就在卡片 body 里），靠留白与标签层级区分 */
.stat-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

/* 数字：大一号、等宽数字（tabular-nums 让"1 小时 23 分"与"45 分"纵向对齐） */
.stat-value {
  font-size: 17px;
  font-weight: 600;
  line-height: 1.25;
  font-variant-numeric: tabular-nums;
}

/* 滚动容器：高度 = 可用区（router-view 是 overflow-hidden，滚动权在这一层）。
   ⚠️ 不能只写 flex-1：n-split 的 pane 是 overflow:hidden 的普通块，
   里面再要滚动就必须有明确高度（与 /works 的 .works-split-scroll 同一教训）。 */
.stats-scroll {
  height: 100%;
}

/* n-split 默认 display:flex + height:100%，补 min-height:0 让它能在 flex 列里收缩 */
.stats-split {
  min-height: 0;
}

/* split 宿主容器：高度写死（用户 2026-11-28「外层卡片 容器大小写死」）——
   不再 flex-1 吃剩余高度，给 split 一个固定大小的盒子，
   min/max 也确定，不会首帧闪烁或拉伸异常。 */
.stats-split-host {
  height: 480px;
  flex-shrink: 0;
}

/* 内层纵向 split：填满外层 split pane2 的高度（pane2 是普通块，必须显式 100% 高度，
   否则高度塌成 0 —— 与 /works 的 .works-split-scroll 同一教训） */
.stats-split-inner {
  height: 100%;
  min-height: 0;
  min-width: 0;
}

/* pane 内部变成 flex 列：卡片才能用 flex-1 吃掉 pane 的高度并自己开滚 */
.stats-pane {
  height: 100%;
  min-height: 0;
}

/* 占满整块 pane 的卡片：header + body 都撑开，图表自适应剩余高度 */
.stats-fill-card {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.stats-fill-card :deep(.n-card__content) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

/* split 卡片的 body 也要是 flex 列：MetaSection 的 body 是普通块，
   不撑开的话 n-scrollbar / 图表的 flex-1 拿不到高度（会被 overflow:hidden 的 content 裁掉）。 */
.stats-body-fill {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

/* 名次列表滚动区：被 split 拖小之后列表自己滚（naive n-scrollbar，与整页同一口径）。 */
.stats-ranks-scroll {
  min-height: 0;
}
</style>
