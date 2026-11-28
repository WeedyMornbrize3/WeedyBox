<!-- frontend/src/components/stats/StatsSplitBar.vue
     数据分析页卡片之间的分割条（2026-11-28 用户要求
     「参考work的split 样式 用来排布不同卡片之间的大小」「可以通过split调整排布」）。

     与 /works 那条 WorksSplitBar 同源（保持一致手感）：
       · 线本身 1px（宽度/高度由 n-split 的 resize-trigger-size 决定，这里不自己设）；
       · 静息**透明**、与页面底色一致，视觉上不切断卡片氛围；
       · hover / 拖动中亮成品牌文字色，告诉使用者「这里能拖」；
       · 1px 太细抓不住，用伪元素向两侧各外扩 4px（共 9px）做热区。

     与 /works 的差别：那里只有竖线，这里支持两个方向 ——
       · horizontal（默认）：横线，分隔**上下**两块，拖动改高度（row-resize）；
       · vertical：竖线，分隔**左右**两块，拖动改宽度（col-resize）。

     为什么两端不做留白/圆角：全站卡片是直角的，分割线也应当贯通整宽，
     加圆角会变成"胶囊"这种装饰物，与 Flat Design 的层级表达冲突。 -->
<template>
  <span
    class="stats-split-bar"
    :class="[`stats-split-bar--${direction}`, { 'stats-split-bar--active': dragging }]"
    aria-hidden="true"
    @mousedown="onMouseDown"
  />
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue'

withDefaults(
  defineProps<{
    /** horizontal = 横线（上下拖）；vertical = 竖线（左右拖） */
    direction?: 'horizontal' | 'vertical'
  }>(),
  { direction: 'horizontal' }
)

/**
 * 拖动中持续高亮：按住后鼠标很可能滑出那条 1px 的线，
 * 只靠 :hover 会中途闪回透明，看起来像「拖着拖着线没了」。
 */
const dragging = ref(false)

function onMouseUp() {
  dragging.value = false
  document.removeEventListener('mouseup', onMouseUp)
}

function onMouseDown() {
  dragging.value = true
  document.addEventListener('mouseup', onMouseUp)
}

onUnmounted(() => document.removeEventListener('mouseup', onMouseUp))
</script>

<style scoped>
.stats-split-bar {
  display: block;
  position: relative;
  /* 静息透明：与页面底色一致，不画任何线 */
  background: transparent;
  transition: background-color var(--motion-fast) var(--ease-enter);
}

/* 横线：占满宽、1px 高由 n-split 给，上下外扩热区 */
.stats-split-bar--horizontal {
  width: 100%;
  height: 100%;
  cursor: row-resize;
}

.stats-split-bar--horizontal::before {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  top: -4px;
  bottom: -4px;
}

/* 竖线：占满高、1px 宽由 n-split 给，左右外扩热区 */
.stats-split-bar--vertical {
  width: 100%;
  height: 100%;
  cursor: col-resize;
}

.stats-split-bar--vertical::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: -4px;
  right: -4px;
}

/* hover 或按住拖动时才亮起来 */
.stats-split-bar:hover,
.stats-split-bar--active {
  background: var(--color-brand-text);
}
</style>
