// frontend/src/services/libraryService.ts
// 游戏库（缓存优先）的前端封装。
// 对应后端 internal/service/library_service.go（Wails 服务名 LibraryService）。
//
// 与 metaService.ts 的分工（很重要，别合并）：
//   metaApi    —— 永远联网。调试页 /testing 用它，为的是核对「后端解析得对不对」。
//   libraryApi —— 缓存优先。作品库 /works 与游戏库 /library 用它：库里有的直接给，
//                 没有才联网，抓完自动入库。
//
// 注意：frontend/bindings/ 是 wails3 generate bindings 生成的（已被 .gitignore 忽略），
// 新克隆仓库后必须先跑一次生成命令，否则下面的 import 会解析失败。
import { LibraryService } from '../../bindings/WeedyBox/internal/service'
import type { LibraryWork } from '../../bindings/WeedyBox/internal/model/models.js'
import type { WorkMeta } from './metaService'

export type { LibraryWork }

/** 当前唯一的元数据来源；加新来源时改后端 database.SourceDLsite 一类的常量即可 */
export const LIBRARY_SOURCE_DLSITE = 'dlsite'

/**
 * 缓存优先取回来的作品 = 后端抓到的原始 json + 这几个库侧字段。
 * 后端是在同一份 map 上"额外注入"这些键，所以字段与 WorkMeta 兼容。
 */
export interface LibraryMeta extends WorkMeta {
  /** true = 这次没联网，数据来自本地库 */
  cached?: boolean
  workId?: string
  libraryCreatedAt?: string
  libraryUpdatedAt?: string
  coverDir?: string
  coverPath?: string
  coverAbsPath?: string
  coverBytes?: number
  coverSuffix?: string
}

/** 辅助函数：确保返回值不为 null（绑定层会返回 `T | null`） */
function ensure<T>(value: T | null | undefined, message: string): T {
  if (value === null || value === undefined) {
    throw new Error(message)
  }
  return value
}

export const libraryApi = {
  /**
   * 取一个作品：**缓存优先**。
   * refresh = true 表示"我要最新的，别给我缓存"，会跳过本地库直接联网。
   */
  async fetch(keyword: string, workId = '', refresh = false): Promise<LibraryMeta> {
    const result = await LibraryService.Fetch({
      keyword,
      workId,
      source: LIBRARY_SOURCE_DLSITE,
      refresh,
    })
    return ensure<LibraryMeta>(result as LibraryMeta | null, '未返回作品数据')
  },

  /** 列出游戏库里已入库的作品（最近入库的在前）；source 传空串 = 所有来源 */
  async list(source = ''): Promise<LibraryWork[]> {
    return (await LibraryService.List(source)) ?? []
  },

  /** 从游戏库删掉一条记录（磁盘上的图片不动） */
  async remove(workId: string, source = LIBRARY_SOURCE_DLSITE): Promise<void> {
    await LibraryService.Remove(workId, source)
  },

  /**
   * 本地封面的 data URL（base64），给 <img> 直接显示。
   * 没有下载过封面 / 文件被删了 → 返回空串（不是错误）。
   */
  async coverDataUrl(workId: string, source = LIBRARY_SOURCE_DLSITE): Promise<string> {
    return (await LibraryService.CoverDataURL(workId, source)) ?? ''
  },
}
