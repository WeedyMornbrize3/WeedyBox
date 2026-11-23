<!-- frontend/src/views/LibraryPage.vue
     游戏库（/library）：把"抓过的作品"当成一个可浏览的收藏夹。
     交互（用户明确要求）：
       - 封面横向排列成网格；
       - 名称平时不占地方，鼠标悬停时用半透明面板从下往上淡入；
       - 点封面进入「作品库」详情页（/works，复用原页面，点谁就加载谁）；
       - **卡片上没有任何操作按钮**（用户 2026-11-23 要求）：删除等操作只在
         「批量操作」模式下的操作栏里出现，操作栏展开在刷新按钮所在的那张卡里；
         进入批量模式后点封面 = 选中（选中态是封面内侧一圈品牌色描边）；
       - **「添加游戏」入口在顶栏「游戏库」卡片里**（2026-11-23 起去掉网格末尾的圆形加号），点它弹出「添加游戏」窗（「扫描添加」扫 Steam 库 / 本地文件夹，点名字即去搜索；「手动搜索」是原来的搜索流程）
         （原来顶栏的「搜索」按钮与网格末尾的圆形加号都已移除）。
     数据全来自本地库（stores/libraryStore.ts → libraryApi.list），列表本身不发网络请求，
     断网也能看；封面是后端读文件转成的 base64 data URL（WebView 里一定显示得出来）。 -->
