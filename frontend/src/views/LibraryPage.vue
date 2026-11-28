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
        <!-- 已入库数量放在标题栏右侧；进入批量模式后这行换成操作提示（同一行替换，不挤压布局） -->
        <span v-if="batchMode" class="text-xs text-tertiary whitespace-nowrap">
          点击封面选择 · 已选
          <span class="text-primary font-medium tabular-nums">{{ selectedCount }}</span>
          款
        </span>
        <span v-else class="text-xs text-tertiary whitespace-nowrap">
          已入库 <span class="text-primary font-medium tabular-nums">{{ library.count }}</span> 款
        </span>
        <n-button text size="tiny" :disabled="library.loading" @click="library.refresh">
          <template #icon>
            <span class="i-lucide-refresh-cw icon-xs" aria-hidden="true" />
          </template>
          {{ library.loading ? '刷新中…' : '刷新' }}
        </n-button>
        <!-- 「添加游戏」入口（用户 2026-11-23 要求：不再用网格末尾的圆形加号，
             改放到「游戏库」标题所在的这张卡里） -->
        <n-button
          text
          size="tiny"
          title="添加游戏：扫描 Steam 库 / 本地文件夹，或手动搜索"
          @click="openAddGame"
        >
          <template #icon>
            <span class="i-lucide-plus icon-xs" aria-hidden="true" />
          </template>
          添加游戏
        </n-button>
        <n-button
          v-if="!batchMode"
          text
          size="tiny"
          :disabled="!library.count"
          title="进入批量操作：选几张卡一起删除"
          @click="enterBatch"
        >
          <template #icon>
            <span class="i-lucide-list-checks icon-xs" aria-hidden="true" />
          </template>
          批量操作
        </n-button>
        <n-button v-else text size="tiny" title="退出批量操作" @click="exitBatch">
          <template #icon>
            <span class="i-lucide-x icon-xs" aria-hidden="true" />
          </template>
          退出批量
        </n-button>
      </template>

    </MetaSection>

    <!-- 读取/删除失败统一走 toast（libraryStore 里弹） -->

    <!-- ── 网格：自己开滚动（MainLayout 的 router-view 是 overflow-hidden） ──
         scroll-fade：上下边缘按「还能不能继续滚」渐隐（参考 shadcn 的 scroll-fade，
         指令见 utils/scrollFade.ts；只有真的还有内容在视口外时才淡）。 -->
    <div class="flex-1 min-h-0 flex gap-4">
      <div v-scroll-fade class="scroll-fade flex-1 min-h-0 overflow-y-auto scrollbar-theme pr-2">
        <!-- 网格在外层渲染（空库/加载中也要占位）：加载中先铺骨架屏卡片，
             不再先闪一下「游戏库还是空的」再冒出内容（用户 2026-11-27 要求用 naive 的 Skeleton）。 -->
        <div
          class="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3"
          :class="{ 'lib-batch-mode': batchMode }"
        >
          <!-- 数据加载中的骨架屏：与真实封面同比例（2:3）、同间距 -->
          <template v-if="library.loading && !library.works.length">
            <div v-for="i in 12" :key="`lib-skeleton-${i}`" class="lib-skeleton">
              <n-skeleton height="100%" :sharp="true" />
            </div>
          </template>
          <!-- 一张卡片 = 一个封面按钮（**卡片上不再有任何操作按钮**，用户要求）。
               封面统一 2:3 等比裁切（Steam / DLsite / 以后所有来源都一样）：
               Steam 的 library_600x900 本身就是 2:3，DLsite 的 4:3 图会被 object-cover 居中裁掉两边。
               批量模式：点卡片 = 选中 / 取消选中（不再进详情）；左上角出现方形勾选框、
               未选中的卡片压暗去饱和（「进入批量操作后卡片样式变化」），
               选中时**边框与遮罩都用品牌色**。
               平时：hover 轻微上浮 + 封面放大 + 底部名称浮层（仿 Steam 库卡片），
               点击从指针位置扩散一圈水纹（lib-ripple）。 -->
          <div v-for="item in library.works" :key="itemKey(item)" class="group relative">
            <button
              type="button"
              class="lib-card relative block w-full overflow-hidden rounded-none bg-secondary cursor-pointer p-0"
              :class="{ 'lib-selected': isSelected(item) }"
              :aria-label="
                batchMode
                  ? `${isSelected(item) ? '取消选择' : '选择'}「${item.name}」`
                  : `打开「${item.name}」的作品详情`
              "
              :aria-pressed="batchMode ? isSelected(item) : undefined"
              @click="handleCardClick(item, $event)"
            >
              <span class="lib-card-media block w-full aspect-[2/3]">
                <img
                  v-if="library.coverFor(item)"
                  :src="library.coverFor(item)"
                  :alt="item.name"
                  class="lib-card-img w-full h-full object-cover"
                  draggable="false"
                />
                <span v-else class="flex-center w-full h-full text-muted">
                  <span class="i-lucide-image-off icon-lg" aria-hidden="true" />
                </span>
              </span>

              <!-- 名称浮层：hover 时从下方淡入（Steam 库卡片的信息面板） -->
              <span class="lib-card-caption" aria-hidden="true" :title="item.name">
                {{ item.name }}
              </span>

              <!-- 批量模式的方形勾选框：未选为空框、选中填品牌色（卡片样式变化的提示） -->
              <span v-if="batchMode" class="lib-check" :class="{ 'lib-check--on': isSelected(item) }" aria-hidden="true">
                <span v-if="isSelected(item)" class="i-lucide-check icon-xs" />
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
        <n-empty
          v-if="!library.loading && !library.works.length"
          class="py-8"
          description="游戏库还是空的"
        >
          <template #extra>
            <p class="m-0 text-xs text-tertiary max-w-[46ch] text-center">
              点顶部「游戏库」卡片里的「添加游戏」：扫描 Steam 库 / 本地文件夹，也可以手动输入 RJ 编号或作品名；点开详情、再点「入库」才会收进这里。
            </p>
          </template>
        </n-empty>
      </div>
      <!-- 批量操作长条：**浮空的独立小卡片**（fixed 定位，垂直位置在「游戏库」顶栏卡片
           下方、水平位置在游戏卡片右侧的留白里，宽 56px、高随内容）。
           **完全不参与布局** —— 网格宽度、卡片排布、滚动条位置都不变（用户 2026-11-23
           要求「在游戏库字样下方游戏卡片右侧显示 并且使用长条浮空设计 作为独立卡片弹出」
           「不遮盖现有ui」「不挤压现有ui」「太宽了 遮挡住了ui」）。
           条内只放得下图标：标题图标 + 已选数量 + 全选 / 删除 / 退出三个图标按钮，
           文字说明走 title / aria-label；选中与否由卡片自己的品牌色边框 + 遮罩表达。 -->
      <Transition name="lib-drawer">
        <aside v-if="batchMode" class="lib-batch-panel" aria-label="批量操作" title="批量操作：点击封面选择作品">
          <span class="i-lucide-list-checks icon-sm text-brand flex-center" aria-hidden="true" />

          <p class="lib-batch-count" :title="`已选 ${selectedCount} / ${library.count}`">
            <span class="lib-batch-num">{{ selectedCount }}</span>
            <span class="lib-batch-total">/{{ library.count }}</span>
          </p>

          <!-- 操作竖向排布（用户要求）：全选 / 删除所选 / 退出 -->
          <div class="lib-batch-actions">
            <n-tooltip placement="left" :delay="300">
              <template #trigger>
                <n-button
                  size="small"
                  quaternary
                  :disabled="!library.count || library.removing"
                  :aria-label="allSelected ? '取消全选' : '全选'"
                  @click="toggleSelectAll"
                >
                  <template #icon>
                    <span :class="allSelected ? 'i-lucide-square-check-big' : 'i-lucide-square'" class="icon-sm" aria-hidden="true" />
                  </template>
                </n-button>
              </template>
              {{ allSelected ? '取消全选' : '全选' }}
            </n-tooltip>

            <n-popconfirm
              positive-text="删除"
              negative-text="取消"
              :positive-button-props="{ type: 'error' }"
              @positive-click="handleBatchDelete"
            >
              <template #trigger>
                <n-button
                  size="small"
                  quaternary
                  type="error"
                  :disabled="!selectedCount || library.removing"
                  :loading="library.removing"
                  aria-label="删除所选"
                >
                  <template #icon>
                    <span class="i-lucide-trash-2 icon-sm" aria-hidden="true" />
                  </template>
                </n-button>
              </template>
              删除选中的 {{ selectedCount }} 款游戏？封面文件也会一并删除。
            </n-popconfirm>

            <n-tooltip placement="left" :delay="300">
              <template #trigger>
                <n-button size="small" quaternary aria-label="退出批量操作" @click="exitBatch">
                  <template #icon>
                    <span class="i-lucide-x icon-sm" aria-hidden="true" />
                  </template>
                </n-button>
              </template>
              退出批量
            </n-tooltip>
          </div>
        </aside>
      </Transition>
    </div>

    <!-- ── 添加游戏弹窗（扫描添加 / 手动搜索）：改用 naive 的 n-modal ────────
         点遮罩不关闭（用户要求）：只有右上角 ✕ 与 Esc 能关。
         mask-style 里加一层模糊，保留原来「周围虚化、中间清晰」的观感。 -->
    <n-modal
      v-model:show="searchOpen"
      preset="card"
      title="添加游戏"
      :mask-closable="false"
      :style="{ width: 'min(560px, 92vw)' }"
      :mask-style="{ backdropFilter: 'blur(8px)' }"
      class="add-game-modal"
      :content-style="{ display: 'flex', flexDirection: 'column', minHeight: 0, flex: '1 1 auto', overflow: 'hidden' }"
    >
      <div
        class="flex min-h-0 flex-1 flex-col gap-4"
        :class="addMode === 'manual' ? 'overflow-hidden' : 'overflow-y-auto scrollbar-theme pr-1'"
      >
        <!-- 两种添加方式：默认「扫描添加」，「手动搜索」保留原来的搜索流程 -->
        <n-radio-group
          class="shrink-0"
          :value="addMode"
          size="small"
          name="add-game-mode"
          aria-label="添加方式"
          @update:value="(value: string | number | boolean) => (addMode = value as AddMode)"
        >
          <n-radio-button value="scan">扫描添加</n-radio-button>
          <n-radio-button value="manual">手动搜索</n-radio-button>
        </n-radio-group>

        <!-- 扫描添加：扫出来的名字点一下就进搜索（名字同时会写进手动搜索的输入框） -->
        <ScanPanel v-if="addMode === 'scan'" class="min-h-0" @detail="openScanResult" />

        <!-- 手动搜索：**一次同时查 DLsite 与 Steam**（用户 2026-11-23 要求
             「手动搜索只显示 DLsite 数据源，应该也显示 steam 数据源」），
             两个来源各自成页、用上面的源切换器查看（2026-11-24），不再把两源塞进同一列表。 -->
        <template v-else>
          <p class="m-0 shrink-0 text-xs text-tertiary">{{ searchHint }}</p>
          <MetaSearchBar class="shrink-0" v-model="meta.keyword" :loading="meta.loading" @submit="submitSearch" />
        </template>

        <!-- 搜索/分页/游戏库的一次性反馈全部走 toast（2026-11-25 用户要求） -->

        <!-- 候选：**不同来源分开显示**（用户 2026-11-24 要求：不要把多源数据归并到一页里）。
             用源切换器选看 Steam 或 DLsite，分页只属于当前源（Steam 最多 10 条、没有分页）。
             列表与分页仍是一张卡片：列表滚动、分页固定在卡片底部。 -->
        <template v-if="addMode === 'manual' && sourceOptions.length">
          <!-- 源切换：只有存在候选的源才出现，按钮上带条数 -->
          <n-radio-group
            class="shrink-0"
            :value="activeSource"
            size="small"
            name="candidate-source"
            aria-label="候选来源"
            @update:value="(value: string | number | boolean) => (searchSource = value as LibrarySource)"
          >
            <n-radio-button v-for="option in sourceOptions" :key="option.value" :value="option.value">
              {{ option.label }}（{{ option.count }}）
            </n-radio-button>
          </n-radio-group>

          <div class="flex min-h-[140px] flex-1 flex-col overflow-hidden rounded-none border-theme bg-card">
            <div v-scroll-fade class="scroll-fade min-h-0 flex-1 overflow-y-auto scrollbar-theme">
              <MetaCandidateList
                v-if="activeSource === 'dlsite'"
                :candidates="meta.pagedCandidates"
                :covers="meta.candidateCovers"
                @select="chooseCandidate"
              />
              <MetaCandidateList
                v-else
                :candidates="meta.steamCandidates"
                @select="chooseSteamCandidate"
              />
            </div>

            <!-- 分页只属于当前源；Steam 没有分页，所以只在 DLsite 下出现。
                 翻页改成顶部那条很细的「处理中」进度条（原来是常驻 loading toast，
                 2026-11-25 用户收紧 toast 口径后放回页面内，不再弹浮层）。 -->
            <div
              v-if="activeSource === 'dlsite' && meta.candidateTotal > 0"
              class="shrink-0 border-t-theme-light px-2 py-2"
            >
              <n-progress
                v-if="meta.pageLoading"
                class="mb-2"
                type="line"
                :percentage="100"
                processing
                :show-indicator="false"
                :height="2"
              />
              <MetaPager
                :page="meta.candidatePage"
                :page-count="meta.candidatePageCount"
                @change="meta.setCandidatePage"
              />
            </div>
          </div>
        </template>
      </div>
    </n-modal>

  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  NButton,
  NEmpty,
  NModal,
  NPopconfirm,
  NProgress,
  NRadioButton,
  NRadioGroup,
  NSkeleton,
  NTooltip,
} from 'naive-ui'
import { useMetaStore } from '../stores/metaStore'
import { useLibraryStore } from '../stores/libraryStore'
import type { LibrarySource, LibraryWork } from '../services/libraryService'
import { toastError, toastSuccess } from '../utils/toast'
import MetaCandidateList from '../components/meta/MetaCandidateList.vue'
import MetaPager from '../components/meta/MetaPager.vue'
import MetaSearchBar from '../components/meta/MetaSearchBar.vue'
import MetaSection from '../components/meta/MetaSection.vue'
import ScanPanel from '../components/library/ScanPanel.vue'

