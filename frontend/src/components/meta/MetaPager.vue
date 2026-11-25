<!-- frontend/src/components/meta/MetaPager.vue
     分页条（2026-11-24 按 shadcn-vue 的 Pagination 重新设计，并取代此前那套自定义逻辑）：
       · 结构完全照 shadcn：上一页（chevron-left + 文字）→ 页码组（gap-1）→ 下一页（文字 + chevron-right）；
       · 当前页 = 描边按钮（shadcn 的 variant="outline"），其余页 = 幽灵按钮（variant="ghost"）；
       · 省略号用 MoreHorizontal 图标（lucide 里叫 ellipsis），首尾固定、当前页两侧各留 1 页；
       · **不再有**「共 N 个结果」、跳页输入、前 3/后 3 的旧窗口逻辑（用户 2026-11-24 要求忽略旧逻辑）；
       · 不再用 `v-if="pageCount > 1"` 隐藏 —— 只有 1 页时也显示（前后按钮禁用），
         避免「结果有、分页却不见」的观感问题。
     状态仍在 stores/metaStore.ts（candidatePage / setCandidatePage），这里只 emit 目标页码。
     实现用 naive 的 n-button（项目控件统一 naive），shadcn 的 outline 外观通过内联
     覆盖 --n-* 变量实现（naive 的主题变量是内联 cssVars，外部 class 覆盖不掉）。 -->
<template>
  <nav class="flex flex-wrap items-center justify-center gap-x-3 gap-y-2" aria-label="分页">
    <div class="flex items-center gap-1">
      <!-- shadcn PaginationPrevious：ghost + gap-1 -->
      <n-button quaternary size="small" :disabled="page <= 1" aria-label="上一页" @click="go(page - 1)">
        <template #icon>
          <span class="i-lucide-chevron-left icon-sm" aria-hidden="true" />
        </template>
        上一页
      </n-button>

      <!-- shadcn PaginationContent：flex items-center gap-1 -->
      <template v-for="(item, index) in items" :key="index">
        <!-- shadcn PaginationEllipsis：MoreHorizontal + sr-only -->
        <span
          v-if="item === '...'"
          class="flex items-center justify-center text-tertiary"
          :style="pagerBtnSize"
          aria-hidden="true"
        >
          <span class="i-lucide-ellipsis icon-sm" />
        </span>

        <!-- shadcn PaginationItem：当前页 outline，其余 ghost，size="icon"（这里 28px 见方） -->
        <n-button
          v-else
          size="small"
          :quaternary="item !== page"
          :bordered="item === page"
          :style="item === page ? activeLinkStyle : pagerBtnSize"
          :aria-current="item === page ? 'page' : undefined"
          :aria-label="`第 ${item} 页`"
          @click="go(item)"
        >
          {{ item }}
        </n-button>
      </template>

      <!-- shadcn PaginationNext：文字在左、箭头在右 -->
      <n-button quaternary size="small" :disabled="page >= totalPages" aria-label="下一页" @click="go(page + 1)">
        下一页
        <span class="i-lucide-chevron-right icon-sm" aria-hidden="true" />
      </n-button>
    </div>

    <!-- 跳页（用户 2026-11-24 要求恢复）：输入页码后回车或点「跳转」 -->
    <form class="flex items-center gap-1.5 text-xs text-tertiary" @submit.prevent="jump">
      <label :for="jumpId">跳至</label>
      <n-input
        v-model:value="draft"
        size="small"
        placeholder="页码"
        :input-props="{ id: jumpId, 'aria-label': '跳转到指定页码' }"
        :style="{ width: '52px' }"
      />
      <span>页</span>
      <n-button quaternary size="small" attr-type="submit">跳转</n-button>
    </form>
  </nav>
</template>

<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import type { CSSProperties } from 'vue'
import { NButton, NInput } from 'naive-ui'

const props = defineProps<{ page: number; pageCount: number }>()

const emit = defineEmits<{ change: [page: number] }>()

/** 「跳至某页」输入框的 id（label 关联用）与草稿值（提交后才生效） */
const jumpId = useId()
const draft = ref('')

/** 页码可能被外部传成 0（还没搜到结果时）：按 1 页处理，避免渲染出「第 0 页」 */
const totalPages = computed(() => Math.max(1, Math.floor(props.pageCount) || 1))
const currentPage = computed(() => Math.min(Math.max(1, props.page), totalPages.value))

/** 页码按钮/省略号统一 28px 见方（与 n-button small 同高，整行对齐） */
const pagerBtnSize = { width: '28px', height: '28px', padding: 0 } as CSSProperties

/**
 * 当前页的 shadcn outline 外观：卡片底 + 1px 描边。
 * naive 把主题变量写成**内联** cssVars，所以必须在同一个元素上以内联样式覆盖同名变量。
 */
const activeLinkStyle = {
  ...pagerBtnSize,
  '--n-color': 'var(--color-bg-card)',
  '--n-color-hover': 'var(--color-bg-hover)',
  '--n-color-pressed': 'var(--color-bg-tertiary)',
  '--n-border': '1px solid var(--color-border)',
  '--n-border-hover': '1px solid var(--color-border-dark)',
  '--n-border-pressed': '1px solid var(--color-border-dark)',
  '--n-text-color': 'var(--color-text-primary)',
  '--n-text-color-hover': 'var(--color-text-primary)',
  '--n-text-color-pressed': 'var(--color-text-primary)',
} as CSSProperties

/**
 * 页码窗口（shadcn 的 ellipsis 版本）：页数少就全列出；
 * 多了就固定首尾，当前页两侧各留 1 页，断开处插省略号。
 */
const items = computed<Array<number | '...'>>(() => {
  const count = totalPages.value
  const page = currentPage.value
  if (count <= 7) {
    return Array.from({ length: count }, (_, i) => i + 1)
  }

  const out: Array<number | '...'> = [1]
  const start = Math.max(2, page - 1)
  const end = Math.min(count - 1, page + 1)
  if (start > 2) out.push('...')
  for (let p = start; p <= end; p += 1) out.push(p)
  if (end < count - 1) out.push('...')
  out.push(count)
  return out
})

function go(target: number) {
  const next = Math.min(Math.max(1, target), totalPages.value)
  if (next === currentPage.value) return
  emit('change', next)
}

/** 跳页：非数字忽略；超出范围按 1 / 最后一页收口 */
function jump() {
  const value = Number.parseInt(draft.value.trim(), 10)
  draft.value = ''
  if (!Number.isFinite(value)) return
  go(value)
}
</script>
