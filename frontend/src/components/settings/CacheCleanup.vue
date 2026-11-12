<!-- frontend/src/components/settings/CacheCleanup.vue
     设置页的「清理缓存」控件（放在 SettingsItem 的默认插槽里）。
     用户要求：「设置界面添加清理缓存功能（将缓存 cache 内所有文件删去，并在进入设置界面时
     扫描 cache 文件夹大小，并显示在清理缓存功能旁边）」。

     职责很窄：
       - onMounted 扫一次 cache/ 占用（后端 CacheInfo → 文件数 + 字节），显示在按钮左边；
       - 点「清理缓存」调后端 ClearCache（只删 cache/ 里的文件，目录保留，preImg/ 不动），
         然后把占用重新扫一遍，以真实结果为准（有删不掉的文件也能立刻看见）；
       - 操作结果用一行小字反馈（成功绿 / 失败红），4 秒后自动消失。

     为什么不在进入路由时扫而是 onMounted：设置页是动态导入的路由页，每次进来都会重新挂载，
     onMounted 等价于「进入设置界面」，不用额外写路由钩子。
     数据全部走 services/metaService.ts（Wails 绑定的唯一入口），components 里不直接 import 绑定。 -->
<template>
  <div class="flex flex-col items-end gap-1">
    <div class="flex items-center gap-3">
      <!-- 占用：进页面时统计一次；hover 显示缓存目录绝对路径，省得用户去猜在哪 -->
      <span class="cache-size" :title="cache.dir ? `缓存目录：${cache.dir}` : undefined">
        {{ sizeText }}
      </span>

      <button
        type="button"
        class="cache-btn"
        :disabled="!canClear"
        :aria-label="busy ? '正在清理缓存' : '清理缓存'"
        @click="clearCache"
      >
        <span
          :class="busy ? 'i-lucide-loader-circle spin' : 'i-lucide-trash-2'"
          class="icon-xs"
          aria-hidden="true"
        />
        {{ busy ? '清理中…' : '清理缓存' }}
      </button>
    </div>

    <!-- 一次性反馈：4 秒后自己消失，不占常驻空间 -->
    <span v-if="note" class="cache-note" :class="noteTone === 'error' ? 'is-error' : 'is-success'">
      {{ note }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { metaApi, type CacheInfo } from '../../services/metaService'
import { formatBytes, toMessage } from '../../utils/metaFormat'

const cache = ref<CacheInfo>({ dir: '', files: 0, bytes: 0 })
/** 首次/重新统计中（按钮要禁用，避免拿旧数字点下去） */
const loading = ref(false)
/** 正在删除 */
const busy = ref(false)
const note = ref('')
const noteTone = ref<'success' | 'error'>('success')
let noteTimer: ReturnType<typeof setTimeout> | undefined

const sizeText = computed(() => {
  if (loading.value) return '统计中…'
  if (cache.value.files === 0) return '当前无缓存'
  return `占用 ${formatBytes(cache.value.bytes)}（${cache.value.files} 个文件）`
})

/** 没有缓存、正在统计、正在删除时都不能点 */
const canClear = computed(() => !loading.value && !busy.value && cache.value.files > 0)

function flashNote(text: string, tone: 'success' | 'error' = 'success') {
  note.value = text
  noteTone.value = tone
  if (noteTimer) clearTimeout(noteTimer)
  noteTimer = setTimeout(() => {
    note.value = ''
  }, 4000)
}

/** 读一次真实占用（进页面、删完之后各调一次） */
async function refresh() {
  loading.value = true
  try {
    cache.value = await metaApi.cacheInfo()
  } catch (err) {
    flashNote(toMessage(err), 'error')
  } finally {
    loading.value = false
  }
}

async function clearCache() {
  if (!canClear.value) return
  busy.value = true
  try {
    const result = await metaApi.clearCache()
    if (result.files > 0) {
      flashNote(`已清理 ${formatBytes(result.bytes)}（${result.files} 个文件）`)
    } else {
      flashNote('缓存已经是空的')
    }
    // 以重新统计的结果为准：有文件删不掉时（rest > 0）用户能立刻看到还剩多少
    await refresh()
  } catch (err) {
    flashNote(toMessage(err), 'error')
  } finally {
    busy.value = false
  }
}

onMounted(refresh)
onBeforeUnmount(() => {
  if (noteTimer) clearTimeout(noteTimer)
})
</script>

<style scoped>
/* 占用文案：小字、等宽数字（跳动时不抖宽度） */
.cache-size {
  font-size: 12px;
  line-height: 1;
  color: var(--color-text-tertiary);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

/* 清理按钮：与项目里其他破坏性按钮同一套颜色语言 ——
   透明底 + danger 红描边红图标 → hover 填红 + 随主题翻转的深色文字（--color-text-on-danger）。
   形状（圆角/内边距/字号）跟设置页里的紧凑控件一致，不用 btn-brand（那是 35.2px 的大按钮）。 */
.cache-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  border: 1px solid var(--color-danger);
  border-radius: 8px;
  background-color: transparent;
  color: var(--color-danger);
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color var(--motion-fast) var(--ease-enter),
    border-color var(--motion-fast) var(--ease-enter),
    color var(--motion-fast) var(--ease-enter),
    opacity var(--motion-fast) var(--ease-enter);
}

.cache-btn:hover:not(:disabled) {
  background-color: var(--color-danger);
  border-color: var(--color-danger);
  color: var(--color-text-on-danger);
}

.cache-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* 结果反馈：成功用 success、失败用 danger，都是 token 色 */
.cache-note {
  font-size: 11px;
  line-height: 1.2;
}

.cache-note.is-success {
  color: var(--color-success);
}

.cache-note.is-error {
  color: var(--color-danger);
}
</style>
