<template>
  <!-- min-h-0 + overflow-y-auto：设置页内容会略高于可用高度。
       由于 MainLayout 的 router-view 是 overflow-hidden（为支撑 TODO 页的 flex 高度链），
       这里必须自己滚，否则底部会被裁掉。 -->
  <div class="settings-page select-none min-h-0 overflow-y-auto scrollbar-theme">
    <h1 class="settings-title">
      <slot name="title">设置</slot>
    </h1>
    <div class="settings-container">
      <slot />
    </div>
  </div>
</template>
