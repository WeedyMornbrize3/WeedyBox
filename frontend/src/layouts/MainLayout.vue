<template>
  <div class="flex h-screen w-screen bg-primary pt-[30px] overflow-hidden">
    <TitleBar />
    <Sidebar class="mt-[30px] !h-[calc(100vh-30px)]" />
    <router-view 
      class="flex-1 min-w-0 min-h-0 overflow-y-auto bg-primary transition-[padding-left] duration-300 ease-[cubic-bezier(0.4,0,0.2,1)] p-6"
      :style="{ paddingLeft: sidebarStore.mainPaddingLeft + 'px' }"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useSidebarStore } from '../stores/Sidebar'
import Sidebar from '../components/Sidebar.vue'
import TitleBar from '../components/TitleBar.vue'

const sidebarStore = useSidebarStore()

// 组件挂载时恢复侧边栏状态
onMounted(() => {
  sidebarStore.restoreState()
})
</script>

<style scoped>
@media (max-width: 768px) {
  .main-content {
    padding: 16px;
  }
}
</style>