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
import type { DownloadResult, SearchCandidate, SteamStats, WorkMeta } from '../services/metaService'
import { LIBRARY_SOURCE_DLSITE, LIBRARY_SOURCE_STEAM, libraryApi } from '../services/libraryService'
import type { LibraryMeta, LibrarySource } from '../services/libraryService'
import { gameApi } from '../services/gameService'
import type { LibraryLaunchMode } from '../services/gameService'
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
  const coversPath = ref('')
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
  /**
   * 当前作品来自哪个源（dlsite / steam）。
   * 入库、启动、标签、封面都要按它取数据 —— 同一个编号在不同来源里是两条库记录，
   * 所以从游戏库点进 /works 时必须把来源带过来（loadById 的第三个参数）。
   */
  const source = ref<LibrarySource>(LIBRARY_SOURCE_DLSITE)
  /** 当前作品在库里的入库 / 更新时间（空 = 不在库里） */
  const libraryUpdatedAt = ref('')
  /** 库里已有这个作品的封面时，后端注入的封面绝对路径（空 = 还没入过库/没下过封面） */
  const coverAbsPath = ref('')
  /**
   * 库里记录的游戏程序绝对路径（空 = 还没设置）。
   * 位置就是 library_works 上的 game_path 列，随 Fetch/Store 的返回一起带回来；
   * 入库后「游戏启动」按钮按它启动，为空时第一次点击会弹系统文件框让用户选 exe。
   */
  const gamePath = ref('')
  /**
   * 库里记录的启动方式（2026-11-23）：'' = 没选过（按来源取默认），
   * 'steam' = steam://rungameid/<appid>，'exe' = 本地程序。
   * 与 gamePath 同表（library_works.launch_mode），随 Fetch/Store 带回来。
   */
  const launchMode = ref<LibraryLaunchMode>('')
  /** 记录的 exe 文件此刻是否还在（进入详情时由后端检测；false = 已删除/变更位置） */
  const gamePathExists = ref(true)
  /** 游戏此刻是否正在运行（进入详情时检测 + 页面定时轮询） */
  const gameRunning = ref(false)
  /** 正在查询游戏状态（防轮询重叠） */
  const gameStatusLoading = ref(false)
  /** exe 不存在时的提示文案（用户要求：提示游戏已删除或变更位置） */
  const gameWarning = ref('')
  /** 正在启动游戏（按钮 loading + 防连点） */
  const launching = ref(false)
  /**
   * 当前作品的标签列表（**编辑稿**）：抓取来的 tags 与用户自己加的标签合成一份，
   * 删/增都只改这里，点「保存」才整体写进后端 work_tags 表。
   */
  const tagList = ref<string[]>([])
  /** 上一次写入数据库的那份（或刚载入时的初始列表）；与 tagList 不同 = 有未保存改动 */
  const tagSaved = ref<string[]>([])
  /** 库里存的是不是用户整理过的整份列表（false = 只是旧版逐条加的标签） */
  const tagsEdited = ref(false)
  const tagsSaving = ref(false)
  const tagError = ref('')

  /** 搜索/入库结果的附加提示（某个来源失败但整体还有结果、或需要用户注意的说明） */
  const searchNotice = ref('')
  /** Steam 搜索候选（storesearch 最多 10 条，没有分页） */
  const steamCandidates = ref<SearchCandidate[]>([])
  const steamSearchTotal = ref(0)
  /** /works「Steam 动态」：评价汇总 + 在线人数 + 最近公告（只有 Steam 源才有） */
  const steamStats = ref<SteamStats | null>(null)
  const steamStatsLoading = ref(false)
  /**
   * Steam 源作品的 hero 背景图（data URL）：后端在本机 Steam 客户端缓存
   * appcache\librarycache 里按 appid 找文件名含 hero 的图片并读成 base64。
   * 空串 = 本机没有（游戏没装 / 缓存被清），详情页回落到商店接口的预览图。
   */
  const heroImage = ref('')

  /** 后端返回的完整结构（原样展示，调试页用） */
  const raw = ref<unknown>(null)
  /** covers/ 绝对路径是否已读过（只在第一次进页面时问一次后端） */
  const coversLoaded = ref(false)

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
   * 是否已入库：库里已经有这个作品的封面文件（打开页面时后端经 coverAbsPath 告知），
   * 或本次会话刚点过入库（saved），或后端带了库里这条记录的更新时间（libraryUpdatedAt）。
   * 按钮据此从「入库」切换成「游戏启动」（不再有禁用态）。
   * 判定刻意不用"在游戏库里"：元数据随抓取就可能已有缓存行，按此判定按钮永远不可点。
   *
   * 为什么要带 libraryUpdatedAt：Steam 游戏可能本机没有 librarycache 封面（用户要求
   * 没有就不使用图像），此时 coverAbsPath 为空，只看 coverAbsPath 会误判成"没入库"。
   */
  const stored = computed(
    () => Boolean(saved.value) || Boolean(coverAbsPath.value) || Boolean(libraryUpdatedAt.value),
  )
  /**
   * 生效的启动方式（2026-11-23）：
   *   - 用户显式选过（库里 launch_mode 非空）就用它；
   *   - 否则按来源取默认 —— Steam 源走 steam://rungameid/<appid>（用户要求：默认 URL），
   *     DLsite 源只能跑本地 exe。
   */
  const effectiveLaunchMode = computed<Exclude<LibraryLaunchMode, ''>>(() => {
    if (launchMode.value === 'steam' || launchMode.value === 'exe') return launchMode.value
    return source.value === LIBRARY_SOURCE_STEAM ? 'steam' : 'exe'
  })
  /** 当前是否「通过 Steam 启动」（决定按钮提示、是否还需要本地 exe） */
  /**
   * 启动按钮是否该禁用：exe 方式下已经记了位置、但 exe 文件不存在了（被删除或移走）。
   * Steam 方式不需要本地 exe（游戏装没装、装在哪由 Steam 客户端管），所以不禁用。
   */
  const launchBlocked = computed(
    () =>
      stored.value &&
      effectiveLaunchMode.value === 'exe' &&
      Boolean(gamePath.value) &&
      !gamePathExists.value,
  )
  /**
   * 标签是否有未保存的改动（决定「保存」按钮显不显示）。
   * 按**完整列表**比较而不是集合：调换顺序也算改动（顺序就是保存顺序）。
   */
  const tagsDirty = computed(() => {
    if (tagList.value.length !== tagSaved.value.length) return true
    return tagList.value.some((tag, i) => tag !== tagSaved.value[i])
  })

  /**
   * 详情页 hero 区域的大背景图（用户 2026-11-23 定稿）：
   *   - Steam 源：优先本机扫描到的 hero 图（后端转 data URL）；没有就用商店接口的
   *     预览图（background_raw / background / 第一张截图），最后才退到封面；
   *   - DLsite 源：用画廊的**第二张**预览图（第一张是游戏库卡片封面，见
   *     getFromDlsite.go 的 previewImages）。
   * 旧缓存（2026-11-23 之前入库的）没有预览图列表，从封面 URL 推一张 `_img_smp1`
   * 兜底 —— 推出来的图不一定存在，<img> 加载失败时 hero 会退成纯渐变底。
   */
  const backgroundUrl = computed(() => {
    const current = work.value
    if (!current) return ''
    const previews = Array.isArray(current.previewImages) ? current.previewImages : []

    if (source.value === LIBRARY_SOURCE_STEAM) {
      if (heroImage.value) return heroImage.value
      const steam = (current.steam ?? {}) as Record<string, unknown>
      const screenshots = Array.isArray(steam.screenshots)
        ? (steam.screenshots as Array<Record<string, unknown>>)
        : []
      // 用户要求「没有本地 hero 就使用 api 获取的预览图填充」：预览图 = 商店截图，
      // 比 page_bg_raw（Steam 页头背景，很多游戏是一张很暗的纹理）更像"游戏画面"。
      // background* 放在截图之后兜底，最后才是 16:9 header 封面。
      const url = firstImageUrl([
        ...screenshots.map((shot) => shot.path_full),
        steam.background_raw,
        steam.background,
        ...previews,
      ])
      return url || current.coverUrl || ''
    }

    // DLsite：第二张预览图作背景（用户要求）；只有一张预览图时就用它
    if (previews.length > 1) return previews[1]
    if (previews.length === 1) return previews[0]
    return deriveDlsitePreviewUrl(current.coverUrl) || current.coverUrl || ''
  })

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
    gamePath.value = ''
    launchMode.value = ''
    gamePathExists.value = true
    gameRunning.value = false
    gameStatusLoading.value = false
    gameWarning.value = ''
    launching.value = false
    tagList.value = []
    tagSaved.value = []
    tagsEdited.value = false
    tagsSaving.value = false
    tagError.value = ''
    searchNotice.value = ''
    steamCandidates.value = []
    steamSearchTotal.value = 0
    steamStats.value = null
    steamStatsLoading.value = false
    heroImage.value = ''
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
   *
   * ⚠️ **返回"这次是否真的发起了搜索"**：空关键词只提示、不搜索（返回 false）。
   * 调用方不能只看 `work` 判断要不要跳详情页 —— store 里还留着上一次的 work，
   * 空关键词点「搜索」会拿旧结果跳详情（用户报的 bug），必须靠这个布尔值把关。
   */
  async function runSearch(query?: string, refresh = false): Promise<boolean> {
    const q = (query ?? keyword.value).trim()
    if (!q) {
      // 用户要求：搜索栏为空时点「搜索」只提示「搜索栏不能为空」，不做任何跳转
      error.value = '搜索栏不能为空'
      return false
    }
    keyword.value = q
    // 这里只跑 DLsite 分支（调试页 /testing 用它核对 DLsite 解析结果）；
    // 游戏库「手动搜索」要两个来源一起看，走 runMultiSearch。
    source.value = LIBRARY_SOURCE_DLSITE
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
        return true
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
      return true
    } catch (err) {
      error.value = toMessage(err)
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * 手动搜索（游戏库「添加游戏 → 手动搜索」）：**一次同时查 DLsite 与 Steam**。
   *
   * 用户 2026-11-23 要求「手动搜索只显示 DLsite 数据源，应该也显示 steam 数据源」，
   * 所以不再让用户先选来源再搜，而是把两边的候选并列出来：
   *   - DLsite：名字模糊搜（每页 30 条，前端 10 条一页，可翻页）；
   *   - Steam：storesearch（最多 10 条、没有分页）。
   * 两路各取各的，一路失败不影响另一路（失败原因写进 searchNotice 显示为警告）。
   * 编号输入（RJ/RE/VJ）仍然只走 DLsite 精确抓取 —— 它本来就不是"搜索"。
   *
   * 返回值语义与 runSearch 一致：true = 这次真的发了请求（空关键词只提示、不搜索）。
   */
  async function runMultiSearch(query?: string): Promise<boolean> {
    const q = (query ?? keyword.value).trim()
    if (!q) {
      error.value = '搜索栏不能为空'
      return false
    }
    keyword.value = q
    source.value = LIBRARY_SOURCE_DLSITE
    const token = ++searchToken
    clearResult()
    loading.value = true
    const notices: string[] = []
    try {
      // 编号：DLsite 精确抓详情（Steam 没有这种编号，不用搜）
      if (looksLikeWorkId(q)) {
        const data = await libraryApi.fetch(q, '', false)
        applyWork(data)
        status.value = cached.value
          ? `按编号 ${data.id} 从本地库读取（缓存）`
          : `按编号 ${data.id} 精确抓取成功（还没入库，点「入库」才收进游戏库）`
        return true
      }

      const [dlsite, steam] = await Promise.allSettled([
        metaApi.searchWorks(q, 1),
        metaApi.searchSteam(q),
      ])
      // 期间又搜了一次：这次的结果整个作废（别把旧关键词的候选塞进新列表）
      if (token !== searchToken) return false

      const parts: string[] = []
      if (dlsite.status === 'fulfilled') {
        const resp = dlsite.value
        raw.value = resp
        sourcePages.value = { 1: resp.results ?? [] }
        sourcePageCount.value = Math.max(1, resp.pageCount ?? 1)
        searchTotal.value = resp.total ?? 0
        candidatePage.value = 1
        // 只给当前页（第一页）下预览图
        void loadCandidateCovers(pagedCandidates.value)
        // 模糊搜索命中单个作品时后端也会带上 work
        if (resp.work) applyWork(resp.work)
        parts.push(`DLsite ${candidateTotal.value} 条`)
      } else {
        notices.push(`DLsite 搜索失败：${toMessage(dlsite.reason)}`)
      }

      if (steam.status === 'fulfilled') {
        steamCandidates.value = steam.value.items ?? []
        steamSearchTotal.value = steam.value.total ?? 0
        parts.push(`Steam ${steamCandidates.value.length} 个候选`)
      } else {
        notices.push(`Steam 搜索失败：${toMessage(steam.reason)}`)
      }

      // 两边都没成功：当成一次失败搜索（error 条会显示具体原因）
      if (!parts.length) {
        error.value = notices.join('；') || '没有搜索到结果'
        return false
      }
      searchNotice.value = notices.join('；')
      status.value = `同时搜索 DLsite 与 Steam：${parts.join(' · ')}；点「详情」核对后再入库`
      return true
    } catch (err) {
      error.value = toMessage(err)
      return false
    } finally {
      // 期间又搜了一次就别动 loading：新请求还在飞
      if (token === searchToken) loading.value = false
    }
  }

  /**
   * 读当前作品的「Steam 动态」：评价汇总 / 在线人数 / 最近公告（后端一次取回）。
   * 只有 Steam 源才有；失败就留空（详情页照常显示，不打断浏览）。
   */
  async function loadSteamStats() {
    const current = work.value
    if (!current || source.value !== LIBRARY_SOURCE_STEAM) {
      steamStats.value = null
      return
    }
    if (steamStatsLoading.value) return
    steamStatsLoading.value = true
    try {
      const stats = await metaApi.steamStats(current.id)
      // 期间换作品了：丢弃旧响应
      if (work.value?.id !== current.id || source.value !== LIBRARY_SOURCE_STEAM) return
      steamStats.value = stats
    } catch {
      steamStats.value = null
    } finally {
      steamStatsLoading.value = false
    }
  }

  /**
   * 读 Steam 源作品的 hero 背景图（本机 Steam 客户端缓存，后端转 data URL）。
   * 只有 Steam 源才有；本机没有就留空 —— 详情页会回落到商店接口的预览图。
   */
  async function loadSteamHeroImage(appID: string) {
    try {
      const url = await libraryApi.steamHeroImage(appID)
      if (!url) return
      // 期间换作品了：丢弃旧响应（否则会把上一部的大图贴到当前作品上）
      if (work.value?.id !== appID || source.value !== LIBRARY_SOURCE_STEAM) return
      heroImage.value = url
    } catch {
      // 背景图是装饰，拿不到不影响详情页
    }
  }

  /** 把取回来的作品塞进 state（同时记住"这是不是缓存"） */
  function applyWork(data: WorkMeta) {
    const meta = data as LibraryMeta
    // 换作品前记一下是谁：同一个作品重新 apply（比如刚点完「入库」）时下面要保住未保存的标签
    const previousId = work.value?.id
    work.value = data
    raw.value = data
    cached.value = Boolean(meta.cached)
    libraryUpdatedAt.value = meta.libraryUpdatedAt ?? ''
    coverAbsPath.value = meta.coverAbsPath ?? ''
    // 游戏启动位置就在库记录上（与元数据同表），随 Fetch/Store 一起带回来。
    // 先按"文件存在/未运行"占位，避免按钮闪一下禁用态；紧接着再问后端真实状态。
    gamePath.value = meta.gamePath ?? ''
    const storedMode = (meta.launchMode ?? '').toLowerCase()
    launchMode.value = storedMode === 'steam' || storedMode === 'exe' ? storedMode : ''
    gamePathExists.value = Boolean(gamePath.value)
    gameRunning.value = false
    gameWarning.value = ''
    // 用户要求：进入相关界面时先检测 exe 是否存在、游戏是否在运行
    void refreshGameStatus()
    // 标签跟着作品走：换作品时重新拉一份（抓取标签 + 库里保存过的整理结果）。
    // 同一个作品重新 apply（例如刚点完「入库」）且手上还有没保存的标签改动时跳过，
    // 否则那点编辑会被服务器状态覆盖掉。
    if (previousId !== data.id || !tagsDirty.value) void loadTags(data.id)
    // Steam 源额外取「评价 / 在线人数 / 公告」（DLsite 分支里直接清空）
    void loadSteamStats()
    // Steam 源额外取本机 hero 背景图（DLsite 的背景图直接从作品的预览图列表取）
    heroImage.value = ''
    if (source.value === LIBRARY_SOURCE_STEAM) void loadSteamHeroImage(data.id)
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
   * fromSource 把「从哪个来源点进来的」带进来（游戏库卡片 / 扫描结果行都知道来源）；
   * 不传就沿用当前来源（例如 /works 里的「重新抓取」按钮）。
   * **不写回 keyword**：搜索框只反映用户自己输入过的内容。打开一个作品（尤其从游戏库
   * 点封面进来）不该把 RJ 编号灌进搜索框，否则下次弹搜索窗会莫名带着一串编号。
   */
  async function loadById(id: string, refresh = false, fromSource?: LibrarySource) {
    if (fromSource) source.value = fromSource
    clearResult({ keepCandidates: true })
    loading.value = true
    try {
      const data = await libraryApi.fetch('', id, refresh, source.value)
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
   * 后端抓文本元数据、把封面（优先 webp）落到 covers/ 并记账，返回带 cover_* 的完整数据。
   * 搜索、打开详情都只读不写，所以游戏库里只会出现点过入库的作品。
   *
   * 手动搜索入库时**先请用户选游戏程序**（用户要求「手动搜索添加则在入库时便向用户请求路径」）：
   *   - 选了 → 入库时一起记录，之后「游戏启动」直接跑；
   *   - 取消 → 照常入库但不记位置，启动时再问（用户要求里的兜底）。
   * 已经记录过位置的（老数据 / 从扫描详情进来）不重复问，沿用原路径。
   */
  async function storeIn() {
    const current = work.value
    if (!current || stored.value || downloading.value) return
    downloading.value = true
    error.value = ''
    try {
      // 手动搜索入库：先请用户选游戏程序；取消则照常入库，只是不记位置。
      // Steam 源例外：默认走 steam://rungameid/<appid>，本机 exe 不是必需品，
      // 入库时不再追问（用户要切「本地程序启动」时在作品页再选）。
      let pickedPath = gamePath.value && gamePathExists.value ? gamePath.value : ''
      if (!pickedPath && source.value !== LIBRARY_SOURCE_STEAM) {
        try {
          pickedPath = await gameApi.pickExecutable(`选择「${current.name}」的游戏程序（取消则启动时再选）`)
        } catch {
          pickedPath = ''
        }
      }
      const data = await libraryApi.storeIn(current.id, { gamePath: pickedPath }, source.value)
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
      if (source.value === LIBRARY_SOURCE_STEAM) {
        status.value = pickedPath
          ? `已入库：${data.id}（元数据已载入数据库，封面存到 covers/；默认通过 Steam 启动，已记录本地程序可切换）`
          : `已入库：${data.id}（元数据已载入数据库，封面存到 covers/；启动默认通过 Steam）`
      } else {
        status.value = pickedPath
          ? `已入库：${data.id}（元数据已载入数据库，封面存到 covers/，游戏位置已记录）`
          : `已入库：${data.id}（元数据已载入数据库，封面存到 covers/；未记录游戏位置，启动时会再让你选）`
      }
    } catch (err) {
      error.value = `入库失败：${toMessage(err)}`
    } finally {
      downloading.value = false
    }
  }

  /**
   * 查询当前作品的游戏状态：exe 是否还在、是否正在运行。
   *
   * 进入作品详情时调一次；页面打开期间定时轮询（游戏启动/退出要能实时反映到
   * 「游戏启动 ↔ 运行中」）。exe 不存在时给一句提示，启动按钮由 `launchBlocked` 禁用。
   * 状态是附加信息，查询失败不打扰用户（保留上一次状态）。
   */
  async function refreshGameStatus(): Promise<void> {
    const current = work.value
    if (!current || !stored.value || !gamePath.value || gameStatusLoading.value) return
    if (typeof document !== 'undefined' && document.hidden) return

    gameStatusLoading.value = true
    try {
      const status = await gameApi.status(current.id, source.value)
      if (work.value?.id !== current.id) return // 期间换作品了，丢弃旧响应
      gamePath.value = status.path || gamePath.value
      gamePathExists.value = status.exists
      gameRunning.value = status.running
      // 生效方式以后端为准（默认值已按来源解析）：切换过方式后状态轮询也能对齐
      const reportedMode = (status.launchMode ?? '').toLowerCase()
      if (reportedMode === 'steam' || reportedMode === 'exe') launchMode.value = reportedMode
      // Steam 方式不依赖本地 exe：没装/换了目录都不该提示"程序已被删除"
      gameWarning.value =
        effectiveLaunchMode.value === 'steam' || status.exists || !status.path
          ? ''
          : '游戏程序已被删除或移动了位置，请点「更换位置」重新选择游戏程序'
    } catch {
      // 静默：轮询偶尔失败不该弹错误打断浏览
    } finally {
      gameStatusLoading.value = false
    }
  }

  /**
   * 「游戏启动」按钮：用库里记录的位置启动游戏。
   *
   * 没有位置（老数据 / 手动搜索入库）时弹系统文件框选一次 exe，存好再启动；
   * 已经记录位置 → 直接启动。exe 不存在的禁用态由 `launchBlocked` 在按钮层挡掉。
   */
  async function launchGame(): Promise<void> {
    const current = work.value
    if (!current || !stored.value || launching.value || launchBlocked.value) return
    launching.value = true
    error.value = ''
    status.value = ''
    try {
      let result = await gameApi.launch(current.id, source.value)
      // 只有 exe 方式才需要"第一次先选程序"；Steam 方式不需要本地 exe
      if (result.missing && effectiveLaunchMode.value === 'exe') {
        const picked = await gameApi.pickExecutable(`选择「${current.name}」的游戏程序`)
        if (!picked) return // 用户取消：什么都不做
        const savedPath = await gameApi.setPath(current.id, picked, source.value)
        gamePath.value = savedPath.path
        gamePathExists.value = true
        result = await gameApi.launch(current.id, source.value)
      }
      if (!result.started) throw new Error(result.message || '启动失败')
      if (result.path) {
        gamePath.value = result.path
        gamePathExists.value = true
      }
      gameRunning.value = result.running
      gameWarning.value = ''
      status.value = result.message || `已启动游戏：${result.path}`
      // 稍后复核一次（进程可能秒退；Steam 方式下 Steam 客户端拉起游戏也需要几秒）
      void refreshGameStatus()
    } catch (err) {
      error.value = `启动游戏失败：${toMessage(err)}`
      // 启动失败可能正是因为 exe 变了，刷新状态让按钮/提示回到真实情况
      void refreshGameStatus()
    } finally {
      launching.value = false
    }
  }

  /** 更换游戏程序（已经设置过位置、但选错了 exe / 换了安装目录时用；选择框沿用原来的） */
  async function changeGamePath(): Promise<void> {
    const current = work.value
    if (!current || !stored.value || launching.value) return
    error.value = ''
    try {
      const picked = await gameApi.pickExecutable(`选择「${current.name}」的游戏程序`)
      if (!picked) return
      const savedPath = await gameApi.setPath(current.id, picked, source.value)
      gamePath.value = savedPath.path
      gamePathExists.value = true
      gameRunning.value = false
      gameWarning.value = ''
      status.value = `已记录游戏位置：${savedPath.path}`
      void refreshGameStatus()
    } catch (err) {
      error.value = `设置游戏位置失败：${toMessage(err)}`
    }
  }

  /**
   * 切换启动方式（Steam 源作品才有两种可选，用户 2026-11-23 要求）：
   *   - 'steam'：写库后下次启动跑 steam://rungameid/<appid>；
   *   - 'exe'：还没记录本地程序时先弹系统文件框选一次 exe（取消则保持原样），再写库。
   * 只改「下次怎么启动」，不会顺手启动游戏。
   */
  async function setGameLaunchMode(mode: 'steam' | 'exe'): Promise<void> {
    const current = work.value
    if (!current || !stored.value || launching.value) return
    // 已经在目标方式上（且 exe 方式下已经有程序）就不必重复写库
    if (effectiveLaunchMode.value === mode && (mode === 'steam' || Boolean(gamePath.value))) return

    error.value = ''
    try {
      if (mode === 'exe' && !gamePath.value) {
        const picked = await gameApi.pickExecutable(`选择「${current.name}」的游戏程序`)
        if (!picked) return // 用户取消：保持原来的启动方式
        const savedPath = await gameApi.setPath(current.id, picked, source.value)
        gamePath.value = savedPath.path
        gamePathExists.value = true
        gameWarning.value = ''
      }
      const state = await gameApi.setLaunchMode(current.id, mode, source.value)
      const reported = (state.mode ?? '').toLowerCase()
      launchMode.value = reported === 'steam' || reported === 'exe' ? reported : mode
      status.value =
        mode === 'steam'
          ? `启动方式已切换：通过 Steam 启动（steam://rungameid/${current.id}）`
          : `启动方式已切换：本地程序启动（${gamePath.value}）`
      void refreshGameStatus()
    } catch (err) {
      error.value = `切换启动方式失败：${toMessage(err)}`
    }
  }

  /** 标签去重（保序）：抓取来的 tags 与库里存的标签可能重叠 */
  function dedupeTags(list: string[]): string[] {
    const seen = new Set<string>()
    const out: string[] = []
    for (const tag of list) {
      const value = tag.trim()
      if (!value || seen.has(value)) continue
      seen.add(value)
      out.push(value)
    }
    return out
  }

  /**
   * 读当前作品的标签集合，拼出编辑稿：
   *   - 库里存的是用户整理过的整份列表（edited=true）→ 直接用它（重抓也不会被冲掉）；
   *   - 还没整理过 → 抓取来的 tags 打底，旧版逐条加过的标签接在后面（去重）。
   * 失败就当没有：标签是辅助功能，不该让详情页整体报错。
   */
  async function loadTags(workId: string) {
    tagError.value = ''
    const fallback = () => {
      tagsEdited.value = false
      const initial = dedupeTags(work.value?.tags ?? [])
      tagSaved.value = initial
      tagList.value = [...initial]
    }
    try {
      const state = await libraryApi.getTags(workId, source.value)
      // 期间换作品了：丢弃旧响应（否则会把上一部的标签贴到当前作品上）
      if (work.value?.id !== workId) return
      const saved = dedupeTags(state.tags ?? [])
      const initial = state.edited ? saved : dedupeTags([...(work.value?.tags ?? []), ...saved])
      tagsEdited.value = state.edited
      tagSaved.value = initial
      tagList.value = [...initial]
    } catch {
      if (work.value?.id !== workId) return
      fallback()
    }
  }

  /** 添加一个标签：只改本地编辑稿，等「保存」才写库；重复的不加 */
  function addTag(tag: string): boolean {
    const value = tag.trim()
    if (!value || tagList.value.includes(value)) return false
    tagList.value = [...tagList.value, value]
    return true
  }

  /** 删掉一个标签：同样只改编辑稿，等「保存」才写库 */
  function removeTag(tag: string) {
    tagList.value = tagList.value.filter((item) => item !== tag)
  }

  /**
   * 把当前编辑稿**整份**写进数据库（用户点「保存」时唯一会写标签的入口）。
   * 成功后用后端返回的列表当新的基准（后端会清洗/去重，顺序以它为准）。
   */
  async function saveTags(): Promise<boolean> {
    const current = work.value
    if (!current || tagsSaving.value) return false
    tagError.value = ''
    tagsSaving.value = true
    try {
      const state = await libraryApi.setTags(current.id, tagList.value, source.value)
      if (work.value?.id !== current.id) return false
      const saved = dedupeTags(state.tags ?? [])
      tagSaved.value = saved
      tagList.value = [...saved]
      tagsEdited.value = state.edited
      return true
    } catch (err) {
      tagError.value = `保存标签失败：${toMessage(err)}`
      return false
    } finally {
      tagsSaving.value = false
    }
  }

  /** 读取 covers/ 的绝对路径（失败就留空，不打扰用户） */
  async function refreshCoversDir() {
    try {
      coversPath.value = await metaApi.coversDir()
    } catch {
      coversPath.value = ''
    }
    coversLoaded.value = true
  }

  /** 进页面时调：只问一次后端，之后靠「刷新」按钮手动更新 */
  function ensureCoversDir() {
    if (!coversLoaded.value) void refreshCoversDir()
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
    coversPath,
    candidateCovers,
    candidatePage,
    pageLoading,
    raw,
    cached,
    source,
    libraryUpdatedAt,
    coverAbsPath,
    gamePath,
    launchMode,
    gamePathExists,
    gameRunning,
    gameWarning,
    launching,
    tagList,
    tagsEdited,
    tagsSaving,
    tagError,
    searchNotice,
    steamCandidates,
    steamSearchTotal,
    steamStats,
    steamStatsLoading,
    heroImage,
    // Getters
    rawJson,
    hasResult,
    stored,
    launchBlocked,
    effectiveLaunchMode,
    tagsDirty,
    backgroundUrl,
    candidates,
    candidateTotal,
    candidatePageCount,
    pagedCandidates,
    // Actions
    runSearch,
    runMultiSearch,
    loadSteamStats,
    loadById,
    searchBySample,
    storeIn,
    launchGame,
    changeGamePath,
    setGameLaunchMode,
    refreshGameStatus,
    addTag,
    removeTag,
    saveTags,
    setCandidatePage,
    refreshCoversDir,
    ensureCoversDir,
    clearResult,
  }
})

/** 从若干候选里挑第一个非空字符串（Steam 原始数据的字段类型是 unknown） */
function firstImageUrl(values: unknown[]): string {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return ''
}

/**
 * 2026-11-23 之前入库的作品没有 previewImages：从封面 URL 推一张预览大图
 * （`_img_main.jpg` → `_img_smp1.jpg`）。推出来的图未必存在，加载失败时由 <img> 报错兜底。
 */
function deriveDlsitePreviewUrl(coverUrl: unknown): string {
  if (typeof coverUrl !== 'string' || !coverUrl) return ''
  return coverUrl.replace(/_img_main(?:_\d+x\d+)?\.[a-z0-9]+$/i, '_img_smp1.jpg')
}
