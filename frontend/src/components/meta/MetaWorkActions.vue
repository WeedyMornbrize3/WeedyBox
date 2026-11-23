<!-- frontend/src/components/meta/MetaWorkActions.vue
     作品详情页的动作区：**入库 / 游戏启动按钮 + 启动方式切换**。
     从 MetaWorkOverview 里抽出来单独成组件：/works 的新排版（hero 背景 + 左信息栏 + 右主区）
     把动作放在主区顶部，调试页（/testing）仍把动作放在概览块里 —— 两处共用同一份，
     避免"按钮长什么样/什么状态"被抄两遍（改一处忘另一处的经典坑）。
     真正的状态判断在 MetaDownloadButton 与 stores/metaStore.ts，这里只负责摆位置。 -->
<template>
  <div class="flex flex-col gap-3">
    <div class="flex items-center gap-3 flex-wrap">
      <MetaDownloadButton
        :disabled="disabled"
        :loading="downloading"
        :stored="stored"
        :game-path="gamePath"
        :game-path-exists="gamePathExists"
        :launch-mode="launchMode"
        :running="running"
        :launching="launching"
        @click="emit('download')"
        @launch="emit('launch')"
        @change-path="emit('changePath')"
      />
    </div>

    <!-- 启动方式（只有 Steam 源的作品有两选一；默认「通过 Steam 启动」= steam://rungameid/<appid>） -->
    <div v-if="stored && steamSource" class="flex items-center gap-2 flex-wrap" role="group" aria-label="启动方式">
      <span class="text-xs text-tertiary">启动方式</span>
      <button
        v-for="option in LAUNCH_MODE_OPTIONS"
        :key="option.value"
        type="button"
        class="chip"
        :class="{ 'chip-active': launchMode === option.value }"
        :aria-pressed="launchMode === option.value"
        :title="option.hint"
        @click="emit('setLaunchMode', option.value)"
      >
        {{ option.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import MetaDownloadButton from './MetaDownloadButton.vue'

withDefaults(
  defineProps<{
    /** 未入库时「入库」按钮是否禁用（没有封面 URL 时不能入库） */
    disabled?: boolean
    downloading?: boolean
    stored?: boolean
    /** 库里记录的游戏程序路径（空 = 还没设置） */
    gamePath?: string
    /** 记录的 exe 是否还存在（false = 被删除/移走） */
    gamePathExists?: boolean
    /** 生效的启动方式（Steam 源才有 steam 这个选项） */
    launchMode?: 'steam' | 'exe'
    /** 当前作品是否来自 Steam 源：只有它能选「通过 Steam 启动 / 本地程序启动」 */
    steamSource?: boolean
    /** 游戏是否正在运行 */
    running?: boolean
    /** 正在启动游戏 */
    launching?: boolean
  }>(),
  {
    disabled: false,
    downloading: false,
    stored: false,
    gamePath: '',
    gamePathExists: true,
    launchMode: 'exe',
    steamSource: false,
    running: false,
    launching: false,
  },
)

const emit = defineEmits<{
  download: []
  launch: []
  changePath: []
  setLaunchMode: [mode: 'steam' | 'exe']
}>()

/** 启动方式的两个选项（Steam 源专用）：默认走 Steam URL，也可以退回本地 exe */
const LAUNCH_MODE_OPTIONS: { value: 'steam' | 'exe'; label: string; hint: string }[] = [
  { value: 'steam', label: '通过 Steam 启动', hint: '运行 steam://rungameid/<appid>，由 Steam 客户端拉起游戏' },
  { value: 'exe', label: '本地程序启动', hint: '直接运行本机记录的游戏程序（没有记录时会先让你选一次 exe）' },
]
</script>
