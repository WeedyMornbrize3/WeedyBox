// frontend/src/utils/metaFormat.ts
// 元数据展示用的纯函数：不依赖组件、不发请求，卡片组件与页面都能直接用。
// 从 MetaApiTestCard.vue 抽出，作品库（/works）与调试页（/testing）共用同一套格式化口径，
// 避免两个页面各写一份、改了一处忘另一处。
import type { WorkMeta } from '../services/metaService'

/** 一行键值：展示层的统一结构，MetaKeyValueList 直接消费 */
export interface MetaRow {
  label: string
  value: string
}

/**
 * 把错误对象整成一句可读的话。
 * 注意：Wails 运行时在拿不到后端响应时抛的是 **message 为空字符串的 Error**
 * （实测 `String(e) === "Error"`），原样透传会得到一个空串，
 * 于是 `v-if="error"` 判假、界面上什么都不显示 —— 必须在这里兜底。
 */
export function toMessage(err: unknown): string {
  if (err instanceof Error && err.message) return err.message
  if (typeof err === 'string' && err) return err
  if (err && typeof err === 'object') {
    const maybe = err as { message?: unknown }
    if (typeof maybe.message === 'string' && maybe.message) return maybe.message
    try {
      const text = JSON.stringify(err)
      if (text && text !== '{}') return text
    } catch {
      /* 落到下面 */
    }
  }
  return `${String(err)}：未收到后端响应（后端未注册该服务，或调用被取消）`
}

/** 字节数转人类可读容量（封面一般几十到几百 KB） */
export function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '0 B'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

/** 本地绝对路径 → webview 能直接打开的 file:// 链接（Windows 反斜杠要换成斜杠） */
export function fileUrl(absPath: string): string {
  return `file:///${absPath.replace(/\\/g, '/')}`
}

/**
 * 详情卡最常看的几行字段。
 * 刻意只挑"人想一眼确认"的：社团、系列、形式、年龄分级、文件、容量、发售日、价格、评分、抓取时间；
 * 各种内部 id 之类看调试页的原始 JSON 即可，不必都塞进卡片。
 */
export function buildSummaryRows(work: WorkMeta): MetaRow[] {
  const rows: MetaRow[] = []
  const push = (label: string, value?: string | null) => {
    if (value) rows.push({ label, value })
  }

  push('社团', work.maker ? `${work.maker}${work.makerId ? `（${work.makerId}）` : ''}` : '')
  push('系列', work.series ? `${work.series}${work.seriesId ? `（${work.seriesId}）` : ''}` : '')
  push('作品形式', work.workType)
  push('年龄指定', work.ageRating)
  push('文件格式', work.fileFormats?.length ? work.fileFormats.join(' / ') : '')
  push('文件容量', work.fileSize)
  push('发售日', work.releaseDate)
  push('价格', work.price ? `${work.price} ${work.currency ?? ''}${work.priceUntil ? `（至 ${work.priceUntil}）` : ''}` : '')
  push('评分', work.rating ? `${work.rating}（${work.ratingCount ?? 0} 条）` : '')
  push('抓取时间', work.fetchedAt ? new Date(work.fetchedAt).toLocaleString() : '')
  return rows
}
