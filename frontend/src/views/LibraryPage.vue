<!-- frontend/src/views/LibraryPage.vue
     游戏库（/library）：把"抓过的作品"当成一个可浏览的收藏夹。
     交互（用户明确要求）：
       - 封面横向排列成网格；
       - 名称平时不占地方，鼠标悬停时用半透明面板从下往上淡入；
       - 点封面进入「作品库」详情页（/works，复用原页面，点谁就加载谁）；
       - 每个封面右上角一个删除按钮（连点两下确认，与 TODO 的删除同套路）。
     数据全来自本地库（stores/libraryStore.ts → libraryApi.list），列表本身不发网络请求，
     断网也能看；封面是后端读文件转成的 base64 data URL（WebView 里一定显示得出来）。 -->
<template>
  <div class="w-full max-w-5xl mx-auto h-full min-h-0 flex flex-col gap-4 select-text">
    <!-- ── 顶部：标题 + 已入库数量 + 搜索入口 + 刷新 ───────────────
         只有这一行（没有内容区），MetaSection 会把上下内边距收到 header 上，
         卡片高度就由 35.2px 的按钮决定，标题/数量/按钮靠 items-center 垂直居中。 -->
    <MetaSection icon="i-lucide-gamepad-2" title="游戏库">
      <template #actions>
        <!-- 已入库数量放在标题栏右侧，和按钮同一行居中 -->
        <span class="text-xs text-tertiary whitespace-nowrap">
          已入库 <span class="text-primary font-medium tabular-nums">{{ library.count }}</span> 款
        </span>
        <!-- 刷新在左、搜索在右（用户要求调换位置） -->
        <button
          class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none disabled:opacity-50"
          :disabled="library.loading"
          @click="library.load(true)"
        >
          <span class="i-lucide-refresh-cw icon-xs" aria-hidden="true" />
          {{ library.loading ? '刷新中…' : '刷新' }}
        </button>
        <!-- 搜索入口：作品库页面已没有搜索卡，统一从这里弹出居中弹窗 -->
        <button
          class="btn-brand flex items-center gap-1.5"
          :disabled="meta.loading"
          @click="openSearch"
        >
          <span class="i-lucide-search icon-sm" aria-hidden="true" />
          搜索
        </button>
      </template>
    </MetaSection>

    <MetaMessage v-if="library.error" tone="error" :text="library.error" />

    <!-- ── 网格：自己开滚动（MainLayout 的 router-view 是 overflow-hidden） ── -->
    <div class="flex-1 min-h-0 overflow-y-auto scrollbar-theme pr-2">
      <div v-if="library.works.length" class="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-3">
        <!-- 一张卡片 = 封面按钮 + 右上角删除按钮（同级，不嵌套：button 里不能再放 button） -->
        <div v-for="item in library.works" :key="item.source + item.workId" class="group relative">
          <button
            type="button"
            class="relative block w-full overflow-hidden rounded-lg border-theme bg-secondary cursor-pointer p-0"
            :aria-label="`打开「${item.name}」的作品详情`"
            @click="openWork(item.workId)"
          >
            <span class="block w-full aspect-[4/3]">
              <img
                v-if="library.covers[item.workId]"
                :src="library.covers[item.workId]"
                :alt="item.name"
                class="w-full h-full object-cover"
                draggable="false"
              />
              <span v-else class="flex-center w-full h-full text-muted">
                <span class="i-lucide-image-off icon-lg" aria-hidden="true" />
              </span>
            </span>

            <!-- 名称：平时藏在封面下沿之外，hover / 键盘聚焦时半透明面板向上淡入 -->
            <span class="lib-name translate-y-full opacity-0 transition-[transform,opacity] duration-200 ease-out group-hover:translate-y-0 group-hover:opacity-100 group-focus-within:translate-y-0 group-focus-within:opacity-100" aria-hidden="true">
              {{ item.name }}
            </span>
          </button>

          <!-- 右上角删除：连点两下确认（armed 态变红 + 勾），
               鼠标移开或按钮失焦就撤销待确认态 —— 与 TodoItem 的删除按钮同一套逻辑。
               外观也照搬 TodoItem：btn-row 尺寸 + 静息 75% 不透明 + hover/聚焦提亮，
               颜色状态（红描边 / hover 填红 / armed 填红 + 轻微放大）见下方 .lib-del。 -->
          <button
            type="button"
            class="lib-del btn-row absolute top-1.5 right-1.5 opacity-75 group-hover:opacity-100 group-focus-within:opacity-100 hover:opacity-100 focus-visible:opacity-100"
            :class="{ 'is-armed opacity-100 scale-110': armedId === item.workId }"
            :disabled="library.removingId === item.workId"
            :title="deleteTitle(item.workId)"
            :aria-label="deleteTitle(item.workId)"
            @click.stop="handleRemove(item.workId)"
            @mouseleave="cancelArm(item.workId)"
            @blur="cancelArm(item.workId)"
          >
            <span
              :class="
                library.removingId === item.workId
                  ? 'i-lucide-loader-circle spin'
                  : armedId === item.workId
                    ? 'i-lucide-check'
                    : 'i-lucide-trash-2'
              "
              class="icon-xs"
              aria-hidden="true"
            />
          </button>
        </div>
      </div>

      <!-- 空态 / 首次加载 -->
      <MetaEmptyState
        v-else-if="!library.loading"
        icon="i-lucide-gamepad-2"
        title="游戏库还是空的"
        hint="点上方「搜索」，输入 RJ 编号或作品名；搜到后点开详情、再点「入库」才会收进这里。"
      />
      <p v-else class="m-0 text-sm text-tertiary py-6 text-center">正在读取游戏库…</p>
    </div>

    <!-- ── 搜索弹窗：居中，周围模糊 ─────────────────────────────────
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
          aria-label="搜索作品"
          @keydown.esc="closeSearch"
        >
          <div class="lib-dialog card-flat p-5 w-[min(560px,92vw)] max-h-[80vh] flex flex-col gap-4">
            <header class="flex items-center gap-2.5">
              <span class="i-lucide-search icon-md section-icon" aria-hidden="true" />
              <h2 class="section-title">搜索作品</h2>
              <button
                type="button"
                class="ml-auto w-7 h-7 rounded-md flex-center bg-transparent border-none text-tertiary cursor-pointer hover:text-primary hover:bg-hover transition-colors duration-200"
                aria-label="关闭搜索"
                @click="closeSearch"
              >
                <span class="i-lucide-x icon-sm" aria-hidden="true" />
              </button>
            </header>

            <p class="m-0 text-xs text-tertiary">
              输入 RJ 编号精确抓取（如 RJ01721474），或输入作品名做模糊搜索；搜到的作品<b class="font-medium text-secondary">不会自动入库</b>，点开详情后再点「入库」。
            </p>

            <MetaSearchBar v-model="meta.keyword" :loading="meta.loading" @submit="submitSearch" />
            <MetaMessage v-if="meta.error" tone="error" :text="meta.error" />
            <MetaMessage v-if="meta.status" :text="meta.status" />
            <MetaMessage v-if="meta.pageLoading" :text="`正在加载第 ${meta.candidatePage} 页…`" />

            <!-- 模糊搜索的候选：每页 10 个，翻到哪页才取那一页的 30 条里的 10 条并下预览图 -->
            <template v-if="meta.candidateTotal > 0">
              <div class="min-h-0 overflow-y-auto scrollbar-theme -mx-5 border-t-theme-light">
                <MetaCandidateList
                  :candidates="meta.pagedCandidates"
                  :covers="meta.candidateCovers"
                  @select="chooseCandidate"
                />
              </div>

              <!-- 分页：前 3 页 / 当前页 / 后 3 页可点，右侧输入页码直接跳转 -->
              <MetaPager
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
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMetaStore } from '../stores/metaStore'
import { useLibraryStore } from '../stores/libraryStore'
import MetaCandidateList from '../components/meta/MetaCandidateList.vue'
import MetaEmptyState from '../components/meta/MetaEmptyState.vue'
import MetaMessage from '../components/meta/MetaMessage.vue'
import MetaPager from '../components/meta/MetaPager.vue'
import MetaSearchBar from '../components/meta/MetaSearchBar.vue'
import MetaSection from '../components/meta/MetaSection.vue'

