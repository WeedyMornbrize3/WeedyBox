<!-- frontend/src/components/meta/MetaTagEditor.vue
     作品详情里的「标签」：抓取来的 tags 与用户自己加的标签**合并成一份列表**，
     改动先攒在 store 的编辑稿里，点「保存」才整份写进数据库。
     2026-11-24 起标签的增删改用 naive 的 n-dynamic-tags：
     点末尾「+」原地输入、点标签上的 × 删除，都交给组件本身处理；
     这里只负责把差值同步进 store（tagList / addTag / removeTag / saveTags）。 -->
<template>
  <div class="flex flex-col gap-2">
    <div class="flex items-center gap-2">
      <span class="i-lucide-tags icon-sm text-brand flex-center" aria-hidden="true" />
      <h3 class="m-0 text-sm font-medium text-primary">标签</h3>
      <span class="text-xs text-tertiary">{{ meta.tagList.length }} 个</span>
      <!-- 有改动才出现：平时不占位置，也提醒"标签不是改一下就写库的" -->
      <n-button
        v-if="meta.tagsDirty"
        size="tiny"
        type="primary"
        class="ml-auto"
        :loading="meta.tagsSaving"
        :disabled="meta.tagsSaving"
        title="把当前标签列表写进数据库"
        @click="save"
      >
        <template #icon>
          <span class="i-lucide-save icon-xs" aria-hidden="true" />
        </template>
        {{ meta.tagsSaving ? '保存中' : '保存' }}
      </n-button>
    </div>

    <!-- 保存失败等反馈走 toast（metaStore.saveTags 里弹） -->

    <n-dynamic-tags
      :value="meta.tagList"
      size="small"
      :max="128"
      @update:value="onTagsChange"
    />

    <p v-if="!meta.tagList.length" class="m-0 text-xs text-tertiary">
      还没有标签 —— 点「+」添加，改完点「保存」写进数据库。
    </p>
  </div>
</template>

<script setup lang="ts">
import { NButton, NDynamicTags } from 'naive-ui'
import { useMetaStore } from '../../stores/metaStore'

const meta = useMetaStore()

/**
 * n-dynamic-tags 给的是「新列表」，而 store 提供的是逐条增删。
 * 这里算差值再落到 store 上：不新增 store action，也不让组件绕过编辑稿直接写库。
 */
function onTagsChange(next: string[]) {
  const current = [...meta.tagList]
  for (const tag of current) {
    if (!next.includes(tag)) meta.removeTag(tag)
  }
  for (const tag of next) {
    if (!current.includes(tag)) meta.addTag(tag)
  }
}

function save() {
  void meta.saveTags()
}
</script>
