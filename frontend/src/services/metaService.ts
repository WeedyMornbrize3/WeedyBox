// frontend/src/services/metaService.ts
// 元数据抓取（DLsite）的前端封装。
// 对应后端 internal/service/getMeta/meta_service.go（Wails 服务名 MetaApiService）。
//
// 刻意不加工数据：后端返回什么就透传什么，测试页直接展示原始 JSON，
// 便于对着 DLsite 页面核对解析结果。
//
// 注意：frontend/bindings/ 是 wails3 generate bindings 生成的（已被 .gitignore 忽略），
// 新克隆仓库后必须先跑一次生成命令，否则下面的 import 会解析失败。
import { MetaApiService } from '../../bindings/WeedyBox/internal/service/getMeta'

/** 后端返回的作品数据（DLsiteWork 的 json 形态）。字段与后端 json tag 一一对应。 */
export interface WorkMeta {
  id: string
  name: string
  alternateId?: string
  maker?: string
  makerId?: string
  series?: string
  seriesId?: string
  workType?: string
  ageRating?: string
  fileFormats?: string[]
  fileSize?: string
  releaseDate?: string
  price?: string
  currency?: string
  priceUntil?: string
  rating?: string
  ratingCount?: string
  coverUrl?: string
  summary?: string
  tags?: string[]
  outline?: Array<{ label: string; value: string }>
  sourceUrl?: string
  fetchedAt?: string
  [key: string]: unknown
}

export interface SearchCandidate {
  id: string
  name: string
  cover?: string
}

export interface SearchResponse {
  query: string
  /** "id" = 按编号精确抓取（work 有值）；"keyword" = 按名字模糊搜索（只有 results） */
  kind: string
  matched: boolean
  message?: string
  work?: WorkMeta
  results?: SearchCandidate[] | null
  /** 当前页（从 1 开始） */
  page?: number
  /** DLsite 搜索的总页数（后端从分页块解析） */
  pageCount?: number
  /** 每页条数（DLsite 固定 30，改 per_page 无效） */
  perPage?: number
  /** 搜索结果总数（页面 data-search-result-count，精确值） */
  total?: number
}

export interface DownloadResult {
  id: string
  /** 相对工作目录的目录名，固定为 preImg */
  dir: string
  /** 相对路径，如 preImg/RJ01721474_cover.jpg */
  path: string
  /** 实际绝对位置，方便对照 */
  absPath: string
  bytes: number
  suffix: string
  coverUrl: string
}

/** 辅助函数：确保返回值不为 null（绑定层会返回 `T | null`） */
function ensure<T>(value: T | null | undefined, message: string): T {
  if (value === null || value === undefined) {
    throw new Error(message)
  }
  return value
}

export const metaApi = {
  /**
   * 关键词含 RJ/RE/VJ 编号 → 精确抓详情；否则模糊搜索返回候选。
   * page = 第几页（DLsite 每页固定 30 条，翻到哪页才取哪页，结果总数因此不受 30 限制）。
   */
  async searchWorks(keyword: string, page = 1): Promise<SearchResponse> {
    return ensure(await MetaApiService.SearchWorks(keyword, page), '搜索返回为空')
  },

  /** 只抓一个作品：workId 优先，否则按 keyword 搜到最匹配的再抓 */
  async fetchWork(keyword: string, workId = ''): Promise<WorkMeta> {
    return ensure<WorkMeta>(await MetaApiService.FetchWork({ keyword, workId }), '未返回作品数据')
  },

  /** 图片落地目录的绝对路径 */
  async preImgDir(): Promise<string> {
    return (await MetaApiService.PreImgDir()) ?? 'preImg'
  },

  /**
   * 搜索结果缩略图：后端把它缓存到工作目录的 `cache/`（目录不存在会建），返回 data URL。
   * 已缓存过的作品不会重复联网；拿不到图时返回空串（列表退回远程直链）。
   */
  async cacheCover(workId: string, coverUrl: string): Promise<string> {
    return (await MetaApiService.CacheCover(workId, coverUrl)) ?? ''
  },
}