<template>
  <div class="w-full max-w-5xl mx-auto h-full min-h-0 flex flex-col gap-4 select-text">
    <!-- ── 顶部：标题 + 已入库数量 + 刷新 + 添加游戏 + 批量操作 ───────────────
         平时只有标题这一行（没有内容区），MetaSection 会把上下内边距收到 header 上；
         「添加游戏」入口与「批量操作」按钮都放在这张卡里（用户 2026-11-23 要求）；
         进入批量模式后，操作项在右侧的浮空长条卡片里（lib-batch-panel，图标竖排）。 -->
    <MetaSection icon="i-lucide-gamepad-2" title="游戏库">
      <template #actions>
        <!-- 已入库数量放在标题栏右侧，和按钮同一行居中 -->
        <span class="text-xs text-tertiary whitespace-nowrap">
          已入库 <span class="text-primary font-medium tabular-nums">{{ library.count }}</span> 款
        </span>
        <button
          class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none disabled:opacity-50"
          :disabled="library.loading"
          @click="library.load(true)"
        >
          <span class="i-lucide-refresh-cw icon-xs" aria-hidden="true" />
          {{ library.loading ? '刷新中…' : '刷新' }}
        </button>
        <!-- 「添加游戏」入口（用户 2026-11-23 要求：不再用网格末尾的圆形加号，
             改放到「游戏库」标题所在的这张卡里） -->
        <button
          type="button"
          class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none"
          title="添加游戏：扫描 Steam 库 / 本地文件夹，或手动搜索"
          @click="openAddGame"
        >
          <span class="i-lucide-plus icon-xs" aria-hidden="true" />
          添加游戏
        </button>
        <button
          v-if="!batchMode"
          type="button"
          class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none disabled:opacity-50"
          :disabled="!library.count"
          title="进入批量操作：选几张卡一起删除"
          @click="enterBatch"
        >
          <span class="i-lucide-list-checks icon-xs" aria-hidden="true" />
          批量操作
        </button>
        <button
          v-else
          type="button"
          class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none"
          title="退出批量操作"
          @click="exitBatch"
        >
          <span class="i-lucide-x icon-xs" aria-hidden="true" />
          退出批量
        </button>
      </template>

    </MetaSection>

    <MetaMessage v-if="library.error" tone="error" :text="library.error" />

    <!-- ── 网格：自己开滚动（MainLayout 的 router-view 是 overflow-hidden） ── -->
    <div class="flex-1 min-h-0 flex gap-4">
      <div class="flex-1 min-h-0 overflow-y-auto scrollbar-theme pr-2">
        <!-- 网格在有作品或已读完时都渲染：空库也要能看到末尾那个「添加游戏」圆钮 -->
        <div
          v-if="library.works.length || !library.loading"
          class="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3"
        >
          <!-- 一张卡片 = 一个封面按钮（**卡片上不再有任何操作按钮**，用户要求）。
               封面统一 2:3 等比裁切（Steam / DLsite / 以后所有来源都一样）：
               Steam 的 library_600x900 本身就是 2:3，DLsite 的 4:3 图会被 object-cover 居中裁掉两边。
               批量模式：点卡片 = 选中 / 取消选中（不再进详情）；选中时**边框与遮罩都用品牌色**
               （用户 2026-11-23 要求），边框/遮罩的具体写法在 scoped 样式里。 -->
          <div v-for="item in library.works" :key="itemKey(item)" class="group relative">
            <button
              type="button"
              class="lib-card relative block w-full overflow-hidden rounded-lg bg-secondary cursor-pointer p-0"
              :class="{ 'lib-selected': isSelected(item) }"
              :aria-label="
                batchMode
                  ? `${isSelected(item) ? '取消选择' : '选择'}「${item.name}」`
                  : `打开「${item.name}」的作品详情`
              "
              :aria-pressed="batchMode ? isSelected(item) : undefined"
              @click="handleCardClick(item)"
            >
              <span class="block w-full aspect-[2/3]">
                <img
                  v-if="library.coverFor(item)"
                  :src="library.coverFor(item)"
                  :alt="item.name"
                  class="w-full h-full object-cover"
                  draggable="false"
                />
                <span v-else class="flex-center w-full h-full text-muted">
                  <span class="i-lucide-image-off icon-lg" aria-hidden="true" />
                </span>
              </span>

              <!-- 选中遮罩：品牌色半透明铺满封面 + 居中勾选（与品牌色边框一起表达"已选中"）。
                   平时 opacity 0 且不接收指针事件，只在 .lib-selected 时浮现。
                   ⚠️ 类名不能叫 lib-mask：搜索弹窗的全屏遮罩已经是这个名字。 -->
              <span class="lib-select-mask" aria-hidden="true">
                <span class="i-lucide-check icon-lg" />
              </span>

            </button>
          </div>
        </div>

        <!-- 空库提示：添加入口在顶栏「游戏库」卡片里，这里只补一句"下一步做什么" -->
        <p
          v-if="!library.loading && !library.works.length"
          class="m-0 py-6 text-center text-xs text-tertiary"
        >
          游戏库还是空的 —— 点顶部「游戏库」卡片里的「添加游戏」：扫描 Steam 库 / 本地文件夹，也可以手动输入 RJ 编号或作品名；点开详情、再点「入库」才会收进这里。
        </p>
        <p v-else-if="library.loading && !library.works.length" class="m-0 text-sm text-tertiary py-6 text-center">正在读取游戏库…</p>
      </div>
      <!-- 批量操作长条：**浮空的独立小卡片**（fixed 定位，垂直位置在「游戏库」顶栏卡片
           下方、水平位置在游戏卡片右侧的留白里，宽 56px、高随内容）。
           **完全不参与布局** —— 网格宽度、卡片排布、滚动条位置都不变（用户 2026-11-23
           要求「在游戏库字样下方游戏卡片右侧显示 并且使用长条浮空设计 作为独立卡片弹出」
           「不遮盖现有ui」「不挤压现有ui」「太宽了 遮挡住了ui」）。
           条内只放得下图标：标题图标 + 已选数量 + 全选 / 删除 / 退出三个图标按钮，
           文字说明走 title / aria-label；选中与否由卡片自己的品牌色边框 + 遮罩表达。 -->
      <Transition name="lib-drawer">
        <aside v-if="batchMode" class="lib-batch-panel" aria-label="批量操作">
          <span class="i-lucide-list-checks icon-sm section-icon" aria-hidden="true" />

          <p class="lib-batch-count" :title="`已选 ${selectedCount} / ${library.count}`">
            <span class="lib-batch-num">{{ selectedCount }}</span>
            <span class="lib-batch-total">/{{ library.count }}</span>
          </p>

          <!-- 操作竖向排布（用户要求）：全选 / 删除所选 / 退出（删除要连点两下确认） -->
          <div class="lib-batch-actions">
            <button
              type="button"
              class="lib-batch-btn"
              :disabled="!library.count || library.removing"
              :title="allSelected ? '取消全选' : '全选'"
              :aria-label="allSelected ? '取消全选' : '全选'"
              @click="toggleSelectAll"
            >
              <span :class="allSelected ? 'i-lucide-square-check-big' : 'i-lucide-square'" class="icon-sm" aria-hidden="true" />
            </button>

            <button
              type="button"
              class="lib-batch-btn lib-batch-btn-danger"
              :class="{ 'is-armed': batchArmed }"
              :disabled="!selectedCount || library.removing"
              :title="batchArmed ? '再点一次确认删除' : `删除所选（${selectedCount}）—— 连点两下确认`"
              :aria-label="batchArmed ? '再点一次确认删除' : '删除所选'"
              @click="handleBatchDelete"
              @mouseleave="batchArmed = false"
              @blur="batchArmed = false"
            >
              <span
                :class="
                  library.removing
                    ? 'i-lucide-loader-circle spin'
                    : batchArmed
                      ? 'i-lucide-check'
                      : 'i-lucide-trash-2'
                "
                class="icon-sm"
                aria-hidden="true"
              />
            </button>

            <button
              type="button"
              class="lib-batch-btn"
              title="退出批量操作"
              aria-label="退出批量操作"
              @click="exitBatch"
            >
              <span class="i-lucide-x icon-sm" aria-hidden="true" />
            </button>
          </div>
        </aside>
      </Transition>
    </div>

    <!-- ── 添加游戏弹窗（扫描添加 / 手动搜索）：居中，周围模糊 ────────
         Teleport 到 body：避免被页面/侧栏的层级与 overflow 裁掉。
         Esc 关闭：焦点在输入框里，keydown 会冒泡到遮罩层。
         **点遮罩不关闭**（用户要求）：只有右上角 ✕ 与 Esc 能关，避免复制/选中文字时
         手一滑点在旁边就把整段搜索关掉。 -->
    <Teleport to="body">
      <Transition name="lib-modal">
        <div
          v-if="searchOpen"
          class="lib-mask fixed inset-0 z-[1300] flex items-center justify-center p-6"
          role="dialog"
          aria-modal="true"
          aria-label="添加游戏"
          @keydown.esc="closeSearch"
        >
          <div class="lib-dialog card-flat p-5 w-[min(560px,92vw)] max-h-[80vh] flex flex-col gap-4 overflow-hidden">
            <header class="flex items-center gap-2.5">
              <span class="i-lucide-gamepad-2 icon-md section-icon" aria-hidden="true" />
              <h2 class="section-title">添加游戏</h2>
              <button
                type="button"
                class="ml-auto w-7 h-7 rounded-md flex-center bg-transparent border-none text-tertiary cursor-pointer hover:text-primary hover:bg-hover transition-colors duration-200"
                aria-label="关闭添加游戏"
                @click="closeSearch"
              >
                <span class="i-lucide-x icon-sm" aria-hidden="true" />
              </button>
            </header>

            <!-- 两种添加方式：默认「扫描添加」（新方法），「手动搜索」保留原来的搜索流程 -->
            <div class="flex items-center gap-1" role="group" aria-label="添加方式">
              <button
                type="button"
                class="chip"
                :class="{ 'chip-active': addMode === 'scan' }"
                :aria-pressed="addMode === 'scan'"
                @click="addMode = 'scan'"
              >
                扫描添加
              </button>
              <button
                type="button"
                class="chip"
                :class="{ 'chip-active': addMode === 'manual' }"
                :aria-pressed="addMode === 'manual'"
                @click="addMode = 'manual'"
              >
                手动搜索
              </button>
            </div>

            <!-- 扫描添加：扫出来的名字点一下就进搜索（名字同时会写进手动搜索的输入框） -->
            <ScanPanel v-if="addMode === 'scan'" class="min-h-0" @detail="openScanResult" />

            <!-- 手动搜索：**一次同时查 DLsite 与 Steam**（用户 2026-11-23 要求
                 「手动搜索只显示 DLsite 数据源，应该也显示 steam 数据源」），
                 两个来源的候选分组列出，不再需要先选来源。 -->
            <template v-else>
              <p class="m-0 text-xs text-tertiary">{{ searchHint }}</p>
              <MetaSearchBar v-model="meta.keyword" :loading="meta.loading" @submit="submitSearch" />
            </template>

            <MetaMessage v-if="addMode === 'manual' && meta.error" tone="error" :text="meta.error" />
            <MetaMessage v-if="addMode === 'manual' && meta.searchNotice" tone="warning" :text="meta.searchNotice" />
            <MetaMessage v-if="addMode === 'manual' && meta.status" :text="meta.status" />
            <MetaMessage v-if="addMode === 'manual' && meta.pageLoading" :text="`正在加载第 ${meta.candidatePage} 页…`" />

            <!-- 候选：Steam 组（最多 10 条、无分页）+ DLsite 组（每页 10 条，可翻页） -->
            <template v-if="addMode === 'manual' && (meta.steamCandidates.length || meta.candidateTotal > 0)">
              <div class="min-h-0 flex-1 overflow-y-auto scrollbar-theme -mx-5 border-t-theme-light">
                <template v-if="meta.steamCandidates.length">
                  <p class="m-0 px-5 pt-3 pb-1.5 text-xs text-tertiary border-b-theme-light">
                    Steam 商店 · {{ meta.steamCandidates.length }} 个候选（最多 10 个，没有分页）
                  </p>
                  <MetaCandidateList :candidates="meta.steamCandidates" @select="chooseSteamCandidate" />
                </template>

                <template v-if="meta.candidateTotal > 0">
                  <p class="m-0 px-5 pt-3 pb-1.5 text-xs text-tertiary border-b-theme-light">
                    DLsite · 共 {{ meta.candidateTotal }} 条
                  </p>
                  <MetaCandidateList
                    :candidates="meta.pagedCandidates"
                    :covers="meta.candidateCovers"
                    @select="chooseCandidate"
                  />
                </template>
              </div>

              <!-- 分页只作用于 DLsite（Steam 的 storesearch 没有分页参数） -->
              <MetaPager
                v-if="meta.candidateTotal > 0"
                class="-mt-1"
                :page="meta.candidatePage"
                :page-count="meta.candidatePageCount"
                :total="meta.candidateTotal"
                @change="meta.setCandidatePage"
              />
            </template>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMetaStore } from '../stores/metaStore'
