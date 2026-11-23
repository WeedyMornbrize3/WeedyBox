<!-- frontend/src/components/library/ScanPanel.vue
     「扫描添加」面板：选来源（Steam 库 / 本地文件夹）→ 扫描 → **自动逐个搜索** →
     结果按扫描顺序排列；每行可勾选，底部两个入口：「全部入库」与「入库所选」。
     点「详情」只是去 /works 核对作品，不写库 —— 写库只发生在「入库」动作里。
     数据与批量逻辑都在 stores/scanStore.ts（为什么独立 store 见那里注释）。 -->
<template>
  <div class="flex flex-col gap-3">
    <!-- 来源：Steam 自动定位；本地文件夹要选路径（用户取消就保持原路径） -->
    <div class="flex items-center gap-2 flex-wrap">
      <span class="text-xs text-secondary">扫描来源</span>
      <button
        v-for="option in SOURCE_OPTIONS"
        :key="option.value"
        type="button"
        class="chip"
        :class="{ 'chip-active': scan.source === option.value }"
        :aria-pressed="scan.source === option.value"
        @click="scan.setSource(option.value)"
      >
        {{ option.label }}
      </button>
    </div>

    <!-- 本地文件夹：路径 + 系统选择框 + 识别方式 -->
    <template v-if="scan.source === 'folder'">
      <div class="flex items-end gap-2">
        <label class="flex-1 min-w-0 flex flex-col gap-1.5">
          <span class="text-xs text-secondary">游戏目录</span>
          <input
            v-model="scan.folderPath"
            type="text"
            class="input-theme w-full"
            placeholder="例如 D:\Games"
            autocomplete="off"
            spellcheck="false"
            @keydown.enter.prevent="runScan"
          />
        </label>
        <button
          type="button"
          class="btn-brand flex items-center gap-1.5 shrink-0"
          :disabled="scan.picking"
          @click="scan.pickFolder()"
        >
          <span :class="scan.picking ? 'i-lucide-loader-circle spin' : 'i-lucide-folder-open'" class="icon-sm" aria-hidden="true" />
          {{ scan.picking ? '选择中' : '选择…' }}
        </button>
      </div>

      <div class="flex items-center gap-2 flex-wrap">
        <span class="text-xs text-secondary">识别方式</span>
        <button
          v-for="option in MODE_OPTIONS"
          :key="option.value"
          type="button"
          class="chip"
          :class="{ 'chip-active': scan.mode === option.value }"
          :aria-pressed="scan.mode === option.value"
          @click="scan.setMode(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
      <p class="m-0 text-xs text-tertiary">{{ modeHint }}</p>
    </template>

    <p v-else class="m-0 text-xs text-tertiary">
      自动读取 Steam 的 <code class="font-mono">steamapps\appmanifest_&lt;id&gt;.acf</code>，按 appid 从 Steam 商店取游戏信息；
      封面取本机 <code class="font-mono">appcache\librarycache</code> 里该 id 目录下的 600x900 / capsule 图（没有就不用封面）。
      没装 Steam 就换成「选择文件夹」扫本地库。
    </p>

    <!-- 扫描 → 自动搜索/取详情；进行中显示进度与「停止」 -->
    <div class="flex items-center gap-2 flex-wrap">
      <button
        type="button"
        class="btn-brand flex items-center gap-1.5"
        :disabled="scan.scanning || (scan.source === 'folder' && !scan.folderPath.trim())"
        @click="runScan"
      >
        <span :class="scan.scanning ? 'i-lucide-loader-circle spin' : 'i-lucide-search'" class="icon-sm" aria-hidden="true" />
        {{ scan.scanning ? '扫描中…' : '扫描' }}
      </button>
      <span v-if="scan.searching" class="text-xs text-tertiary">{{ searchLabel }} {{ scan.searched }}/{{ scan.count }}…</span>
      <span v-else-if="scan.detailing" class="text-xs text-tertiary">获取详情 {{ scan.detailed }}/{{ scan.detailTotal }}…</span>
      <button v-if="scan.searching || scan.detailing" type="button" class="chip" @click="scan.stopSearch()">停止</button>
      <span v-else-if="scan.matches.length" class="text-xs text-tertiary">匹配 {{ scan.matchedCount }}/{{ scan.count }}</span>
    </div>

    <MetaMessage v-if="scan.error" tone="error" :text="scan.error" />
    <MetaMessage v-if="scan.status" :text="scan.status" />

    <!-- 批量入库：一键全部 / 勾选部分（搜索未完成时先别动，避免只入到一半的结果） -->
    <div v-if="scan.storableCount" class="flex items-center gap-2 flex-wrap">
      <label class="flex items-center gap-1.5 text-xs text-secondary cursor-pointer select-none">
        <input
          type="checkbox"
          class="todo-checkbox flex-shrink-0 w-[18px] h-[18px] rounded-[5px] cursor-pointer appearance-none flex-center text-transparent transition-colors duration-200 border-2 border-theme-dark bg-transparent hover:border-brand checked:border-brand focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand disabled:cursor-not-allowed disabled:opacity-50"
          :checked="scan.allSelected"
          :disabled="scan.storingAll || scan.searching || scan.detailing"
          @change="toggleAll"
        />
        全选
      </label>
      <button
        type="button"
        class="chip"
        :disabled="scan.storingAll || scan.searching || scan.detailing"
        @click="storeAll"
      >
        {{ scan.storingAll ? `入库中 ${scan.storeProgress}/${scan.storeTotal}` : `全部入库（${scan.storableCount}）` }}
      </button>
      <button
        type="button"
        class="chip chip-active"
        :disabled="scan.storingAll || scan.searching || !scan.selectedCount"
        @click="storeSelected"
      >
        入库所选（{{ scan.selectedCount }}）
      </button>
    </div>
    <MetaMessage v-if="scan.storeError" tone="error" :text="scan.storeError" />

    <!-- 搜索结果：按扫描顺序排列，一行 = 一个扫描名 + 匹配到的作品 -->
    <div v-if="scan.matches.length" class="min-h-0 max-h-[34vh] overflow-y-auto scrollbar-theme -mx-5 border-t-theme-light">
      <ul class="list-none m-0 p-0">
        <li
          v-for="match in scan.matches"
          :key="match.entryIndex + ':' + match.source"
          class="flex items-center gap-3 px-5 py-2.5 border-b-theme-light last:border-none"
        >
          <!-- 勾选框：只有「有匹配且未入库」的行可勾；其余留同宽占位保证封面左对齐。
               外观类串与 components/todo/TodoItem.vue 的选择框一字不差（改一处必须同步另一处）。 -->
          <input
            v-if="match.state === 'matched' && !scan.isStored(match)"
            type="checkbox"
            class="todo-checkbox flex-shrink-0 w-[18px] h-[18px] rounded-[5px] cursor-pointer appearance-none flex-center text-transparent transition-colors duration-200 border-2 border-theme-dark bg-transparent hover:border-brand checked:border-brand focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand disabled:cursor-not-allowed disabled:opacity-50"
            :checked="match.selected"
            :disabled="scan.storingAll || match.storing"
            :aria-label="`选择「${match.workName || match.entry.name}」（${sourceLabel(match)}）`"
            @change="scan.toggleSelect(match)"
          />
          <span v-else-if="match.state === 'searching'" class="w-[18px] shrink-0 flex-center" aria-hidden="true">
            <span class="i-lucide-loader-circle spin icon-xs text-tertiary" />
          </span>
          <span v-else class="w-[18px] shrink-0" aria-hidden="true" />

          <MetaCover :src="match.cover" :alt="match.workName" size="sm" />

          <div class="flex-1 min-w-0">
            <p class="m-0 text-sm text-primary truncate" :title="match.message || matchTitle(match)">{{ matchTitle(match) }}</p>
            <p class="m-0 text-xs text-tertiary truncate">
              <!-- 来源徽标：同一个文件夹最多两行（Steam 行 + DLsite 行），靠它区分 -->
              <span class="src-tag">{{ sourceLabel(match) }}</span>
              <template v-if="match.state === 'matched'">
                <span class="font-mono">{{ match.workId }}</span>
                <span class="mx-1">·</span>
              </template>
              <span>{{ match.entry.name }}</span>
            </p>
            <!-- 入库后回填的作品信息（社团 · 形式 · 发售日 · 容量）：搜索完会逐条抓详情填进来 -->
            <p v-if="match.info" class="m-0 text-xs text-tertiary truncate" :title="match.info">{{ match.info }}</p>
            <p v-if="match.message" class="m-0 text-xs text-warning truncate" :title="match.message">{{ match.message }}</p>
            <p v-if="match.storeError" class="m-0 text-xs text-danger truncate" :title="match.storeError">{{ match.storeError }}</p>
          </div>

          <span v-if="scan.isStored(match)" class="text-xs text-success flex items-center gap-1 shrink-0">
            <span class="i-lucide-circle-check-big icon-xs" aria-hidden="true" />
            已入库
          </span>
          <button v-else-if="match.state === 'matched'" type="button" class="chip shrink-0" @click="openDetail(match)">
            详情
          </button>
          <button
            v-else-if="match.state === 'error' || match.state === 'stopped'"
            type="button"
            class="chip shrink-0"
            @click="scan.retryMatch(match)"
          >
            重试
          </button>
        </li>
      </ul>
    </div>
    <p v-else-if="scan.done && !scan.searching" class="m-0 text-xs text-tertiary">{{ emptyHint }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useScanStore } from '../../stores/scanStore'
