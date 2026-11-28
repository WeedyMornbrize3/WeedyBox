<!-- frontend/src/components/settings/ThemePalettePicker.vue
     设置页「主题配色」（2026-11-28，最终口径，用户原话：
       「一个圆形 四角显示配色 点击选择配色 自定义以外颜色圆盘不能选中」）

     结构：
       · 一个圆形，用 conic-gradient 把当前主题的四色显示在四个角（左上主背景 /
         右上卡片 / 左下控件底 / 右下品牌色）；
       · 点圆形 → 弹出 naive 的 n-popselect 选择配色（内置主题 + 自定义）；
       · 只有「自定义」态下，圆的四个象限才各自可点 —— 点哪个象限就用
         naive 的 n-color-picker 现调那一档颜色（RGB 三个数字框）；
       · 非自定义态：四个色块是纯显示，不可点，整圆只负责「选配色」。

     为什么用 n-popselect 而不是自绘菜单：naive 的 Popselect 现成、带深色适配，
     与全站控件观感一致（项目规矩：控件外观唯一来源 naive 主题）。
     四个象限是无样式的 hit-area（透明 quarter 按钮），它们没有 naive 等价物，
     所以用原生 button + 全局 focus-visible 兜底焦点。 -->
<template>
  <div class="palette-root">
    <div class="palette-wrap">
      <!-- 非自定义：整圆就是「选择配色」的触发器 -->
      <n-popselect
        v-if="!isCustom"
        :value="presetValue"
        :options="presetOptions"
        trigger="click"
        placement="bottom-start"
        @update:value="themeStore.setThemeId"
      >
        <button
          type="button"
          class="palette-disc"
          :style="{ background: quadrantBg }"
          :title="`主题配色：${presetName}`"
          :aria-label="`主题配色：${presetName}，点击选择配色`"
        />
      </n-popselect>

      <!-- 自定义：圆只做显示；四象限各自可点开取色面板；中心 hub 仍能选配色 -->
      <template v-else>
        <div class="palette-disc" :style="{ background: quadrantBg }" aria-hidden="true" />

        <n-color-picker
          v-for="q in QUADRANTS"
          :key="q.key"
          :value="hexToRgbString(palette[q.key])"
          :show-alpha="false"
          :modes="COLOR_MODES"
          :swatches="swatches"
          :actions="['confirm']"
          :show-preview="true"
          size="small"
          placement="bottom-start"
          @update:value="(value: string | null) => themeStore.setCustomColor(q.key, value)"
        >
          <template #trigger="tp">
            <button
              :ref="tp.ref"
              type="button"
              class="palette-quadrant"
              :class="`palette-quadrant--${q.pos}`"
              :title="`${q.label}：${q.hint}（${palette[q.key]}）`"
              :aria-label="`编辑${q.label}，当前 ${palette[q.key]}`"
              @click="tp.onClick"
            />
          </template>
        </n-color-picker>

        <n-popselect
          :value="presetValue"
          :options="presetOptions"
          trigger="click"
          placement="bottom-start"
          @update:value="themeStore.setThemeId"
        >
          <button type="button" class="palette-hub" title="选择配色" aria-label="选择配色">
            <span class="i-lucide-chevron-down icon-xs" aria-hidden="true" />
          </button>
        </n-popselect>
      </template>
    </div>

    <!-- 自定义态才出现：把当前明暗的四色退回内置默认主题那一套 -->
    <n-button v-if="isCustom" text size="tiny" @click="themeStore.resetCustomColors()">
      恢复默认
    </n-button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NColorPicker, NPopselect } from 'naive-ui'
import { useThemeStore } from '../../stores/Theme'
import type { PaletteKey } from '../../stores/Theme'
import { CUSTOM_THEME_ID, CUSTOM_THEME_NAME, THEME_PRESETS } from '../../styles/themePresets'
import { hexToRgbString, paletteToQuadrant } from '../../utils/color'

const themeStore = useThemeStore()

const isCustom = computed(() => themeStore.isCustom)

/** 当前明暗下正在生效的四色（圆盘四象限 + 取色面板都基于它） */
const palette = computed(() => themeStore.activePalette)

/** 四色 → 四象限圆的背景（见 utils/color.ts 的 paletteToQuadrant） */
const quadrantBg = computed(() => paletteToQuadrant(palette.value))

