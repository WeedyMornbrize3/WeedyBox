<!-- frontend/src/components/meta/MetaWorkSidebar.vue
     作品详情页左侧信息栏（/works 新排版，用户 2026-11-23 按 Steam 商店页布局定稿）：
     从上到下依次是「作品信息」「游戏」「快速链接」三组，都是只读展示 ——
     操作（入库 / 启动 / 启动方式）在右侧主区顶部，标签编辑也在主区。
     布局参考 Steam 商店页左栏的 Stats / Quick Links：窄列、小字、键值对。 -->
<template>
  <aside class="flex flex-col gap-6">
    <!-- 作品信息：社团 / 系列 / 形式 / 年龄 / 格式 / 容量 / 发售日 / 价格 / 评分 -->
    <section class="flex flex-col gap-2">
      <h2 class="work-side-title">作品信息</h2>
      <MetaKeyValueList :rows="rows" dense />
    </section>

    <!-- 游戏：入库状态 + 生效的启动方式 + 本机程序位置（位置只对"本地程序启动"有意义） -->
    <section class="flex flex-col gap-2">
      <h2 class="work-side-title">游戏</h2>
      <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 m-0 text-xs">
        <dt class="text-tertiary whitespace-nowrap">入库状态</dt>
        <dd class="m-0 text-secondary">{{ stored ? '已入库' : '未入库' }}</dd>

        <template v-if="stored">
          <dt class="text-tertiary whitespace-nowrap">启动方式</dt>
          <dd class="m-0 text-secondary">{{ launchMode === 'steam' ? '通过 Steam' : '本地程序' }}</dd>

          <template v-if="launchMode !== 'steam'">
            <dt class="text-tertiary whitespace-nowrap">程序位置</dt>
            <dd class="m-0 text-secondary break-all">{{ gamePath || '还没有记录（启动时再选）' }}</dd>
            <dt class="text-tertiary whitespace-nowrap">文件状态</dt>
            <dd class="m-0" :class="fileStateClass">{{ fileStateText }}</dd>
          </template>
        </template>
      </dl>
    </section>

    <!-- 快速链接：原始作品页（DLsite / Steam 商店页由后端给的 sourceUrl 决定） -->
    <section class="flex flex-col gap-2">
      <h2 class="work-side-title">快速链接</h2>
      <a
        v-if="work.sourceUrl"
        :href="work.sourceUrl"
        target="_blank"
        rel="noreferrer noopener"
        class="text-xs text-brand hover:underline flex items-center gap-1.5"
      >
        <span class="i-lucide-external-link icon-xs" aria-hidden="true" />
        {{ sourceLabel ? `${sourceLabel} 原始页面` : '原始页面' }}
      </a>
      <span v-else class="text-xs text-tertiary">没有可用的链接</span>

      <p v-if="libraryUpdatedAt" class="m-0 text-[11px] text-tertiary leading-relaxed">
        入库时间：{{ savedAtText }}
      </p>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WorkMeta } from '../../services/metaService'
import { buildSummaryRows } from '../../utils/metaFormat'
import MetaKeyValueList from './MetaKeyValueList.vue'

const props = withDefaults(
  defineProps<{
    work: WorkMeta
    /** 是否已入库（决定「游戏」一组显示多少行） */
    stored?: boolean
    /** 库里记录的游戏程序路径（空 = 还没记录） */
    gamePath?: string
    /** 记录的 exe 是否还存在 */
    gamePathExists?: boolean
    /** 生效的启动方式（steam = 不需要本地 exe） */
    launchMode?: 'steam' | 'exe'
    /** 来源名称（DLsite / Steam），用在「快速链接」的文字上 */
    sourceLabel?: string
    /** 入库时间（ISO 字符串） */
    libraryUpdatedAt?: string
  }>(),
  {
    stored: false,
    gamePath: '',
    gamePathExists: true,
    launchMode: 'exe',
    sourceLabel: '',
    libraryUpdatedAt: '',
  },
)

/** 左侧「作品信息」用与详情卡同一套字段口径（buildSummaryRows） */
const rows = computed(() => buildSummaryRows(props.work))

/** 本地 exe 的状态：没记录 / 存在 / 被删除或移动（后者用危险色，和启动按钮的禁用态呼应） */
const fileStateText = computed(() => {
  if (!props.gamePath) return '未记录'
  return props.gamePathExists ? '存在' : '已被删除或移动'
})
const fileStateClass = computed(() =>
  props.gamePath && !props.gamePathExists ? 'text-danger' : 'text-secondary',
)

const savedAtText = computed(() => {
  const time = new Date(props.libraryUpdatedAt)
  return Number.isNaN(time.getTime()) ? props.libraryUpdatedAt : time.toLocaleString('zh-CN')
})
</script>

<style scoped>
/* 左栏分组标题：比正文略重、带一条很轻的下边线，模仿 Steam 页左栏的 Stats / Quick Links */
.work-side-title {
  margin: 0;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--color-border-light);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.4px;
  color: var(--color-text-primary);
}
</style>
