<!-- frontend/src/components/stats/WorkRankList.vue
     「名次统计」：按时长降序的作品榜（2026-11-28 用户要求「名次统计」）。

     为什么不用 n-data-table：
       · 这里只有 4 列、10 行，且要做"名次徽章 + 时长占比条"这种定制单元格；
         n-data-table 的列渲染函数反而更长，还得处理它自己的加载态与空态。
       · 全站骨架/空态统一走 n-skeleton / n-empty，自己排一遍更容易与其它卡片一致。

     名次前三名给金/银/铜的强调色，其余用普通序号 —— 只用文字颜色区分，
     不加背景块（用户 2026-11-28 的视觉口径：破坏性之外不铺色块，靠文字与线区分层级）。
-->
<template>
  <div class="flex flex-col">
    <!-- 加载中：铺行骨架（与真实行同高，避免加载完页面跳一下） -->
    <template v-if="loading && !ranks.length">
      <div v-for="i in skeletonRows" :key="`rank-skeleton-${i}`" class="rank-row">
        <n-skeleton text style="width: 20px" />
        <n-skeleton text style="width: 40%;" />
        <n-skeleton text style="width: 60px; margin-left: auto" />
      </div>
    </template>

    <template v-else-if="ranks.length">
      <div v-for="(item, index) in ranks" :key="`${item.source}:${item.workId}`" class="rank-row">
        <!-- 名次徽章：1-3 名给金/银/铜，其余是灰色序号 -->
        <span class="rank-badge" :class="rankClass(index)" aria-hidden="true">{{ index + 1 }}</span>

        <div class="flex flex-col min-w-0 flex-1">
          <!-- 作品名：太长的用 ellipsis（全站统一截断方式），完整名字进 title -->
          <span class="rank-name text-primary" :title="item.name">{{ item.name }}</span>
          <!-- 副信息：来源 + 次数 + 最后一次。次数为 0 说明这段只有日桶老数据，不显示 -->
          <span class="rank-meta text-tertiary">
            {{ sourceLabel(item.source) }}
            <template v-if="item.sessions > 0"> · 启动 {{ item.sessions }} 次</template>
            <template v-if="item.lastPlayedAt"> · 最后 {{ lastPlayedText(item.lastPlayedAt) }}</template>
          </span>
        </div>

        <!-- 右侧：时长 + 占比条。占比条用数据表格的 5 色（与上方饼图同源），
             长度 = 相对第一名的比例；颜色按名次取同一套 chartSeriesColors。 -->
        <div class="flex flex-col items-end gap-1 shrink-0 ml-3">
          <span class="rank-value text-primary tabular-nums">{{ formatDuration(item.seconds) }}</span>
          <span
            class="rank-bar"
            :style="{ width: `${barWidth(item.seconds)}%`, backgroundColor: barColor(index) }"
          />
        </div>
      </div>
    </template>

    <!-- 空态：只在**加载完成后**才出现（用户口径） -->
    <p v-else class="m-0 py-6 text-center text-xs text-tertiary">这段时间还没有游玩记录。</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NSkeleton } from 'naive-ui'
import { formatDuration } from '../../utils/statsChart'
import { chartSeriesColors } from '../../utils/chartColors'
import { useThemeStore } from '../../stores/Theme'
import type { PlayWorkRank } from '../../services/statsService'

const props = withDefaults(
  defineProps<{
    ranks: PlayWorkRank[]
    loading?: boolean
  }>(),
  { loading: false }
)

const theme = useThemeStore()

/** 骨架行数 = 名次上限（10），与真实列表高度一致 */
const skeletonRows = 10

/** 第一名 = 100%，其余按比例；最小 2% 让"玩了一点点"也能看见一条短线 */
const maxSeconds = computed(() => Math.max(1, ...props.ranks.map((item) => item.seconds)))

function barWidth(seconds: number): number {
  if (seconds <= 0) return 0
  return Math.max(2, Math.round((seconds / maxSeconds.value) * 100))
}

/** 占比条颜色：数据表格的 5 色，按名次取（与上方饼图同源）；超过 5 名循环取色 */
function barColor(index: number): string {
  const colors = chartSeriesColors(theme.isDark)
  return colors[index % colors.length]
}

/** 前三名：金 / 银 / 铜（用主题的功能色，不硬编码） */
function rankClass(index: number): string {
  if (index === 0) return 'rank-badge--gold'
  if (index === 1) return 'rank-badge--silver'
  if (index === 2) return 'rank-badge--bronze'
  return ''
}

function sourceLabel(source: string): string {
  if (source === 'steam') return 'Steam'
  if (source === 'dlsite') return 'DLsite'
  return source || '未知来源'
}

/**
 * 「最后 11-27」：只显示月-日。
 * 完整时间没必要 —— 名次榜关心的是"最近还在玩"，精确到分钟反而占地方。
 */
function lastPlayedText(raw: string): string {
  const time = new Date(raw)
  if (Number.isNaN(time.getTime())) return ''
  return `${String(time.getMonth() + 1).padStart(2, '0')}-${String(time.getDate()).padStart(2, '0')}`
}
</script>

<style scoped>
/* 一行名次：与 TODO 行同一套紧凑口径（py-2 + 行间 1px 分隔线） */
.rank-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 0;
  min-height: 40px;
}

/* 行之间只用 1px 分隔线，最后一行不画（与全站卡片堆栈口径一致） */
.rank-row + .rank-row {
  border-top: 1px solid var(--color-border-light);
}

/* 名次徽章：定宽 + tabular-nums，序号位数不同也不会让名字起点跳动 */
.rank-badge {
  flex-shrink: 0;
  width: 20px;
  text-align: center;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-tertiary);
}

/* 前三名：只换文字颜色 + 加粗，不铺背景块 */
.rank-badge--gold {
  color: var(--color-warning);
  font-weight: 600;
}

.rank-badge--silver {
  color: var(--color-text-secondary);
  font-weight: 600;
}

.rank-badge--bronze {
  color: var(--color-text-secondary);
  font-weight: 600;
  opacity: 0.85;
}

.rank-name {
  font-size: 13px;
  line-height: 1.35;
  /* 单行截断：全站统一用这个组合，别用 overflow-hidden + text-ellipsis 分开写 */
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rank-meta {
  font-size: 11px;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rank-value {
  font-size: 12px;
}

/* 占比条：颜色由内联 backgroundColor 决定（数据表格 5 色），这里只留形状 */
.rank-bar {
  display: block;
  height: 2px;
  max-width: 100%;
  transition: width 240ms cubic-bezier(0.4, 0, 0.2, 1);
}
</style>
