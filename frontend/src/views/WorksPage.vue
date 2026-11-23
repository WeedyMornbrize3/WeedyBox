<!-- frontend/src/views/WorksPage.vue
     作品库（/works）：只展示「一个作品的信息」。
     入口：游戏库（/library）里点封面（点谁就 loadById 谁，走缓存优先）；侧栏没有入口。
     搜索不在这页 —— 已经移到游戏库网格末尾的透明圆形加号按钮（居中弹窗，见 LibraryPage.vue）。

     排版（2026-11-23 用户按 Steam 商店页定稿）：
       ┌ hero 大背景图（本机 Steam librarycache 的 hero 图 / DLsite 第二张预览图）
       │   左下：来源·发售日·类型 + 作品名 + 标签；右下：竖版封面（游戏库卡片那张）
       ├ 左信息栏（228px）         │ 右主区
       │   作品信息 / 游戏 / 快速链接 │   动作条（入库·启动 + 启动方式 + 重新抓取）
       └                            │   Steam 动态 / 标签 / 简介 + Outline
     与调试页（views/test.vue → MetaApiTestCard.vue）共用 stores/metaStore.ts 与 components/meta/*。 -->
<template>
  <div class="w-full max-w-6xl mx-auto h-full min-h-0 flex flex-col">
    <!-- 自己开滚动（MainLayout 的 router-view 是 overflow-hidden） -->
    <div class="flex-1 min-h-0 overflow-y-auto scrollbar-theme pr-2 flex flex-col gap-5 pb-2 select-text">
      <template v-if="meta.work">
        <!-- 顶部大图：背景图 + 作品名 + 标签 + 竖版封面 -->
        <MetaWorkHero
          :work="meta.work"
          :background-src="meta.backgroundUrl"
          :cover-src="coverSrc"
          :tags="meta.tagList"
          :source-label="sourceLabel"
        />

        <div class="flex items-start gap-6">
          <!-- 左栏：只读信息披露（操作区在右栏顶部） -->
          <MetaWorkSidebar
            class="w-[228px] shrink-0"
            :work="meta.work"
            :stored="meta.stored"
            :game-path="meta.gamePath"
            :game-path-exists="meta.gamePathExists"
            :launch-mode="meta.effectiveLaunchMode"
            :source-label="sourceLabel"
            :library-updated-at="meta.libraryUpdatedAt"
          />

          <!-- 右栏主区 -->
          <main class="flex-1 min-w-0 flex flex-col gap-5">
            <!-- 动作条：入库 / 游戏启动 + 启动方式 + 缓存标记 + 重新抓取 -->
            <section class="card-flat px-5 py-4 flex items-start gap-4 flex-wrap">
              <MetaWorkActions
                :disabled="!meta.work.coverUrl"
                :downloading="meta.downloading"
                :stored="meta.stored"
                :game-path="meta.gamePath"
                :game-path-exists="meta.gamePathExists"
                :launch-mode="meta.effectiveLaunchMode"
                :steam-source="meta.source === 'steam'"
                :running="meta.gameRunning"
                :launching="meta.launching"
                @download="handleStoreIn"
                @launch="meta.launchGame"
                @change-path="meta.changeGamePath"
                @set-launch-mode="meta.setGameLaunchMode"
              />

              <div class="ml-auto flex items-center gap-3 shrink-0 pt-1">
                <span
                  v-if="meta.cached"
                  class="meta-cache-badge"
                  :title="meta.libraryUpdatedAt ? `入库时间：${new Date(meta.libraryUpdatedAt).toLocaleString()}` : '来自本地游戏库'"
                >
                  缓存
                </span>
                <code class="font-mono text-[11px] text-tertiary">{{ meta.work.id }}</code>
                <button
                  class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none disabled:opacity-50"
                  :disabled="meta.loading"
                  @click="meta.loadById(meta.work.id, true)"
                >
                  <span class="i-lucide-refresh-cw icon-xs" aria-hidden="true" />
                  重新抓取
                </button>
              </div>
            </section>

            <!-- exe 被删除/移走：提示用户用「更换位置」重新选程序（启动按钮同时禁用） -->
            <MetaMessage v-if="meta.gameWarning" tone="error" :text="meta.gameWarning" />
            <!-- 入库失败 / 启动失败要看得见：这页没有别的提示条，静默失败会让人以为点了没反应 -->
            <MetaMessage v-if="meta.error" tone="error" :text="meta.error" />
            <!-- 启动成功 / 换了游戏位置的回执 -->
            <MetaMessage v-if="meta.status" :text="meta.status" />

            <!-- Steam 源的动态：评价汇总 / 当前在线 / 最近公告（DLsite 源没有这块） -->
            <MetaSteamStats v-if="meta.source === 'steam'" />

            <!-- 标签：抓取来的 tags 与用户自己加的合并成一份，改完点「保存」才写库 -->
            <section class="card-flat px-5 py-4">
              <MetaTagEditor />
            </section>

            <!-- 简介 + Outline -->
            <section class="card-flat px-5 py-4">
              <MetaWorkDetails :work="meta.work" />
            </section>
          </main>
        </div>
      </template>

      <MetaEmptyState
        v-else
        icon="i-lucide-inbox"
        title="还没有选中作品"
        hint="去「游戏库」点一张封面进来，或在那里用加号按钮添加新作品。"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useMetaStore } from '../stores/metaStore'
import { useLibraryStore } from '../stores/libraryStore'
import MetaEmptyState from '../components/meta/MetaEmptyState.vue'
import MetaMessage from '../components/meta/MetaMessage.vue'
import MetaSteamStats from '../components/meta/MetaSteamStats.vue'
import MetaTagEditor from '../components/meta/MetaTagEditor.vue'
import MetaWorkActions from '../components/meta/MetaWorkActions.vue'
import MetaWorkDetails from '../components/meta/MetaWorkDetails.vue'
import MetaWorkHero from '../components/meta/MetaWorkHero.vue'
import MetaWorkSidebar from '../components/meta/MetaWorkSidebar.vue'

// 状态与请求全在 store 里；本页面只做排布。
const meta = useMetaStore()
const library = useLibraryStore()

/** 来源名称：hero 的小字与左栏「原始页面」链接都用它 */
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
  if (meta.stored && meta.work) await library.load(true)
}

onMounted(() => {
  // 列表本身不在这页渲染；这里只是让游戏库 store 有数据（hero 的竖版封面、入库后新卡片都要它）
  void library.load()
  // 从别的页面进来时 store 里可能已经有作品（applyWork 已检测过），这里再对齐一次
  void meta.refreshGameStatus()
  statusTimer = window.setInterval(pollGameStatus, GAME_STATUS_POLL_MS)
  window.addEventListener('focus', pollGameStatus)
})

onUnmounted(() => {
  if (statusTimer !== null) window.clearInterval(statusTimer)
  statusTimer = null
  window.removeEventListener('focus', pollGameStatus)
})
</script>

<style scoped>
/* 「缓存」标记：与原来详情卡标题旁的小徽标同一长相（半透明底 + 细描边） */
.meta-cache-badge {
  padding: 1px 6px;
  border: 1px solid var(--color-border-light);
  border-radius: 4px;
  font-size: 11px;
  line-height: 1.5;
  color: var(--color-text-tertiary);
  background: var(--color-bg-secondary);
  white-space: nowrap;
}
</style>
