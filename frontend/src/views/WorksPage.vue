<!-- frontend/src/views/WorksPage.vue
     作品库（/works）：正式浏览 DLsite 作品元数据的页面。
     与调试页（views/test.vue → components/test/MetaApiTestCard.vue）刻意不同构：
       调试页 = 单列、保留原始 JSON，用来核对解析结果；
       作品库 = 「左详情 / 右候选栏」，用来真正看作品。
     两者共用 stores/metaStore.ts（状态与请求）和 components/meta/*（积木），
     所以换排布只需改这里和那里的模板，不动逻辑。 -->
<template>
  <div class="w-full max-w-5xl mx-auto h-full min-h-0 flex flex-col gap-4 select-text">
    <!-- ── 搜索区：常驻，不随结果滚动 ─────────────────────────────── -->
    <MetaSection icon="i-lucide-library-big" title="作品库 · DLsite 元数据" body-class="p-5 flex flex-col gap-4">
      <MetaSearchBar v-model="meta.keyword" :loading="meta.loading" @submit="meta.runSearch()" />
      <MetaMessage v-if="meta.error" tone="error" :text="meta.error" />
      <MetaMessage v-if="meta.status" :text="meta.status" />
    </MetaSection>

    <!-- ── 结果区：自己开滚动（MainLayout 的 router-view 是 overflow-hidden） ── -->
    <div class="flex-1 min-h-0 overflow-y-auto scrollbar-theme pr-2 flex flex-col gap-4">
      <div v-if="meta.hasResult" class="grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_280px] gap-4 items-start">
        <!-- 左：有作品放详情，只有候选（还没精确抓取）就把候选铺在左边 -->
        <div class="flex flex-col gap-4 min-w-0">
          <MetaSection v-if="meta.work" icon="i-lucide-file-json" body-class="p-5 flex flex-col gap-5">
            <template #title>
              <code class="font-mono">{{ meta.work.id }}</code>
              <span class="text-xs text-tertiary font-normal ml-2">作品详情</span>
            </template>
            <template #actions>
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
              @download="meta.downloadCover"
            />
            <MetaWorkDetails :work="meta.work" />
          </MetaSection>

          <MetaSection
            v-else-if="meta.candidates.length"
            icon="i-lucide-list"
            :title="`搜索候选（${meta.candidates.length}）`"
            body-class="p-0"
          >
            <template #actions>
              <span class="text-xs text-tertiary">点「详情」按编号精确抓取</span>
            </template>
            <MetaCandidateList :candidates="meta.candidates" @select="meta.loadById" />
          </MetaSection>
        </div>

        <!-- 右：候补栏 + 封面落地位置 -->
        <div class="flex flex-col gap-4 min-w-0">
          <MetaSection
            v-if="meta.work && meta.candidates.length"
            icon="i-lucide-list"
            :title="`搜索候选（${meta.candidates.length}）`"
            body-class="p-0"
          >
            <MetaCandidateList :candidates="meta.candidates" @select="meta.loadById" />
          </MetaSection>

          <MetaSection icon="i-lucide-folder-open" title="封面保存位置" body-class="p-5">
            <MetaPreImgPath :path="meta.preImgPath" @refresh="meta.refreshPreImgDir" />
          </MetaSection>
        </div>
      </div>

      <MetaEmptyState
        v-else
        icon="i-lucide-inbox"
        title="还没有搜索结果"
        hint="输入 RJ 编号精确抓取，或输入作品名做模糊搜索。"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useMetaStore } from '../stores/metaStore'
import MetaCandidateList from '../components/meta/MetaCandidateList.vue'
import MetaEmptyState from '../components/meta/MetaEmptyState.vue'
import MetaMessage from '../components/meta/MetaMessage.vue'
import MetaPreImgPath from '../components/meta/MetaPreImgPath.vue'
import MetaSearchBar from '../components/meta/MetaSearchBar.vue'
import MetaSection from '../components/meta/MetaSection.vue'
import MetaWorkDetails from '../components/meta/MetaWorkDetails.vue'
import MetaWorkOverview from '../components/meta/MetaWorkOverview.vue'

// 状态与请求全在 store 里；本页面只做排布。
const meta = useMetaStore()

onMounted(() => meta.ensurePreImgDir())
</script>
