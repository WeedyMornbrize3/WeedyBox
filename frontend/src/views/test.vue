<template>
  <!-- 测试页（可删）：保留但统一到设计系统，不再使用行内硬编码颜色 -->
  <div class="h-full min-h-0 flex-col-center gap-3 select-none">
    <span class="i-lucide-flask-conical icon-xl text-tertiary" aria-hidden="true" />
    <p class="text-sm text-secondary m-0">测试页 · aloha</p>
    <button type="button" class="btn-ghost flex items-center gap-1.5" @click="goTodo">
      <span class="i-lucide-arrow-left icon-sm" aria-hidden="true" />
      返回 TODO
    </button>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
const goTodo = () => router.push('/todo')
</script>
