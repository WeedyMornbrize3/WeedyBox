<!-- frontend/src/components/meta/MetaCandidateList.vue
     模糊搜索候选列表：小封面 + 名称 + 编号 + 「详情」。
     点「详情」emit 编号，由页面决定"精确抓取"还是别的行为 —— 列表本身不关心。
     行分隔用 border-b-theme-light + last:border-none，最后一行不留尾巴。 -->
<template>
  <ul class="list-none m-0 p-0">
    <li
      v-for="item in candidates"
      :key="item.id"
      class="flex items-center gap-3 px-5 py-3 border-b-theme-light last:border-none"
    >
      <MetaCover :src="item.cover" :alt="item.name" size="sm" />
      <div class="flex-1 min-w-0">
        <p class="m-0 text-sm text-primary truncate" :title="item.name">{{ item.name }}</p>
        <p class="m-0 text-xs text-tertiary font-mono">{{ item.id }}</p>
      </div>
      <button type="button" class="btn-ghost shrink-0 text-xs" @click="emit('select', item.id)">详情</button>
    </li>
  </ul>
</template>

<script setup lang="ts">
import type { SearchCandidate } from '../../services/metaService'
import MetaCover from './MetaCover.vue'

defineProps<{ candidates: SearchCandidate[] }>()

const emit = defineEmits<{ select: [id: string] }>()
</script>
