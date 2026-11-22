<!-- frontend/src/components/meta/MetaTagEditor.vue
     作品详情里的「标签」：抓取来的 tags 与用户自己加的标签**合并成一份列表**，
     可以随手删/加，改动先攒在 store 的编辑稿里，点「保存」才整份写进数据库。
     交互（2026-11-22 用户定稿）：
       - 鼠标移到标签上，右上角浮出小 ×，点它把这枚标签从编辑稿里删掉；
       - 列表末尾固定一颗同款 chip 的「+」，点它原地变成输入框（先保持原大小、随输入变宽），
         回车或失焦提交，输入为空则无视这次修改；
       - 有改动（增/删）时标题右侧出现「保存」，点它把整份列表写进 work_tags。
     数据与请求都在 stores/metaStore.ts（tagList / tagsDirty / addTag / removeTag / saveTags）。 -->
<template>
  <div class="flex flex-col gap-2">
    <div class="flex items-center gap-2">
      <span class="i-lucide-tags icon-sm section-icon" aria-hidden="true" />
      <h3 class="m-0 text-sm font-medium text-primary">标签</h3>
      <span class="text-xs text-tertiary">{{ meta.tagList.length }} 个</span>
      <!-- 有改动才出现：平时不占位置，也提醒"标签不是改一下就写库的" -->
      <button
        v-if="meta.tagsDirty"
        type="button"
        class="tag-save"
        :disabled="meta.tagsSaving"
        title="把当前标签列表写进数据库"
        @click="save"
      >
        <span
          :class="meta.tagsSaving ? 'i-lucide-loader-circle spin' : 'i-lucide-save'"
          class="icon-xs"
          aria-hidden="true"
        />
        {{ meta.tagsSaving ? '保存中' : '保存' }}
      </button>
    </div>

    <MetaMessage v-if="meta.tagError" tone="error" :text="meta.tagError" />

    <!-- 标签 + 末尾占位加号，同一行 flex-wrap：加号永远排在最后一个 -->
    <ul class="list-none m-0 p-0 flex flex-wrap items-center gap-x-2.5 gap-y-3">
      <li v-for="tag in meta.tagList" :key="tag" class="chip tag-chip cursor-default">
        {{ tag }}
        <button
          type="button"
          class="tag-remove"
          :title="`删除「${tag}」`"
          :aria-label="`删除标签「${tag}」`"
          @click="remove(tag)"
        >
          <span class="i-lucide-x" aria-hidden="true" />
        </button>
      </li>

      <!-- 占位加号：外观与标签完全一致（同一个 chip），点开就在原地输入 -->
      <li v-if="!editing" class="tag-add">
        <button
          type="button"
          class="chip cursor-pointer tag-add-btn"
          title="添加标签"
          aria-label="添加标签"
          @click="startEditing"
        >
          <!-- 16px：与标签文字的行盒同高，加号 chip 与真标签 chip 等高 -->
          <span class="i-lucide-plus icon-sm" aria-hidden="true" />
        </button>
      </li>
      <li v-else class="chip tag-add-editing !border-brand">
        <input
          ref="inputEl"
          v-model="draft"
          class="tag-input"
          type="text"
          maxlength="32"
          placeholder="标签名"
          aria-label="新标签"
          autocomplete="off"
          spellcheck="false"
          @keydown.enter.prevent="submit"
          @keydown.esc.prevent="cancel"
          @blur="submit"
        />
        <!-- 隐藏镜像：输入框与它叠在同一个网格单元，外壳宽度跟着输入内容长 -->
        <span class="tag-mirror" aria-hidden="true">{{ draft || ' ' }}</span>
      </li>
    </ul>

    <p v-if="!meta.tagList.length && !editing" class="m-0 text-xs text-tertiary">
      还没有标签 —— 点「+」添加，改完点「保存」写进数据库。
    </p>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useMetaStore } from '../../stores/metaStore'
import MetaMessage from './MetaMessage.vue'

const meta = useMetaStore()
// editing = 末尾那颗「+」是否已展开成输入框；draft = 正在输入的内容
const editing = ref(false)
const draft = ref('')
const inputEl = ref<HTMLInputElement | null>(null)

