<template>
  <!-- 首页：项目默认落在 /todo，这里只作过渡引导，不再是一行裸文本 -->
  <div class="h-full min-h-0 flex-col-center select-none">
    <div class="card-flat p-8 flex flex-col items-center gap-4 max-w-[420px] text-center">
      <span class="i-lucide-sprout icon-xl text-brand" aria-hidden="true" />
      <h2 class="text-lg font-semibold text-primary m-0">WeedyBox</h2>
      <p class="text-sm text-tertiary m-0 leading-relaxed">
        一个本地优先的待办工具。数据存在本机 SQLite，不联网、不上传。
      </p>
      <button type="button" class="btn-brand flex items-center gap-1.5" @click="goTodo">
        <span class="i-lucide-list-todo icon-sm" aria-hidden="true" />
        打开 TODO 列表
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
const goTodo = () => router.push('/todo')
</script>