import { useLibraryStore } from '../stores/libraryStore'
import type { LibrarySource, LibraryWork } from '../services/libraryService'
import { dismissToast, toastError, toastLoading, toastSuccess } from '../utils/toast'
import type { ToastId } from '../utils/toast'
import MetaCandidateList from '../components/meta/MetaCandidateList.vue'
import MetaMessage from '../components/meta/MetaMessage.vue'
import MetaPager from '../components/meta/MetaPager.vue'
import MetaSearchBar from '../components/meta/MetaSearchBar.vue'
import MetaSection from '../components/meta/MetaSection.vue'
import ScanPanel from '../components/library/ScanPanel.vue'

/** 「添加游戏」弹窗的两种方式：扫描添加（默认）/ 手动搜索（原搜索流程） */
type AddMode = 'scan' | 'manual'

const library = useLibraryStore()
const meta = useMetaStore()
const router = useRouter()

/** 两个来源的输入提示：DLsite 支持 RJ 精确抓，Steam 按名字搜（候选最多 10 条、无分页） */
const searchHint = computed(
  () =>
    '一次同时搜索 DLsite 与 Steam：DLsite 支持 RJ 编号精确抓取（如 RJ01721474），Steam 按游戏名搜（最多 10 个候选）。两边结果分组列出，点「详情」核对后再入库。',
)

