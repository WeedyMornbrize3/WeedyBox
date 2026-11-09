<!-- frontend/src/views/LibraryPage.vue
     游戏库（/library）：把"抓过的作品"当成一个可浏览的收藏夹。
     与作品库（/works）的分工：
       /works    —— 搜索、抓取、看详情（一次一部，带候选栏）
       /library  —— 只读列表：库里现在有什么，封面在哪，删掉哪条
     数据全来自本地库（stores/libraryStore.ts → libraryApi.list），不发任何网络请求，
     所以断网也能看；封面是后端读文件转成的 base64 data URL（WebView 里一定显示得出来）。 -->
<template>
  <div class="w-full max-w-5xl mx-auto h-full min-h-0 flex flex-col gap-4 select-text">
    <!-- ── 顶部：标题 + 计数 + 刷新 ─────────────────────────────── -->
    <MetaSection icon="i-lucide-gamepad-2" title="游戏库" body-class="px-5 py-4 flex flex-col gap-3">
      <template #actions>
        <button
          class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none disabled:opacity-50"
          :disabled="library.loading"
          @click="library.load(true)"
        >
          <span class="i-lucide-refresh-cw icon-xs" aria-hidden="true" />
          {{ library.loading ? '刷新中…' : '刷新' }}
        </button>
      </template>

      <p class="m-0 text-sm text-secondary">
        已入库 <span class="text-primary font-medium">{{ library.count }}</span> 部作品
        <span v-if="library.count" class="text-tertiary">
          （{{ library.withCoverCount }} 部已下载封面）
        </span>
      </p>
      <p class="m-0 text-xs text-tertiary">
        抓取成功的作品会自动入库；这里只改数据库记录，删除时不会动磁盘
        <code class="font-mono">preImg/</code> 里的图片。
      </p>
      <MetaMessage v-if="library.error" tone="error" :text="library.error" />
    </MetaSection>

    <!-- ── 列表：自己开滚动（MainLayout 的 router-view 是 overflow-hidden） ── -->
    <div class="flex-1 min-h-0 overflow-y-auto scrollbar-theme pr-2 flex flex-col gap-3">
      <template v-if="library.works.length">
        <article
          v-for="item in library.works"
          :key="item.source + item.workId"
          class="card-flat p-4 flex gap-4 items-start"
        >
          <!-- 封面：没有本地封面就退成图标占位，不留破图 -->
          <div class="w-[84px] h-[63px] shrink-0 rounded-md overflow-hidden bg-secondary flex-center">
            <img
              v-if="library.covers[item.workId]"
              :src="library.covers[item.workId]"
              :alt="item.name"
              class="w-full h-full object-cover"
            />
            <span v-else class="i-lucide-image-off icon-md text-muted" aria-hidden="true" />
          </div>

          <!-- 主体 -->
          <div class="min-w-0 flex-1 flex flex-col gap-1.5">
            <div class="flex items-center gap-2 flex-wrap">
              <code class="text-xs font-mono text-tertiary">{{ item.workId }}</code>
              <span v-if="item.maker" class="text-[11px] px-2 py-0.5 rounded bg-secondary text-secondary">
                {{ item.maker }}
              </span>
              <span v-if="item.workType" class="text-[11px] px-2 py-0.5 rounded bg-secondary text-tertiary">
                {{ item.workType }}
              </span>
            </div>
            <h3 class="m-0 text-[15px] font-semibold text-primary leading-snug">
              {{ item.name }}
            </h3>
            <p class="m-0 text-xs text-tertiary">
              <span v-if="item.series">系列 {{ item.series }}</span>
              <span v-if="item.series && item.releaseDate"> · </span>
              <span v-if="item.releaseDate">发售 {{ item.releaseDate }}</span>
              <span v-if="item.createdAt"> · 入库 {{ formatTime(item.createdAt) }}</span>
            </p>
          </div>

          <!-- 操作 -->
          <div class="flex flex-col items-end gap-2 shrink-0">
            <button class="btn-ghost text-xs" @click="openDetail(item.workId)">查看详情</button>
            <button
              class="px-3 py-1.5 text-xs rounded-lg cursor-pointer bg-transparent border-none transition-colors duration-200"
              :class="
                armedId === item.workId
                  ? 'text-danger bg-hover'
                  : 'text-tertiary hover:text-danger hover:bg-hover'
              "
              :disabled="library.removingId === item.workId"
              @click="handleRemove(item.workId)"
            >
              {{ removeLabel(item.workId) }}
            </button>
          </div>
        </article>
      </template>

      <!-- 空态 / 首次加载 -->
      <MetaEmptyState
        v-else-if="!library.loading"
        icon="i-lucide-gamepad-2"
        title="游戏库还是空的"
        hint="去「作品库」搜一个 RJ 编号，抓取成功后会自动入库。"
      />
      <p v-else class="m-0 text-sm text-tertiary py-6 text-center">正在读取游戏库…</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useLibraryStore } from '../stores/libraryStore'
import { useMetaStore } from '../stores/metaStore'
import MetaEmptyState from '../components/meta/MetaEmptyState.vue'
import MetaMessage from '../components/meta/MetaMessage.vue'
import MetaSection from '../components/meta/MetaSection.vue'

const library = useLibraryStore()
const meta = useMetaStore()
const router = useRouter()

/** 删除是破坏性操作，第一次点只"上膛"，再点一次才真删（与 TODO 批量删除同一套路） */
const armedId = ref('')

function removeLabel(workId: string): string {
  if (library.removingId === workId) return '删除中…'
  return armedId.value === workId ? '确认删除' : '删除'
}

async function handleRemove(workId: string) {
  if (armedId.value !== workId) {
    armedId.value = workId
    return
  }
  armedId.value = ''
  await library.remove(workId)
}

/** 查看详情：复用作品库的详情加载（缓存优先，库里已有就不会联网），然后跳过去 */
async function openDetail(workId: string) {
  await router.push('/works')
  await meta.loadById(workId)
}

function formatTime(value: string): string {
  if (!value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

onMounted(() => {
  // 每次进页面都重新读一次：别的页面刚抓完的作品要能立刻看到
  void library.load(true)
})
</script>
