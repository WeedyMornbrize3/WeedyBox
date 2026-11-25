<!-- frontend/src/components/meta/MetaTags.vue
     标签云。这里不可点（DLsite 的 tag 我们没做筛选路由），所以用普通 n-tag，
     不给 checkable（避免看起来能点进去）。
     limit = 最多显示几个（0 = 不限）；超出的折成一个「+N」tag，鼠标悬停给出提示。
     数量由设置页「标签显示数量」控制（stores/App.ts 的 tagLimit）。 -->
<template>
  <ul v-if="tags.length" class="list-none m-0 p-0 flex flex-wrap gap-1.5">
    <li v-for="tag in visibleTags" :key="tag">
      <n-tag size="small" :bordered="true">{{ tag }}</n-tag>
    </li>
    <li
      v-if="hiddenCount > 0"
      :title="`还有 ${hiddenCount} 个标签未显示，可在设置里调整「标签显示数量」`"
    >
      <n-tag size="small" :bordered="true" type="primary">+{{ hiddenCount }}</n-tag>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NTag } from 'naive-ui'

const props = withDefaults(defineProps<{ tags?: string[]; limit?: number }>(), {
  tags: () => [],
  limit: 0,
})

const visibleTags = computed(() => (props.limit > 0 ? props.tags.slice(0, props.limit) : props.tags))
const hiddenCount = computed(() => Math.max(0, props.tags.length - visibleTags.value.length))
</script>
