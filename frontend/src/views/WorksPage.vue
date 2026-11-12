<!-- frontend/src/views/WorksPage.vue
     作品库（/works）：只展示「一个作品的信息」。
     入口：游戏库（/library）里点封面（点谁就 loadById 谁，走缓存优先）；侧栏没有入口。
     搜索不在这页 —— 已经移到游戏库网格末尾的透明圆形加号按钮（居中弹窗，见 LibraryPage.vue）。
     所以这里不再有搜索卡、候选栏、封面保存位置卡、游戏库卡，只留作品信息本身。
     与调试页（views/test.vue → MetaApiTestCard.vue）共用 stores/metaStore.ts 与 components/meta/*。 -->
<template>
  <div class="w-full max-w-4xl mx-auto h-full min-h-0 flex flex-col gap-4 select-text">
    <!-- 自己开滚动（MainLayout 的 router-view 是 overflow-hidden） -->
    <div class="flex-1 min-h-0 overflow-y-auto scrollbar-theme pr-2 flex flex-col gap-4">
      <MetaSection v-if="meta.work" icon="i-lucide-file-json" body-class="p-5 flex flex-col gap-5">
        <template #title>
          <code class="font-mono">{{ meta.work.id }}</code>
          <span class="text-xs text-tertiary font-normal ml-2">作品详情</span>
          <!-- 缓存标记：库里已有就直接读，不联网 -->
          <span
            v-if="meta.cached"
            class="ml-2 text-[11px] px-1.5 py-0.5 rounded bg-secondary text-tertiary border-theme-light"
            :title="meta.libraryUpdatedAt ? `入库时间：${new Date(meta.libraryUpdatedAt).toLocaleString()}` : '来自本地游戏库'"
          >
            缓存
          </span>
        </template>
        <template #actions>
          <button
            class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none"
            :disabled="meta.loading"
            @click="meta.loadById(meta.work.id, true)"
          >
            <span class="i-lucide-refresh-cw icon-xs" aria-hidden="true" />
            重新抓取
          </button>
          <a
            v-if="meta.work.sourceUrl"
            :href="meta.work.sourceUrl"
            target="_blank"
            rel="noreferrer noopener"
            class="text-xs text-brand hover:underline flex items-center gap-1"
          >
            <span class="i-lucide-external-link icon-xs" aria-hidden="true" />
            原始页面
          </a>
        </template>

        <MetaWorkOverview
          :work="meta.work"
          :downloading="meta.downloading"
          :saved="meta.saved"
          :stored="meta.stored"
          @download="handleStoreIn"
        />
        <!-- 我的标签：用户自己加，存后端 work_tags 表（与 DLsite 抓来的 tags 分开） -->
        <MetaUserTags />
        <!-- 入库失败要看得见：这页没有别的提示条，静默失败会让人以为点了没反应 -->
        <MetaMessage v-if="meta.error" tone="error" :text="meta.error" />
        <MetaWorkDetails :work="meta.work" />
      </MetaSection>

      <MetaEmptyState
        v-else
        icon="i-lucide-inbox"
        title="还没有选中作品"
        hint="去「游戏库」点一张封面进来，或在那里用顶部的「搜索」添加新作品。"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useMetaStore } from '../stores/metaStore'
import { useLibraryStore } from '../stores/libraryStore'
import MetaEmptyState from '../components/meta/MetaEmptyState.vue'
import MetaMessage from '../components/meta/MetaMessage.vue'
import MetaSection from '../components/meta/MetaSection.vue'
import MetaUserTags from '../components/meta/MetaUserTags.vue'
import MetaWorkDetails from '../components/meta/MetaWorkDetails.vue'
import MetaWorkOverview from '../components/meta/MetaWorkOverview.vue'

// 状态与请求全在 store 里；本页面只做排布。
const meta = useMetaStore()
const library = useLibraryStore()

/**
 * 「入库」按钮 —— 这是**唯一**会把作品收进游戏库的动作：
 *   后端 `Store` 抓文本元数据写进库、把封面原样落到 preImg/ 并记账；
 *   搜索和打开详情都只读不写，所以游戏库里只会出现点过入库的作品。
 * 入库成功后重拉一次游戏库列表，卡片（含封面）立刻可见 —— 否则库里还停在"没有这件作品"。
 */
async function handleStoreIn() {
  if (meta.stored) return
  await meta.storeIn()
  if (meta.stored && meta.work) await library.load(true)
}

onMounted(() => {
  // 列表本身不在这页渲染；这里只是让游戏库 store 有数据（入库成功后要立刻能看到新卡片）
  void library.load()
})
</script>