const library = useLibraryStore()
const meta = useMetaStore()
const router = useRouter()

/** 删除是破坏性操作，第一次点只"上膛"，再点一次才真删（与 TODO 删除同一套路） */
const armedId = ref('')
/** 搜索弹窗开关 */
const searchOpen = ref(false)

function deleteTitle(workId: string): string {
  if (library.removingId === workId) return '删除中…'
  return armedId.value === workId ? '再点一次删除' : '从游戏库删除（连点两下）'
}

/** 移开鼠标 / 按钮失焦就撤销"再点一次"的待确认态（TodoItem 同款逻辑） */
function cancelArm(workId: string) {
  if (armedId.value === workId) armedId.value = ''
}

async function handleRemove(workId: string) {
  if (armedId.value !== workId) {
    armedId.value = workId
    return
  }
  armedId.value = ''
  await library.remove(workId)
}

/**
 * 打开搜索弹窗：清掉上一次的错误/状态，并**清空输入框**。
 * 搜索成功后 `runSearch` 会把关键词（往往是 RJ 编号）留在 store 里，弹窗一开就会
 * 在输入框里显示出来 —— 用户明确不要"上次搜过的编号"自动出现，所以每次开窗都从空白开始。
 */
function openSearch() {
  meta.error = ''
  meta.status = ''
  meta.keyword = ''
  searchOpen.value = true
}

