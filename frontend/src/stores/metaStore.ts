// frontend/src/stores/metaStore.ts
// DLsite 元数据抓取的状态与动作（唯一一份）。
// 调试页（components/test/MetaApiTestCard.vue）现在用它；作品库（views/WorksPage.vue）的
// 路由已摘掉但页面文件与 components/meta/* 保留，等重做浏览入口时直接复用同一个 store：
//   stores/metaStore.ts       状态 + 请求（怎么拿数据）
//   components/meta/*         积木（长什么样）
//   views/*                   排布（放在哪）
// 顺带的好处：页面之间切换时搜索条件与结果都还在，不用重新搜一遍。

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { metaApi } from '../services/metaService'
import type { DownloadResult, SearchCandidate, WorkMeta } from '../services/metaService'
import { libraryApi } from '../services/libraryService'
import type { LibraryMeta } from '../services/libraryService'
import { looksLikeWorkId, toMessage } from '../utils/metaFormat'

/** 快捷示例：省得每次手打编号（编号精确抓取 + 作品名模糊搜索各来一个） */
export const META_SAMPLES = ['RJ01721474', 'RJ01682332', 'エルフ パラダイス']

export const useMetaStore = defineStore('meta', () => {
  // ===== State =====
  const keyword = ref('')
  const loading = ref(false)
  const downloading = ref(false)
  const error = ref('')
  const status = ref('')
  const work = ref<WorkMeta | null>(null)
  const saved = ref<DownloadResult | null>(null)
  const preImgPath = ref('')
  /** 搜索候选的缩略图（编号 → data URL）：来自 workdir 的 cache/，没缓存过才下载 */
  const candidateCovers = ref<Record<string, string>>({})
  /** 已取回的候选：**DLsite 页号 → 该页 30 条**（翻到哪页才取哪页） */
  const sourcePages = ref<Record<number, SearchCandidate[]>>({})
  /** DLsite 搜索的总页数（后端从分页块解析） */
  const sourcePageCount = ref(1)
  /** 搜索结果总数（后端从 data-search-result-count 解析，精确值） */
  const searchTotal = ref(0)
  /** 正在取某一页（同一时间只发一个请求） */
  const pageLoading = ref(false)
  /** 前端分页：每页 10 个，只给当前页下预览图（省网络，见 setCandidatePage） */
  const candidatePage = ref(1)

  /** 当前作品是不是从本地库读出来的（没联网）。详情卡据此显示「缓存」标记 */
  const cached = ref(false)
  /** 当前作品在库里的入库 / 更新时间（空 = 不在库里） */
  const libraryUpdatedAt = ref('')
  /** 库里已有这个作品的封面时，后端注入的封面绝对路径（空 = 还没入过库/没下过封面） */
  const coverAbsPath = ref('')

  /** 后端返回的完整结构（原样展示，调试页用） */
  const raw = ref<unknown>(null)
  /** preImg/ 绝对路径是否已读过（只在第一次进页面时问一次后端） */
  const preImgLoaded = ref(false)

  // ===== Getters =====
  const rawJson = computed(() => (raw.value === null ? '' : JSON.stringify(raw.value, null, 2)))
  /** 已加载的候选（按 DLsite 页序拼起来；调试页展示用） */
  const candidates = computed(() =>
    Object.keys(sourcePages.value)
      .map(Number)
      .sort((a, b) => a - b)
      .flatMap((page) => sourcePages.value[page]),
  )
  const hasResult = computed(() => work.value !== null || candidates.value.length > 0)
  /**
   * 「入库」是否已完成：库里已经有这个作品的封面文件（打开页面时后端经 coverAbsPath 告知），
   * 或本次会话刚点过入库（saved）。按钮据此显示「已入库」并禁用 ——
   * 下载实现刻意不覆盖同名文件（重名加 -2/-3 后缀），再点一下只会多出一份重复的图。
   */
  const stored = computed(() => Boolean(saved.value) || Boolean(coverAbsPath.value))

  /** 前端搜索每页条数 */
  const CANDIDATE_PAGE_SIZE = 10
  /** DLsite 搜索每页条数（服务端固定 30，改 per_page 无效） */
  const SOURCE_PAGE_SIZE = 30
  /** 1 个 DLsite 页 = 3 个前端页（30 / 10，整除，所以前端页永远不会跨 DLsite 页） */
  const FRONT_PAGES_PER_SOURCE = SOURCE_PAGE_SIZE / CANDIDATE_PAGE_SIZE

  /** 结果总数：后端解析出的精确值；拿不到时退回已加载条数（保证分页条还能用） */
  const candidateTotal = computed(() =>
    searchTotal.value > 0 ? searchTotal.value : candidates.value.length,
  )
  /** 页码上限（至少 1 页，空列表也显示"第 1/1 页"） */
  const candidatePageCount = computed(() =>
    Math.max(1, Math.ceil(candidateTotal.value / CANDIDATE_PAGE_SIZE)),
  )
  /** 当前页的候选（列表只渲染这一页） */
  const pagedCandidates = computed(() => {
    const sourcePage = Math.ceil(candidatePage.value / FRONT_PAGES_PER_SOURCE)
    const offset = ((candidatePage.value - 1) % FRONT_PAGES_PER_SOURCE) * CANDIDATE_PAGE_SIZE
    return (sourcePages.value[sourcePage] ?? []).slice(offset, offset + CANDIDATE_PAGE_SIZE)
  })

  // ===== Actions =====

  /**
   * 清空上一次的结果。
   * keepCandidates = 保留候选列表：作品库里是「左边详情 / 右边候选栏」，
   * 点候选换作品时列表得留着，否则一点就空了，没法连着看下一部。
   */
  function clearResult(opts: { keepCandidates?: boolean } = {}) {
    work.value = null
    saved.value = null
    error.value = ''
    status.value = ''
    raw.value = null
    cached.value = false
    libraryUpdatedAt.value = ''
    coverAbsPath.value = ''
    if (!opts.keepCandidates) {
      sourcePages.value = {}
      sourcePageCount.value = 1
      searchTotal.value = 0
      candidatePage.value = 1
    }
  }

  /**
   * 搜索：关键词含 RJ/RE/VJ 编号走精确抓取（**缓存优先**，库里有的不打网络），
   * 否则走模糊搜索（一定联网 —— 搜索本身就是问 DLsite 有哪些作品，没有缓存的意义）。
   * refresh = true 时跳过本地库强制联网（调试页用，要核对后端解析结果）。
   */
  async function runSearch(query?: string, refresh = false) {
    const q = (query ?? keyword.value).trim()
    if (!q) {
      error.value = '请输入作品名或 RJ 编号'
      return
    }
    keyword.value = q
    searchToken += 1
    clearResult()
    loading.value = true
    try {
      if (looksLikeWorkId(q)) {
        const data = await libraryApi.fetch(q, '', refresh)
        applyWork(data)
        status.value = cached.value
          ? `按编号 ${data.id} 从本地库读取（缓存）`
          : `按编号 ${data.id} 精确抓取成功（还没入库，点「入库」才收进游戏库）`
        return
      }
      const resp = await metaApi.searchWorks(q, 1)
      raw.value = resp
      status.value = resp.message ?? ''
      sourcePages.value = { 1: resp.results ?? [] }
      sourcePageCount.value = Math.max(1, resp.pageCount ?? 1)
      searchTotal.value = resp.total ?? 0
      candidatePage.value = 1
      // 只给当前页（第一页）下预览图
      void loadCandidateCovers(pagedCandidates.value)
      // 模糊搜索命中单个作品时后端也会带上 work
      if (resp.work) applyWork(resp.work)
    } catch (err) {
      error.value = toMessage(err)
    } finally {
      loading.value = false
    }
  }

  /** 把取回来的作品塞进 state（同时记住"这是不是缓存"） */
  function applyWork(data: WorkMeta) {
    const meta = data as LibraryMeta
    work.value = data
    raw.value = data
    cached.value = Boolean(meta.cached)
    libraryUpdatedAt.value = meta.libraryUpdatedAt ?? ''
    coverAbsPath.value = meta.coverAbsPath ?? ''
    // 命中的作品从候选里剔除（已加载的每一页都要剔，否则翻页还会看到）
    if (work.value) {
      const id = work.value.id
      const next: Record<number, SearchCandidate[]> = {}
      for (const [page, list] of Object.entries(sourcePages.value)) {
        next[Number(page)] = list.filter((c) => c.id !== id)
      }
      sourcePages.value = next
    }
  }

  /**
   * 点候选的「详情」：按编号精确抓（保留候选列表，方便连着换）。
   * refresh = true 时跳过本地库，强制联网重新抓一份（并覆盖库里的旧数据）。
   * **不写回 keyword**：搜索框只反映用户自己输入过的内容。打开一个作品（尤其从游戏库
   * 点封面进来）不该把 RJ 编号灌进搜索框，否则下次弹搜索窗会莫名带着一串编号。
   */
  async function loadById(id: string, refresh = false) {
    clearResult({ keepCandidates: true })
    loading.value = true
    try {
      const data = await libraryApi.fetch('', id, refresh)
      applyWork(data)
      status.value = cached.value
        ? `按编号 ${id} 从本地库读取（缓存）`
        : `按编号 ${id} 精确抓取成功（还没入库，点「入库」才收进游戏库）`
    } catch (err) {
      error.value = toMessage(err)
    } finally {
      loading.value = false
    }
  }

  /**
   * 载入候选缩略图：后端把图缓存到 workdir 的 cache/（目录不存在会建，已缓存则不联网），
   * 返回 data URL。失败就留空 —— 列表会退回 DLsite 直链，加载不出来时显示占位图。
   */
  async function loadCandidateCovers(list: SearchCandidate[]) {
    const pending = list.filter((c) => c.cover && !candidateCovers.value[c.id])
    if (!pending.length) return
    const entries = await Promise.all(
      pending.map(async (c) => {
        try {
          return [c.id, await metaApi.cacheCover(c.id, c.cover ?? '')] as const
        } catch {
          return [c.id, ''] as const
        }
      }),
    )
    const next = { ...candidateCovers.value }
    for (const [id, url] of entries) if (url) next[id] = url
    candidateCovers.value = next
  }

  /** 正在请求中的 DLsite 页（同一页并发点两次也只发一个请求） */
  const inflightPages = new Map<number, Promise<boolean>>()
  /** 搜索代次：换关键词后，旧的预取/翻页响应不能写进新的候选里 */
  let searchToken = 0

  /**
   * 取 DLsite 的某一页候选（已取过直接返回）。silent = 预取，不把错误弹给用户。
   * 只有用户翻到那一页（或翻到一组的最后一页触发预取）时才调用。
   */
  function loadSourcePage(page: number, silent = false): Promise<boolean> {
    if (sourcePages.value[page]) return Promise.resolve(true)
    const running = inflightPages.get(page)
    if (running) return running
    const token = searchToken

    const task = (async () => {
      try {
        const resp = await metaApi.searchWorks(keyword.value, page)
        // 期间换过关键词：丢弃这次结果，别把旧关键词的候选塞进新列表
        if (token !== searchToken) return false
        sourcePages.value = { ...sourcePages.value, [page]: resp.results ?? [] }
        if (resp.pageCount && resp.pageCount > 0) sourcePageCount.value = resp.pageCount
        if (resp.total && resp.total > 0) searchTotal.value = resp.total
        return true
      } catch (err) {
        if (token === searchToken && !silent) error.value = toMessage(err)
        return false
      }
    })()

    inflightPages.set(page, task)
    void task.finally(() => inflightPages.delete(page))
    return task
  }

  /**
   * 当前前端页对应的 DLsite 页没取过就先取，再给这一页下预览图。
   * 因为 30 / 10 = 3，「第 1~3 页 = DLsite 第 1 面」「第 4~6 页 = 第 2 面」……依此类推；
   * 翻到一组的最后一页（第 3、6、9…页）时顺手**预取下一面**，翻过去就不用等。
   */
  async function ensureCandidatesForCurrentPage() {
    const sourcePage = Math.ceil(candidatePage.value / FRONT_PAGES_PER_SOURCE)
    if (!sourcePages.value[sourcePage]) {
      pageLoading.value = true
      try {
        const ok = await loadSourcePage(sourcePage)
        if (!ok) return
        // 最后一页不满 30 条时总数会变小，把当前页收回到新的最后一页
        if (candidatePage.value > candidatePageCount.value) candidatePage.value = candidatePageCount.value
      } finally {
        pageLoading.value = false
      }
    }
    void loadCandidateCovers(pagedCandidates.value)

    // 预取下一面（只在组末页触发；失败不打扰用户，翻过去时还会再试一次）
    if (candidatePage.value % FRONT_PAGES_PER_SOURCE === 0) {
      const next = sourcePage + 1
      if (next <= sourcePageCount.value) void loadSourcePage(next, true)
    }
  }

  /** 翻页：需要时才向后端要那一页，且只给当前页下预览图（缓存过的直接命中 cache/） */
  async function setCandidatePage(page: number) {
    const next = Math.min(Math.max(1, Math.floor(page)), candidatePageCount.value)
    if (next === candidatePage.value) return
    candidatePage.value = next
    await ensureCandidatesForCurrentPage()
  }

  /** 点示例：填进输入框再搜一次（调试页传 refresh=true 走真联网） */
  function searchBySample(sample: string, refresh = false) {
    void runSearch(sample, refresh)
  }

  /**
   * 入库（作品库 /works 与调试页的「入库」按钮）：**只有这一步会把作品收进游戏库** ——
   * 后端抓文本元数据、把封面原样落到 preImg/ 并记账，返回带 cover_* 的完整数据。
   * 搜索、打开详情都只读不写，所以游戏库里只会出现点过入库的作品。
   */
  async function storeIn() {
    const current = work.value
    if (!current || stored.value) return
    downloading.value = true
    error.value = ''
    try {
      const data = await libraryApi.storeIn(current.id)
      applyWork(data)
      saved.value = {
        id: data.id,
        dir: data.coverDir ?? '',
        path: data.coverPath ?? '',
        absPath: data.coverAbsPath ?? '',
        bytes: data.coverBytes ?? 0,
        suffix: data.coverSuffix ?? '',
        coverUrl: data.coverUrl ?? '',
      }
      if (data.coverAbsPath) coverAbsPath.value = data.coverAbsPath
      status.value = `已入库：${data.id}（元数据已载入数据库，封面存到 preImg/）`
    } catch (err) {
      error.value = `入库失败：${toMessage(err)}`
    } finally {
      downloading.value = false
    }
  }

  /** 读取 preImg/ 的绝对路径（失败就留空，不打扰用户） */
  async function refreshPreImgDir() {
    try {
      preImgPath.value = await metaApi.preImgDir()
    } catch {
      preImgPath.value = ''
    }
    preImgLoaded.value = true
  }

  /** 进页面时调：只问一次后端，之后靠「刷新」按钮手动更新 */
  function ensurePreImgDir() {
    if (!preImgLoaded.value) void refreshPreImgDir()
  }

  return {
    // State
    keyword,
    loading,
    downloading,
    error,
    status,
    work,
    saved,
    preImgPath,
    candidateCovers,
    candidatePage,
    pageLoading,
    raw,
    cached,
    libraryUpdatedAt,
    coverAbsPath,
    // Getters
    rawJson,
    hasResult,
    stored,
    candidates,
    candidateTotal,
    candidatePageCount,
    pagedCandidates,
    // Actions
    runSearch,
    loadById,
    searchBySample,
    storeIn,
    setCandidatePage,
    refreshPreImgDir,
    ensurePreImgDir,
    clearResult,
  }
})
