// frontend/src/styles/theme.ts
// ===== 四色主题引擎 =====
//
// 输入：styles/themePresets.ts 里的一套四色 + 当前明暗模式；
// 输出：整站所有 --color-* / --sidebar-* CSS 变量。
//
// 为什么在运行时算而不是在 global.css 里手写：
//   用户要求「以后主题包括现在，通过四色决定主题，其他次要颜色通过全局计算得出」。
//   手写几十个颜色无法保证新增主题也一致，所以这里以四色为唯一输入，
//   文字/边框/悬停/状态色/滚动条全部由对比度目标推导：
//     · 文字按 WCAG AA（小字 4.5:1）往上取，主文字额外拉到 9:1 以上；
//     · 控件描边按 3:1 取，装饰分隔线只做轻微提亮；
//     · 品牌色直接作填充，作文字时自动调亮/调暗到 4.5:1。
//   这样换一套四色，整站自动得到一套可读的层次，不会出现「新主题下小字看不清」。
//
// 新增主题：只改 themePresets.ts；本文件不用动。
import type { ThemePalette, ThemePreset } from './themePresets'

export type ThemeScheme = 'light' | 'dark'

// ===== 颜色工具（只在本文件内部使用）=====

interface Rgba {
  r: number
  g: number
  b: number
  a: number
}

function clampByte(value: number): number {
  return Math.min(255, Math.max(0, Math.round(value)))
}

/** 解析 #rgb / #rrggbb / rgb() / rgba()；解析失败返回 null */
function parseColor(input: string): Rgba | null {
  const value = input.trim()
  if (!value) return null
  if (value.startsWith('#')) {
    const hex = value.slice(1)
    const expand = (part: string) =>
      part.length === 1 ? parseInt(part + part, 16) : parseInt(part.slice(0, 2), 16)
    if (hex.length === 3 || hex.length === 4) {
      return {
        r: expand(hex[0]),
        g: expand(hex[1]),
        b: expand(hex[2]),
        a: hex.length === 4 ? expand(hex[3]) / 255 : 1,
      }
    }
    if (hex.length === 6 || hex.length === 8) {
      return {
        r: parseInt(hex.slice(0, 2), 16),
        g: parseInt(hex.slice(2, 4), 16),
        b: parseInt(hex.slice(4, 6), 16),
        a: hex.length === 8 ? parseInt(hex.slice(6, 8), 16) / 255 : 1,
      }
    }
    return null
  }
  const match = value.match(/^rgba?\(([^)]+)\)$/i)
  if (!match) return null
  const parts = match[1]
    .split(/[\s,/]+/)
    .filter(Boolean)
    .map(Number)
  if (parts.length < 3 || parts.some((n) => !Number.isFinite(n))) return null
  return { r: parts[0], g: parts[1], b: parts[2], a: parts.length > 3 ? parts[3] : 1 }
}

/** 统一输出：不透明用 #rrggbb（CSS 变量好读），半透明用 rgba() */
function formatColor({ r, g, b, a }: Rgba): string {
  if (a >= 0.999) {
    const hex = (value: number) => clampByte(value).toString(16).padStart(2, '0')
    return `#${hex(r)}${hex(g)}${hex(b)}`
  }
  return `rgba(${clampByte(r)}, ${clampByte(g)}, ${clampByte(b)}, ${Number(a.toFixed(3))})`
}

/** a → b 插值，weight = b 的占比（0 = 全 a，1 = 全 b） */
function mix(a: string, b: string, weight: number): string {
  const ca = parseColor(a)
  const cb = parseColor(b)
  if (!ca || !cb) return a
  const w = Math.min(1, Math.max(0, weight))
  return formatColor({
    r: ca.r + (cb.r - ca.r) * w,
    g: ca.g + (cb.g - ca.g) * w,
    b: ca.b + (cb.b - ca.b) * w,
    a: ca.a + (cb.a - ca.a) * w,
  })
}

