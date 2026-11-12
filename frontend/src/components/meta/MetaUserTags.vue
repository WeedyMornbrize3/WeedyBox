<!-- frontend/src/components/meta/MetaUserTags.vue
     作品详细页里的「我的标签」：用户自己添加的标签（存后端 work_tags 表），
     与 DLsite 抓来的 tags 分开显示 —— 抓取标签会随元数据刷新变，自己加的标签不会丢。
     数据与请求都在 stores/metaStore.ts（customTags / addCustomTag / removeCustomTag）。 -->
<template>
  <div class="flex flex-col gap-2">
    <div class="flex items-center gap-2">
      <span class="i-lucide-tags icon-sm section-icon" aria-hidden="true" />
      <h3 class="m-0 text-sm font-medium text-primary">我的标签</h3>
      <span class="text-xs text-tertiary">{{ meta.customTags.length }} 个</span>
    </div>

    <form class="flex items-end gap-2" @submit.prevent="submit">
      <label class="flex-1 min-w-0 flex flex-col gap-1.5">
        <span class="text-xs text-secondary">添加标签（最多 32 字，回车直接添加）</span>
        <input
          v-model="draft"
          type="text"
          class="input-theme w-full"
          placeholder="例如：已通关 / 想再玩 / 神作"
          maxlength="32"
          autocomplete="off"
          spellcheck="false"
        />
      </label>
      <button type="submit" class="btn-brand flex items-center gap-1.5 shrink-0" :disabled="busy || !draft.trim()">
        <span :class="busy ? 'i-lucide-loader-circle spin' : 'i-lucide-plus'" class="icon-sm" aria-hidden="true" />
        添加
      </button>
    </form>

    <MetaMessage v-if="meta.customTagError" tone="error" :text="meta.customTagError" />

    <ul v-if="meta.customTags.length" class="list-none m-0 p-0 flex flex-wrap gap-1.5">
      <li v-for="tag in meta.customTags" :key="tag" class="chip cursor-default flex items-center gap-1">
        {{ tag }}
        <button type="button" class="tag-remove" :aria-label="`移除标签「${tag}」`" :title="`移除「${tag}」`" @click="remove(tag)">
          <span class="i-lucide-x icon-xs" aria-hidden="true" />
        </button>
      </li>
    </ul>
    <p v-else class="m-0 text-xs text-tertiary">
      还没有自己的标签 —— 加上后会存进本地数据库，重新抓取元数据也不会丢。
    </p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useMetaStore } from '../../stores/metaStore'
import MetaMessage from './MetaMessage.vue'

const meta = useMetaStore()
const draft = ref('')
const busy = ref(false)

async function submit() {
  const text = draft.value.trim()
  if (!text || busy.value) return
  busy.value = true
  try {
    if (await meta.addCustomTag(text)) draft.value = ''
  } finally {
    busy.value = false
  }
}

async function remove(tag: string) {
  if (busy.value) return
  busy.value = true
  try {
    await meta.removeCustomTag(tag)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
/* 标签上的小 ×：跟 chip 的配色走，静息淡一点、hover 提亮 */
.tag-remove {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: none;
  background: transparent;
  color: inherit;
  opacity: 0.55;
  cursor: pointer;
  transition: opacity var(--motion-fast) var(--ease-enter);
}

.tag-remove:hover,
.tag-remove:focus-visible {
  opacity: 1;
}
</style>
