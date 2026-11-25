<!-- frontend/src/components/meta/MetaWorkEditorForm.vue
     作品详情页「编辑」页里的元数据表单（2026-11-24 用户要求：
     「编辑界面中所有数据库数据都可以改变」「日期选择等信息可以用组件中的日历等组件修改」）。
     覆盖 library_works 里所有**人可读**的元数据字段：名称 / 社团 / 系列 / 形式 / 年龄 /
     文件格式 / 容量 / 发售日 / 价格与有效期 / 评分 / 简介 / 链接 / 封面 URL / Outline。
     预览图、Steam 平台原始数据、抓取时间属于抓取产物，不在这里改（改了也没有意义）。

     交互约定：
       - 表单先攒在本地 draft 里，点「保存」才写库（与标签编辑同一套口径），另有「还原」；
       - 日期用 n-date-picker（带日历），保存统一成 yyyy-MM-dd；原来不是标准日期的
         旧值不会被吞掉 —— 不动它就不提交这个字段，避免一次无关编辑把日期改坏；
       - 只提交**改过的字段**（打补丁），空值 = 清空这一项（后端会连 raw_json 一起清）。
     请求在 stores/metaStore.ts 的 saveMeta()，本组件只管长相与草稿。 -->
<template>
  <div class="flex flex-col gap-4">
    <div class="flex items-center gap-2 flex-wrap">
      <span class="i-lucide-square-pen icon-sm text-brand" aria-hidden="true" />
      <h3 class="m-0 text-sm font-medium text-primary">元数据</h3>
      <span class="text-xs text-tertiary">
        {{ dirty ? '有未保存的改动' : '改完点右侧「保存」写回数据库' }}
      </span>

      <div class="ml-auto flex items-center gap-2">
        <n-button size="tiny" :disabled="!canEdit || !dirty || meta.metaSaving" @click="resetDraft">
          还原
        </n-button>
        <n-button
          size="tiny"
          type="primary"
          :disabled="!canEdit || !dirty || meta.metaSaving"
          :loading="meta.metaSaving"
          @click="save"
        >
          <template #icon>
            <span class="i-lucide-save icon-xs" aria-hidden="true" />
          </template>
          保存
        </n-button>
      </div>
    </div>

    <p v-if="!canEdit" class="m-0 text-xs text-tertiary">
      作品还没入库 —— 先点顶部动作栏的「入库」，之后才能编辑并保存元数据。
    </p>

    <template v-else>
      <p class="m-0 text-[11px] text-tertiary">
        编号 <code class="font-mono">{{ work.id }}</code> 是库记录的定位键，不可修改；
        预览图、Steam 平台原始数据属于抓取产物，不在这里编辑。
      </p>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-x-4 gap-y-3">
        <label class="work-field md:col-span-2">
          <span class="work-field-label">名称 <span class="text-danger">*</span></span>
          <n-input v-model:value="draft.name" size="small" placeholder="作品名称" />
        </label>

        <label class="work-field">
          <span class="work-field-label">社团 / 开发商</span>
          <n-input v-model:value="draft.maker" size="small" placeholder="社团名 / 开发商" />
        </label>
        <label class="work-field">
          <span class="work-field-label">社团编号</span>
          <n-input v-model:value="draft.makerId" size="small" placeholder="如 RG51931" />
        </label>

        <label class="work-field">
          <span class="work-field-label">系列</span>
          <n-input v-model:value="draft.series" size="small" placeholder="系列名" />
        </label>
        <label class="work-field">
          <span class="work-field-label">系列编号</span>
          <n-input v-model:value="draft.seriesId" size="small" placeholder="如 SRI0000043774" />
        </label>

        <label class="work-field">
          <span class="work-field-label">作品形式</span>
          <n-input v-model:value="draft.workType" size="small" placeholder="如 游戏 / ボイス・ASMR" />
        </label>
        <label class="work-field">
          <span class="work-field-label">年龄指定</span>
          <n-input v-model:value="draft.ageRating" size="small" placeholder="如 R18 / 全年龄" />
        </label>

        <label class="work-field">
          <span class="work-field-label">文件容量</span>
          <n-input v-model:value="draft.fileSize" size="small" placeholder="如 1.2GB" />
        </label>
        <div class="work-field">
          <span class="work-field-label">文件格式（可增删）</span>
          <n-dynamic-tags v-model:value="draft.fileFormats" size="small" :max="32" />
        </div>

        <label class="work-field">
          <span class="work-field-label">发售日</span>
          <n-date-picker
            v-model:formatted-value="draft.releaseDate"
            type="date"
            value-format="yyyy-MM-dd"
            size="small"
            clearable
            placeholder="选择发售日"
          />
          <span v-if="rawReleaseDate && !initialReleaseDate" class="work-field-hint">
            原值「{{ rawReleaseDate }}」不是标准日期；重新选一次才会覆盖它
          </span>
        </label>
        <label class="work-field">
          <span class="work-field-label">价格有效期至</span>
          <n-date-picker
            v-model:formatted-value="draft.priceUntil"
            type="date"
            value-format="yyyy-MM-dd"
            size="small"
            clearable
            placeholder="没有则留空"
          />
          <span v-if="rawPriceUntil && !initialPriceUntil" class="work-field-hint">
            原值「{{ rawPriceUntil }}」不是标准日期；重新选一次才会覆盖它
          </span>
        </label>

        <label class="work-field">
          <span class="work-field-label">价格</span>
          <n-input v-model:value="draft.price" size="small" placeholder="如 1540" />
        </label>
        <label class="work-field">
          <span class="work-field-label">币种</span>
          <n-input v-model:value="draft.currency" size="small" placeholder="如 JPY" />
        </label>

        <label class="work-field">
          <span class="work-field-label">评分</span>
          <n-input v-model:value="draft.rating" size="small" placeholder="如 4.43" />
        </label>
        <label class="work-field">
          <span class="work-field-label">评价数</span>
          <n-input v-model:value="draft.ratingCount" size="small" placeholder="如 1484" />
        </label>

        <label class="work-field md:col-span-2">
          <span class="work-field-label">原始页面链接</span>
          <n-input v-model:value="draft.sourceUrl" size="small" placeholder="https://…" />
        </label>

        <label class="work-field md:col-span-2">
          <span class="work-field-label">封面 URL</span>
          <n-input v-model:value="draft.coverUrl" size="small" placeholder="https://…" />
          <span class="work-field-hint">只改元数据里的地址；covers/ 里已下载的封面文件不会自动替换。</span>
        </label>

        <label class="work-field md:col-span-2">
          <span class="work-field-label">简介（summary）</span>
          <n-input
            v-model:value="draft.summary"
            type="textarea"
            size="small"
            :autosize="{ minRows: 4, maxRows: 12 }"
            placeholder="作品简介"
          />
        </label>

        <label class="work-field md:col-span-2">
          <span class="work-field-label">Outline 作品信息表（每行「标签 | 值」，留空即清空）</span>
          <n-input
            v-model:value="draft.outlineText"
            type="textarea"
            size="small"
            :autosize="{ minRows: 4, maxRows: 14 }"
            placeholder="販売日 | 2026年06月18日&#10;作品形式 | ゲーム"
          />
        </label>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton, NDatePicker, NDynamicTags, NInput } from 'naive-ui'