/** 「添加游戏」弹窗的两种方式：扫描添加（默认）/ 手动搜索（原搜索流程） */
type AddMode = 'scan' | 'manual'

const library = useLibraryStore()
const meta = useMetaStore()
const router = useRouter()

/**
 * 翻页加载提示：2026-11-25 用户收紧 toast 口径后不再弹浮层，
 * 改为分页条上方一条 2px 的「处理中」进度条（见模板），加载期间不打断视线。
 */

/** 两个来源的输入提示：DLsite 支持 RJ 精确抓，Steam 按名字搜（候选最多 10 条、无分页） */
const searchHint = computed(
  () =>
    '一次同时搜索 DLsite 与 Steam：DLsite 支持 RJ 编号精确抓取（如 RJ01721474），Steam 按游戏名搜（最多 10 个候选）。两边结果分开显示（上方切换来源），点「详情」核对后再入库。',
)

/** 一张卡片的唯一 key（编号在两个来源里可能重复，所以带上 source） */
function itemKey(item: Pick<LibraryWork, 'source' | 'workId'>): string {
  return `${item.source}:${item.workId}`
}

/** 搜索弹窗开关 */
const searchOpen = ref(false)
/** 弹窗当前方式：默认扫描添加；切到手动搜索走原来的输入框流程 */
const addMode = ref<AddMode>('scan')
/** 手动搜索当前查看的候选来源：不同源不归并到同一页，用切换器分开看 */
const searchSource = ref<LibrarySource>('dlsite')

