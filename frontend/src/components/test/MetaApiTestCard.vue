<!-- frontend/src/components/test/MetaApiTestCard.vue
     接口测试卡片 · DLsite 元数据抓取（MetaApiService）
     由 views/test.vue 挂载（视图层只留一个薄壳，与 TodoPage → TodoList 同一结构）。

     定位：这是"调试用"的页面 —— 原始 JSON 只在这里出现，方便对着 DLsite 页面核对解析结果。
     2026-11-25 布局改为 **naive 的 n-split 左右分栏**（用户要求「用 naive ui split 分割卡片」）：
       左栏：请求（搜索/示例/封面目录）→ 精确命中 → 模糊搜索候选
       右栏：原始 JSON（未加工），两侧各自滚动、可拖动中间的分隔条调宽度
     卡片改为方形（全局 Card.borderRadius=0），左栏用 `.card-stack` 让相邻卡片只共享一条 1px 边框。

     滚动：MainLayout 的 router-view 是 overflow-hidden；n-split 的左右面板各自 overflow:hidden，
     所以两栏内部各开一个滚动容器（scrollbar-theme 恢复全局被隐藏的滚动条）。
     注意：数据不做二次加工，右栏直接展示后端返回的原始 JSON。 -->
<template>
  <div class="w-full max-w-6xl mx-auto h-full min-h-0 select-text">
    <n-split
      class="h-full"
      direction="horizontal"
      :default-size="0.62"
      :min="0.4"
      :max="0.78"
      :resize-trigger-size="4"
    >
      <!-- 左栏：请求 + 结果 -->
      <template #1>
        <div v-scroll-fade class="scroll-fade h-full min-h-0 overflow-y-auto scrollbar-theme pr-2 card-stack">
          <!-- 卡片 1：请求 -->
          <MetaSection icon="i-lucide-plug" title="接口测试 · DLsite 元数据" body-class="flex flex-col gap-4">
            <template #actions>
              <span class="text-xs text-tertiary font-mono">MetaApiService</span>
            </template>

            <MetaSearchBar v-model="meta.keyword" :loading="meta.loading" @submit="meta.runSearch(undefined, true)" />
            <MetaSampleChips :samples="META_SAMPLES" @pick="pickSample" />
            <MetaCoversPath :path="meta.coversPath" @refresh="meta.refreshCoversDir" />
          </MetaSection>

          <!-- 卡片 2：精确命中（按编号） -->
          <MetaSection v-if="meta.work" icon="i-lucide-file-json" body-class="flex flex-col gap-5">
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
              :game-path="meta.gamePath"
              :game-path-exists="meta.gamePathExists"
              :launch-mode="meta.effectiveLaunchMode"
              :steam-source="meta.source === 'steam'"
              :running="meta.gameRunning"
              :launching="meta.launching"
              @download="meta.storeIn"
              @launch="meta.launchGame"
              @change-path="meta.changeGamePath"
              @set-launch-mode="meta.setGameLaunchMode"
            />
            <MetaWorkDetails :work="meta.work" />
          </MetaSection>

          <!-- 卡片 3：模糊搜索候选 -->
          <MetaSection
            v-if="!meta.work && meta.candidates.length"
            icon="i-lucide-list"
            :title="`搜索候选（${meta.candidates.length}）`"
            body-class="-mx-4 -my-4"
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
        </div>
      </template>

      <!-- 右栏：原始返回数据（与左栏并排，方便对照解析结果） -->
      <template #2>
        <div v-scroll-fade class="scroll-fade h-full min-h-0 overflow-y-auto scrollbar-theme pl-2">
          <MetaRawJson v-if="meta.raw !== null" :value="meta.raw" />
          <MetaSection v-else icon="i-lucide-braces" title="原始 JSON">
            <p class="m-0 text-xs text-tertiary">
              搜索或抓取后，这里显示后端返回的完整 JSON，方便对着 DLsite 页面核对解析结果。
            </p>
          </MetaSection>
        </div>
      </template>
    </n-split>
  </div>
</template>

<script setup lang="ts">
// 调试页：数据不做二次加工，直接展示后端返回的原始 JSON。
import { onMounted } from 'vue'
import { NSplit } from 'naive-ui'
import { META_SAMPLES, useMetaStore } from '../../stores/metaStore'
import MetaCandidateList from '../meta/MetaCandidateList.vue'
import MetaCoversPath from '../meta/MetaCoversPath.vue'
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
  // 调试页只核对 DLsite 解析：显式指定来源，避免刚从 Steam 作品切过来时用错 source
  void meta.loadById(id, true, 'dlsite')
}

onMounted(() => meta.ensureCoversDir())
</script>
