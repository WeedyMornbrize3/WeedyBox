// frontend/src/utils/color.ts
// 设置页「自定义四色」用到的几个纯函数：校验、hex ↔ rgb 互转、明暗判断。
//
// 为什么单独放一个文件而不是塞进 styles/theme.ts：
//   theme.ts 是「四色 → 全站变量」的计算引擎，只认已经合法的 ThemePalette；
//   这里处理的是**用户输入**（ColorPicker 可能给出 rgba() / #rrggbbaa / 半截 hex），
//   职责是"把任意输入收敛成 #rrggbb"。两者混在一起会让 theme.ts 也背上容错分支。

export interface Rgb {
  r: number
  g: number
  b: number
}

function clampByte(value: number): number {
  return Math.min(255, Math.max(0, Math.round(value)))
}

function toHexByte(value: number): string {
  return clampByte(value).toString(16).padStart(2, '0')
}

/** rgb 三元组 → #rrggbb（小写，与 themePresets.ts 里手写的色值同一种写法） */
export function rgbToHex({ r, g, b }: Rgb): string {
  return `#${toHexByte(r)}${toHexByte(g)}${toHexByte(b)}`
}

/**
 * 任意 CSS 颜色 → #rrggbb；解析失败返回 null。
 *
 * 支持 #rgb / #rgba / #rrggbb / #rrggbbaa 与 rgb() / rgba()。
 * 半透明一律**丢掉 alpha**：四色主题的四个槽位必须是实色 —— theme.ts 会拿它们
 * 算对比度、做混色，带 alpha 的底色会让整站文字对比度失去意义（也存不进 localStorage 的
 * 简单格式）。ColorPicker 那边已经把 show-alpha 关掉了，这里只是兜底。
 */
export function normalizeHex(input: string | null | undefined): string | null {
  if (!input) return null
  const value = input.trim().toLowerCase()
  if (!value) return null

  if (value.startsWith('#')) {
    const hex = value.slice(1)
    const expand = (part: string) => parseInt(part.length === 1 ? part + part : part, 16)
    if (!/^[0-9a-f]+$/.test(hex)) return null
    if (hex.length === 3 || hex.length === 4) {
      return rgbToHex({ r: expand(hex[0]), g: expand(hex[1]), b: expand(hex[2]) })
    }
    if (hex.length === 6 || hex.length === 8) {
      return rgbToHex({
        r: parseInt(hex.slice(0, 2), 16),
        g: parseInt(hex.slice(2, 4), 16),
        b: parseInt(hex.slice(4, 6), 16),
      })
    }
    return null
  }

  const match = value.match(/^rgba?\(([^)]+)\)$/)
  if (!match) return null
  const parts = match[1]
    .split(/[\s,/]+/)
    .filter(Boolean)
    .map(Number)
  if (parts.length < 3 || parts.slice(0, 3).some((n) => !Number.isFinite(n))) return null
  return rgbToHex({ r: parts[0], g: parts[1], b: parts[2] })
}

/**
 * #rrggbb → `rgb(r, g, b)`；解析失败原样返回。
 *
 * 设置页的取色面板用它当 v-model 传入值：naive 的 ColorPicker 会按传进来的格式
 * 决定打开时的输入模式（`#…` → HEX、含 rgb → RGB），所以传 rgb() 就能让面板
 * 直接以「RGB 三个数字框」打开（用户要求「提供自定义 rgb 选色」）。
 */
export function hexToRgbString(hex: string): string {
  const normalized = normalizeHex(hex)
  if (!normalized) return hex
  const r = parseInt(normalized.slice(1, 3), 16)
  const g = parseInt(normalized.slice(3, 5), 16)
  const b = parseInt(normalized.slice(5, 7), 16)
  return `rgb(${r}, ${g}, ${b})`
}

/**
 * 一套四色 → 四象限圆（conic-gradient）的 background 值。
 *
 * 设置页的配色圆盘用它画「一个圆形、四角显示四种颜色」：
 *   conic-gradient 从 12 点方向顺时针走，所以
 *     0–90°  = 右上 → bgSecondary
 *     90–180° = 右下 → brand
 *     180–270° = 左下 → bgTertiary
 *     270–360° = 左上 → bgPrimary
 *   正好是「左上主背景 → 右上卡片 → 左下控件底 → 右下品牌色」的读图顺序。
 *
 * 传进来的是已经合法的 ThemePalette（theme.ts 侧同款数据结构），
 * 这里不做容错：颜色来源要么是内置主题常量，要么是 store 里 sanitize 过的值。
 */
export function paletteToQuadrant(palette: {
  bgPrimary: string
  bgSecondary: string
  bgTertiary: string
  brand: string
}): string {
  const { bgPrimary, bgSecondary, bgTertiary, brand } = palette
  return `conic-gradient(${bgSecondary} 0deg 90deg, ${brand} 90deg 180deg, ${bgTertiary} 180deg 270deg, ${bgPrimary} 270deg 360deg)`
}

/**
 * 颜色是不是"偏深"（用来给设置页圆盘里的文字选黑/白）。
 * 用 WCAG 相对亮度而不是 `r+g+b < 384` 这类经验值，浅黄/深蓝这类极端色才不会判错。
 */
export function isDarkColor(hex: string): boolean {
  const normalized = normalizeHex(hex)
  if (!normalized) return true
  const channel = (value: number) => {
    const s = value / 255
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
  }
  const r = channel(parseInt(normalized.slice(1, 3), 16))
  const g = channel(parseInt(normalized.slice(3, 5), 16))
  const b = channel(parseInt(normalized.slice(5, 7), 16))
  return 0.2126 * r + 0.7152 * g + 0.0722 * b < 0.36
}
