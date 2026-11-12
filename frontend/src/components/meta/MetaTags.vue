<!-- frontend/src/components/meta/MetaTags.vue
     标签云。这里不可点（DLsite 的 tag 我们没做筛选路由），所以 chip 配 cursor-default，
     避免鼠标变手型误导用户以为能点进去。
     limit = 最多显示几个（0 = 不限）；超出的折成一个「+N」chip，鼠标悬停给出提示。
     数量由设置页「标签显示数量」控制（stores/App.ts 的 tagLimit）。 -->
<template>
  <ul v-if="tags.length" class="list-none m-0 p-0 flex flex-wrap gap-1.5">
    <li v-for="tag in visibleTags" :key="tag" class="chip cursor-default">{{ tag }}</li>
    <li
      v-if="hiddenCount > 0"
      class="chip cursor-default"
      :title="`还有 ${hiddenCount} 个标签未显示，可在设置里调整「标签显示数量」`"
    >
      +{{ hiddenCount }}
    </li>
  </ul>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ tags?: string[]; limit?: number }>(), {
  tags: () => [],
  limit: 0,
})

const visibleTags = computed(() => (props.limit > 0 ? props.tags.slice(0, props.limit) : props.tags))
const hiddenCount = computed(() => Math.max(0, props.tags.length - visibleTags.value.length))
</script>
