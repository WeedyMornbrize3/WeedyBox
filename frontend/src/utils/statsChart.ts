// frontend/src/utils/statsChart.ts
//
// 分析页图表与时长文案的**共用工具**（纯函数，不含组件）。
//
// 为什么单独一个文件：
//   · 图表配色必须跟着四色主题走 —— 全站外观的唯一来源是 styles/naiveTheme.ts 的
//     readDesignTokens()（它读的是 theme.ts 写到 <html> 上的 --color-*），
//     所以这里也只能从它取色，绝不能在这里硬编码色值或再调一次 getComputedStyle。
//   · 时长格式（"1 小时 23 分" / "45 分" / "不到 1 分"）在总览、名次、单日表三处都要用，
//     写三份迟早不一致。
import { readDesignTokens } from '../styles/naiveTheme'

/** 时长格式化的档位：秒 → 人话（中文口径，与全站一致） */
export function formatDuration(seconds: number): string {
  const total = Math.max(0, Math.round(seconds || 0))
  if (total === 0) return '0 分'
  if (total < 60) return '不到 1 分'
  const hours = Math.floor(total / 3600)
  const minutes = Math.floor((total % 3600) / 60)
  if (hours === 0) return `${minutes} 分`
  if (minutes === 0) return `${hours} 小时`
  return `${hours} 小时 ${minutes} 分`
}

/** 图表纵轴要用的紧凑刻度：秒 → "2h" / "30m"（轴标签空间有限，不能写一长串中文） */
export function formatAxisDuration(seconds: number): string {
  const total = Math.max(0, Math.round(seconds || 0))
  if (total < 3600) return `${Math.round(total / 60)}m`
  const hours = total / 3600
  return Number.isInteger(hours) ? `${hours}h` : `${hours.toFixed(1)}h`
}

/** "2026-11-27" → "11-27"（横轴标签用短格式） */
export function shortDay(day: string): string {
  return day.length >= 10 ? day.slice(5) : day
}

/**
 * 图表配色：全部从主题 token 派生。
 *
 * 为什么每次调用都重新读：换主题/换明暗时 --color-* 会被 theme.ts 重写，
 * 缓存住的话图表会停在上一个主题的颜色。读 token 只是 getComputedStyle 一次，
 * 在"用户点开报表页/换主题"这种低频场景里完全不是瓶颈。
 */
export interface ChartPalette {
  brand: string
  brandText: string
  grid: string
  axis: string
  text: string
  textTertiary: string
  tooltipBg: string
  tooltipBorder: string
  areaFrom: string
  areaTo: string
}

export function chartPalette(): ChartPalette {
  const t = readDesignTokens()
  return {
    brand: t.primary,
    brandText: t.brandText,
    grid: t.borderLight,
    axis: t.border,
    text: t.textPrimary,
    textTertiary: t.textTertiary,
    tooltipBg: t.bgSecondary,
    tooltipBorder: t.border,
    // 面积图渐变：品牌色 → 透明。用 rgba 拼，避免再引一个颜色库
    areaFrom: hexToRgba(t.primary, 0.28),
    areaTo: hexToRgba(t.primary, 0.02),
  }
}

/**
 * 面积渐变对象（ECharts 的 LinearGradient）。
 *
 * 为什么不直接写颜色字符串：areaStyle 支持 `new echarts.graphic.LinearGradient(...)`，
 * 但那样要把 echarts 的 graphic 也引入组件层；用等价的普通对象字面量即可
 * （ECharts 认 {type,s,x,y,x2,y2,colorStops} 这种结构），组件侧就不用依赖 echarts 命名空间。
 */
export function verticalAreaGradient(from: string, to: string) {
  return {
    type: 'linear' as const,
    x: 0,
    y: 0,
    x2: 0,
    y2: 1,
    colorStops: [
      { offset: 0, color: from },
      { offset: 1, color: to },
    ],
  }
}

/**
 * #rrggbb / #rgb → rgba(r,g,b,a)。
 * 主题色都是 hex（theme.ts 的 mix/ensureContrast 输出 hex），这里不做到处兜底；
 * 遇到不认识的写法（rgb()/颜色名）就返回原色 —— 渐变退化成纯色，但不会白屏。
 */
export function hexToRgba(color: string, alpha: number): string {
  const hex = color.trim()
  if (!hex.startsWith('#')) return hex
  const expand = (s: string) => (s.length === 1 ? s + s : s)
  const body = hex.slice(1)
  const parts =
    body.length === 3
      ? [expand(body[0]), expand(body[1]), expand(body[2])]
      : body.length >= 6
        ? [body.slice(0, 2), body.slice(2, 4), body.slice(4, 6)]
        : []
  if (!parts.length) return hex
  const [r, g, b] = parts.map((p) => parseInt(p, 16))
  if ([r, g, b].some((n) => Number.isNaN(n))) return hex
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

/**
 * ECharts 的通用基础配置（四个图表共用）：
 * 透明背景（卡片底色由 naive 的 n-card 提供）、统一字体、不显示缩放动画。
 *
 * textStyle 的 fontFamily 直接继承页面字体：ECharts 默认是 sans-serif，
 * 与全站的 Inter/系统字体不一致，中文会明显"变了一个字重"。
 */
export function baseChartOption() {
  const p = chartPalette()
  return {
    backgroundColor: 'transparent',
    textStyle: { color: p.text, fontFamily: 'inherit' },
    animationDuration: 320,
    animationEasing: 'cubicOut' as const,
  }
}

/** 统一的 tooltip 外观（与 naive 的浮层口径一致：卡片底 + 细描边 + 小圆角） */
export function chartTooltip() {
  const p = chartPalette()
  return {
    backgroundColor: p.tooltipBg,
    borderColor: p.tooltipBorder,
    borderWidth: 1,
    padding: [8, 10] as [number, number],
    textStyle: { color: p.text, fontSize: 12 },
    extraCssText: 'border-radius:6px; box-shadow:var(--color-shadow);',
  }
}
