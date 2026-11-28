<!-- frontend/src/components/WorksSplitBar.vue
     分割条的「可见线」+「加宽热区」，放在 n-split 的 #resize-trigger 插槽里用。

     用户 2026-11-28：「为 work 界面的左右两侧添加 面板分割 Split 供使用者可以自行调整
     work 卡片显示范围  Split 条调为 1px」——视觉上就是一条 1px 的竖线（宽度由
     n-split 的 resize-trigger-size 决定，这里不再自己设宽，避免两处打架）；
     但 1px 的拖动热区几乎抓不住，所以用伪元素向两侧各外扩 4px（共 9px）做热区：
     线本身仍是 1px，摸上去够得着。

     n-split 的 mousedown 监听在外层 wrapper 上，伪元素冒泡上来的事件同样能触发拖动，
     所以热区扩大不需要改 naive 的结构。

     配色（用户 2026-11-28：「split 条默认与背景颜色一致  hover时才显示出颜色」）：
       静息 = 透明，与页面底色完全一致，视觉上不存在，不切断 hero / 卡片的连续氛围；
       hover / 拖动中 = 品牌文字色，1px 线亮起来，告诉使用者「这里能拖」。
     静息不可见，所以 4px 的透明热区是唯一能被摸到的地方（线本身只有 1px）。 -->
<template>
  <span
    class="works-split-bar"
    :class="{ 'works-split-bar--active': dragging }"
    aria-hidden="true"
    @mousedown="onMouseDown"
  />
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue'

/**
 * 拖动中给它加个持续高亮：鼠标按住后很可能滑出那 1px 的线，
 * 只靠 :hover 会中途闪回透明，看起来像「拖着拖着线没了」。
 * 事件冒泡到 n-split 的外层 wrapper 照样能触发它的拖动逻辑，不影响手的操作。
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
.works-split-bar {
  display: block;
  position: relative;
  width: 100%;
  height: 100%;
  /* 静息透明：与页面底色一致，不画任何线（用户 2026-11-28） */
  background: transparent;
  /* 颜色反馈用 design token（全站 hover 统一口径，见 global.css 动效基调） */
  transition: background-color var(--motion-fast) var(--ease-enter);
  cursor: col-resize;
}

/* 透明热区：左右各外扩 4px，1px 的线也摸得着（不参与视觉） */
.works-split-bar::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: -4px;
  right: -4px;
}

/* hover 或按住拖动时才亮起来 */
.works-split-bar:hover,
.works-split-bar--active {
  background: var(--color-brand-text);
}
</style>
