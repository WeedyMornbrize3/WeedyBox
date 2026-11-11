<!-- frontend/src/components/test/MetaApiTestCard.vue
     接口测试卡片 · DLsite 元数据抓取（MetaApiService）
     由 views/test.vue 挂载（视图层只留一个薄壳，与 TodoPage → TodoList 同一结构）。

     定位：这是"调试用"的单列排布 —— 原始 JSON 只在这里出现，方便对着 DLsite 页面核对解析结果。
     作品库（/works）的路由已摘掉、页面文件保留（views/WorksPage.vue 与 components/meta/* 后续复用）；
     现阶段抓取入口只剩本页。两者共用：
       stores/metaStore.ts   状态 + 请求
       components/meta/*     卡片外壳、搜索行、封面、键值表等积木
     所以本文件现在只描述"摆什么"，不再自己写请求逻辑。

     滚动：MainLayout 的 router-view 是 overflow-hidden，所以本组件必须自己开滚动容器；
     而又因为 global.css 用 !important 隐藏了全局滚动条，必须配 scrollbar-theme 恢复。 -->
<template>
  <div class="w-full max-w-4xl mx-auto h-full min-h-0 overflow-y-auto scrollbar-theme pr-2 flex flex-col gap-4 select-text">
    <!-- ── 卡片 1：请求 ───────────────────────────────────────────── -->
    <MetaSection icon="i-lucide-plug" title="接口测试 · DLsite 元数据" body-class="p-5 flex flex-col gap-4">
      <template #actions>
        <span class="text-xs text-tertiary font-mono">MetaApiService</span>
      </template>

      <MetaSearchBar v-model="meta.keyword" :loading="meta.loading" @submit="meta.runSearch(undefined, true)" />
      <MetaSampleChips :samples="META_SAMPLES" @pick="pickSample" />
      <MetaPreImgPath :path="meta.preImgPath" @refresh="meta.refreshPreImgDir" />
      <MetaMessage v-if="meta.error" tone="error" :text="meta.error" />
      <MetaMessage v-if="meta.status" :text="meta.status" />
    </MetaSection>

    <!-- ── 卡片 2：精确命中（按编号） ─────────────────────────────── -->
    <MetaSection v-if="meta.work" icon="i-lucide-file-json" body-class="p-5 flex flex-col gap-5">
      <template #title>
        <code class="font-mono">{{ meta.work.id }}</code>
        <span class="text-xs text-tertiary font-normal ml-2">按编号精确抓取</span>
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
        :stored="meta.stored"
        @download="meta.storeIn"
      />
      <MetaWorkDetails :work="meta.work" />
    </MetaSection>

    <!-- ── 卡片 3：模糊搜索候选 ───────────────────────────────────── -->
    <MetaSection
      v-if="!meta.work && meta.candidates.length"
      icon="i-lucide-list"
      :title="`搜索候选（${meta.candidates.length}）`"
      body-class="p-0"
    >
      <template #actions>
        <span class="text-xs text-tertiary">点「详情」按编号精确抓取</span>
      </template>
      <MetaCandidateList
        :candidates="meta.candidates"
        :covers="meta.candidateCovers"
        @select="loadCandidate"
      />
    </MetaSection>

    <!-- ── 卡片 4：原始数据 ───────────────────────────────────────── -->
    <MetaRawJson v-if="meta.raw !== null" :value="meta.raw" />
  </div>
</template>

<script setup lang="ts">
// 调试页：数据不做二次加工，直接展示后端返回的原始 JSON。
import { onMounted } from 'vue'
import { META_SAMPLES, useMetaStore } from '../../stores/metaStore'
import MetaCandidateList from '../meta/MetaCandidateList.vue'
import MetaMessage from '../meta/MetaMessage.vue'
import MetaPreImgPath from '../meta/MetaPreImgPath.vue'
import MetaRawJson from '../meta/MetaRawJson.vue'
import MetaSampleChips from '../meta/MetaSampleChips.vue'
import MetaSearchBar from '../meta/MetaSearchBar.vue'
import MetaSection from '../meta/MetaSection.vue'
import MetaWorkDetails from '../meta/MetaWorkDetails.vue'
import MetaWorkOverview from '../meta/MetaWorkOverview.vue'

const meta = useMetaStore()

// 调试页刻意**跳过缓存**（refresh=true）：这一页的存在意义就是「对着 DLsite 页面核对
// 后端解析得对不对」，读本地缓存会掩盖解析改动；抓到的作品仍会正常入库（游戏库 /library 可见）。
function pickSample(sample: string) {
  meta.searchBySample(sample, true)
}

function loadCandidate(id: string) {
  void meta.loadById(id, true)
}

onMounted(() => meta.ensurePreImgDir())
</script>
