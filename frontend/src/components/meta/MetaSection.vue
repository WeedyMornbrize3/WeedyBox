<!-- frontend/src/components/meta/MetaSection.vue
     统一卡片外壳：header（图标 + 标题 + 右侧动作）+ 内容区。
     2026-11-24 起改用 naive 的 n-card 承载「卡片长相」（底色/描边/内边距
     全部来自 styles/naiveTheme.ts 的 Card 段），这里只负责头部结构与插槽。
     没有内容区时（只有标题 + 动作的顶栏）n-card 不会渲染空 body。

     2026-11-25：全站卡片改为方形（Card.borderRadius = 0）；卡片之间要紧凑时，
     把外层容器加上 global.css 的 `.card-stack`，相邻卡片只共享一条 1px 边框。
     这里不再有 square / hover 才显示描边的分支。 -->
<template>
  <n-card
    size="small"
    :bordered="true"
    :content-style="hasBody ? undefined : { display: 'none' }"
  >
    <template #header>
      <div class="flex items-center gap-2.5">
        <span v-if="icon" :class="icon" class="icon-md text-brand flex-center" aria-hidden="true" />
        <h2 class="m-0 text-[15px] font-semibold text-primary">
          <slot name="title">{{ title }}</slot>
        </h2>
      </div>
    </template>

    <template v-if="$slots.actions" #header-extra>
      <div class="flex items-center gap-2 shrink-0">
        <slot name="actions" />
      </div>
    </template>

    <div v-if="$slots.default" :class="bodyClass">
      <slot />
    </div>
  </n-card>
</template>

<script setup lang="ts">
import { computed, useSlots } from 'vue'
import { NCard } from 'naive-ui'

// 只有标题 + 右侧动作的顶栏卡片（如「游戏库」）：n-card 的 content 一定存在，
// 但这里没有 body 内容时会把卡片白白撑高 16px，所以直接把 content 隐藏掉。
const slots = useSlots()
const hasBody = computed(() => Boolean(slots.default))

// 纯外壳组件：不持有状态，标题/动作都走插槽，方便组合出"带标签的标题 + 右边一个链接"这类形态。
withDefaults(
  defineProps<{
    /** 左侧图标类名，如 i-lucide-plug */
    icon?: string
    /** 纯文本标题（需要自定义标题时用 #title 插槽） */
    title?: string
    /** 内容区追加的类名（n-card 已提供内边距，默认不再加 p-5） */
    bodyClass?: string
  }>(),
  { icon: '', title: '', bodyClass: '' }
)
</script>