/** 一张卡片的唯一 key（编号在两个来源里可能重复，所以带上 source） */
function itemKey(item: Pick<LibraryWork, 'source' | 'workId'>): string {
  return `${item.source}:${item.workId}`
}

/** 搜索弹窗开关 */
const searchOpen = ref(false)
/** 弹窗当前方式：默认扫描添加；切到手动搜索走原来的输入框流程 */
const addMode = ref<AddMode>('scan')

// ===== 批量操作（用户 2026-11-23）=====
// 「只有批量操作游戏库时才弹出操作栏」：平时卡片上没有任何按钮，
// 点顶栏的「批量操作」才进入选卡模式，操作栏展开在刷新按钮所在的那张卡里。
/** 是否在批量模式：进入后点卡片 = 选中 / 取消选中，不再进详情 */
const batchMode = ref(false)
/** 已选中的卡片 key（`source:workId`，来源不同的同编号是两条记录） */
const selectedKeys = ref<string[]>([])
/** 「删除所选」的待确认态：第一次点只上膛，第二次才真删 */
const batchArmed = ref(false)

/** 当前选中的库记录（删除时交给 store；按当前列表过滤，刷新后不会带上已不存在的 key） */
const selectedWorks = computed(() =>
  library.works.filter((item) => selectedKeys.value.includes(itemKey(item))),
)
const selectedCount = computed(() => selectedWorks.value.length)
/** 当前列表是否已经全选（空库不算全选） */
const allSelected = computed(
  () => library.works.length > 0 && selectedWorks.value.length === library.works.length,
)

