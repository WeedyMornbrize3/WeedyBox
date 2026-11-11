<!-- frontend/src/components/meta/MetaPager.vue
     搜索结果分页条：上一页/下一页 + 页码（前 3 页、当前页、后 3 页，中间省略号）+ 输入页码跳转。
     状态在 stores/metaStore.ts（candidatePage / setCandidatePage），这里只 emit 目标页码 ——
     「翻到哪页才下哪页的预览图」的逻辑因此留在 store 里，组件不碰数据。
     页码按钮复用 chips 的 chip / chip-active，不另造一套外观。 -->
<template>
  <div v-if="pageCount > 1" class="flex items-center justify-between gap-2 flex-wrap">
    <div class="flex items-center gap-1 flex-wrap">
      <span v-if="total > 0" class="text-xs text-tertiary whitespace-nowrap mr-1 tabular-nums">
        共 {{ total }} 个结果
      </span>
      <button
        type="button"
        class="chip disabled:opacity-40 disabled:cursor-not-allowed"
        :disabled="page <= 1"
        @click="emit('change', page - 1)"
      >
        上一页
      </button>

      <template v-for="(item, index) in items" :key="index">
        <span v-if="item === '...'" class="px-0.5 text-xs text-tertiary" aria-hidden="true">…</span>
        <button
          v-else
          type="button"
          class="chip tabular-nums"
          :class="item === page ? 'chip-active' : ''"
          :aria-current="item === page ? 'page' : undefined"
          @click="emit('change', item)"
        >
          {{ item }}
        </button>
      </template>

      <button
        type="button"
        class="chip disabled:opacity-40 disabled:cursor-not-allowed"
        :disabled="page >= pageCount"
        @click="emit('change', page + 1)"
      >
        下一页
      </button>
    </div>

    <!-- 输入页码跳转：非法输入直接忽略；超范围按 1 / 最后一页收口 -->
    <form class="flex items-center gap-1.5 text-xs text-tertiary" @submit.prevent="jump">
      <label for="meta-pager-jump">跳至</label>
      <input
        id="meta-pager-jump"
        v-model="draft"
        type="text"
        inputmode="numeric"
        placeholder="页码"
        aria-label="跳转到指定页码"
        class="w-12 h-7 px-2 rounded-md bg-secondary border-theme outline-none text-xs text-primary text-center tabular-nums focus:border-brand"
      />
      <span>页</span>
      <button type="submit" class="chip">跳转</button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = withDefaults(defineProps<{ page: number; pageCount: number; total?: number }>(), {
  total: 0,
})

const emit = defineEmits<{ change: [page: number] }>()

/** 输入框里的页码草稿（提交后才生效，避免每敲一个字就翻页） */
const draft = ref('')

/** 要显示哪些页码：前 3 页 + 当前页 + 后 3 页，中间断开处插省略号 */
const items = computed<Array<number | '...'>>(() => {
  const picked = new Set<number>()
  for (let p = 1; p <= Math.min(3, props.pageCount); p += 1) picked.add(p)
  for (let p = Math.max(1, props.pageCount - 2); p <= props.pageCount; p += 1) picked.add(p)
  if (props.page >= 1 && props.page <= props.pageCount) picked.add(props.page)

  const pages = [...picked].sort((a, b) => a - b)
  const out: Array<number | '...'> = []
  let previous = 0
  for (const page of pages) {
    if (previous && page - previous > 1) out.push('...')
    out.push(page)
    previous = page
  }
  return out
})

function jump() {
  const value = Number.parseInt(draft.value.trim(), 10)
  if (!Number.isFinite(value)) return
  const target = Math.min(Math.max(1, value), props.pageCount)
  draft.value = ''
  emit('change', target)
}
</script>