function closeSearch() {
  searchOpen.value = false
}

/**
 * 提交搜索：命中作品（按编号精准抓）就直接进作品库；
 * 只是候选列表（模糊搜索）就留在弹窗里让用户点「详情」。
 */
async function submitSearch() {
  await meta.runSearch()
  if (meta.work) finishSearch()
}

/** 点候选「详情」：精确抓取 → 进作品库 */
async function chooseCandidate(id: string) {
  await meta.loadById(id)
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
 */
async function openWork(workId: string) {
  await router.push('/works')
  await meta.loadById(workId)
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

/* ===== 右上角删除按钮 =====
   ⚠️ 状态色必须与 components/todo/TodoItem.vue 的 .row-btn-delete 保持一致
   （红描边 → hover 填红 → armed 填红 + 放大、文字色随主题翻转）：
   两个删除按钮是同一层级的破坏性操作，用户可以一眼认出"这是删除"。
   改这里要同步改 TodoItem；形状（w-7/h-7/圆角/边框宽度）由 btn-row 提供，这里只管颜色。
   状态色全部写在这里（不在模板里叠 bg-… / text-… 类）：
   UnoCSS 对同一属性只保留一条规则，模板与 scoped 样式混着改颜色容易互相压掉。 */
.lib-del {
  background-color: transparent;
  border-color: var(--color-danger);
  color: var(--color-danger);
  transition:
    background-color var(--motion-fast) var(--ease-enter),
    border-color var(--motion-fast) var(--ease-enter),
    color var(--motion-fast) var(--ease-enter),
    opacity var(--motion-fast) var(--ease-enter),
    transform var(--motion-fast) var(--ease-enter);
}

/* hover 才填红（静息只描边） */
.lib-del:hover:not(:disabled) {
  background-color: var(--color-danger);
  border-color: var(--color-danger);
  color: var(--color-text-on-danger);
}

/* 待确认：实心红（与 hover 同色，靠放大 + 勾图标区分）+ 随主题翻转的深色勾 */
.lib-del.is-armed {
  background-color: var(--color-danger);
  border-color: var(--color-danger);
  color: var(--color-text-on-danger);
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
