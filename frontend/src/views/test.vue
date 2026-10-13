<!-- frontend/src/views/test.vue
     路由 /testing 的视图层：只做挂载，实际内容在 components/test/MetaApiTestCard.vue
     （与 TodoPage.vue → components/todo/TodoList.vue 保持同一结构）。 -->
<template>
  <MetaApiTestCard />
</template>

<script setup lang="ts">
import MetaApiTestCard from '../components/test/MetaApiTestCard.vue'
</script>
