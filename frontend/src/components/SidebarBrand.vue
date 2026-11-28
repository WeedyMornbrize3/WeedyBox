<!-- frontend/src/components/SidebarBrand.vue
    侧边栏的品牌区（用户 2026-11-28 定稿）：
      · **只在展开态显示** logo + 名称（折叠后收起，见 Sidebar.vue 的 n-collapse-transition）；
      · logo 图形来自 config/brand.ts 的接口：LOGO_SRC（图片，可为空）优先，
        为空时退回 LOGO_ICON（UnoCSS 的 lucide 图标类）——换 logo 只改那一个文件；
      · **直接复用 router 内容区卡片 header 的样式口径**（用户 2026-11-28
        「整体调大 与 router 大小一致」「直接复用 router ui css 样式」）：
          - 行高 50px  = n-card header（paddingSmall 12px ×2 + 内容 26px）
          - 图标 20px  = .icon-md（卡片标题图标同尺寸）
          - 文字 15px  = Card.titleFontSizeMedium + fontWeight 600
          - 图标与文字间距 10px = 卡片 header 里 `.gap-2.5`
          - 左右内边距 12px = Card.paddingSmall，与卡片 header 同一内缩
    折叠控件不在这里：它是导航区顶部那一行的普通图标按钮（Sidebar.vue）。 -->
<template>
  <div class="sidebar-brand">
    <span class="sidebar-brand-mark" :style="markStyle">
      <img
        v-if="LOGO_SRC"
        :src="LOGO_SRC"
        :alt="APP_NAME"
        class="w-full h-full object-contain"
        draggable="false"
      />
      <span v-else :class="LOGO_ICON" class="icon-md" aria-hidden="true" />
    </span>
    <span class="sidebar-brand-name">{{ APP_NAME }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { APP_NAME, LOGO_ICON, LOGO_SIZE, LOGO_SRC } from '../config/brand'

/** logo 盒子尺寸由 config/brand.ts 的 LOGO_SIZE 决定（换图/换尺寸不用改样式） */
const markStyle = computed(() => ({
  width: `${LOGO_SIZE}px`,
  height: `${LOGO_SIZE}px`,
  fontSize: `${LOGO_SIZE}px`,
}))
</script>

<style scoped>
/* 尺寸全部对齐 router 内容区的卡片 header（见文件顶部注释）：
   50 行高 / 20 图标 / 15 文字 / 10 间距 / 12 内边距。 */
.sidebar-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  /* min-width: 0 保证名称能用省略号收窄，不把侧栏撑宽 */
  min-width: 0;
  height: 50px;
  padding: 0 12px;
  color: var(--color-brand-text);
}

.sidebar-brand-mark {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

/* 名称：字号/字重/颜色与 n-card 标题同一套（15px / 600 / text-primary） */
.sidebar-brand-name {
  overflow: hidden;
  font-size: 18px;
  font-weight: 600;
  line-height: 1.4;
  letter-spacing: 0;
  white-space: nowrap;
  text-overflow: ellipsis;
  color: var(--color-text-primary);
}
</style>