import type { FolderScanMode, ScanMatch, ScanSource, ScanStoreResult } from '../../stores/scanStore'
import { toastError, toastSuccess } from '../../utils/toast'
import MetaCover from '../meta/MetaCover.vue'
import MetaMessage from '../meta/MetaMessage.vue'

const scan = useScanStore()

/** 详情要带上来源：同一个编号在 dlsite / steam 里是两条库记录 */
const emit = defineEmits<{ detail: [workId: string, source: 'dlsite' | 'steam'] }>()

/** Steam 条目按 appid 取商店详情，说法上不叫「搜索」*/
const searchLabel = computed(() => (scan.source === 'steam' ? '获取信息' : '搜索中'))

const SOURCE_OPTIONS: { value: ScanSource; label: string }[] = [
  { value: 'steam', label: 'Steam 库' },
  { value: 'folder', label: '选择文件夹' },
]

const MODE_OPTIONS: { value: FolderScanMode; label: string }[] = [
  { value: 'exe', label: '按 exe 识别' },
  { value: 'folder', label: '按文件夹名' },
]

/** 两种识别方式的差别（用户要求的"通用 / 节约资源"两种用途） */
const modeHint = computed(() =>
  scan.mode === 'exe'
    ? '扫描每个一级子文件夹里的 .exe，用 exe 文件名当游戏名；不递归，也不含根目录自己的 exe。搜之前会同时查 Steam 商店与 DLsite。'
    : '只读目录第一层，直接用子文件夹名当游戏名。搜之前**同时查 Steam 商店与 DLsite**：Steam 猜得到 appid 就多一行 Steam 结果，DLsite 按原名搜一条；两边都列出来让你挑。',
)

