// frontend/src/styles/themePresets.ts
// ===== 四色主题清单（新增主题只改这一个文件）=====
//
// 用户 2026-11-25 定的口径：**一套主题由四个颜色决定** ——
//   ① 主要背景色（页面底）
//   ② 次一级背景色（卡片 / 浮层）
//   ③ 再次一级背景色（输入框 / 悬停 / 控件底）
//   ④ 品牌色（最次一级，也用于主按钮、勾选、焦点环）
// 其余 token（文字、边框、hover 派生、状态色、阴影、滚动条…）全部由
// styles/theme.ts 的 buildThemeVars() 统一计算，不要在主题清单里手写第二份颜色。
//
// 要加主题：往 THEME_PRESETS 里追加一项（浅色 / 深色各四个颜色即可），
// 设置页会自动出现新的选项；其它代码不用动。
//
// 配色来源：https://colorhunt.co/palettes/popular
//   · 夜岚（默认）：404258 / 474E68 / 50577A / 6B728E
//   · 暖砂：保留迁移前的暖米色 + 青绿，作为回退选项。

/** 一套主题的四个基准色（其余颜色由 theme.ts 推导） */
export interface ThemePalette {
  /** 主要背景色 */
  bgPrimary: string
  /** 次一级背景色 */
  bgSecondary: string
  /** 再次一级背景色 */
  bgTertiary: string
  /** 品牌色（最次一级） */
  brand: string
}

export interface ThemePreset {
  /** 稳定 id：写进 localStorage，改名不影响已保存的选择 */
  id: string
  /** 设置页显示的名字 */
  name: string
  /** 设置页的一句话说明 */
  description: string
  /** 浅色模式下的四色 */
  light: ThemePalette
  /** 深色模式下的四色 */
  dark: ThemePalette
}

export const THEME_PRESETS: readonly ThemePreset[] = [
  {
    id: 'nightfall',
    name: '夜岚',
    description: '雾紫灰四阶，四色决定整体层次',
    light: {
      bgPrimary: '#f2f1f6',
      bgSecondary: '#e6e5ee',
      bgTertiary: '#dad9e6',
      brand: '#6b728e',
    },
    dark: {
      bgPrimary: '#404258',
      bgSecondary: '#474e68',
      bgTertiary: '#50577a',
      brand: '#6b728e',
    },
  },
  {
    id: 'ember',
    name: '暖砂',
    description: '迁移前的暖米色 + 青绿（旧观感）',
    light: {
      bgPrimary: '#f3eee5',
      bgSecondary: '#e6ddcc',
      bgTertiary: '#e6decf',
      brand: '#52621f',
    },
    dark: {
      bgPrimary: '#101418',
      bgSecondary: '#1b222a',
      bgTertiary: '#232c35',
      brand: '#2dd4bf',
    },
  },
]

export const DEFAULT_THEME_ID = 'nightfall'

/**
 * 按 id 取主题；未知 id（旧版本 localStorage / 拼错）回落到默认主题，
 * 保证任何情况下都能拿到一份可用的四色。
 */
export function getThemePreset(id?: string | null): ThemePreset {
  const found = THEME_PRESETS.find((preset) => preset.id === id)
  return found ?? THEME_PRESETS.find((preset) => preset.id === DEFAULT_THEME_ID) ?? THEME_PRESETS[0]
}
