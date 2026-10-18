<!-- frontend/src/components/meta/MetaKeyValueList.vue
     两列键值表（dt/dd 语义化）。三档密度：
       dense      → outline 那种多行表，行距更紧
       scrollable → 限高 220px 自己滚（长 summary / 长 outline 不至于把卡片撑爆）
     限高与滚动条写在一起：global.css 用 !important 隐藏了全局滚动条，
     限高容器必须同时配 scrollbar-theme，否则用户看不到"这里还能滚"。 -->
<template>
  <dl
    class="grid grid-cols-[auto_1fr] m-0 text-xs"
    :class="[dense ? 'gap-x-4 gap-y-1' : 'gap-x-4 gap-y-1.5', scrollable ? 'max-h-[220px] overflow-y-auto scrollbar-theme pr-2' : '']"
  >
    <template v-for="row in rows" :key="`${row.label}:${row.value}`">
      <dt class="text-tertiary whitespace-nowrap">{{ row.label }}</dt>
      <dd class="m-0 text-secondary min-w-0 break-words">{{ row.value }}</dd>
    </template>
  </dl>
</template>

<script setup lang="ts">
import type { MetaRow } from '../../utils/metaFormat'

withDefaults(defineProps<{ rows: MetaRow[]; dense?: boolean; scrollable?: boolean }>(), {
  dense: false,
  scrollable: false,
})
</script>
