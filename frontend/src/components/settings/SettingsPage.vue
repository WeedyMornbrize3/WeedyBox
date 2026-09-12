<template>
  <!-- min-h-0 + overflow-y-auto：设置页内容会略高于可用高度（实测溢出 6px）。
       由于 MainLayout 的 router-view 改为 overflow-hidden 以支撑 TODO 页的 flex 高度链，
       这里必须自己滚，否则底部会被裁掉。 -->
  <div class="max-w-[700px] mx-auto w-full select-none min-h-0 overflow-y-auto scrollbar-theme">
    <h1 class="text-2xl font-semibold text-primary mb-5 tracking-[0.5px]">
      <slot name="title">设置</slot>
    </h1>
    <div class="flex flex-col gap-5">
      <slot />
    </div>
  </div>
</template>