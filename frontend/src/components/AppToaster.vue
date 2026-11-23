<!-- frontend/src/components/AppToaster.vue
     全局轻提示（toast）容器 —— 唯一一份，挂在 App.vue 根部。
     组件库：vue-sonner（2026-11-23 引入，替代 TODO 页原来手写的反馈条）。
     为什么用它：
       - 浮层定位、排队、滑出、悬停展开、读屏播报都由库负责，不再每页各写一套；
       - 原手写反馈条是布局里的一个 flex 子项，出现/消失会把列表往下推 30 多像素，
         改成浮层后完全不参与布局（用户要求「优化 todo 的提示」的根因）。
     外观完全走设计 token（global.css），所以浅色/深色主题自动跟着换，组件里没有硬编码色值。
     每条提示顶端有一条**定时条**（::before 的 2px 语义色横线）：时长来自 toast 的内联变量
     `--toast-duration`（utils/toast.ts 按同一个 duration 写入），悬停/聚焦时与 sonner 的
     自动关闭计时器一起暂停；loading 类型没有剩余时间，不显示条。
     调用入口统一是 utils/toast.ts，不要在业务代码里直接 import vue-sonner。 -->
<template>
  <Toaster
    class="app-toaster"
    :theme="theme.isDark ? 'dark' : 'light'"
    position="top-center"
    :offset="{ top: '42px' }"
    :gap="8"
    :duration="2400"
    :visible-toasts="3"
    :close-button="false"
    :rich-colors="false"
    :style="{ '--width': '320px' }"
    container-aria-label="操作提示"
  >
    <!-- 图标沿用项目统一的 lucide 线性图标（UnoCSS 运行时零请求），颜色在下方按类型映射 -->
    <template #success-icon>
      <span class="i-lucide-circle-check-big icon-sm" aria-hidden="true" />
    </template>
    <template #error-icon>
      <span class="i-lucide-circle-alert icon-sm" aria-hidden="true" />
    </template>
    <template #warning-icon>
      <span class="i-lucide-triangle-alert icon-sm" aria-hidden="true" />
    </template>
    <template #info-icon>
      <span class="i-lucide-info icon-sm" aria-hidden="true" />
    </template>
    <template #loading-icon>
      <span class="i-lucide-loader-circle icon-sm spin" aria-hidden="true" />
    </template>
  </Toaster>
</template>

<script setup lang="ts">
import { Toaster } from 'vue-sonner'
import { useThemeStore } from '../stores/Theme'

/** 主题跟着应用设置走（system 模式在 store 里已解析成实际生效值） */
const theme = useThemeStore()
</script>

<style>
/* =============================================================================
   vue-sonner 外观映射（非 scoped：它把 DOM 渲染在自己的容器里）
   选择器统一带 [data-sonner-toaster].app-toaster[data-sonner-theme]（0,3,0），
   比库自带的 [data-sonner-toaster][data-sonner-theme='light'|'dark']（0,2,0）高一级，
   因此不依赖样式表加载顺序，主题切换也不会被库的默认色盖掉。
   ============================================================================= */

/* ---------- 容器：宽度 / 圆角 / 字体 ---------- */
[data-sonner-toaster].app-toaster[data-sonner-theme] {
  --width: 320px;
  --border-radius: 10px;
  /* 卡片底色、描边、文字色：全部来自 global.css 的三层 token */
  --normal-bg: var(--color-bg-card);
  --normal-bg-hover: var(--color-bg-card);
  --normal-border: var(--color-border);
  --normal-border-hover: var(--color-border);
  --normal-text: var(--color-text-primary);
  font-family: inherit;
}

/* ---------- 单条：更紧凑的卡片（Flat：1px 描边 + 轻浮层阴影） ---------- */
[data-sonner-toaster].app-toaster [data-sonner-toast][data-styled="true"] {
  padding: 10px 12px;
  gap: 8px;
  font-size: 13px;
  border-radius: var(--border-radius);
  box-shadow: var(--color-shadow-heavy);
  /* 有描述时要让图标与标题的第一行对齐，而不是整体居中 */
  align-items: flex-start;
}

