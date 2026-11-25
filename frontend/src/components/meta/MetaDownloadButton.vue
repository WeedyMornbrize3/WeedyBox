<!-- frontend/src/components/meta/MetaDownloadButton.vue
     入库 / 启动按钮：只负责「长相 + 状态判断」，真正的请求在 store 里。
     2026-11-24 起按钮改用 naive 的 n-button；「启动方式」下拉从自绘浮层换成 n-dropdown。
     两个阶段：
       - 还没入库：显示「入库」，点击把作品收进游戏库；没有封面 URL 时禁用；
       - 已经入库：**不再显示「已入库」**，替换成「游戏启动」——
         点击按库里记录的启动方式启动（stores/metaStore.ts 的 launchGame）。
     已入库后的状态：
         · exe 方式下文件不存在（被删除/移走）→ 按钮禁用，由页面提示"游戏已被删除或变更位置"；
         · 游戏正在运行 → 显示「运行中」并禁用（防止重复启动）；
         · 没有记录位置 → 可点，第一次点击弹系统文件框选一次 exe。
     启动方式（Steam 源）：默认走 steam://rungameid/<appid>，不需要本地 exe，
     方式本身由「游戏启动」右端挂的 n-dropdown 切换（分体按钮，见下方 split-* 类）。 -->
<template>
  <div class="flex items-center gap-2 flex-wrap">
    <!-- 已入库：启动游戏（+ 启动方式 + 更换位置） -->
    <template v-if="stored">
      <div class="flex items-stretch">
        <n-button
          type="primary"
          :disabled="launching || running || launchBlocked"
          :loading="launching"
          :title="launchHint"
          :style="steamSource ? { borderTopRightRadius: 0, borderBottomRightRadius: 0 } : undefined"
          @click="emit('launch')"
        >
          <template #icon>
            <span :class="launchIcon" class="icon-sm" aria-hidden="true" />
          </template>
          {{ launchText }}
        </n-button>

        <!-- 右段：启动方式下拉（只有 Steam 源才有两种启动方式）。
             2026-11-24 用户要求「去除游戏启动旁启动方式简称」：按钮上只留一个下拉箭头，
             当前方式改用 title / aria-label 表达，选项里的勾选仍然标出当前方式。 -->
        <n-dropdown
          v-if="steamSource"
          v-model:show="modeMenuOpen"
          trigger="click"
          :options="modeOptions"
          :disabled="launching || running"
          @select="onSelectMode"
        >
          <n-button
            type="primary"
            :disabled="launching || running"
            :title="`启动方式：${modeLabel}`"
            :aria-label="`启动方式：${modeLabel}，点击切换`"
            :aria-expanded="modeMenuOpen"
            :style="{
              borderTopLeftRadius: 0,
              borderBottomLeftRadius: 0,
              borderLeft: '1px solid var(--color-divider-on-brand)',
              padding: '0 8px',
            }"
          >
            <!-- 箭头：展开时转 180°（用户 2026-11-25 要求） -->
            <span
              class="i-lucide-chevron-down icon-sm launch-mode-chevron"
              :class="{ 'launch-mode-chevron--open': modeMenuOpen }"
              aria-hidden="true"
            />
          </n-button>
        </n-dropdown>
      </div>

      <n-button
        v-if="gamePath && launchMode !== 'steam'"
        text
        size="tiny"
        :disabled="launching || running"
        title="重新选择游戏程序（.exe）"
        @click="emit('changePath')"
      >
        <template #icon>
          <span class="i-lucide-folder-search icon-xs" aria-hidden="true" />
        </template>
        更换位置
      </n-button>
    </template>

    <!-- 未入库：入库 -->
    <n-button
      v-else
      type="primary"
      :disabled="disabled || loading"
      :loading="loading"
      @click="emit('click')"
    >
      <template #icon>
        <span class="i-lucide-download icon-sm" aria-hidden="true" />
      </template>
      {{ text }}
    </n-button>
  </div>
</template>

<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { NButton, NDropdown } from 'naive-ui'

