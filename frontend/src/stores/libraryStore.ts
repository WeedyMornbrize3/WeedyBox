// frontend/src/stores/libraryStore.ts
// 游戏库：已入库作品列表 + 本地封面缓存。
//
// 为什么单独一个 store 而不是塞进 metaStore：
//   metaStore 描述「一次搜索/抓取的过程」（关键词、候选、当前作品、原始 json），
//   libraryStore 描述「库里现在有什么」（列表、封面 data URL、删除）。
//   两者生命周期完全不同 —— 退出一部作品不该把游戏库列表清空。
//
// 封面走 base64 data URL（后端 libraryApi.coverDataUrl）：Wails webview 对 file:// 的
// 支持至今没实机验证过，而 data URL 一定显示得出来，封面本来也只有几百 KB。
//
// 多来源（2026-11-21 加 Steam）：列表**不按来源过滤**（dlsite / steam 一起显示），
// 封面缓存的 key 用 `来源:编号`，避免两个来源的编号撞车。

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { libraryApi } from '../services/libraryService'
import type { LibrarySource, LibraryWork } from '../services/libraryService'
import { toMessage } from '../utils/metaFormat'

/** 封面缓存的 key：来源 + 编号（dlsite: RJ… / steam: appid） */
function coverKey(work: Pick<LibraryWork, 'source' | 'workId'>): string {
  return `${work.source}:${work.workId}`
}

export const useLibraryStore = defineStore('library', () => {
  // ===== State =====
  const works = ref<LibraryWork[]>([])
  const loading = ref(false)
  const error = ref('')
  /** `来源:编号` → 本地封面 data URL（空串表示"查过了、没有封面"） */
  const covers = ref<Record<string, string>>({})
  /** 批量删除进行中（整表删除期间不许重复点） */
  const removing = ref(false)
  /** 是否已经加载过一次（避免每次进页面都闪一下 loading） */
  const loaded = ref(false)

  // ===== Getters =====
  /** 已入库作品数（游戏库顶栏右侧显示） */
  const count = computed(() => works.value.length)

  // ===== Actions =====

  /** 拉列表（**所有来源**）+ 并发把每条的本地封面取成 data URL */
  async function load(force = false) {
    if (loading.value) return
    if (loaded.value && !force) return
    loading.value = true
    error.value = ''
    try {
      works.value = await libraryApi.list()
      loaded.value = true
      await loadCovers()
    } catch (err) {
      error.value = `读取游戏库失败：${toMessage(err)}`
    } finally {
      loading.value = false
    }
  }

  /** 只取封面（新入库的作品、或封面刚下载完时单独调） */
  async function loadCovers() {
    // 逐条 await 会串行几十次 IPC；这里一次性并发，失败了那条就先留空
    const entries = await Promise.all(
      works.value.map(async (work) => {
        const key = coverKey(work)
        if (covers.value[key]) return [key, covers.value[key]] as const
        try {
          const url = await libraryApi.coverDataUrl(work.workId, work.source as LibrarySource)
          return [key, url] as const
        } catch {
          return [key, ''] as const
        }
      }),
    )
    const next = { ...covers.value }
    for (const [key, url] of entries) next[key] = url
    covers.value = next
  }

  /** 某张卡片该显示的封面（模板只调这一个） */
  function coverFor(work: Pick<LibraryWork, 'source' | 'workId'>): string {
    return covers.value[coverKey(work)] ?? ''
  }

  /**
   * 批量从游戏库里删除（页面上不再有「单卡删除」，只有批量操作栏的「删除所选」）。
   * 逐条调后端：后端会同时删掉这条作品在 covers/ 里的封面文件；
   * 成功的从本地列表与封面缓存里移除，失败的汇总进 error（其余照删，不整体回滚）。
   * 返回实际删掉的条数。
   */
  async function removeMany(items: LibraryWork[]): Promise<number> {
    const unique = new Map<string, LibraryWork>()
    for (const item of items) unique.set(coverKey(item), item)
    if (!unique.size) return 0

    removing.value = true
    error.value = ''
    const removedKeys = new Set<string>()
    const failures: string[] = []
    try {
      for (const item of unique.values()) {
        try {
          await libraryApi.remove(item.workId, item.source as LibrarySource)
          removedKeys.add(coverKey(item))
        } catch (err) {
          failures.push(`${item.name || item.workId}（${toMessage(err)}）`)
        }
      }
      if (removedKeys.size) {
        works.value = works.value.filter((w) => !removedKeys.has(coverKey(w)))
        const next = { ...covers.value }
        for (const key of removedKeys) delete next[key]
        covers.value = next
      }
      if (failures.length) {
        error.value = `有 ${failures.length} 款删除失败：${failures.join('；')}`
      }
    } finally {
      removing.value = false
    }
    return removedKeys.size
  }

  /** 作品刚被抓进库（或刚下完封面）时调：只刷新这一条的封面，不重拉整个列表 */
  async function refreshCover(workId: string, source: LibrarySource = 'dlsite') {
    try {
      const url = await libraryApi.coverDataUrl(workId, source)
      covers.value = { ...covers.value, [`${source}:${workId}`]: url }
    } catch {
      // 取不到就是没有封面，不打扰用户
    }
  }

  /** 判断某个作品是否已在库里（作品库详情用） */
  function has(workId: string, source?: string): boolean {
    return works.value.some(
      (w) => w.workId === workId && (source === undefined || w.source === source),
    )
  }

  return {
    // State
    works,
    loading,
    error,
    covers,
    removing,
    loaded,
    // Getters
    count,
    // Actions
    load,
    loadCovers,
    coverFor,
    removeMany,
    refreshCover,
    has,
  }
})