function isSelected(item: Pick<LibraryWork, 'source' | 'workId'>): boolean {
  return selectedKeys.value.includes(itemKey(item))
}

/** 进入批量模式：清空上次的选择与待确认态 */
function enterBatch() {
  batchMode.value = true
  selectedKeys.value = []
  batchArmed.value = false
}

/** 退出批量模式：卡片恢复成"点开详情" */
function exitBatch() {
  batchMode.value = false
  selectedKeys.value = []
  batchArmed.value = false
}

function toggleSelect(item: LibraryWork) {
  const key = itemKey(item)
  selectedKeys.value = selectedKeys.value.includes(key)
    ? selectedKeys.value.filter((k) => k !== key)
    : [...selectedKeys.value, key]
  // 选择变了就撤销待确认态：避免「看到的是 A，确认时删的是 B」
  batchArmed.value = false
}

function toggleSelectAll() {
  selectedKeys.value = allSelected.value ? [] : library.works.map((item) => itemKey(item))
  batchArmed.value = false
}

/** 点卡片：批量模式下只切换选中；平时进作品详情 */
function handleCardClick(item: LibraryWork) {
  if (batchMode.value) {
    toggleSelect(item)
    return
  }
  void openWork(item)
}

/** 删除所选：连点两下确认（与 TODO 的删除同一套路，不弹确认框）
 *  2026-11-23：删除是游戏库的「移除」，耗时与条数相关（逐条删库记录 + 封面文件），
 *  所以操作超过 300ms 时弹一条 loading 提示占位；结束时**先关掉它、再另弹一条结果**。
 *  ⚠️ 不能把 loading 那条用同一个 id 原地改成结果：sonner 的 loading toast 不启动自动关闭计时器，
 *  原地更新后计时器也不会重启，提示会永远停在屏幕上（用户报的 bug）。详见 utils/toast.ts。 */
async function handleBatchDelete() {
  if (!selectedCount.value || library.removing) return
  if (!batchArmed.value) {
    batchArmed.value = true
    return
  }
  batchArmed.value = false

  const count = selectedCount.value
  // 「正在移除」占位提示：只有操作真的慢（>300ms）才弹，删一两款时不会闪一下再消失。
  // 结束时必须 dismissToast 关掉它 —— loading toast 不会自动消失，也不能原地改成结果（见 utils/toast.ts）。
  const slowNotice: { id: ToastId | null } = { id: null }
  const loadingTimer = window.setTimeout(() => {
    slowNotice.id = toastLoading(count > 1 ? `正在移除 ${count} 款游戏…` : '正在从游戏库移除…')
  }, 300)
  let removed = 0
  try {
    removed = await library.removeMany(selectedWorks.value)
  } catch (e) {
    // removeMany 自己会兜住逐条失败；走到这里说明是意外错误（此时选择集留着，方便重试）
    window.clearTimeout(loadingTimer)
    if (slowNotice.id !== null) dismissToast(slowNotice.id)
    toastError('删除失败', { description: e instanceof Error ? e.message : '请稍后再试' })
    return
  }
  window.clearTimeout(loadingTimer)
  if (slowNotice.id !== null) dismissToast(slowNotice.id)
  selectedKeys.value = []

  if (removed > 0) {
    toastSuccess(removed > 1 ? `已移除 ${removed} 款游戏` : '已移除 1 款游戏', {
      description: '封面文件也一并删除了',
    })
    // 部分失败时 store.error 里有逐条原因（页面上的提示条也会显示，这里补一条浮层提醒）
    if (library.error) toastError('有游戏没能删除', { description: library.error })
  } else if (library.error) {
    toastError('删除失败', { description: library.error })
  }

  // 全删光了就没有"选卡"的意义了，自动退回普通浏览
  if (removed > 0 && !library.works.length) exitBatch()
}

/**
 * 打开「添加游戏」弹窗：清掉上一次的搜索结果/错误/状态，并**清空输入框**，默认落在扫描添加。
 * 搜索成功后 `runSearch` 会把关键词（往往是 RJ 编号）留在 store 里，弹窗一开就会
 * 在输入框里显示出来 —— 用户明确不要"上次搜过的编号"自动出现，所以每次开窗都从空白开始。
 * 候选也一起清掉：现在是 DLsite + Steam 两组，留着上次的两组候选比只剩 DLsite 时更让人困惑。
 * 扫描结果保留（scanStore 独立于弹窗），重新打开还能接着点上次扫到的名字。
 */
