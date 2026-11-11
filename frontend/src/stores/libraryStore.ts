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

import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { LIBRARY_SOURCE_DLSITE, libraryApi } from '../services/libraryService'
import type { LibraryWork } from '../services/libraryService'
import { toMessage } from '../utils/metaFormat'

export const useLibraryStore = defineStore('library', () => {
  // ===== State =====
  const works = ref<LibraryWork[]>([])
  const loading = ref(false)
  const error = ref('')
  /** 作品编号 → 本地封面 data URL（空串表示"查过了、没有封面"） */
  const covers = ref<Record<string, string>>({})
  const removingId = ref('')
  /** 是否已经加载过一次（避免每次进页面都闪一下 loading） */
  const loaded = ref(false)

  // ===== Getters =====
  /** 已入库作品数（游戏库顶栏右侧显示） */
  const count = computed(() => works.value.length)

  // ===== Actions =====

  /** 拉列表 + 并发把每条的本地封面取成 data URL */
  async function load(force = false) {
    if (loading.value) return
    if (loaded.value && !force) return
    loading.value = true
    error.value = ''
    try {
      works.value = await libraryApi.list(LIBRARY_SOURCE_DLSITE)
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
        if (covers.value[work.workId]) return [work.workId, covers.value[work.workId]] as const
        try {
          const url = await libraryApi.coverDataUrl(work.workId)
          return [work.workId, url] as const
        } catch {
          return [work.workId, ''] as const
        }
      }),
    )
    const next = { ...covers.value }
    for (const [id, url] of entries) next[id] = url
    covers.value = next
  }

  /** 从库里删一条记录（后端不删磁盘上的图片） */
  async function remove(workId: string) {
    removingId.value = workId
    error.value = ''
    try {
      await libraryApi.remove(workId)
      works.value = works.value.filter((w) => w.workId !== workId)
      const next = { ...covers.value }
      delete next[workId]
      covers.value = next
    } catch (err) {
      error.value = `删除失败：${toMessage(err)}`
    } finally {
      removingId.value = ''
    }
  }

  /** 作品刚被抓进库（或刚下完封面）时调：只刷新这一条的封面，不重拉整个列表 */
  async function refreshCover(workId: string) {
    try {
      const url = await libraryApi.coverDataUrl(workId)
      covers.value = { ...covers.value, [workId]: url }
    } catch {
      // 取不到就是没有封面，不打扰用户
    }
  }

  /** 判断某个作品是否已在库里（作品库详情用） */
  function has(workId: string): boolean {
    return works.value.some((w) => w.workId === workId)
  }

  return {
    // State
    works,
    loading,
    error,
    covers,
    removingId,
    loaded,
    // Getters
    count,
    // Actions
    load,
    loadCovers,
    remove,
    refreshCover,
    has,
  }
})
