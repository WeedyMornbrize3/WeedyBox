// frontend/src/stores/metaStore.ts
// DLsite 元数据抓取的状态与动作（唯一一份）。
// 作品库（views/WorksPage.vue）与调试页（components/test/MetaApiTestCard.vue）共用同一个 store，
// 所以两个页面只负责「怎么摆」，不各写一套请求逻辑：
//   stores/metaStore.ts       状态 + 请求（怎么拿数据）
//   components/meta/*         积木（长什么样）
//   views/*                   排布（放在哪）
// 顺带的好处：在两个页面之间切换时搜索条件与结果都还在，不用重新搜一遍。

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { metaApi } from '../services/metaService'
import type { DownloadResult, SearchCandidate, WorkMeta } from '../services/metaService'
import { toMessage } from '../utils/metaFormat'

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
  const candidates = ref<SearchCandidate[]>([])
  const saved = ref<DownloadResult | null>(null)
  const preImgPath = ref('')

  /** 后端返回的完整结构（原样展示，调试页用） */
  const raw = ref<unknown>(null)
  /** preImg/ 绝对路径是否已读过（只在第一次进页面时问一次后端） */
  const preImgLoaded = ref(false)

  // ===== Getters =====
  const rawJson = computed(() => (raw.value === null ? '' : JSON.stringify(raw.value, null, 2)))
  const hasResult = computed(() => work.value !== null || candidates.value.length > 0)

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
    if (!opts.keepCandidates) candidates.value = []
  }

  /** 搜索：关键词含 RJ/RE/VJ 编号走精确抓取，否则模糊搜索返回候选 */
  async function runSearch(query?: string) {
    const q = (query ?? keyword.value).trim()
    if (!q) {
      error.value = '请输入作品名或 RJ 编号'
      return
    }
    keyword.value = q
    clearResult()
    loading.value = true
    try {
      const resp = await metaApi.searchWorks(q)
      raw.value = resp
      status.value = resp.message ?? ''
      candidates.value = resp.results ?? []
      // 按编号精确抓取时后端会带上 work
      work.value = resp.work ?? null
      if (work.value) candidates.value = []
    } catch (err) {
      error.value = toMessage(err)
    } finally {
      loading.value = false
    }
  }

  /** 点候选的「详情」：按编号精确抓（保留候选列表，方便连着换） */
  async function loadById(id: string) {
    clearResult({ keepCandidates: true })
    keyword.value = id
    loading.value = true
    try {
      const data = await metaApi.fetchWork('', id)
      work.value = data
      raw.value = data
      status.value = `按编号 ${id} 精确抓取成功`
    } catch (err) {
      error.value = toMessage(err)
    } finally {
      loading.value = false
    }
  }

  /** 点示例：填进输入框再搜一次 */
  function searchBySample(sample: string) {
    void runSearch(sample)
  }

  /** 下载封面到 preImg/ */
  async function downloadCover() {
    const current = work.value
    if (!current?.coverUrl) return
    downloading.value = true
    error.value = ''
    try {
      saved.value = await metaApi.downloadCover(current.id, current.coverUrl)
    } catch (err) {
      error.value = `下载封面失败：${toMessage(err)}`
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
    candidates,
    saved,
    preImgPath,
    raw,
    // Getters
    rawJson,
    hasResult,
    // Actions
    runSearch,
    loadById,
    searchBySample,
    downloadCover,
    refreshPreImgDir,
    ensurePreImgDir,
    clearResult,
  }
})
