<!-- frontend/src/components/meta/MetaLaunchModeMenu.vue
     「启动方式」下拉段（2026-11-23 用户要求：DropdownMenu 样式，并「通过 group 连接到
     游戏启动右端」）。它是分体按钮的右段：左段是「游戏启动」主按钮（MetaDownloadButton），
     本组件紧贴在它右边，两段共用 btn-brand 外观、中间一条分隔线。
     左段由调用方负责 `btn-brand-split`（右侧直角），本组件自己用 `btn-brand-attach`。
     选项只有 Steam 源才有的两个：通过 Steam 启动 / 本地程序启动。
     两处细节都按项目既有口径：
       · 浮层 **fixed 定位**（Teleport 到 body，量触发按钮的 rect 后摆位），
         不参与布局、不被卡片的 overflow 裁掉（与「添加游戏」弹窗同一套做法）；
       · 菜单宽度不小于触发按钮、右缘贴住按钮右缘；下方放不下时（窗口矮）翻到上方。
     语义用 menu / menuitemradio + aria-checked（单选菜单），键盘：Esc 关闭（焦点回按钮）、
     ↓ 从按钮展开、↑↓ 在选项间移动、Enter/Space 选中（原生 button 行为）、
     点组件外或 Tab 离开 / 页面滚动都会关闭。
     **状态与请求都不在这里**：选中后只 emit select，由 metaStore.setGameLaunchMode 写库。 -->
<template>
  <button
    ref="triggerEl"
    type="button"
    class="btn-brand-attach"
    aria-haspopup="menu"
    :aria-expanded="open"
    :aria-label="`启动方式：${currentOption.label}`"
    :title="currentOption.hint"
    :disabled="disabled"
    @click="toggle"
    @keydown.down.prevent="openMenu(true)"
    @keydown.tab="close()"
  >
    <span
      class="i-lucide-chevron-down icon-xs transition-transform duration-200"
      :class="open ? 'rotate-180' : ''"
      aria-hidden="true"
    />

    <Teleport to="body">
      <Transition name="menu-pop">
        <div
          v-if="open"
          ref="menuEl"
          class="fixed z-[1200] rounded-md border-theme bg-card p-1 shadow-theme-heavy"
          role="menu"
          aria-label="启动方式"
          :style="menuStyle"
          @keydown.esc.prevent="close(true)"
          @keydown.down.prevent="moveFocus(1)"
          @keydown.up.prevent="moveFocus(-1)"
          @keydown.tab="close()"
        >
          <button
            v-for="option in LAUNCH_MODE_OPTIONS"
            :key="option.value"
            type="button"
            role="menuitemradio"
            :aria-checked="launchMode === option.value"
            tabindex="-1"
            class="flex w-full items-center gap-2 px-2 py-1.5 rounded-md text-left text-sm bg-transparent border-none outline-none cursor-pointer transition-colors duration-200 hover:bg-hover hover:text-primary focus:bg-hover focus:text-primary"
            :class="launchMode === option.value ? 'text-brand font-medium' : 'text-secondary'"
            :title="option.hint"
            @click="choose(option.value)"
          >
            <span class="flex-1">{{ option.label }}</span>
            <span v-if="launchMode === option.value" class="i-lucide-check icon-xs" aria-hidden="true" />
          </button>
        </div>
      </Transition>
    </Teleport>
  </button>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    /** 生效的启动方式（Steam 源才有 steam 这个选项） */
    launchMode?: 'steam' | 'exe'
    /** 启动中 / 运行中时禁用切换（与「更换位置」的禁用口径一致） */
    disabled?: boolean
  }>(),
  { launchMode: 'exe', disabled: false },
)

const emit = defineEmits<{ select: [mode: 'steam' | 'exe'] }>()

/** 启动方式的两个选项：默认走 Steam URL，也可以退回本地 exe。 */
const LAUNCH_MODE_OPTIONS: { value: 'steam' | 'exe'; label: string; hint: string }[] = [
  { value: 'steam', label: '通过 Steam 启动', hint: '运行 steam://rungameid/<appid>，由 Steam 客户端拉起游戏' },
  { value: 'exe', label: '本地程序启动', hint: '直接运行本机记录的游戏程序（没有记录时会先让你选一次 exe）' },
]

/** 菜单最小宽度 / 与按钮的间距 / 离窗口边缘的安全距离 */
const MENU_MIN_WIDTH = 176
const MENU_GAP = 4
const MENU_EDGE = 8

const open = ref(false)
const triggerEl = ref<HTMLButtonElement | null>(null)
const menuEl = ref<HTMLElement | null>(null)
const menuStyle = ref<Record<string, string>>({})

const currentOption = computed(
  () => LAUNCH_MODE_OPTIONS.find((o) => o.value === props.launchMode) ?? LAUNCH_MODE_OPTIONS[0],
)

