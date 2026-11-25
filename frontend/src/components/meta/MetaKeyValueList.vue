<!-- frontend/src/components/meta/MetaKeyValueList.vue
     键值表：改用 naive 的 n-descriptions（单列 + 带边框）。
     两档密度：dense → size=small；scrollable → 限高 220px 自己滚（长 summary / 长 outline
     不至于把卡片撑爆）。限高容器必须同时配 scrollbar-theme ——
     global.css 用 !important 隐藏了全局滚动条，不配就看不到"这里还能滚"。 -->
<template>
  <div :class="scrollable ? 'max-h-[220px] overflow-y-auto scrollbar-theme pr-2' : ''">
    <n-descriptions :column="1" :size="dense ? 'small' : 'medium'" bordered label-placement="left">
      <n-descriptions-item v-for="row in rows" :key="`${row.label}:${row.value}`" :label="row.label">
        <span class="break-words">{{ row.value }}</span>
      </n-descriptions-item>
    </n-descriptions>
  </div>
</template>

<script setup lang="ts">
import { NDescriptions, NDescriptionsItem } from 'naive-ui'
import type { MetaRow } from '../../utils/metaFormat'

withDefaults(defineProps<{ rows: MetaRow[]; dense?: boolean; scrollable?: boolean }>(), {
  dense: false,
  scrollable: false,
})
</script>