const emptyHint = computed(() => {
  if (scan.source === 'steam') {
    return '没有读到已安装的 Steam 游戏（steamapps\\appmanifest_*.acf）。没装 Steam 的话，换成「选择文件夹」扫本地库。'
  }
  if (scan.mode === 'exe') return '这些子文件夹里没有发现 .exe（根目录自己的 exe 不算）。'
  return '该目录下没有一级子文件夹。'
})

/** 一行的来源文字（Steam / DLsite） */
function sourceLabel(match: ScanMatch): string {
  return match.source === 'steam' ? 'Steam' : 'DLsite'
}

function matchTitle(match: ScanMatch): string {
  if (match.state === 'matched') return match.workName || match.entry.name
  if (match.state === 'searching') return searchLabel.value + '…'
  if (match.state === 'error') return match.source === 'steam' ? '获取 Steam 信息失败' : 'DLsite 搜索失败'
  if (match.state === 'stopped') return match.source === 'steam' ? '未获取（已停止）' : '未搜索（已停止）'
  if (match.state === 'empty') return match.message ? '已跳过' : '未找到匹配作品'
  return match.entry.name
}

/** 点「详情」：带上这一行的来源进 /works 核对（Steam 行按 appid 读，DLsite 行按编号读） */
function openDetail(match: ScanMatch) {
  emit('detail', match.workId, match.source)
}

function toggleAll(event: Event) {
  scan.toggleSelectAll((event.target as HTMLInputElement).checked)
}

function runScan() {
  void scan.scan()
}

/**
 * 批量入库的收尾提示（游戏库「添加」的 toast，2026-11-23 用户要求）。
 * 只在整批结束时弹一次汇总：每行成功都弹会把提示刷屏，行内本身已有「已入库」状态。
 */
function reportStoreResult(result: ScanStoreResult) {
  if (result.stored > 0) {
    toastSuccess(`已入库 ${result.stored} 款游戏`, { description: '已经收进游戏库，可以在网格里看到' })
  }
  if (result.failed > 0) {
    toastError(`${result.failed} 款入库失败`, {
      description: scan.storeError || '展开结果行可以看到具体原因',
    })
  }
}

async function storeAll() {
  reportStoreResult(await scan.storeAll())
}

async function storeSelected() {
  reportStoreResult(await scan.storeSelected())
}
</script>

<style scoped>
/* 来源徽标（Steam / DLsite）：10px 小字 + 1px 描边，和行内的编号一样是辅助信息，
   不抢封面与作品名的视觉重心；但用 secondary 色 + 500 字重保证一眼能分清两行。
   颜色只取全局 token（两套主题自动适配）。 */
.src-tag {
  display: inline-block;
  margin-right: 4px;
  padding: 0 4px;
  border: 1px solid var(--color-border-light);
  border-radius: 4px;
  font-size: 10px;
  font-weight: 500;
  line-height: 14px;
  color: var(--color-text-secondary);
  vertical-align: 1px;
}
</style>