/** 摆位：菜单右侧与触发按钮右缘对齐、贴它下方弹出；越界往左收，下方放不下就翻到上方 */
function placeMenu() {
  const trigger = triggerEl.value
  if (!trigger) return
  const rect = trigger.getBoundingClientRect()
  const width = Math.max(rect.width, MENU_MIN_WIDTH)
  // 首帧还没渲染出菜单，用「选项数 × 单行高 + 内边距」估算
  const height = menuEl.value?.offsetHeight ?? LAUNCH_MODE_OPTIONS.length * 34 + 8
  const openUp = rect.bottom + MENU_GAP + height > window.innerHeight - MENU_EDGE
  const top = openUp ? Math.max(MENU_EDGE, rect.top - MENU_GAP - height) : rect.bottom + MENU_GAP
  // 右段按钮本来就窄、菜单比它宽 → 让菜单右缘贴住按钮右缘（向左展开）；
  // 越出窗口时再夹回界内。
  const left = Math.min(Math.max(MENU_EDGE, rect.right - width), window.innerWidth - width - MENU_EDGE)
  menuStyle.value = { top: `${top}px`, left: `${left}px`, width: `${width}px` }
}

/** 打开后把焦点放到当前选中的那项（键盘用户连按 Enter 不会误选） */
function focusChecked() {
  const items = Array.from(menuEl.value?.querySelectorAll<HTMLElement>('[role="menuitemradio"]') ?? [])
  if (!items.length) return
  const checked = items.find((item) => item.getAttribute('aria-checked') === 'true')
  ;(checked ?? items[0]).focus()
}

async function openMenu(focus = false) {
  if (open.value || props.disabled) return
  // 先用「估算高度」摆好位：菜单此刻还没渲染，若等 nextTick 再定位，
  // 中间会有一帧按 style 为空（fixed，左上角）画出来 —— 肉眼就是闪一下。
  placeMenu()
  open.value = true
  await nextTick()
  // 拿到真实高度后校正一次（是否需要翻到上方，靠的就是真实高度）
  placeMenu()
  if (focus) focusChecked()
}

function close(returnFocus = false) {
  if (!open.value) return
  open.value = false
  if (returnFocus) triggerEl.value?.focus()
}

function toggle() {
  if (open.value) close(true)
  else void openMenu()
}

function choose(value: 'steam' | 'exe') {
  close(true)
  if (value !== props.launchMode) emit('select', value)
}

/** ↑↓ 在选项间循环移动焦点（菜单打开时焦点在某一项上） */
function moveFocus(direction: 1 | -1) {
  const items = Array.from(menuEl.value?.querySelectorAll<HTMLElement>('[role="menuitemradio"]') ?? [])
  if (!items.length) return
  const index = items.indexOf(document.activeElement as HTMLElement)
  const next = index === -1 ? 0 : (index + direction + items.length) % items.length
  items[next]?.focus()
}

// ===== 关闭时机：点组件以外的地方 / 页面滚动 / Esc / 窗口尺寸变化时重新摆位 =====
function onPointerDown(event: PointerEvent) {
  const target = event.target as Node | null
  if (!target) return
  if (triggerEl.value?.contains(target) || menuEl.value?.contains(target)) return
  close()
}

function onScroll() {
  close()
}

function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  event.preventDefault()
  close(true)
}

function onResize() {
  placeMenu()
}

function bindWindow(active: boolean) {
  // ⚠️ 不要写成 `const fn = active ? window.addEventListener : ...` 再 fn(...)：
  //    脱离 window 调用的 DOM 方法在 Chromium 里会抛 Illegal invocation。
  if (active) {
    window.addEventListener('pointerdown', onPointerDown, true)
    window.addEventListener('scroll', onScroll, true)
    window.addEventListener('keydown', onKeydown, true)
    window.addEventListener('resize', onResize)
  } else {
    window.removeEventListener('pointerdown', onPointerDown, true)
    window.removeEventListener('scroll', onScroll, true)
    window.removeEventListener('keydown', onKeydown, true)
    window.removeEventListener('resize', onResize)
  }
}

watch(open, (isOpen) => bindWindow(isOpen))

// 被禁用（启动中/运行中）时收起菜单，避免"菜单开着但按钮已禁用"的怪状态
watch(
  () => props.disabled,
  (disabled) => {
    if (disabled) close()
  },
)

onBeforeUnmount(() => {
  if (open.value) bindWindow(false)
})
</script>

<style scoped>
/* 菜单进出场：轻微下移 + 淡入（很短，避免"弹出菜单还要等"的感觉） */
.menu-pop-enter-active,
.menu-pop-leave-active {
  transition: opacity var(--motion-fast) ease, transform var(--motion-fast) ease;
}

.menu-pop-enter-from,
.menu-pop-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