function openAddGame() {
  meta.clearResult()
  meta.keyword = ''
  addMode.value = 'scan'
  searchOpen.value = true
}

function closeSearch() {
  searchOpen.value = false
}

/**
 * 提交搜索：**DLsite 与 Steam 一起搜**（metaStore.runMultiSearch）。
 * 命中作品（按编号精准抓）就直接进作品库；只是候选列表就留在弹窗里让用户点「详情」。
 *
 * ⚠️ 必须用 `runMultiSearch()` 的返回值把关，不能只看 `meta.work`：
 * store 里可能还留着上一次搜索的作品，搜索栏为空时点「搜索」会拿它跳详情页
 * （用户报的 bug）。空关键词时只提示「搜索栏不能为空」并返回 false。
 */
async function submitSearch() {
  const searched = await meta.runMultiSearch()
  if (searched && meta.work) finishSearch()
}

/**
 * 点扫描结果里的「详情」：按编号精确抓取后进 /works 核对。
 * Steam 条目按 appid 读商店详情（source='steam'），其余仍走 DLsite 缓存优先。
 * 写库不在这里 —— 是否入库由 ScanPanel 的「全部入库 / 入库所选」决定。
 */
async function openScanResult(workId: string, source: LibrarySource = 'dlsite') {
  await meta.loadById(workId, false, source)
  if (meta.work) finishSearch()
}

/** 点候选「详情」：精确抓取 → 进作品库（候选永远来自 DLsite 手动搜索） */
async function chooseCandidate(id: string) {
  await meta.loadById(id, false, 'dlsite')
  if (meta.work) finishSearch()
}

/** 点 Steam 候选「详情」：按 appid 取商店详情（source='steam'）→ 进作品库 */
async function chooseSteamCandidate(id: string) {
  await meta.loadById(id, false, 'steam')
  if (meta.work) finishSearch()
}

/** 搜索成功：清空输入框 + 关弹窗 + 进作品库看详情（关键词不留到下一次开窗） */
function finishSearch() {
  meta.keyword = ''
  searchOpen.value = false
  void router.push('/works')
}

/**
 * 点封面进入作品库详情：
 * 先进 /works 再 loadById —— loadById 是缓存优先（库里有就不联网），所以详情几乎是瞬间出来的。
 * 必须把卡片的来源一并带过去：同一个编号在 dlsite / steam 里是两条记录。
 */
async function openWork(item: LibraryWork) {
  await router.push('/works')
  await meta.loadById(item.workId, false, item.source as LibrarySource)
}

onMounted(() => {
  // 每次进页面都重新读一次：别的页面刚抓完的作品要能立刻看到
  void library.load(true)
})
</script>

<style scoped>
/* ===== 名称面板 =====
   半透明背景必须用 color-mix：项目不允许 `bg-secondary/50` 这类透明度后缀
   （UnoCSS rules 是精确匹配，写了会被静默丢弃），而色值本身只能来自全局 token。
   基础配色随主题：浅色=浅底深字，深色=深底浅字，所以用 bg-base + text-primary。
   位置固定在封面下沿，向上淡入的位移交给模板里的 translate-y-full → translate-y-0。 */
.lib-name {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 6px 8px;
  font-size: 12px;
  line-height: 1.35;
  text-align: left;
  color: var(--color-text-primary);
  background: color-mix(in srgb, var(--color-bg-base) 88%, transparent);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
  /* 最多两行，超出省略 */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* ===== 卡片边框 + 批量模式选中态 =====
   基础边框写在这里（模板不再挂 border-theme）：选中时要一次改 width/color，
   和 UnoCSS 的 border-theme 简写叠在一起会争同一属性（同属性只留一条，容易互相压掉）。
   选中态 = **品牌色边框 + 品牌色遮罩 + 居中勾选**（用户 2026-11-23 要求两者都用品牌色）。
   边框从 1px 变 2px：border-box 下只让内容缩 1px，不改变卡片外尺寸与圆角。 */
.lib-card {
  border: 1px solid var(--color-border);
  transition: border-color var(--motion-fast) var(--ease-enter);
}

.lib-selected {
  border: 2px solid var(--color-primary);
}

/* 选中遮罩：铺满封面，品牌色半透明；勾选图标用反色文字（浅色品牌配米白、深色品牌配深绿） */
.lib-select-mask {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--color-primary) 38%, transparent);
  color: var(--color-text-on-brand);
  opacity: 0;
  pointer-events: none;
  transition: opacity var(--motion-fast) var(--ease-enter);
}