/** 有候选的来源（切换器只出现有结果的源，按钮上带条数） */
const sourceOptions = computed(() => {
  const options: Array<{ value: LibrarySource; label: string; count: number }> = []
  if (meta.candidateTotal > 0) {
    options.push({ value: 'dlsite', label: 'DLsite', count: meta.candidateTotal })
  }
  if (meta.steamCandidates.length) {
    options.push({ value: 'steam', label: 'Steam 商店', count: meta.steamCandidates.length })
  }
  return options
})

/** 生效的源：用户选的源已经没有候选时，回落到第一个有候选的源 */
const activeSource = computed<LibrarySource>(() => {
  const values = sourceOptions.value.map((option) => option.value)
  return values.includes(searchSource.value) ? searchSource.value : (values[0] ?? 'dlsite')
})

// ===== 批量操作（用户 2026-11-23）=====
// 「只有批量操作游戏库时才弹出操作栏」：平时卡片上没有任何按钮，
// 点顶栏的「批量操作」才进入选卡模式，操作栏展开在刷新按钮所在的那张卡里。
/** 是否在批量模式：进入后点卡片 = 选中 / 取消选中，不再进详情 */
const batchMode = ref(false)
/** 已选中的卡片 key（`source:workId`，来源不同的同编号是两条记录） */
const selectedKeys = ref<string[]>([])

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