const props = withDefaults(
  defineProps<{
    label?: string
    loadingText?: string
    loading?: boolean
    disabled?: boolean
    /** 已入库：按钮改成「游戏启动」 */
    stored?: boolean
    /** 库里记录的游戏程序路径（空 = 还没设置，点击时会先弹选择框） */
    gamePath?: string
    /** 记录的 exe 是否还存在（false = 被删除/移走 → exe 方式下启动按钮禁用） */
    gamePathExists?: boolean
    /** 生效的启动方式：steam = 交给 Steam 客户端（不需要本地 exe）；exe = 跑本机程序 */
    launchMode?: 'steam' | 'exe'
    /** 游戏是否正在运行（true → 显示「运行中」并禁用） */
    running?: boolean
    /** 正在启动游戏 */
    launching?: boolean
    /** 是否 Steam 源：决定「游戏启动」右边挂不挂启动方式下拉段 */
    steamSource?: boolean
  }>(),
  {
    label: '入库',
    loadingText: '入库中',
    loading: false,
    disabled: false,
    stored: false,
    gamePath: '',
    gamePathExists: true,
    launchMode: 'exe',
    running: false,
    launching: false,
    steamSource: false,
  }
)

const emit = defineEmits<{
  click: []
  launch: []
  changePath: []
  /** 下拉里选了另一种启动方式（透传给页面 → metaStore.setGameLaunchMode） */
  setLaunchMode: [mode: 'steam' | 'exe']
}>()

/** 启动方式下拉是否展开：只用来给右侧箭头做旋转动画（展开 180°、收起回正） */
const modeMenuOpen = ref(false)

/** 未入库时的文案：加载中优先（入库成功那一轮 loading 已落回 false，不会说错话） */
const text = computed(() => (props.loading ? props.loadingText : props.label))

/** 已入库但 exe 不存在：exe 方式下不能启动，等用户换位置；Steam 方式不需要本地 exe */
const launchBlocked = computed(
  () =>
    props.stored &&
    props.launchMode !== 'steam' &&
    Boolean(props.gamePath) &&
    !props.gamePathExists,
)

const launchText = computed(() =>
  props.launching ? '启动中' : props.running ? '运行中' : '游戏启动',
)
const launchIcon = computed(() =>
  props.running ? 'i-lucide-activity' : 'i-lucide-play',
)

/** 当前方式的完整说法（只用于 title / aria-label，按钮上不再显示简称） */
const modeLabel = computed(() => (props.launchMode === 'steam' ? '通过 Steam' : '本地程序'))

/** 下拉选项：当前方式前面打勾（原来是自绘菜单的勾选标记） */
const modeOptions = computed(() => [
  {
    key: 'steam',
    label: '通过 Steam 启动',
    icon: () =>
      props.launchMode === 'steam' ? h('span', { class: 'i-lucide-check icon-xs text-brand' }) : null,
  },
  {
    key: 'exe',
    label: '本地程序启动',
    icon: () =>
      props.launchMode === 'exe' ? h('span', { class: 'i-lucide-check icon-xs text-brand' }) : null,
  },
])

function onSelectMode(key: string | number) {
  emit('setLaunchMode', key as 'steam' | 'exe')
}

/** 悬停提示：直接告诉用户这次点下去会用哪个程序 / 为什么点不了 */
const launchHint = computed(() => {
  if (launchBlocked.value) return '游戏程序不存在（可能已被删除或移动），请点「更换位置」'
  if (props.running) return '游戏正在运行'
  if (props.launchMode === 'steam') return '通过 Steam 启动（steam://rungameid/<appid>）'
  if (!props.gamePath) return '启动游戏（还没记录游戏位置，点击后选择游戏程序）'
  return `启动游戏：${props.gamePath}`
})
</script>

<style scoped>
/* 启动方式箭头：展开时翻转（时长/缓动统一走全局动效 token） */
.launch-mode-chevron {
  transition: transform var(--motion-base) var(--ease-enter);
}

.launch-mode-chevron--open {
  transform: rotate(180deg);
}
</style>