.lib-selected .lib-select-mask {
  opacity: 1;
}

/* ===== 右侧批量操作长条（批量模式下浮出的独立小卡片）=====
   用户 2026-11-23 要求：「在游戏库字样下方游戏卡片右侧显示 并且使用长条浮空设计
   作为独立卡片弹出」「不遮盖现有ui」「不挤压现有ui」「太宽了」——
   所以它是一张 **fixed 浮空的竖向长条卡片**：
     · 高度自适应内容（不是整条侧边栏），圆角 + 描边 + 阴影，像一张独立卡片；
     · 垂直位置 = 「游戏库」顶栏卡片下方（top 112px），水平位置 = 网格右侧的留白；
     · **完全不参与布局**（网格宽度、卡片排布、滚动条位置都不变），宽只有 56px，
       常规窗口内容两侧的留白（页面是 max-w-5xl 居中）就能容下它。
   z-index 1050：低于标题栏（1100）、高于侧栏（1000）与页面内容。 */
.lib-batch-panel {
  position: fixed;
  right: 15px;
  z-index: 1050;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  width: 56px;
  padding: 12px 0;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-bg-card);
  box-shadow: var(--color-shadow-heavy);
}

.lib-batch-count {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1px;
  margin: 0;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}

.lib-batch-num {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.lib-batch-total {
  font-size: 10px;
  color: var(--color-text-tertiary);
}

/* 操作竖向排布（用户要求）：一条竖线三个图标按钮 */
.lib-batch-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

/* 长条里的图标按钮：36×36，描边口径与标题栏/行内图标按钮一致 */
.lib-batch-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  border: 1px solid var(--color-border-dark);
  border-radius: 10px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition:
    background-color var(--motion-fast) var(--ease-enter),
    border-color var(--motion-fast) var(--ease-enter),
    color var(--motion-fast) var(--ease-enter),
    opacity var(--motion-fast) var(--ease-enter);
}

.lib-batch-btn:hover:not(:disabled) {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.lib-batch-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* 删除：红描边红图标 → hover / armed 填红 + 反色（与 TODO 行内删除同一套语义） */
.lib-batch-btn-danger {
  border-color: var(--color-danger);
  color: var(--color-danger);
}

.lib-batch-btn-danger:hover:not(:disabled),
.lib-batch-btn-danger.is-armed {
  background: var(--color-danger);
  border-color: var(--color-danger);
  color: var(--color-text-on-danger);
}

/* 弹出/收起：从右侧轻微滑入（独立卡片，不需要整条移出视口） */
.lib-drawer-enter-active,
.lib-drawer-leave-active {
  transition:
    opacity var(--motion-slow) var(--ease-enter),
    transform var(--motion-slow) var(--ease-enter);
}

.lib-drawer-enter-from,
.lib-drawer-leave-to {
  opacity: 0;
  transform: translateX(12px);
}

/* ===== 搜索弹窗 =====
   遮罩用半透明 + 模糊：让"周围虚化、中间清晰"，色值同样只能来自 token
   （bg-base 在浅色主题是浅底、深色主题是深底，无需两套写法）。 */
.lib-mask {
  background: color-mix(in srgb, var(--color-bg-base) 62%, transparent);
  -webkit-backdrop-filter: blur(8px);
  backdrop-filter: blur(8px);
}

.lib-dialog {
  box-shadow: var(--color-shadow-heavy);
}

/* 进出场：遮罩淡入 + 面板轻微缩放（时长/缓动走全局动效 token） */
.lib-modal-enter-active,
.lib-modal-leave-active {
  transition: opacity var(--motion-base) var(--ease-enter);
}

.lib-modal-enter-active .lib-dialog,
.lib-modal-leave-active .lib-dialog {
  transition:
    transform var(--motion-base) var(--ease-enter),
    opacity var(--motion-base) var(--ease-enter);
}

.lib-modal-enter-from,
.lib-modal-leave-to {
  opacity: 0;
}

.lib-modal-enter-from .lib-dialog,
.lib-modal-leave-to .lib-dialog {
  transform: scale(0.96);
  opacity: 0;
}
</style>