/** 进入批量模式：清空上次的选择 */
function enterBatch() {
  batchMode.value = true
  selectedKeys.value = []
}

/** 退出批量模式：卡片恢复成"点开详情" */
function exitBatch() {
  batchMode.value = false
  selectedKeys.value = []
}

function toggleSelect(item: LibraryWork) {
  const key = itemKey(item)
  selectedKeys.value = selectedKeys.value.includes(key)
    ? selectedKeys.value.filter((k) => k !== key)
    : [...selectedKeys.value, key]
}

function toggleSelectAll() {
  selectedKeys.value = allSelected.value ? [] : library.works.map((item) => itemKey(item))
}

/**
 * 点击水纹：从指针落点扩散一个圆再淡出（用户 2026-11-25 要求「cover 添加点击水纹扩散动画」）。
 * 为什么用命令式 DOM 而不是 v-for 状态数组：水纹是一次性的装饰，动画结束即销毁；
 * 放进响应式状态会让整张网格跟着重渲染，而卡片本身不需要知道有几个水纹。
 * 圆用「卡片对角线 × 2」当直径，保证从任意落点扩散都能盖满整张封面。
 */
function spawnRipple(card: HTMLElement, event: MouseEvent) {
  const rect = card.getBoundingClientRect()
  const size = Math.hypot(rect.width, rect.height) * 2
  const ripple = document.createElement('span')
  ripple.className = 'lib-ripple'
  ripple.style.width = `${size}px`
  ripple.style.height = `${size}px`
  ripple.style.left = `${event.clientX - rect.left}px`
  ripple.style.top = `${event.clientY - rect.top}px`
  card.appendChild(ripple)
  // animationend 为主（动画放完立刻收）；再挂一个兜底定时器：
  // 窗口被遮挡/动画被节流时 animationend 可能迟迟不来，不能把水纹节点留在 DOM 里。
  ripple.addEventListener('animationend', () => ripple.remove(), { once: true })
  window.setTimeout(() => ripple.remove(), 800)
}

