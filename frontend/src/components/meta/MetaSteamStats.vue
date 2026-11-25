<!-- frontend/src/components/meta/MetaSteamStats.vue
     /works 的「Steam 动态」：评价汇总 + 当前在线人数 + 最近公告。
     数据来自 MetaApiService.FetchSteamStats（GetItems 的 reviews + 在线人数 + ISteamNews），
     由 stores/metaStore.ts 的 loadSteamStats() 读取；只有 Steam 源的作品才由父组件渲染。
     三项各自独立：某项拿不到就少显示一块，不整块报错。 -->
<template>
  <MetaSection icon="i-lucide-activity" title="Steam 动态">
    <template #actions>
      <span v-if="stats?.fetchedAt" class="text-[11px] text-tertiary whitespace-nowrap">{{ updatedText }}</span>
      <n-button text size="tiny" :loading="meta.steamStatsLoading" :disabled="meta.steamStatsLoading" @click="meta.loadSteamStats()">
        <template #icon>
          <span class="i-lucide-refresh-cw icon-xs" aria-hidden="true" />
        </template>
        刷新
      </n-button>
    </template>

    <p v-if="meta.steamStatsLoading" class="m-0 text-xs text-tertiary">正在获取 Steam 动态…</p>
    <template v-else-if="stats">
      <div class="flex items-start gap-8 flex-wrap">
        <n-statistic label="评价" :value="reviewText" />
        <n-statistic label="当前在线" :value="playersText" />
      </div>

      <div v-if="news.length" class="flex flex-col gap-1.5 mt-4">
        <p class="m-0 text-xs text-tertiary">最近公告</p>
        <ul class="list-none m-0 p-0 flex flex-col gap-1.5">
          <li v-for="item in news" :key="item.gid" class="flex items-baseline gap-2 min-w-0">
            <a
              :href="item.url"
              target="_blank"
              rel="noreferrer noopener"
              class="text-sm text-brand hover:underline truncate"
              :title="item.title"
            >
              {{ item.title }}
            </a>
            <span class="text-[11px] text-tertiary whitespace-nowrap shrink-0">{{ dateText(item.date) }}</span>
          </li>
        </ul>
      </div>
      <p v-else class="m-0 text-xs text-tertiary mt-4">这个游戏还没有公开公告。</p>
    </template>
    <p v-else class="m-0 text-xs text-tertiary">暂时拿不到 Steam 动态（可能网络不通，或该条目没有公开数据）。</p>
  </MetaSection>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NStatistic } from 'naive-ui'
import { useMetaStore } from '../../stores/metaStore'
import MetaSection from './MetaSection.vue'

const meta = useMetaStore()

const stats = computed(() => meta.steamStats)

/** 评价一行：好评如潮 97%（218,393 篇） */
const reviewText = computed(() => {
  const reviews = stats.value?.reviews
  if (!reviews || !reviews.count) return '暂无评价'
  const label = reviews.label?.trim() || '评价'
  const count = reviews.count.toLocaleString('zh-CN')
  return `${label} ${reviews.percentPositive}%（${count} 篇）`
})

const playersText = computed(() => {
  const players = stats.value?.players
  return typeof players === 'number' ? players.toLocaleString('zh-CN') : '未知'
})

const news = computed(() => stats.value?.news ?? [])

const updatedText = computed(() => {
  const fetchedAt = stats.value?.fetchedAt
  if (!fetchedAt) return ''
  const time = new Date(fetchedAt)
  return Number.isNaN(time.getTime()) ? '' : `更新于 ${time.toLocaleTimeString('zh-CN')}`
})

/** 公告时间是 Unix 秒，只显示日期 */
function dateText(unix: number): string {
  if (!unix) return ''
  const date = new Date(unix * 1000)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleDateString('zh-CN')
}
</script>
