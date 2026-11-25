<template>
  <div class="flex h-screen w-screen bg-primary pt-[30px]">
    <TitleBar />
    <Sidebar class="mt-[30px] !h-[calc(100vh-30px)]" />
    <!-- 内容左内边距 = 侧栏当前宽度（aside 是 fixed，盖在内容上方，不给让位就会压住内容）。
         p-6 的 24px 才是内容与侧栏之间的间距，两者合起来随折叠/展开一起过渡。 -->
    <router-view 
      class="flex-1 flex flex-col min-w-0 min-h-0 overflow-hidden bg-primary transition-[padding-left] duration-300 ease-[cubic-bezier(0.4,0,0.2,1)] p-6"
      :style="{ paddingLeft: sidebarStore.sidebarWidth + 'px' }"
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