/** 改透明度（解析失败原样返回） */
function withAlpha(color: string, alpha: number): string {
  const parsed = parseColor(color)
  if (!parsed) return color
  return formatColor({ ...parsed, a: alpha })
}

/** WCAG 相对亮度 */
function luminance(color: string): number {
  const parsed = parseColor(color)
  if (!parsed) return 0
  const channel = (value: number) => {
    const s = value / 255
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
  }
  return 0.2126 * channel(parsed.r) + 0.7152 * channel(parsed.g) + 0.0722 * channel(parsed.b)
}

/** WCAG 对比度（1–21） */
function contrast(a: string, b: string): number {
  const la = luminance(a)
  const lb = luminance(b)
  const high = Math.max(la, lb)
  const low = Math.min(la, lb)
  return (high + 0.05) / (low + 0.05)
}

/**
 * 从 color 出发朝 target（白/黑）混色，直到与 bg 的对比度 ≥ min。
 * 已经达标就原样返回；二分 24 次足够精确，且结果稳定（同一套四色每次算出来一致）。
 */
function ensureContrast(color: string, bg: string, min: number, target: string): string {
  if (contrast(color, bg) >= min) return color
  let low = 0
  let high = 1
  for (let i = 0; i < 24; i += 1) {
    const mid = (low + high) / 2
    if (contrast(mix(color, target, mid), bg) >= min) high = mid
    else low = mid
  }
  return mix(color, target, high)
}

/**
 * 压在某个底色上的文字色：在近白 / 近黑里选对比度更高的那个。
 * 用于品牌实心按钮、功能色实心按钮、tooltip 反色文字。
 */
function readableOn(background: string): string {
  const white = '#ffffff'
  const black = '#0b0b0f'
  return contrast(white, background) >= contrast(black, background) ? white : black
}

// ===== 四色 → 全站 CSS 变量 =====

export type ThemeVars = Record<string, string>

/**
 * 把一套四色展开成所有主题 CSS 变量。
 * 纯函数：不碰 DOM，方便以后加主题预览 / 单元测试。
 */
