<!-- frontend/src/components/meta/MetaSection.vue
     统一卡片外壳：header（图标 + 标题 + 右侧动作）+ 内容区。
     作品库与调试页的所有卡片都走它，保证「卡片长相」只有一处定义 ——
     页面只决定里面放什么、放几个，这就是「方便重新排列」的前提。
     没有内容区时（只有标题 + 动作的顶栏）不渲染空的 body，并把上下内边距放到
     header 自己身上 —— 否则空 body 的 p-* 会把卡片撑高、标题看着头重脚轻。 -->
<template>
  <section class="card-flat">
    <header class="section-header px-5" :class="$slots.default ? 'pt-4' : 'py-3'">
      <span v-if="icon" :class="icon" class="icon-md section-icon" aria-hidden="true" />
      <h2 class="section-title">
        <slot name="title">{{ title }}</slot>
      </h2>
      <div v-if="$slots.actions" class="ml-auto flex items-center gap-2 shrink-0">
        <slot name="actions" />
      </div>
    </header>

    <div v-if="$slots.default" :class="bodyClass">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
// 纯外壳组件：不持有状态，标题/动作都走插槽，方便组合出"带标签的标题 + 右边一个链接"这类形态。
withDefaults(
  defineProps<{
    /** 左侧图标类名，如 i-lucide-plug */
    icon?: string
    /** 纯文本标题（需要自定义标题时用 #title 插槽） */
    title?: string
    /** 内容区的类名，默认 p-5 */
    bodyClass?: string
  }>(),
  { icon: '', title: '', bodyClass: 'p-5' }
)
</script>