[data-sonner-toaster].app-toaster [data-sonner-toast] [data-icon] {
  /* 16×16 的容器由库固定，这里只把与标题的对齐从"容器居中"抬到"首行居中" */
  margin-top: 1px;
}

/* ---------- 标题 / 描述 ---------- */
[data-sonner-toaster].app-toaster [data-sonner-toast] [data-title] {
  font-size: 13px;
  line-height: 1.5;
  color: var(--color-text-primary);
}

[data-sonner-toaster].app-toaster [data-sonner-toast] [data-description] {
  font-size: 12px;
  line-height: 1.45;
  color: var(--color-text-tertiary);
}

/* ---------- 图标颜色：按提示类型取语义色（不用 richColors 的整块底色，
   与项目「Flat / 只用 1px 描边区分层级」的取向一致） ---------- */
[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="success"] [data-icon] {
  color: var(--color-success);
}

[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="error"] [data-icon] {
  color: var(--color-danger);
}

[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="warning"] [data-icon] {
  color: var(--color-warning);
}

[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="info"] [data-icon] {
  color: var(--color-info);
}

[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="loading"] [data-icon] {
  color: var(--color-primary);
}

/* =============================================================================
   顶端定时条：直观表达「这条提示还剩多久自动消失」
   - 时长由 utils/toast.ts 以 **内联变量** `--toast-duration` 传进来，与 toast 自己的
     duration 是同一个值（改一处文案时不会出现"条走完了提示还挂着"）；
   - loading 类型没有剩余时间概念（要等操作结束），不显示条，只留转圈图标；
   - 悬停 / 键盘聚焦时 sonner 会暂停自动关闭计时器，这里的条也一起暂停，
     否则进度会先走完、提示却还在（用户一眼就能看出不一致）。
   ⚠️ 为什么挂在 [data-content] 的伪元素上、而不是 toast 自己的 ::before / animation：
     sonner 的 `[data-sonner-toast]::before` 是滑动/移除动画的垫层，toast 元素本身还有
     `swipe-out-*` 的 animation；我们的选择器权重更高，写上去会把库的这两套动画顶掉。
     [data-content] 是它的子元素、绝对定位的包含块仍是 toast 自己，正好能用。
   ============================================================================= */
[data-sonner-toaster].app-toaster [data-sonner-toast][data-styled="true"] [data-content]::before {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  /* 条贴在 1px 描边内侧，圆角跟着卡片走 */
  border-radius: calc(var(--border-radius) - 1px) calc(var(--border-radius) - 1px) 0 0;
  background: var(--toast-accent, var(--color-primary));
  transform-origin: left center;
  animation: app-toast-countdown var(--toast-duration, 0ms) linear forwards;
  pointer-events: none;
}

/* 条的颜色跟图标同源（都是语义色），不用变量重复写一遍 */
[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="success"] {
  --toast-accent: var(--color-success);
}

[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="error"] {
  --toast-accent: var(--color-danger);
}

[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="warning"] {
  --toast-accent: var(--color-warning);
}

[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="info"] {
  --toast-accent: var(--color-info);
}

[data-sonner-toaster].app-toaster [data-sonner-toast][data-type="loading"] [data-content]::before {
  display: none;
}

[data-sonner-toaster].app-toaster:hover [data-sonner-toast][data-styled="true"] [data-content]::before,
[data-sonner-toaster].app-toaster:focus-within [data-sonner-toast][data-styled="true"] [data-content]::before {
  animation-play-state: paused;
}

@keyframes app-toast-countdown {
  from {
    transform: scaleX(1);
  }
  to {
    transform: scaleX(0);
  }
}
</style>
