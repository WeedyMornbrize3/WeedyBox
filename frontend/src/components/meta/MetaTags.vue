<!-- frontend/src/components/meta/MetaTags.vue
     标签云。这里不可点（DLsite 的 tag 我们没做筛选路由），所以 chip 配 cursor-default，
     避免鼠标变手型误导用户以为能点进去。 -->
<template>
  <ul v-if="tags.length" class="list-none m-0 p-0 flex flex-wrap gap-1.5">
    <li v-for="tag in tags" :key="tag" class="chip cursor-default">{{ tag }}</li>
  </ul>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ tags?: string[] }>(), { tags: () => [] })
</script>
