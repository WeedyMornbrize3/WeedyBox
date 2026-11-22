// frontend/src/stores/scanStore.ts
// 「扫描添加」的状态与动作：扫描来源（Steam / 本地文件夹）、文件夹路径、
// 扫描方式（按 exe / 按文件夹名）、扫描结果，以及**扫描完自动逐个搜索**的匹配结果。
//
// 为什么单独一个 store：
//   metaStore 描述"用户手动发起的一次搜索/抓取"，libraryStore 描述"库里有什么"，
//   而这里描述"这批扫出来的名字各自匹配到哪个作品、要不要入库" —— 三者生命周期不同，
//   批量搜索不能去覆盖 metaStore 的单次搜索结果（手动搜索还留着呢）。
//
// 流程（用户要求「全部直接搜索」「分为一键全部入库 / 选择部分入库」）：
//   扫描 → 对每个名字并发（3 路）调 SearchWorks → 结果按扫描顺序排列 →
//   用户勾选部分「入库所选」，或直接「全部入库」。入库仍走 LibraryService.Store
//   （唯一写库入口：抓文本 + 下封面 + 写库）。

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { scanApi } from '../services/scanService'
import type { ScannedGame } from '../services/scanService'
import { metaApi } from '../services/metaService'
import { libraryApi } from '../services/libraryService'
import type { LibrarySource } from '../services/libraryService'
import { useLibraryStore } from './libraryStore'
import { toMessage } from '../utils/metaFormat'

export type ScanSource = 'steam' | 'folder'
export type FolderScanMode = 'exe' | 'folder'
/** 一行匹配的状态：待搜 / 搜索中 / 有匹配 / 无匹配 / 出错 / 被停止 */
export type MatchState = 'pending' | 'searching' | 'matched' | 'empty' | 'error' | 'stopped'

/** 一个扫描名 + 它的搜索结果（结果列表的一行） */
export interface ScanMatch {
  entry: ScannedGame
  state: MatchState
  workId: string
  workName: string
  cover: string
  /** 作品信息摘要（社团 · 作品形式 · 发售日）：入库成功/库里已有时才拿得到 */
  info: string
  message: string
  /** 用户勾选（只对"有匹配且未入库"的行有意义） */
  selected: boolean
  /** 入库中（按钮 loading） */
  storing: boolean
  /** 已入库（本次会话刚入的、或匹配时就知道库里已有） */
  stored: boolean
  storeError: string
}

/** 并发搜索数：DLsite 没有批量接口，一个个搜，3 路足够快又不至于把对方打烦 */
const SEARCH_CONCURRENCY = 3