export function buildThemeVars(palette: ThemePalette, scheme: ThemeScheme): ThemeVars {
  const dark = scheme === 'dark'
  /** 深色往白混、浅色往黑混：所有派生的方向都由明暗模式决定 */
  const ink = dark ? '#ffffff' : '#000000'

  const page = palette.bgPrimary
  const raised = palette.bgSecondary
  const active = palette.bgTertiary
  const brand = palette.brand

  // 文字：按对比度目标取「刚好达标」的最浅/最深值，保证每个主题都可读
  const textPrimary = ensureContrast(page, page, dark ? 9 : 12, ink)
  const textSecondary = ensureContrast(page, page, dark ? 6.5 : 8, ink)
  const textTertiary = ensureContrast(page, page, dark ? 4.8 : 5.2, ink)
  const textMuted = ensureContrast(page, page, 4.5, ink)

  // 背景：四色原样落在页面/卡片/控件三层；更外层的 titlebar 与 hover 由它们推导
  const bgCard = dark ? mix(raised, active, 0.45) : mix(page, '#ffffff', 0.55)
  const bgHover = active
  const bgBase = dark ? mix(page, '#000000', 0.16) : mix(page, '#000000', 0.03)
  const sidebarBg = dark ? mix(page, '#000000', 0.08) : mix(page, '#ffffff', 0.35)
  const sidebarBorder = mix(page, textPrimary, 0.08)

  // 边框：分隔线只要轻微可见；控件描边按 3:1（非文本对比度要求）
  const border = mix(page, textPrimary, 0.16)
  const borderLight = mix(page, textPrimary, 0.08)
  const borderDark = ensureContrast(mix(page, textPrimary, 0.3), page, 3, ink)

  // 品牌：填充保留四色原值，文字用的品牌色按 4.5:1 自动调亮/调暗
  const brandText = ensureContrast(brand, page, 4.5, ink)
  const textOnBrand = readableOn(brand)

  // 功能色：语义色保持不变色相，只保证在当前底色上可读
  const statusBase = dark
    ? { success: '#4adea3', warning: '#f4fb24', danger: '#f87185', info: '#60cefa' }
    : { success: '#136431', warning: '#a78913', danger: '#ba1d1d', info: '#234dc3' }
  const success = ensureContrast(statusBase.success, page, 4.5, ink)
  const warning = ensureContrast(statusBase.warning, page, 4.5, ink)
  const danger = ensureContrast(statusBase.danger, page, 4.5, ink)
  const info = ensureContrast(statusBase.info, page, 4.5, ink)

  return {
    // 背景
    '--color-bg-base': bgBase,
    '--color-bg-primary': page,
    '--color-bg-secondary': raised,
    '--color-bg-tertiary': active,
    '--color-bg-hover': bgHover,
    '--color-bg-card': bgCard,
    '--color-bg-secondary-soft': withAlpha(raised, 0.6),

    // 文本
    '--color-text-primary': textPrimary,
    '--color-text-secondary': textSecondary,
    '--color-text-tertiary': textTertiary,
    '--color-text-muted': textMuted,
    '--color-text-inverse': readableOn(textPrimary),

    // 品牌（primary = 填充，brand-text = 文字/图标）
    '--color-primary': brand,
    '--color-primary-hover': mix(brand, textPrimary, 0.14),
    '--color-primary-active': mix(brand, textPrimary, 0.28),
    '--color-primary-light': withAlpha(brand, dark ? 0.16 : 0.12),
    '--color-brand-text': brandText,
    '--color-text-on-brand': textOnBrand,
    '--color-divider-on-brand': withAlpha(textOnBrand, 0.3),
    '--color-ring': brandText,
    '--color-ring-soft': withAlpha(brandText, 0.5),
    '--color-bg-check': brand,
    '--color-check-mark': textOnBrand,

    // 边框
    '--color-border': border,
    '--color-border-light': borderLight,
    '--color-border-dark': borderDark,

    // 功能色
    '--color-success': success,
    '--color-warning': warning,
    '--color-danger': danger,
    '--color-info': info,
    '--color-text-on-success': readableOn(success),
    '--color-text-on-danger': readableOn(danger),

    // 阴影 / 滚动条
    '--color-shadow': dark
      ? '0 1px 2px rgba(0, 0, 0, 0.35)'
      : '0 1px 2px rgba(42, 36, 15, 0.06)',
    '--color-shadow-heavy': dark
      ? '0 12px 32px rgba(0, 0, 0, 0.5)'
      : '0 8px 24px rgba(42, 38, 15, 0.12)',
    '--color-scrollbar': withAlpha(textPrimary, 0.18),
    '--color-scrollbar-hover': withAlpha(textPrimary, 0.32),

    // 侧边栏
    '--sidebar-bg': sidebarBg,
    '--sidebar-border': sidebarBorder,

    // 语义别名（老代码按这些名字取色）
    '--color-text-body': textPrimary,
    '--color-text-hint': textMuted,

    // 压在图片上的文字/遮罩：与主题无关，两套明暗都保持白字 + 黑色遮罩
    '--color-text-on-image': '#ffffff',
    '--color-hero-veil': 'rgba(0, 0, 0, 0.55)',
  }
}

/**
 * 把主题写到 <html>：
 *   1. data-theme = light/dark（给 color-scheme 与少量选择器用）；
 *   2. 所有 --color-* 变量写成内联值，覆盖 global.css 里的兜底。
 * 必须在 naiveTheme.syncNaiveTheme() 之前调用 —— 后者用 getComputedStyle 读这些变量。
 */
export function applyThemeVars(preset: ThemePreset, scheme: ThemeScheme): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  const vars = buildThemeVars(preset[scheme], scheme)
  for (const [name, value] of Object.entries(vars)) {
    root.style.setProperty(name, value)
  }
  root.setAttribute('data-theme', scheme)
}
