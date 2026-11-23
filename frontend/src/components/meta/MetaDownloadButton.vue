<!-- frontend/src/components/meta/MetaDownloadButton.vue
     入库 / 启动按钮：只负责「长相 + 状态判断」，真正的请求在 store 里。
     两个阶段：
       - 还没入库：显示「入库」（原「下载封面」），点击把作品收进游戏库；
         没有封面 URL 时禁用；入库中显示「入库中」；
       - 已经入库：**不再显示「已入库」**，替换成「游戏启动」——
         点击按库里记录的启动方式启动（stores/metaStore.ts 的 launchGame）。
       已入库后的状态（2026-11-16 用户要求）：
         · exe 方式下文件不存在（被删除/移走）→ 按钮**禁用**，由页面提示"游戏已被删除或变更位置"，
           路径选择沿用原来的「更换位置」入口（系统文件框，UI 不改）；
         · 游戏正在运行 → 按钮显示「运行中」并禁用（防止重复启动），状态由页面轮询刷新；
         · 没有记录位置 → 按钮可点，第一次点击弹系统文件框选一次 exe。
       启动方式（2026-11-23）：Steam 源的作品默认走 steam://rungameid/<appid>，
       此时不需要本地 exe，也不显示「更换位置」；方式本身由同目录的
       MetaLaunchModeMenu 切换 —— 它就是「游戏启动」右端的分体按钮右段。
       已经设置过位置时旁边多一个很轻的「更换位置」文字按钮：选错程序或换了安装目录
       时需要能改，但不占视觉重量（运行中时禁用）。
     为什么已入库后不再给「入库」按钮：下载实现刻意不覆盖同名文件
     （重名加 -2/-3 后缀），再点一次只会多出一份重复的图。 -->
<template>
  <div class="flex items-center gap-2 flex-wrap">
    <!-- 已入库：启动游戏（+ 更换位置） -->
    <template v-if="stored">
      <!-- 分体按钮（2026-11-23 用户要求：「启动方式通过 group 连接到游戏启动右端」）：
           左段 = 游戏启动主按钮，右段 = 启动方式下拉。两段之间不能有 gap，
           靠 btn-brand-split / btn-brand-attach 各自的圆角与分隔线拼成一个整体。
           右段只有 Steam 源才有（其它来源只有一种启动方式，没什么可选的）。 -->
      <div class="flex items-stretch">
        <button
          type="button"
          :class="steamSource ? 'btn-brand-split' : 'btn-brand'"
          class="flex items-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="launching || running || launchBlocked"
          :title="launchHint"
          @click="emit('launch')"
        >
          <span :class="[launchIcon, { spin: launching }]" class="icon-sm" aria-hidden="true" />
          {{ launchText }}
        </button>
        <MetaLaunchModeMenu
          v-if="steamSource"
          :launch-mode="launchMode"
          :disabled="launching || running"
          @select="emit('setLaunchMode', $event)"
        />
      </div>
      <button
        v-if="gamePath && launchMode !== 'steam'"
        type="button"
        class="text-xs text-secondary hover:text-primary flex items-center gap-1 cursor-pointer bg-transparent border-none disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="launching || running"
        title="重新选择游戏程序（.exe）"
        @click="emit('changePath')"
      >
        <span class="i-lucide-folder-search icon-xs" aria-hidden="true" />
        更换位置
      </button>
    </template>

    <!-- 未入库：入库 -->
    <button
      v-else
      type="button"
      class="btn-brand flex items-center gap-1.5 disabled:opacity-50 disabled:cursor-not-allowed"
      :disabled="disabled || loading"
      @click="emit('click')"
    >
      <span :class="loading ? 'i-lucide-loader-circle spin' : 'i-lucide-download'" class="icon-sm" aria-hidden="true" />
      {{ text }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import MetaLaunchModeMenu from './MetaLaunchModeMenu.vue'

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
  /** 右段下拉里选了另一种启动方式（透传给页面 → metaStore.setGameLaunchMode） */
  setLaunchMode: [mode: 'steam' | 'exe']
}>()

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
  props.launching
    ? 'i-lucide-loader-circle'
    : props.running
      ? 'i-lucide-activity'
      : 'i-lucide-play',
)

/** 悬停提示：直接告诉用户这次点下去会用哪个程序 / 为什么点不了 */
const launchHint = computed(() => {
  if (launchBlocked.value) return '游戏程序不存在（可能已被删除或移动），请点「更换位置」'
  if (props.running) return '游戏正在运行'
  if (props.launchMode === 'steam') return '通过 Steam 启动（steam://rungameid/<appid>）'
  if (!props.gamePath) return '启动游戏（还没记录游戏位置，点击后选择游戏程序）'
  return `启动游戏：${props.gamePath}`
})
</script>