export const useScanStore = defineStore('scan', () => {
  // ===== State =====
  const source = ref<ScanSource>('steam')
  const mode = ref<FolderScanMode>('exe')
  const folderPath = ref('')
  const scanning = ref(false)
  const picking = ref(false)
  const error = ref('')
  const status = ref('')
  const entries = ref<ScannedGame[]>([])
  const done = ref(false)

  /** 批量搜索：是否还在跑、已完成几个、每一行的结果 */
  const searching = ref(false)
  const searched = ref(0)
  const matches = ref<ScanMatch[]>([])
  /** 详情补全阶段（搜索完再逐条抓社团/形式/发售日） */
  const detailing = ref(false)
  const detailed = ref(0)
  const detailTotal = ref(0)

  /** 批量入库进度（一键全部 / 入库所选共用一个串行队列） */
  const storingAll = ref(false)
  const storeProgress = ref(0)
  const storeTotal = ref(0)
  const storeError = ref('')

  // ===== Getters =====
  const count = computed(() => entries.value.length)
  /** 匹配到作品的行数 */
  const matchedCount = computed(() => matches.value.filter((m) => m.state === 'matched').length)
  /** 还能入库的行（有匹配且未入库） */
  const storableCount = computed(
    () => matches.value.filter((m) => m.state === 'matched' && m.workId && !m.stored).length,
  )
  /** 已勾选且还能入库的行数 */
  const selectedCount = computed(
    () => matches.value.filter((m) => m.selected && m.state === 'matched' && !m.stored).length,
  )
  /** 全选状态（没有可入库项时为 false） */
  const allSelected = computed(
    () => storableCount.value > 0 && selectedCount.value === storableCount.value,
  )

  // ===== 内部 =====
  /** 搜索代次：重新扫描/停止后，旧请求的响应不能再写回列表 */
  let searchToken = 0

  function emptyMatch(entry: ScannedGame): ScanMatch {
    return {
      entry,
      state: 'pending',
      workId: '',
      workName: '',
      cover: '',
      info: '',
      message: '',
      selected: false,
      storing: false,
      stored: false,
      storeError: '',
    }
  }

  /**
   * 作品信息摘要：社团 · 作品形式 · 发售日（缺的字段跳过）。
   * 入库接口返回的是后端刚抓到的完整元数据，用它回填行内信息 —— 这样"扫描入库"
   * 与"手动入库"看到的是同一份作品信息；库里已有的行则直接读库里的列。
   */
  function describeWork(info: {
    maker?: string
    workType?: string
    releaseDate?: string
    fileSize?: string
  }): string {
    return [info.maker, info.workType, info.releaseDate, info.fileSize].filter(Boolean).join(' · ')
  }

  /** 从绝对路径取最后一段（挑主程序时要拿它跟 exe 名比对；Windows/posix 分隔符都认） */
  function folderBaseName(path: string): string {
    const parts = path.split(/[\\/]+/).filter(Boolean)
    return parts.length ? parts[parts.length - 1] : ''
  }

  /**
   * 库里这条记录算不算"已入库"：不只看编号，还要看它有没有详情字段。
   * 旧解析器写进来的空壳（只有名字+封面）不算 —— 让用户能重新点一次入库，
   * 后端会按新模板重抓详情并覆盖库里的空壳，历史数据因此能自愈。
   */
  function libraryEntryComplete(workId: string, source: LibrarySource) {
    const entry = useLibraryStore().works.find((w) => w.workId === workId && w.source === source)
    if (!entry) return null
    const rich = Boolean(entry.maker || entry.workType || entry.releaseDate || entry.series)
    return { entry, rich }
  }

  /** 一行的元数据来源（Steam 条目带 appid，其余都是 DLsite 按名字搜） */
  function matchSource(match: ScanMatch): LibrarySource {
    return match.entry.source === 'steam' ? 'steam' : 'dlsite'
  }

  /** 一行是否已是库里成员（本次刚入的，或库里已有且数据完整） */
  function isStored(match: ScanMatch): boolean {
    if (match.stored) return true
    if (!match.workId) return false
    return Boolean(libraryEntryComplete(match.workId, matchSource(match))?.rich)
  }

  /**
   * 「按文件夹名」扫到的条目先问一次 Steam（2026-11-22）：
   * 文件夹名 → storesearch → 归一化后比对完全一致/前缀重合，命中就把这行改成 Steam 条目
   * （source='steam' + appId）。猜不到返回 false，调用方继续走原来的 DLsite 名字搜索。
   *
   * 为什么要有这一步：Steam 源原本只能靠 appmanifest 识别（= 必须装过），而文件夹里的
   * 游戏经常是"拷过来的/重装没登记"。按名字猜 appid 能补上这个空档；猜错的代价也小 ——
   * 详情页/入库前用户都能看到匹配到的是哪个游戏。
   */
  async function matchSteamEntry(match: ScanMatch): Promise<boolean> {
    if (match.entry.source === 'steam' || match.entry.origin !== 'folder') return false
    try {
      const found = await metaApi.matchSteamApp(match.entry.name)
      if (!found.matched || !found.appId) return false
      match.entry.source = 'steam'
      match.entry.appId = found.appId
      match.entry.detail = `Steam · ${found.appId}`
      return true
    } catch {
      // 网络/接口失败就静默回退 DLsite：多一条 Steam 候选是加分项，不该让整行报错
      return false
    }
  }

  /** 搜一个名字：命中作品（名字里带 RJ）直接用详情，否则取模糊搜索的第一个候选 */
  async function searchOne(match: ScanMatch, token: number) {
    match.state = 'searching'
    match.message = ''
    match.storeError = ''
    // 先试 Steam（只有"按文件夹名"且还不是 Steam 的条目才会真去问），再走原来的分支
    await matchSteamEntry(match)
    if (token !== searchToken) return
    try {
      // Steam 条目：appid 已知，直接取商店详情（缓存优先），不走 DLsite 按名字搜
      if (matchSource(match) === 'steam') {
        const appID = match.entry.appId ?? ''
        if (!appID) {
          match.state = 'empty'
          match.message = '该 Steam 条目没有 appid，已跳过'
          return
        }
        const work = await libraryApi.fetch('', appID, false, 'steam')
        if (token !== searchToken) return
        match.workId = appID
        match.workName = work.name || match.entry.name
        match.cover = work.coverUrl ?? ''
        match.info = describeWork(work)
        // 非游戏条目（Steamworks 共享组件、工具、DLC…）：标成"跳过"，不给入库按钮
        const steamType = String(work.steamType ?? '')
        if (steamType && steamType !== 'game') {
          match.state = 'empty'
          match.message = `Steam 条目类型是 ${steamType}，不是游戏，已跳过`
          return
        }
        match.state = 'matched'
        const existing = libraryEntryComplete(appID, 'steam')
        match.stored = Boolean(existing?.rich)
        if (existing?.rich) match.info = describeWork(existing.entry)
        return
      }

      const resp = await metaApi.searchWorks(match.entry.name, 1)
      if (token !== searchToken) return

      if (resp.work) {
        // 名字里带 RJ：SearchWorks 已经带回完整详情，直接用
        match.workId = resp.work.id
        match.workName = resp.work.name
        match.cover = resp.work.coverUrl ?? ''
        match.info = describeWork(resp.work)
        match.state = 'matched'
      } else {
        const first = resp.results?.[0]
        if (!first) {
          match.state = 'empty'
          return
        }
        match.workId = first.id
        match.workName = first.name
        match.cover = first.cover ?? ''
        match.state = 'matched'
      }
      // 库里已经有的直接标"已入库"，并把库里的作品信息带出来（社团/形式/发售日）；
      // 库里是空壳（没有这些字段）就不算，留给入库时重抓
      const existing = libraryEntryComplete(match.workId, 'dlsite')
      match.stored = Boolean(existing?.rich)
      if (existing?.rich) match.info = describeWork(existing.entry)
    } catch (err) {
      if (token !== searchToken) return
      match.state = 'error'
      match.message = toMessage(err)
    }
  }

  /**
   * 补抓一行的作品详情（缓存优先：库里已有就直接读库，不联网）。
   * 模糊搜索只回"编号/名称/封面"，社团、形式、发售日这些要再抓一次详情页；
   * 这一步是只读的（libraryApi.fetch），**不写库** —— 写库仍然只在点入库时发生。
   */
  async function loadDetail(match: ScanMatch, token: number) {
    try {
      const work = await libraryApi.fetch('', match.workId, false, matchSource(match))
      if (token !== searchToken) return
      match.workName = work.name || match.workName
      match.cover = work.coverUrl || match.cover
      match.info = describeWork(work)
    } catch (err) {
      if (token !== searchToken) return
      // 详情抓不到不影响挑作品：行里仍有编号/名称/封面，入库时后端还会再试一次
      match.message = `详情获取失败：${toMessage(err)}`
    }
  }

  /**
   * 全部直接搜索：两段式 ——
   *   ① 逐个搜名字（3 路并发）拿到匹配的编号/名称/封面，列表先出来；
   *   ② 对没有详情的匹配行再逐条抓详情（缓存优先），把社团/形式/发售日填进行里。
   * 结果始终按扫描顺序留在 matches 里。
   */
  async function searchAll() {
    if (searching.value || !entries.value.length) return
    const token = ++searchToken
    searching.value = true
    searched.value = 0
    matches.value = entries.value.map(emptyMatch)

    const queue = [...matches.value]
    const runWorker = async () => {
      for (;;) {
        if (token !== searchToken) return
        const match = queue.shift()
        if (!match) return
        await searchOne(match, token)
        if (token === searchToken) searched.value += 1
      }
    }

    await Promise.all(
      Array.from({ length: Math.min(SEARCH_CONCURRENCY, queue.length) }, () => runWorker()),
    )
    if (token !== searchToken) return
    searching.value = false

    // ② 详情补全：只补"有匹配但还没信息"的行（名字里带 RJ 的那批已经在 ① 里拿到详情）
    const detailQueue = matches.value.filter((m) => m.state === 'matched' && m.workId && !m.info)
    if (!detailQueue.length) return
    detailing.value = true
    detailed.value = 0
    detailTotal.value = detailQueue.length

    const runDetailWorker = async () => {
      for (;;) {
        if (token !== searchToken) return
        const match = detailQueue.shift()
        if (!match) return
        await loadDetail(match, token)
        if (token === searchToken) detailed.value += 1
      }
    }

    await Promise.all(
      Array.from({ length: Math.min(SEARCH_CONCURRENCY, detailQueue.length) }, () => runDetailWorker()),
    )
    if (token === searchToken) detailing.value = false
  }

  /** 停止批量搜索：正在飞的那几个会跑完，队列里剩下的标成「已停止」 */
  function stopSearch() {
    searchToken += 1
    searching.value = false
    detailing.value = false
    for (const match of matches.value) {
      if (match.state === 'pending' || match.state === 'searching') match.state = 'stopped'
    }
  }

  /** 单行重试（出错或停止的行）：不发起整批；搜到之后顺手补详情 */
  async function retryMatch(match: ScanMatch) {
    if (match.state === 'searching') return
    const token = searchToken
    await searchOne(match, token)
    if (token !== searchToken) return
    if (match.state === 'matched' && !match.info) await loadDetail(match, token)
    searched.value = matches.value.filter((m) => m.state !== 'pending' && m.state !== 'searching').length
  }

  function toggleSelect(match: ScanMatch) {
    if (match.state !== 'matched' || match.stored) return
    match.selected = !match.selected
  }

  /** 全选/全不选：只作用于"有匹配且未入库"的行 */
  function toggleSelectAll(checked: boolean) {
    for (const match of matches.value) {
      if (match.state !== 'matched' || match.stored) {
        match.selected = false
        continue
      }
      match.selected = checked
    }
  }

  /**
   * 批量入库（串行）：一键全部入库 = storeMany(matches)，
   * 选择部分入库 = storeMany(勾选的行)。串行是为了别让几十个"抓文本+下图"
   * 同时压在网络与 SQLite 上；每行独立报错，不影响其他行。
   */
  async function storeMany(targets: ScanMatch[]) {
    if (storingAll.value) return
    const list = targets.filter((m) => m.state === 'matched' && m.workId && !isStored(m) && !m.storing)
    if (!list.length) return

    storingAll.value = true
    storeError.value = ''
    storeProgress.value = 0
    storeTotal.value = list.length
    const library = useLibraryStore()
    try {
      for (const match of list) {
        match.storing = true
        match.storeError = ''
        try {
          // 入库时把启动位置一起带上（用户要求「入库时便存储相关 exe」）：
          //   - 按 exe 名扫到 → 直接给 exe 路径；
          //   - 按文件夹名扫到（Steam / 本地文件夹）→ 给文件夹，后端进文件夹挑主程序
          //     （排除 UnityCrashHandler64、优先 Game.exe、再选与文件夹名匹配最多的 exe）。
          // Steam 条目同样是"有安装目录"的形式，只是元数据来源换成 source='steam'。
          const folder = match.entry.folderPath ?? ''
          const saved = await libraryApi.storeIn(
            match.workId,
            {
              gamePath: match.entry.origin === 'exe' ? (match.entry.path ?? '') : '',
              gameFolder: match.entry.origin === 'folder' ? folder : '',
              gameFolderName: match.entry.origin === 'folder' ? folderBaseName(folder) || match.entry.name : '',
            },
            matchSource(match),
          )
          match.stored = true
          match.selected = false
          // 用后端返回的完整元数据回填：名称/封面以服务端为准，信息行显示刚抓到的作品信息
          match.workName = saved.name || match.workName
          match.cover = saved.coverUrl || match.cover
          match.info = describeWork(saved)
        } catch (err) {
          match.storeError = `入库失败：${toMessage(err)}`
          storeError.value = match.storeError
        } finally {
          match.storing = false
          storeProgress.value += 1
        }
      }
      // 关窗后游戏库网格要立刻出现刚入库的作品
      void library.load(true)
    } finally {
      storingAll.value = false
    }
  }

  function storeAll() {
    return storeMany(matches.value)
  }

  function storeSelected() {
    return storeMany(matches.value.filter((m) => m.selected))
  }

  // ===== 扫描 =====
  function resetResults() {
    // 取消还在跑的批量搜索：代次 +1，旧响应全部作废
    searchToken += 1
    searching.value = false
    detailing.value = false
    searched.value = 0
    detailed.value = 0
    detailTotal.value = 0
    matches.value = []
    entries.value = []
    error.value = ''
    status.value = ''
    done.value = false
    storingAll.value = false
    storeProgress.value = 0
    storeTotal.value = 0
    storeError.value = ''
  }

  function setSource(next: ScanSource) {
    if (source.value === next) return
    source.value = next
    resetResults()
  }

  function setMode(next: FolderScanMode) {
    if (mode.value === next) return
    mode.value = next
    resetResults()
  }

  async function pickFolder() {
    if (picking.value) return
    picking.value = true
    error.value = ''
    try {
      const path = await scanApi.pickDirectory('选择游戏文件夹')
      if (path) {
        folderPath.value = path
        resetResults()
      }
    } catch (err) {
      error.value = `选择文件夹失败：${toMessage(err)}`
    } finally {
      picking.value = false
    }
  }

  async function scan() {
    if (scanning.value) return
    resetResults()
    scanning.value = true
    try {
      if (source.value === 'steam') {
        // 读 appmanifest：拿到 appid + 名称 + 安装目录，每行直接取 Steam 商店详情
        // （不再拿文件夹名去 DLsite 搜；封面由后端从本机 librarycache 找）
        const apps = await scanApi.steamApps()
        entries.value = apps.map((app) => ({
          name: app.name,
          detail: `Steam · ${app.appId}`,
          origin: 'folder',
          folderPath: app.folderPath,
          source: 'steam',
          appId: app.appId,
        }))
      } else {
        const root = folderPath.value.trim()
        if (!root) {
          error.value = '请先选择或填写游戏文件夹'
          return
        }
        if (mode.value === 'exe') {
          const result = await scanApi.executables(root)
          const list: ScannedGame[] = []
          for (const folder of result.folders ?? []) {
            for (const exe of folder.executables ?? []) {
              // 按 exe 名扫到：path 就是这个 exe 的绝对路径，入库直接记录，
              // 之后点「游戏启动」就跑这个程序（不再让用户选一次）
              list.push({
                name: exe.name,
                detail: `${folder.folderName} · ${exe.fileName}`,
                origin: 'exe',
                path: exe.path,
                folderPath: folder.folderPath,
              })
            }
          }
          entries.value = list
          const skipped = result.skipped?.length ?? 0
          if (skipped > 0) status.value = `已跳过 ${skipped} 个读不动的子文件夹`
        } else {
          const folders = await scanApi.subfolders(root)
          // 按文件夹名扫到：入库时后端进这个文件夹挑主程序
          entries.value = folders.map((folder) => ({
            name: folder.name,
            detail: root,
            origin: 'folder',
            folderPath: folder.path,
          }))
        }
      }
      done.value = true
      // 用户要求「全部直接搜索」：扫完立刻逐个搜，不再让用户一一点名字
      void searchAll()
    } catch (err) {
      error.value = `扫描失败：${toMessage(err)}`
    } finally {
      scanning.value = false
    }
  }

  return {
    // State
    source,
    mode,
    folderPath,
    scanning,
    picking,
    error,
    status,
    entries,
    done,
    searching,
    searched,
    matches,
    detailing,
    detailed,
    detailTotal,
    storingAll,
    storeProgress,
    storeTotal,
    storeError,
    // Getters
    count,
    matchedCount,
    storableCount,
    selectedCount,
    allSelected,
    // Actions
    setSource,
    setMode,
    pickFolder,
    scan,
    searchAll,
    stopSearch,
    retryMatch,
    toggleSelect,
    toggleSelectAll,
    storeAll,
    storeSelected,
    isStored,
  }
})
