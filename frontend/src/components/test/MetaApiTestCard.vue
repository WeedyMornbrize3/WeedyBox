<!-- frontend/src/components/test/MetaApiTestCard.vue
     接口测试卡片 · DLsite 元数据抓取（MetaApiService）
     由 views/test.vue 挂载（视图层只留一个薄壳，与 TodoPage → TodoList 同一结构）。

     滚动：MainLayout 的 router-view 是 overflow-hidden，所以本组件必须自己开滚动容器；
     而又因为 global.css 用 !important 隐藏了全局滚动条，必须配 scrollbar-theme 恢复。 -->
<template>
  <div class="w-full max-w-4xl mx-auto h-full min-h-0 overflow-y-auto scrollbar-theme pr-2 flex flex-col gap-4 select-text">
    <!-- ── 卡片 1：请求 ───────────────────────────────────────────── -->
    <section class="card-flat">
      <header class="section-header px-5 pt-4 mb-0">
        <span class="i-lucide-plug icon-md section-icon" aria-hidden="true" />
        <h2 class="section-title">接口测试 · DLsite 元数据</h2>
        <span class="ml-auto text-xs text-tertiary font-mono">MetaApiService</span>
      </header>

      <div class="p-5 flex flex-col gap-4">
        <form class="flex items-end gap-2" @submit.prevent="runSearch">
          <label class="flex-1 min-w-0 flex flex-col gap-1.5">
            <span class="text-xs text-secondary">作品名或 RJ 编号</span>
            <input
              v-model="keyword"
              type="text"
              class="input-theme w-full"
              placeholder="输入RJ号或名称搜索..."
              autocomplete="off"
              spellcheck="false"
            />
          </label>

          <button type="submit" class="btn-brand flex items-center gap-1.5 shrink-0" :disabled="loading">
            <span :class="loading ? 'i-lucide-loader-circle spin' : 'i-lucide-search'" class="icon-sm" aria-hidden="true" />
            {{ loading ? '请求中' : '搜索' }}
          </button>
        </form>

        <!-- 快捷示例：省得每次手打编号 -->
        <div class="flex flex-wrap items-center gap-1.5">
          <span class="text-xs text-tertiary">示例</span>
          <button
            v-for="s in samples"
            :key="s"
            type="button"
            class="chip"
            @click="useSample(s)"
          >
            {{ s }}
          </button>
        </div>

        <!-- 图片落地目录 -->
        <div class="flex items-start gap-2 text-xs text-tertiary">
          <span class="i-lucide-folder-open icon-xs shrink-0 mt-[1px]" aria-hidden="true" />
          <span class="min-w-0 break-all">
            封面保存目录（相对工作目录）：<code class="font-mono text-secondary">preImg/</code>
            <template v-if="preImgPath">
              <br />绝对路径：<code class="font-mono text-secondary">{{ preImgPath }}</code>
            </template>
          </span>
          <button
            type="button"
            class="ml-auto shrink-0 text-xs text-brand hover:underline cursor-pointer bg-transparent border-none"
            @click="refreshPreImgDir"
          >
            刷新
          </button>
        </div>

        <!-- 错误 -->
        <p v-if="error" class="m-0 flex items-start gap-2 text-xs text-danger">
          <span class="i-lucide-triangle-alert icon-xs shrink-0 mt-[1px]" aria-hidden="true" />
          <span class="min-w-0 break-words">{{ error }}</span>
        </p>

        <!-- 状态行 -->
        <p v-if="status" class="m-0 flex items-start gap-2 text-xs text-secondary">
          <span class="i-lucide-info icon-xs shrink-0 mt-[1px]" aria-hidden="true" />
          <span class="min-w-0 break-words">{{ status }}</span>
        </p>
      </div>
    </section>

    <!-- ── 卡片 2：精确命中（按编号） ─────────────────────────────── -->
    <section v-if="work" class="card-flat">
      <header class="section-header px-5 pt-4 mb-0">
        <span class="i-lucide-file-json icon-md section-icon" aria-hidden="true" />
        <h2 class="section-title">
          <code class="font-mono">{{ work.id }}</code>
          <span class="text-xs text-tertiary font-normal ml-2">按编号精确抓取</span>
        </h2>
        <a
          v-if="work.sourceUrl"
          :href="work.sourceUrl"
          target="_blank"
          rel="noreferrer noopener"
          class="ml-auto text-xs text-brand hover:underline flex items-center gap-1 shrink-0"
        >
          <span class="i-lucide-external-link icon-xs" aria-hidden="true" />
          原始页面
        </a>
      </header>

      <div class="p-5 flex flex-col gap-5">
        <!-- 封面 + 关键字段 -->
        <div class="flex gap-5 items-start flex-wrap">
          <figure class="m-0 flex flex-col gap-2 w-[140px] shrink-0">
            <div class="w-[140px] h-[105px] rounded-lg border-theme bg-secondary flex-center overflow-hidden">
              <img
                v-if="work.coverUrl && !coverFailed"
                :src="work.coverUrl"
                :alt="`${work.name} 封面`"
                class="w-full h-full object-cover"
                referrerpolicy="no-referrer"
                @error="coverFailed = true"
              />
              <span v-else class="i-lucide-image-off icon-lg text-muted" aria-hidden="true" />
            </div>
            <figcaption class="text-[11px] text-tertiary break-all leading-snug">
              {{ work.coverUrl || '（该作品未解析到封面）' }}
            </figcaption>
          </figure>

          <div class="flex-1 min-w-[260px] flex flex-col gap-3">
            <h3 class="text-base font-medium text-primary m-0 leading-snug break-words">{{ work.name }}</h3>

            <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 m-0 text-xs">
              <template v-for="row in summaryRows" :key="row.label">
                <dt class="text-tertiary whitespace-nowrap">{{ row.label }}</dt>
                <dd class="m-0 text-secondary min-w-0 break-words">{{ row.value }}</dd>
              </template>
            </dl>

            <div v-if="work.tags?.length" class="flex flex-wrap gap-1.5">
              <span v-for="tag in work.tags" :key="tag" class="chip cursor-default">{{ tag }}</span>
            </div>

            <!-- 下载按钮 + 结果 -->
            <div class="flex items-center gap-3 flex-wrap pt-1">
              <button
                type="button"
                class="btn-brand flex items-center gap-1.5"
                :disabled="!work.coverUrl || downloading"
                @click="downloadCover"
              >
                <span
                  :class="downloading ? 'i-lucide-loader-circle spin' : 'i-lucide-download'"
                  class="icon-sm"
                  aria-hidden="true"
                />
                {{ downloading ? '下载中' : '下载封面' }}
              </button>

              <a
                v-if="saved"
                :href="fileUrl(saved.absPath)"
                target="_blank"
                rel="noreferrer noopener"
                class="text-xs text-brand hover:underline"
              >
                打开已保存的图片
              </a>
            </div>

            <p v-if="saved" class="m-0 text-xs text-success flex items-start gap-1.5">
              <span class="i-lucide-circle-check-big icon-xs shrink-0 mt-[1px]" aria-hidden="true" />
              <span class="min-w-0 break-all">
                已保存 {{ formatBytes(saved.bytes) }} → <code class="font-mono">{{ saved.path }}</code>
              </span>
            </p>
          </div>
        </div>

        <!-- 简介：可能很长，单独一块并限高 -->
        <div v-if="work.summary" class="flex flex-col gap-1.5">
          <h3 class="text-xs text-tertiary m-0 flex items-center gap-1.5">
            <span class="i-lucide-align-left icon-xs" aria-hidden="true" />
            作品内容（summary · {{ work.summary.length }} 字符）
          </h3>
          <p class="m-0 text-xs leading-relaxed text-secondary whitespace-pre-wrap break-words max-h-[220px] overflow-y-auto scrollbar-theme pr-2">
            {{ work.summary }}
          </p>
        </div>

        <!-- 作品信息表（原始日文 kv） -->
        <div v-if="work.outline?.length" class="flex flex-col gap-1.5">
          <h3 class="text-xs text-tertiary m-0 flex items-center gap-1.5">
            <span class="i-lucide-table icon-xs" aria-hidden="true" />
            Outline（{{ work.outline.length }} 行，保留原始日文标签）
          </h3>
          <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 m-0 text-xs max-h-[220px] overflow-y-auto scrollbar-theme pr-2">
            <template v-for="row in work.outline" :key="row.label + row.value">
              <dt class="text-tertiary whitespace-nowrap">{{ row.label }}</dt>
              <dd class="m-0 text-secondary min-w-0 break-words">{{ row.value }}</dd>
            </template>
          </dl>
        </div>
      </div>
    </section>

    <!-- ── 卡片 3：模糊搜索候选 ───────────────────────────────────── -->
    <section v-if="!work && candidates.length" class="card-flat">
      <header class="section-header px-5 pt-4 mb-0">
        <span class="i-lucide-list icon-md section-icon" aria-hidden="true" />
        <h2 class="section-title">搜索候选（{{ candidates.length }}）</h2>
        <span class="ml-auto text-xs text-tertiary">点「详情」按编号精确抓取</span>
      </header>

      <ul class="list-none m-0 p-0">
        <li
          v-for="item in candidates"
          :key="item.id"
          class="flex items-center gap-3 px-5 py-3 border-b-theme-light last:border-none"
        >
          <div class="w-11 h-11 rounded-md border-theme bg-secondary flex-center overflow-hidden shrink-0">
            <img
              v-if="item.cover"
              :src="item.cover"
              alt=""
              class="w-full h-full object-cover"
              referrerpolicy="no-referrer"
            />
            <span v-else class="i-lucide-image icon-sm text-muted" aria-hidden="true" />
          </div>
          <div class="flex-1 min-w-0">
            <p class="m-0 text-sm text-primary truncate" :title="item.name">{{ item.name }}</p>
            <p class="m-0 text-xs text-tertiary font-mono">{{ item.id }}</p>
          </div>
          <button type="button" class="btn-ghost shrink-0 text-xs" @click="loadById(item.id)">详情</button>
        </li>
      </ul>
    </section>

    <!-- ── 卡片 4：原始数据 ───────────────────────────────────────── -->
    <section v-if="rawJson" class="card-flat">
      <header class="section-header px-5 pt-4 mb-0">
        <span class="i-lucide-braces icon-md section-icon" aria-hidden="true" />
        <h2 class="section-title">原始返回数据（未加工）</h2>
        <button
          type="button"
          class="ml-auto flex items-center gap-1.5 text-xs text-secondary hover:text-primary cursor-pointer bg-transparent border-none"
          @click="copyRaw"
        >
          <span :class="copied ? 'i-lucide-clipboard-check' : 'i-lucide-clipboard'" class="icon-xs" aria-hidden="true" />
          {{ copied ? '已复制' : '复制' }}
        </button>
      </header>

      <pre class="m-0 p-5 text-[11px] leading-relaxed font-mono text-secondary overflow-auto max-h-[420px] scrollbar-theme">{{ rawJson }}</pre>
    </section>
  </div>