import type { WorkMeta } from '../../services/metaService'
import { useMetaStore } from '../../stores/metaStore'

const meta = useMetaStore()

/** 表单草稿：字段名与 raw_json 的键一一对应（outline 走 outlineText 这段文本） */
interface Draft {
  name: string
  maker: string
  makerId: string
  series: string
  seriesId: string
  workType: string
  ageRating: string
  fileFormats: string[]
  fileSize: string
  /** 日期用 yyyy-MM-dd；n-date-picker 清空时会给 null */
  releaseDate: string | null
  price: string
  currency: string
  priceUntil: string | null
  rating: string
  ratingCount: string
  summary: string
  sourceUrl: string
  coverUrl: string
  outlineText: string
}

/** 只有已入库的作品能被编辑（后端也只更新库里的记录） */
const canEdit = computed(() => Boolean(meta.work) && meta.stored)
const work = computed(() => meta.work as WorkMeta)

const draft = ref<Draft>(emptyDraft())
/** 上一次的基准（JSON 字符串）：用来判断「有没有改动」与「只提交改过的字段」 */
const baseline = ref(JSON.stringify(draft.value))
/** 原样保留下来的日期字符串（不是标准日期时，界面提示用；不动它就不提交） */
const rawReleaseDate = ref('')
const rawPriceUntil = ref('')
const initialReleaseDate = ref('')
const initialPriceUntil = ref('')

const dirty = computed(() => JSON.stringify(draft.value) !== baseline.value)

function emptyDraft(): Draft {
  return {
    name: '',
    maker: '',
    makerId: '',
    series: '',
    seriesId: '',
    workType: '',
    ageRating: '',
    fileFormats: [],
    fileSize: '',
    // ⚠️ 日期字段的空值必须是 null，不能是 ''：naive 的 n-date-picker 会把空串
    // 当格式化值去解析，setup 里抛 RangeError: Invalid time value，整个组件树渲染中断
    // （表现为「点编辑 tab 后下面空白、连路由都点不动」）。2026-11-24 实测踩过。
    releaseDate: null,
    price: '',
    currency: '',
    priceUntil: null,
    rating: '',
    ratingCount: '',
    summary: '',
    sourceUrl: '',
    coverUrl: '',
    outlineText: '',
  }
}

/** 任意值转成表单用的字符串（raw_json 是 any，不能假设类型） */
function text(value: unknown): string {
  if (value === null || value === undefined) return ''
  return typeof value === 'string' ? value : String(value)
}

