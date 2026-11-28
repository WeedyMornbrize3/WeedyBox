<template>
  <div class="flex h-screen w-screen bg-primary pt-[30px]">
    <TitleBar />

    <!-- 侧栏与页面内容在**同一条 flex 行**里排布（用户 2026-11-28：「与其他 router ui 一起排布」）。
         2026-11-28 之前 aside 是 `fixed` 覆盖层，靠给 router-view 加 padding-left 让位；
         现在改成普通 flex 子项：宽度由 Sidebar 自己过渡（折叠 72 / 展开 250），
         内容区自然跟在后面，不再需要"让位补偿"这种两处耦合的写法。
         右内边距（16px）与 MainLayout 内容的 p-4 同一口径，侧栏与卡片之间是一条统一的留白。 -->
    <div class="flex-1 min-w-0 min-h-0 flex">
      <Sidebar />

      <!-- 内容区自己开滚动（各页面内部还有一层）。留白沿用原来的 p-6(24px)：
           侧栏不再是覆盖层，所以这 24px 就是「侧栏右边缘 → 内容卡片」的真实间距，
           与改动前完全一致（以前是 padding-left=侧栏宽 + p-6）。 -->
      <router-view
        class="flex-1 min-w-0 min-h-0 flex flex-col overflow-hidden bg-primary p-6"
      />
    </div>
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
