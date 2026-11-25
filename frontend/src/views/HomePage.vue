<template>
  <!-- 首页：项目默认落在 /todo，这里只作过渡引导 -->
  <div class="h-full min-h-0 flex-col-center select-none">
    <n-card size="medium" class="max-w-[420px] text-center">
      <div class="flex flex-col items-center gap-4">
        <span class="i-lucide-sprout icon-xl text-brand" aria-hidden="true" />
        <h2 class="text-lg font-semibold text-primary m-0">WeedyBox</h2>
        <p class="text-sm text-tertiary m-0 leading-relaxed">
          一个本地优先的待办工具。数据存在本机 SQLite，不联网、不上传。
        </p>
        <n-button type="primary" @click="goTodo">
          <template #icon>
            <span class="i-lucide-list-todo icon-sm" aria-hidden="true" />
          </template>
          打开 TODO 列表
        </n-button>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { NButton, NCard } from 'naive-ui'

const router = useRouter()
const goTodo = () => router.push('/todo')
</script>