/** 点「+」：把占位 chip 换成输入框，并立刻聚焦（用户点完就能打字） */
async function startEditing() {
  if (editing.value) return
  draft.value = ''
  editing.value = true
  await nextTick()
  inputEl.value?.focus()
}

/** 收起输入框，回到「+」占位（不改编辑稿） */
function closeEditor() {
  editing.value = false
  draft.value = ''
}

function cancel() {
  closeEditor()
}

/**
 * 提交：回车或失焦触发。
 * 输入为空 → 无视这次修改（用户要求），直接收起；
 * 非空 → 追加进编辑稿（重复的忽略），等用户点「保存」才写库。
 */
function submit() {
  if (!editing.value) return
  const text = draft.value.trim()
  if (!text) {
    closeEditor()
    return
  }
  meta.addTag(text)
  closeEditor()
}

function remove(tag: string) {
  meta.removeTag(tag)
}

function save() {
  void meta.saveTags()
}
</script>

<style scoped>
/* 标签 chip：小 × 用绝对定位挂右上角，所以 li 必须是定位上下文 */
.tag-chip {
  position: relative;
}

/* 「保存」：紧凑的品牌色小按钮（不套 btn-brand 那 35.2px 的大按钮） */
.tag-save {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
  padding: 2px 10px;
  border: none;
  border-radius: 6px;
  background: var(--color-primary);
  color: var(--color-text-on-brand);
  font-size: 12px;
  font-weight: 500;
  line-height: 1.5;
  cursor: pointer;
  transition: background-color var(--motion-fast) var(--ease-enter);
}

.tag-save:hover {
  background: var(--color-primary-hover);
}

.tag-save:disabled {
  opacity: 0.6;
  cursor: default;
}

/* 右上角的删除 ×：静息藏起来且不占点击区域（避免误删），
   hover 标签 / 键盘聚焦 / 聚焦到按钮本身时才浮出。 */
.tag-remove {
  position: absolute;
  top: -5px;
  right: -5px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-bg-card);
  color: var(--color-danger);
  opacity: 0;
  pointer-events: none;
  transition:
    opacity var(--motion-fast) var(--ease-enter),
    background-color var(--motion-fast) var(--ease-enter),
    color var(--motion-fast) var(--ease-enter);
}

.tag-chip:hover .tag-remove,
.tag-chip:focus-within .tag-remove,
.tag-remove:focus-visible {
  opacity: 1;
  pointer-events: auto;
}

/* 破坏性语义：hover / 聚焦时直接填红（与游戏库删除、TODO 删除同一套配色口径） */
.tag-remove:hover,
.tag-remove:focus-visible {
  background: var(--color-danger);
  border-color: var(--color-danger);
  color: var(--color-text-on-danger);
}

/* 14px 的圆里放 10px 的 ×（icon-xs 是 14px，会顶满圆） */
.tag-remove .i-lucide-x {
  width: 10px;
  height: 10px;
  font-size: 10px;
}

/* 末尾的「+」：外壳与标签同款 chip，只把加号居中 */
.tag-add {
  display: flex;
}

/* 加号 chip 与其展开后的输入框同宽（38px = 左右 padding 20 + 边框 2 + 16px 图标），
   所以点「+」的瞬间尺寸不变，之后才随输入内容变宽 */
.tag-add-btn,
.tag-add-editing {
  min-width: 2.375rem;
}

.tag-add-btn {
  justify-content: center;
}

/* 点「+」之后：占位 chip 原地变成输入框，外壳仍是 chip（只有边框换成品牌色表示聚焦） */
.tag-add-editing {
  display: inline-grid;
  align-items: center;
  cursor: text;
}

/* 输入框与隐藏镜像共用同一个网格单元 → 单元格宽度 = 两者中较宽者（镜像跟着文字长） */
.tag-add-editing > * {
  grid-area: 1 / 1;
  min-width: 0;
}

.tag-input {
  width: 100%;
  padding: 0;
  margin: 0;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  line-height: inherit;
  outline: none;
}

.tag-mirror {
  visibility: hidden;
  white-space: pre;
}
</style>