/**
 * 面板里给哪几种输入方式，第一个是打开时的默认模式。
 * 用户要求「自定义 rgb 选色」→ 传进去的值是 rgb() 格式（见 hexToRgbString），
 * 面板默认就落在 RGB 三个数字框上；点模式名还能切到十六进制。
 */
const COLOR_MODES: Array<'rgb' | 'hex'> = ['rgb', 'hex']

/** 四个象限与四色槽位的对应（顺序/方位须与 paletteToQuadrant 一致） */
const QUADRANTS: readonly { key: PaletteKey; pos: 'tl' | 'tr' | 'bl' | 'br'; label: string; hint: string }[] = [
  { key: 'bgPrimary', pos: 'tl', label: '主背景', hint: '页面底色' },
  { key: 'bgSecondary', pos: 'tr', label: '次背景', hint: '卡片 / 浮层' },
  { key: 'bgTertiary', pos: 'bl', label: '控件底', hint: '输入框 / 悬停' },
  { key: 'brand', pos: 'br', label: '品牌色', hint: '主按钮 / 选中' },
]

/** 面板下方快捷色块：内置主题在当前明暗下的四色去重 */
const swatches = computed(() => {
  const scheme = themeStore.effectiveTheme
  const colors = new Set<string>()
  for (const preset of THEME_PRESETS) {
    for (const value of Object.values(preset[scheme])) colors.add(value)
  }
  for (const value of Object.values(themeStore.customPalette[scheme])) colors.add(value)
  return Array.from(colors)
})

/** 下拉当前值：内置主题 id 或 'custom' */
const presetValue = computed(() => (isCustom.value ? CUSTOM_THEME_ID : themeStore.themeId))

/** 圆盘 title 里显示的名字 */
const presetName = computed(() => themeStore.activePreset.name)

/** 预设选项：内置主题清单（新增主题自动出现）+ 自定义 */
const presetOptions = computed(() => [
  ...THEME_PRESETS.map((preset) => ({ label: preset.name, value: preset.id })),
  { label: CUSTOM_THEME_NAME, value: CUSTOM_THEME_ID },
])
</script>

<style scoped>
.palette-root {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 圆形本体：40px，四色 conic-gradient 从 inline style 来，这里只管形状与描边 */
.palette-wrap {
  position: relative;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
}

.palette-disc {
  width: 40px;
  height: 40px;
  padding: 0;
  border: none;
  border-radius: 9999px;
  cursor: pointer;
  display: block;
  box-shadow:
    0 0 0 1px var(--color-border-dark),
    inset 0 0 0 1px color-mix(in srgb, var(--color-text-on-image) 22%, transparent);
  /* 过渡写在静息态基础选择器上（挂 :hover 会导致退出无动画，见项目动效口径） */
  transition: box-shadow var(--motion-fast) var(--ease-enter);
}

.palette-disc:hover {
  box-shadow:
    0 0 0 1px var(--color-brand-text),
    inset 0 0 0 1px color-mix(in srgb, var(--color-text-on-image) 22%, transparent);
}

/* 四个象限 hit-area：透明、覆盖圆的一半，只在自定义态渲染 */
.palette-quadrant {
  position: absolute;
  width: 50%;
  height: 50%;
  padding: 0;
  border: none;
  background: transparent;
  cursor: pointer;
}

.palette-quadrant--tl { top: 0; left: 0; border-radius: 9999px 0 0 0; }
.palette-quadrant--tr { top: 0; right: 0; border-radius: 0 9999px 0 0; }
.palette-quadrant--bl { bottom: 0; left: 0; border-radius: 0 0 0 9999px; }
.palette-quadrant--br { bottom: 0; right: 0; border-radius: 0 0 9999px 0; }

.palette-quadrant:hover {
  background: color-mix(in srgb, var(--color-text-on-image) 14%, transparent);
}

/* 中心 hub：自定义态里仍然能「选择配色」的小圆钮 */
.palette-hub {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 9999px;
  background: var(--color-bg-card);
  box-shadow: 0 0 0 1px var(--color-border-dark);
  color: var(--color-text-tertiary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2;
  transition:
    color var(--motion-fast) var(--ease-enter),
    box-shadow var(--motion-fast) var(--ease-enter);
}

.palette-hub:hover {
  color: var(--color-brand-text);
  box-shadow: 0 0 0 1px var(--color-brand-text);
}
</style>