</template>

<script setup lang="ts">
// 接口测试卡片：数据不做二次加工，直接展示后端返回的原始 JSON。
import { computed, onMounted, ref } from 'vue'
import { metaApi, type DownloadResult, type SearchCandidate, type WorkMeta } from '../../services/metaService'

const samples = ['RJ01721474', 'RJ01682332', 'エルフ パラダイス']

const keyword = ref('')
const loading = ref(false)
const downloading = ref(false)
const error = ref('')
const status = ref('')
const work = ref<WorkMeta | null>(null)
const candidates = ref<SearchCandidate[]>([])
const saved = ref<DownloadResult | null>(null)
const preImgPath = ref('')
const coverFailed = ref(false)
const copied = ref(false)

/** 后端返回的完整结构，原样展示 */
const raw = ref<unknown>(null)

const rawJson = computed(() => (raw.value === null ? '' : JSON.stringify(raw.value, null, 2)))

/** 只挑几个最常看的字段，完整数据看下面的原始 JSON */
const summaryRows = computed(() => {
  const w = work.value
  if (!w) return []
  const rows: Array<{ label: string; value: string }> = []
  const push = (label: string, value?: string | null) => {
    if (value) rows.push({ label, value })
  }
  push('社团', w.maker ? `${w.maker}${w.makerId ? `（${w.makerId}）` : ''}` : '')
  push('系列', w.series ? `${w.series}${w.seriesId ? `（${w.seriesId}）` : ''}` : '')
  push('作品形式', w.workType)
  push('年龄指定', w.ageRating)
  push('文件格式', w.fileFormats?.length ? w.fileFormats.join(' / ') : '')
  push('文件容量', w.fileSize)
  push('发售日', w.releaseDate)
  push('价格', w.price ? `${w.price} ${w.currency ?? ''}${w.priceUntil ? `（至 ${w.priceUntil}）` : ''}` : '')
  push('评分', w.rating ? `${w.rating}（${w.ratingCount ?? 0} 条）` : '')
  push('抓取时间', w.fetchedAt ? new Date(w.fetchedAt).toLocaleString() : '')
  return rows
})

