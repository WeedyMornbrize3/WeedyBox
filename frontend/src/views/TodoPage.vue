<!-- frontend/src/views/TodoPage.vue -->
<template>
  <!-- h-full + min-h-0 + flex flex-col：撑满 router-view，让内部卡片能用 flex-1 吃掉剩余高度 -->
  <div class="h-full min-h-0 flex flex-col max-w-3xl mx-auto w-full">
    <TodoList />
  </div>
</template>

<script setup lang="ts">
import TodoList from '../components/todo/TodoList.vue'
</script>