/** 「2020 年 9 月 3 日」/「2020/09/03」/「2020-09-03」→ yyyy-MM-dd；认不出返回空串 */
function toDateValue(raw: string): string {
  const match = raw.match(/(\d{4})\D+(\d{1,2})\D+(\d{1,2})/)
  if (!match) return ''
  const [, year, month, day] = match
  // 必须校验成真实日期：n-date-picker 拿到 "2020-99-99" 这种格式化值会在 setup 里
  // 抛 RangeError（Invalid time value），整个编辑面板直接渲染不出来。
  const date = new Date(Number(year), Number(month) - 1, Number(day))
  if (
    date.getFullYear() !== Number(year) ||
    date.getMonth() !== Number(month) - 1 ||
    date.getDate() !== Number(day)
  ) {
    return ''
  }
  return `${year}-${month.padStart(2, '0')}-${day.padStart(2, '0')}`
}

/** Outline 数组 → 每行「标签 | 值」的文本 */
function outlineToText(outline: unknown): string {
  if (!Array.isArray(outline)) return ''
  return outline
    .map((row) => {
      const label = text((row as { label?: unknown })?.label).trim()
      const value = text((row as { value?: unknown })?.value).trim()
      if (!label) return value
      return value ? `${label} | ${value}` : label
    })
    .filter(Boolean)
    .join('\n')
}

/** 文本 → Outline 数组：按第一个「|」切分；没有竖线的行当成只有值 */
function textToOutline(input: string): Array<{ label: string; value: string }> {
  return input
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const index = line.indexOf('|')
      if (index < 0) return { label: '', value: line }
      return { label: line.slice(0, index).trim(), value: line.slice(index + 1).trim() }
    })
}

/** 用当前作品重置草稿与基准（进入页面、换作品、保存成功后都走这里） */
function resetDraft() {
  const current = meta.work
  if (!current) {
    draft.value = emptyDraft()
    baseline.value = JSON.stringify(draft.value)
    return
  }
  rawReleaseDate.value = text(current.releaseDate)
  rawPriceUntil.value = text(current.priceUntil)
  initialReleaseDate.value = toDateValue(rawReleaseDate.value)
  initialPriceUntil.value = toDateValue(rawPriceUntil.value)
  draft.value = {
    name: text(current.name),
    maker: text(current.maker),
    makerId: text(current.makerId),
    series: text(current.series),
    seriesId: text(current.seriesId),
    workType: text(current.workType),
    ageRating: text(current.ageRating),
    fileFormats: Array.isArray(current.fileFormats) ? current.fileFormats.map(text).filter(Boolean) : [],
    fileSize: text(current.fileSize),
    releaseDate: initialReleaseDate.value || null,
    price: text(current.price),
    currency: text(current.currency),
    priceUntil: initialPriceUntil.value || null,
    rating: text(current.rating),
    ratingCount: text(current.ratingCount),
    summary: text(current.summary),
    sourceUrl: text(current.sourceUrl),
    coverUrl: text(current.coverUrl),
    outlineText: outlineToText(current.outline),
  }
  baseline.value = JSON.stringify(draft.value)
}

// 换作品、或库里这条记录被更新（保存成功 / 重新抓取）后，草稿跟着走
watch(() => [meta.work?.id, meta.work?.libraryUpdatedAt], resetDraft, { immediate: true })

/** 只收集改过的字段（打补丁）；空串 / 空数组表示清空这一项 */
function buildPatch(): Record<string, unknown> {
  const initial = JSON.parse(baseline.value) as Draft
  const next = draft.value
  const patch: Record<string, unknown> = {}

  const textKeys = [
    'name',
    'maker',
    'makerId',
    'series',
    'seriesId',
    'workType',
    'ageRating',
    'fileSize',
    'price',
    'currency',
    'rating',
    'ratingCount',
    'summary',
    'sourceUrl',
    'coverUrl',
  ] as const
  for (const key of textKeys) {
    if (next[key] !== initial[key]) patch[key] = next[key]
  }

  if (JSON.stringify(next.fileFormats) !== JSON.stringify(initial.fileFormats)) {
    patch.fileFormats = next.fileFormats
  }
  if ((next.releaseDate ?? '') !== (initial.releaseDate ?? '')) {
    patch.releaseDate = next.releaseDate ?? ''
  }
  if ((next.priceUntil ?? '') !== (initial.priceUntil ?? '')) {
    patch.priceUntil = next.priceUntil ?? ''
  }
  if (next.outlineText !== initial.outlineText) {
    patch.outline = textToOutline(next.outlineText)
  }
  return patch
}

async function save() {
  if (!canEdit.value || meta.metaSaving) return
  const patch = buildPatch()
  // 没有改动时「保存」本来就是禁用态；这里只兜底，不弹提示（2026-11-25 减少 toast）
  if (!Object.keys(patch).length) return
  const ok = await meta.saveMeta(patch)
  if (!ok) return
  // 保存成功/失败的 toast 由 metaStore.saveMeta 统一弹（2026-11-25 起页面内不再有提示条）
  // 保存后后端返回的就是新作品：草稿以它为新基准（不再有未保存改动）
  resetDraft()
}
</script>

<style scoped>
/* 表单一行：上标签下控件（与设置页的紧凑口径一致，标签走 tertiary 小字） */
.work-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.work-field-label {
  font-size: 11px;
  line-height: 1.4;
  color: var(--color-text-tertiary);
}

.work-field-hint {
  font-size: 11px;
  line-height: 1.4;
  color: var(--color-text-muted);
}
</style>