/**
 * 把错误对象整成一句可读的话。
 * 注意：Wails 运行时在拿不到后端响应时抛的是 **message 为空字符串的 Error**
 * （实测 `String(e) === "Error"`），原样透传会得到一个空串，
 * 于是 `v-if="error"` 判假、界面上什么都不显示 —— 必须在这里兜底。
 */
function toMessage(err: unknown): string {
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

function resetResult() {
  work.value = null
  candidates.value = []
  saved.value = null
  coverFailed.value = false
  error.value = ''
  status.value = ''
  raw.value = null
}

async function runSearch() {
  const q = keyword.value.trim()
  if (!q) {
    error.value = '请输入作品名或 RJ 编号'
    return
  }
  resetResult()
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

/** 从候选列表点「详情」：按编号精确抓 */
async function loadById(id: string) {
  resetResult()
  loading.value = true
  keyword.value = id
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

function useSample(s: string) {
  keyword.value = s
  void runSearch()
}

async function downloadCover() {
  const w = work.value
  if (!w?.coverUrl) return
  downloading.value = true
  error.value = ''
  try {
    const result = await metaApi.downloadCover(w.id, w.coverUrl)
    saved.value = result
  } catch (err) {
    error.value = `下载封面失败：${toMessage(err)}`
  } finally {
    downloading.value = false
  }
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

/** 把本地绝对路径转成 webview 能打开的 file:// 链接 */
function fileUrl(absPath: string): string {
  return `file:///${absPath.replace(/\\/g, '/')}`
}

async function copyRaw() {
  try {
    await navigator.clipboard.writeText(rawJson.value)
    copied.value = true
    window.setTimeout(() => (copied.value = false), 1500)
  } catch {
    error.value = '复制失败：剪贴板不可用'
  }
}

async function refreshPreImgDir() {
  try {
    preImgPath.value = await metaApi.preImgDir()
  } catch {
    preImgPath.value = ''
  }
}

onMounted(refreshPreImgDir)
</script>
