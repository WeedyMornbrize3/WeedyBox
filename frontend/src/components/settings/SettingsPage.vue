<template>
  <!-- min-h-0 + overflow-y-auto：设置页内容会略高于可用高度。
       由于 MainLayout 的 router-view 是 overflow-hidden（为支撑 TODO 页的 flex 高度链），
       这里必须自己滚，否则底部会被裁掉。上下边缘按滚动位置渐隐（v-scroll-fade）。 -->
  <div v-scroll-fade class="scroll-fade settings-page select-none min-h-0 overflow-y-auto scrollbar-theme">
    <h1 class="settings-title">
      <slot name="title">设置</slot>
    </h1>
    <div class="settings-container card-stack">
      <slot />
    </div>
  </div>
</template>