/** 点卡片：批量模式下只切换选中；平时进作品详情。两种情况都先扩散一圈水纹。 */
function handleCardClick(item: LibraryWork, event: MouseEvent) {
  const card = event.currentTarget as HTMLElement | null
  if (card) spawnRipple(card, event)
  if (batchMode.value) {
    toggleSelect(item)
    return
  }
  void openWork(item)
}

/** 删除所选：确认由 n-popconfirm 气泡承担。
 *  删除是游戏库的「移除」，耗时与条数相关（逐条删库记录 + 封面文件）。
 *  2026-11-25 用户收紧 toast 口径后，进行中不再弹 loading 提示 ——
 *  删除按钮自己会转圈（:loading="library.removing"）；结束后只留一条结果提示。 */
async function handleBatchDelete() {
  if (!selectedCount.value || library.removing) return

  let removed = 0
  try {
    removed = await library.removeMany(selectedWorks.value)
  } catch (e) {
    // removeMany 自己会兜住逐条失败；走到这里说明是意外错误（此时选择集留着，方便重试）
    toastError('删除失败', { description: e instanceof Error ? e.message : '请稍后再试' })
    return
  }
  selectedKeys.value = []

  if (removed > 0) {
    toastSuccess(removed > 1 ? `已移除 ${removed} 款游戏` : '已移除 1 款游戏', {
      description: '封面文件也一并删除了',
    })
    // 部分失败时 libraryStore 已弹「有游戏没能删除」的 toast，这里不再重复
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
  searchSource.value = 'dlsite'
  searchOpen.value = true
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
  if (!searched) return
  if (meta.work) {
    finishSearch()
    return
  }
  // 两边都搜完后默认看 DLsite；DLsite 没结果时自动落到 Steam（不把两源塞进同一页）
  searchSource.value = meta.candidateTotal > 0 ? 'dlsite' : 'steam'
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
 * **先把作品取回来、再切路由**（用户 2026-11-27：打开作品页会短暂闪「游戏库还是空的」）。
 * 反过来的话 /works 会先挂载一帧「还没有选中作品」的空状态：store 里还没有 work，
 * 缓存读取也至少要走一次 IPC。取不到作品就留在游戏库（错误已由 store 弹 toast）。
 * 必须把卡片的来源一并带过去：同一个编号在 dlsite / steam 里是两条记录。
 */
async function openWork(item: LibraryWork) {
  await meta.loadById(item.workId, false, item.source as LibrarySource)
  if (meta.work) await router.push('/works')
}

onMounted(() => {
  // 每次进页面都重新读一次：别的页面刚抓完的作品要能立刻看到
  void library.load(true)
})
</script>

<style scoped>
/* ===== 加载骨架屏（2026-11-27 用户要求：数据加载中用 naive 的 Skeleton）=====
   与真实封面同比例（2:3）、同约宽，加载完成后卡片在原地替换它，不跳布局。 */
.lib-skeleton {
  width: 100%;
  aspect-ratio: 2 / 3;
}

.lib-skeleton :deep(.n-skeleton) {
  height: 100%;
  border-radius: 0;
}

/* ===== 封面卡片（仿 Steam 库卡片，2026-11-25）=====
   · hover：上浮 2px + 阴影 + 封面轻微放大 + 底部名称浮层淡入；
   · 点击：从指针落点扩散一圈水纹（.lib-ripple）；
   · 批量模式：未选中压暗去饱和、左上角常驻方形勾选框（见 .lib-batch-mode）。
   位移只用 transform，不参与网格布局 —— 相邻卡片不会被挤走。
   半透明色一律 color-mix（UnoCSS 规则是精确匹配，不能写 bg-secondary/50）。 */
.lib-card {
  border: 1px solid var(--color-border);
  transition:
    border-color var(--motion-fast) var(--ease-enter),
    box-shadow var(--motion-base) var(--ease-enter),
    transform var(--motion-base) var(--ease-enter),
    opacity var(--motion-fast) var(--ease-enter),
    filter var(--motion-fast) var(--ease-enter);
}

.lib-card:hover {
  z-index: 2;
  transform: translateY(-2px);
  border-color: var(--color-border-dark);
  box-shadow: var(--color-shadow-heavy);
}

.lib-card-img {
  transition: transform var(--motion-slow) var(--ease-enter);
  will-change: transform;
}

.lib-card:hover .lib-card-img {
  transform: scale(1.045);
}

/* 名称浮层：平时在封面下沿外、透明，hover / 键盘聚焦时淡入。
   图片上压白字 → 用图片专用 token（两套主题都是白字 + 黑色渐变）。 */
.lib-card-caption {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 3;
  padding: 22px 8px 7px;
  overflow: hidden;
  font-size: 11px;
  line-height: 1.35;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-text-on-image);
  background: linear-gradient(
    to top,
    color-mix(in srgb, var(--color-hero-veil) 92%, transparent),
    transparent
  );
  opacity: 0;
  transform: translateY(8px);
  pointer-events: none;
  transition:
    opacity var(--motion-base) var(--ease-enter),
    transform var(--motion-base) var(--ease-enter);
}

.lib-card:hover .lib-card-caption,
.lib-card:focus-visible .lib-card-caption {
  opacity: 1;
  transform: translateY(0);
}

/* 点击水纹：位置由 JS 写在 left/top（相对卡片），直径 = 卡片对角线 × 2。 */
.lib-ripple {
  position: absolute;
  z-index: 5;
  border-radius: 50%;
  background: radial-gradient(
    circle,
    color-mix(in srgb, var(--color-text-on-image) 46%, transparent) 0%,
    color-mix(in srgb, var(--color-text-on-image) 18%, transparent) 45%,
    transparent 70%
  );
  transform: translate(-50%, -50%) scale(0);
  pointer-events: none;
  animation: lib-ripple-spread 560ms var(--ease-enter) forwards;
}

@keyframes lib-ripple-spread {
  from {
    transform: translate(-50%, -50%) scale(0);
    opacity: 0.9;
  }
  to {
    transform: translate(-50%, -50%) scale(1);
    opacity: 0;
  }
}

/* ===== 批量模式 =====
   卡片样式变化本身就是「现在点封面是选择」的提示：
   未选中的压暗 + 去饱和，hover 恢复；左上角常驻方形勾选框（选中填品牌色）。 */
.lib-batch-mode .lib-card:not(.lib-selected) {
  opacity: 0.6;
  filter: saturate(0.55);
}

.lib-batch-mode .lib-card:not(.lib-selected):hover {
  opacity: 1;
  filter: none;
}

/* 选卡时不上浮、不显示名称浮层：避免点错，也避免名称盖住勾选框 */
.lib-batch-mode .lib-card:hover {
  transform: none;
}

.lib-batch-mode .lib-card-caption {
  display: none;
}

.lib-check {
  position: absolute;
  top: 8px;
  left: 8px;
  z-index: 4;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: 1px solid color-mix(in srgb, var(--color-text-on-image) 70%, transparent);
  border-radius: 4px;
  background: color-mix(in srgb, var(--color-hero-veil) 55%, transparent);
  color: var(--color-text-on-image);
  transition:
    background var(--motion-fast) var(--ease-enter),
    border-color var(--motion-fast) var(--ease-enter);
}

.lib-check--on {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-text-on-brand);
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
  border-radius: 0;
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

/* ===== 添加游戏弹窗（n-modal preset=card）=====
   目标：结果多时只让候选列表滚动、分页固定在卡片底部；弹窗整体不超过 86vh 且垂直居中，
   不必滚动整个窗口才能看到分页。 */
</style>

<!-- n-modal 会 teleport 到 body：这条规则必须非 scoped，否则不一定命中卡片根元素 -->
<style>
.add-game-modal {
  display: flex;
  flex-direction: column;
  max-height: 86vh;
  overflow: hidden;
}

.add-game-modal .n-card-header {
  flex-shrink: 0;
}
</style>
