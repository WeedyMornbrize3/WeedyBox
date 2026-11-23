// frontend/src/services/libraryService.ts
// 游戏库（缓存优先）的前端封装。
// 对应后端 internal/service/library_service.go（Wails 服务名 LibraryService）。
//
// 与 metaService.ts 的分工（很重要，别合并）：
//   metaApi    —— 永远联网。调试页 /testing 用它，为的是核对「后端解析得对不对」。
//   libraryApi —— 缓存优先。作品库 /works 与游戏库 /library 用它：库里有的直接给，
//                 没有才联网；但**抓取不会自动入库**，只有 storeIn() 才写库。
//
// 注意：frontend/bindings/ 是 wails3 generate bindings 生成的（已被 .gitignore 忽略），
// 新克隆仓库后必须先跑一次生成命令，否则下面的 import 会解析失败。
import { LibraryService } from '../../bindings/WeedyBox/internal/service'
import type { LibraryWork, TagSet } from '../../bindings/WeedyBox/internal/model/models.js'
import type { WorkMeta } from './metaService'

export type { LibraryWork, TagSet }

/** 支持的元数据来源：dlsite（RJ 编号，联网抓页面）/ steam（appid，商店接口 + 本机封面） */
export type LibrarySource = 'dlsite' | 'steam'

export const LIBRARY_SOURCE_DLSITE: LibrarySource = 'dlsite'
export const LIBRARY_SOURCE_STEAM: LibrarySource = 'steam'

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
  /** 库里记录的游戏程序绝对路径（空 = 还没设置；与元数据同表，重新抓取不会丢） */
  gamePath?: string
  /**
   * 库里记录的启动方式（2026-11-23）：'' = 还没选过（按来源取默认 ——
   * steam 源走 Steam URL，其它源跑本地 exe）；'steam' / 'exe' = 用户显式选过。
   */
  launchMode?: string
}

/** 辅助函数：确保返回值不为 null（绑定层会返回 `T | null`） */
function ensure<T>(value: T | null | undefined, message: string): T {
  if (value === null || value === undefined) {
    throw new Error(message)
  }
  return value
}

/** 入库时要记录的游戏程序位置：扫描添加会带 exe 或文件夹，手动搜索由用户选 exe */
export interface StoreInOptions {
  /** exe 绝对路径（按 exe 名扫到 / 手动搜索时用户选了程序） */
  gamePath?: string
  /** 游戏文件夹绝对路径（按文件夹名扫到，后端进文件夹挑主程序） */
  gameFolder?: string
  /** 文件夹名，用来在多个 exe 里挑与它匹配最多的主程序 */
  gameFolderName?: string
}

export const libraryApi = {
  /**
   * 取一个作品：**缓存优先**（只读）。
   * refresh = true 表示"我要最新的，别给我缓存"，会跳过本地库直接联网。
   * source = steam 时 workId 是 appid（纯数字），后端走 Steam 商店接口。
   * 注意：这一步**不会**把作品写进游戏库 —— 入库是 `storeIn()` 的事。
   */
  async fetch(
    keyword: string,
    workId = '',
    refresh = false,
    source: LibrarySource = LIBRARY_SOURCE_DLSITE,
  ): Promise<LibraryMeta> {
    const result = await LibraryService.Fetch({
      keyword,
      workId,
      source,
      refresh,
    })
    return ensure<LibraryMeta>(result as LibraryMeta | null, '未返回作品数据')
  },

  /**
   * 入库：**只有这一步会把作品收进游戏库** ——
   * 后端抓文本元数据、把封面（优先 webp）落到 `covers/`、写进 `library_works`。
   * 搜索与打开详情（fetch）都只读不写，所以游戏库里只会出现"点过入库"的作品。
   *
   * options 描述「启动位置从哪来」（用户要求入库时就记录 exe）：
   *   - gamePath：按 exe 名扫到 / 手动搜索时用户选了程序 → 直接记录；
   *   - gameFolder + gameFolderName：按文件夹名扫到 → 后端进文件夹挑主程序；
   *   - 全空：手动搜索且用户取消选择 → 位置留空，第一次启动时再问。
   */
  async storeIn(
    workId: string,
    options: StoreInOptions = {},
    source: LibrarySource = LIBRARY_SOURCE_DLSITE,
  ): Promise<LibraryMeta> {
    const result = await LibraryService.Store({
      workId,
      source,
      gamePath: options.gamePath ?? '',
      gameFolder: options.gameFolder ?? '',
      gameFolderName: options.gameFolderName ?? '',
    })
    return ensure<LibraryMeta>(result as LibraryMeta | null, '入库失败：后端未返回作品数据')
  },

  /** 列出游戏库里已入库的作品（最近入库的在前）；source 传空串 = 所有来源 */
  async list(source = ''): Promise<LibraryWork[]> {
    return (await LibraryService.List(source)) ?? []
  },

  /** 从游戏库删掉一条记录（磁盘上的图片不动） */
  async remove(workId: string, source: LibrarySource = LIBRARY_SOURCE_DLSITE): Promise<void> {
    await LibraryService.Remove(workId, source)
  },

  /**
   * 本地封面的 data URL（base64），给 <img> 直接显示。
   * 没有下载过封面 / 文件被删了 → 返回空串（不是错误）。
   */
  async coverDataUrl(workId: string, source: LibrarySource = LIBRARY_SOURCE_DLSITE): Promise<string> {
    return (await LibraryService.CoverDataURL(workId, source)) ?? ''
  },

  /**
   * Steam 源作品的 hero 背景图（data URL）：后端在本机 Steam 客户端缓存
   * `appcache\librarycache` 里按 appid 找文件名含 hero 的图片并读成 base64。
   * 本机没有（游戏没装 / 缓存被清）返回空串 —— 详情页回落到商店接口的预览图。
   */
  async steamHeroImage(appID: string): Promise<string> {
    return (await LibraryService.SteamHeroImage(appID)) ?? ''
  },

  /**
   * 一个作品的标签集合（作品详情页那一排 chip 的数据源）。
   *
   * 标签本来有两个来源：抓取来的（元数据里的 tags）与用户自己加的。详情页把它们
   * 合成一份列表让用户整理，点「保存」才整体写库 —— 所以这里的语义是"整份"：
   *   - `edited=true`：库里存的就是用户整理后的那一份（重新抓取也不会被冲掉）；
   *   - `edited=false`：还没整理过，`tags` 里只有旧版本逐条加过的标签，
   *     前端要再与作品自带的 `tags` 合并成初始列表。
   */
  async getTags(workId: string, source: LibrarySource = LIBRARY_SOURCE_DLSITE): Promise<TagSet> {
    return ensure<TagSet>(await LibraryService.GetTags(workId, source), '未返回标签数据')
  },

  /**
   * 用整份列表覆盖保存标签（唯一的写标签入口，编辑过程中不写库）。
   * 返回保存后的集合（后端已清洗、去重），前端据此更新"已保存"基准。
   */
  async setTags(
    workId: string,
    tags: string[],
    source: LibrarySource = LIBRARY_SOURCE_DLSITE,
  ): Promise<TagSet> {
    return ensure<TagSet>(await LibraryService.SetTags(workId, source, tags), '保存标签失败：后端未返回结果')
  },
}
