// frontend/src/styles/naiveTheme.ts
// 设计 token → naive-ui 主题的桥接层（2026-11-24 引入 naive-ui）。
//
// 为什么要在运行时读 CSS 变量，而不是在 TS 里再抄一份色值：
//   naive-ui 会拿主题里的颜色做二次计算（hover/pressed 派生、半透明叠加、
//   color-mix 之类它自己也算不了），所以 themeOverrides 里必须给**可直接解析的
//   具体色值**。如果写 `var(--color-primary)`，seemly 解析失败会静默退化成
//   默认色（或 NaN），表现为「按钮 hover 变成 naive 默认绿」这类不着痕迹的错。
//   因此这里在主题切换时从 documentElement 读一次计算后的变量值，
//   唯一的色值来源仍然是 styles/global.css。
//
// 层级：global.css（token 唯一来源）→ 本文件（token → naive 主题）→ 组件。
// ⚠️ 本文件不出现任何硬编码色值（`#fff` / `rgba(0,0,0,...)` 一律由 token 计算）。
import type { GlobalThemeOverrides } from 'naive-ui'

/** 读取到的设计 token（颜色全部是浏览器计算后的字符串，可能是 hex / rgba） */
export interface DesignTokens {
  bgBase: string
  bgPrimary: string
  bgSecondary: string
  bgTertiary: string
  bgHover: string
  bgCard: string
  bgSecondarySoft: string
  textPrimary: string
  textSecondary: string
  textTertiary: string
  textMuted: string
  textInverse: string
  primary: string
  primaryHover: string
  primaryActive: string
  primaryLight: string
  /** 品牌色作为文字/细线时的版本（theme.ts 按当前底色调到 4.5:1） */
  brandText: string
  border: string
  borderLight: string
  borderDark: string
  success: string
  warning: string
  danger: string
  info: string
  shadow: string
  shadowHeavy: string
  textOnBrand: string
  textOnSuccess: string
  textOnDanger: string
  checkMark: string
  scrollbar: string
  scrollbarHover: string
  fontFamily: string
}

/** 取不到 CSS 变量时的兜底色（正常情况下永远不会用到） */
const FALLBACK: DesignTokens = {
  bgBase: '#101418',
  bgPrimary: '#101418',
  bgSecondary: '#1b222a',
  bgTertiary: '#232c35',
  bgHover: '#232c35',
  bgCard: '#171c21',
  bgSecondarySoft: 'rgba(27, 34, 42, 0.6)',
  textPrimary: '#e6edf3',
  textSecondary: '#c3cdd6',
  textTertiary: '#9aa6b2',
  textMuted: '#8a94a0',
  textInverse: '#0b1116',
  primary: '#2dd4bf',
  primaryHover: '#5eead4',
  primaryActive: '#14b8a6',
  primaryLight: 'rgba(45, 212, 191, 0.14)',
  brandText: '#5eead4',
  border: '#2b3540',
  borderLight: '#232c35',
  borderDark: '#4a5764',
  success: '#4adea3',
  warning: '#f4fb24',
  danger: '#f87185',
  info: '#60cefa',
  shadow: '0 1px 2px rgba(0, 0, 0, 0.4)',
  shadowHeavy: '0 8px 24px rgba(0, 0, 0, 0.55)',
  textOnBrand: '#04211f',
  textOnSuccess: '#04211f',
  textOnDanger: '#2a0a0a',
  checkMark: '#04211f',
  scrollbar: 'rgba(149, 155, 171, 0.16)',
  scrollbarHover: 'rgba(31, 35, 48, 0.3)',
  fontFamily: 'system-ui, sans-serif',
}

/** 从计算样式里读一个 CSS 变量；空值回落到兜底值 */
function readVar(style: CSSStyleDeclaration, name: string, fallback: string): string {
  const value = style.getPropertyValue(name).trim()
  return value || fallback
}

/**
 * 读当前生效主题（浅色 :root / 深色 [data-theme="dark"]）下的全部 token。
 * 调用前必须确保 data-theme 已经写到 documentElement 上，否则会读到上一套主题。
 */
export function readDesignTokens(): DesignTokens {
  if (typeof window === 'undefined' || typeof document === 'undefined') return FALLBACK
  const style = getComputedStyle(document.documentElement)
  const bodyFont = getComputedStyle(document.body).fontFamily.trim()
  return {
    bgBase: readVar(style, '--color-bg-base', FALLBACK.bgBase),
    bgPrimary: readVar(style, '--color-bg-primary', FALLBACK.bgPrimary),
    bgSecondary: readVar(style, '--color-bg-secondary', FALLBACK.bgSecondary),
    bgTertiary: readVar(style, '--color-bg-tertiary', FALLBACK.bgTertiary),
    bgHover: readVar(style, '--color-bg-hover', FALLBACK.bgHover),
    bgCard: readVar(style, '--color-bg-card', FALLBACK.bgCard),
    bgSecondarySoft: readVar(style, '--color-bg-secondary-soft', FALLBACK.bgSecondarySoft),
    textPrimary: readVar(style, '--color-text-primary', FALLBACK.textPrimary),
    textSecondary: readVar(style, '--color-text-secondary', FALLBACK.textSecondary),
    textTertiary: readVar(style, '--color-text-tertiary', FALLBACK.textTertiary),
    textMuted: readVar(style, '--color-text-muted', FALLBACK.textMuted),
    textInverse: readVar(style, '--color-text-inverse', FALLBACK.textInverse),
    primary: readVar(style, '--color-primary', FALLBACK.primary),
    primaryHover: readVar(style, '--color-primary-hover', FALLBACK.primaryHover),
    primaryActive: readVar(style, '--color-primary-active', FALLBACK.primaryActive),
    primaryLight: readVar(style, '--color-primary-light', FALLBACK.primaryLight),
    brandText: readVar(style, '--color-brand-text', FALLBACK.brandText),
    border: readVar(style, '--color-border', FALLBACK.border),
    borderLight: readVar(style, '--color-border-light', FALLBACK.borderLight),
    borderDark: readVar(style, '--color-border-dark', FALLBACK.borderDark),
    success: readVar(style, '--color-success', FALLBACK.success),
    warning: readVar(style, '--color-warning', FALLBACK.warning),
    danger: readVar(style, '--color-danger', FALLBACK.danger),
    info: readVar(style, '--color-info', FALLBACK.info),
    shadow: readVar(style, '--color-shadow', FALLBACK.shadow),
    shadowHeavy: readVar(style, '--color-shadow-heavy', FALLBACK.shadowHeavy),
    textOnBrand: readVar(style, '--color-text-on-brand', FALLBACK.textOnBrand),
    textOnSuccess: readVar(style, '--color-text-on-success', FALLBACK.textOnSuccess),
    textOnDanger: readVar(style, '--color-text-on-danger', FALLBACK.textOnDanger),
    checkMark: readVar(style, '--color-check-mark', FALLBACK.checkMark),
    scrollbar: readVar(style, '--color-scrollbar', FALLBACK.scrollbar),
    scrollbarHover: readVar(style, '--color-scrollbar-hover', FALLBACK.scrollbarHover),
    fontFamily: bodyFont || FALLBACK.fontFamily,
  }
}

// ===== 颜色工具（只用 token 推算，不写死色值）=====

interface Rgba {
  r: number
  g: number
  b: number
  a: number
}

/** 解析 #rgb / #rrggbb / #rrggbbaa / rgb() / rgba()；解析不了返回 null */
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

function toRgbaString({ r, g, b, a }: Rgba): string {
  return `rgba(${Math.round(r)}, ${Math.round(g)}, ${Math.round(b)}, ${Number(a.toFixed(3))})`
}

/** 把颜色改成指定透明度（解析失败时原样返回，交给 naive 自己兜底） */
function withAlpha(color: string, alpha: number): string {
  const parsed = parseColor(color)
  return parsed ? toRgbaString({ ...parsed, a: alpha }) : color
}

/**
 * 颜色插值：weight 是 b 的占比（0 = 全 a，1 = 全 b）。
 * 用来从 token 推 hover / pressed 这类 naive 需要的派生色，避免写死第二份色值。
 * 往 text-primary 方向混的做法在两套主题下都成立：浅色主文字是近黑 → 变深；
 * 深色主文字是近白 → 变亮。正好符合 naive 对 hover/pressed 的取向。
 */
function mix(a: string, b: string, weight: number): string {
  const ca = parseColor(a)
  const cb = parseColor(b)
  if (!ca || !cb) return a
  const w = Math.min(1, Math.max(0, weight))
  return toRgbaString({
    r: ca.r + (cb.r - ca.r) * w,
    g: ca.g + (cb.g - ca.g) * w,
    b: ca.b + (cb.b - ca.b) * w,
    a: ca.a + (cb.a - ca.a) * w,
  })
}

/**
 * 把设计 token 展开成 naive-ui 的 themeOverrides。
 *
 * 统一口径：
 *   - 控件高度 35.2px（= 2.2rem，原 control-h 的口径），圆角 6px（按钮）/ 8–12px（容器）；
 *   - 所有「面」的颜色都走 token：卡片/弹层 = bg-card，输入框 = bg-secondary，
 *     页面底 = bg-primary，悬停 = bg-hover；
 *   - 语义色只用于「状态」，不整块铺底（延续项目 Flat 取向）。
 */
export function buildThemeOverrides(t: DesignTokens): GlobalThemeOverrides {
  const hoverOf = (color: string) => mix(color, t.textPrimary, 0.18)
  const pressOf = (color: string) => mix(color, t.textPrimary, 0.34)
  const accentSoft = withAlpha(t.primary, 0.12)
  const neutralSoft = withAlpha(t.textPrimary, 0.06)
  const neutralSoftHover = withAlpha(t.textPrimary, 0.1)

  // 输入框外观（Input / InputNumber / Pagination 的跳页框共用一套）
  const inputLike = {
    heightMedium: '35.2px',
    heightSmall: '28px',
    heightLarge: '40px',
    borderRadius: '8px',
    paddingMedium: '0 12px',
    paddingSmall: '0 10px',
    fontSizeMedium: '14px',
    fontSizeSmall: '12px',
    textColor: t.textPrimary,
    textColorDisabled: t.textMuted,
    placeholderColor: t.textMuted,
    placeholderColorDisabled: t.textMuted,
    color: t.bgSecondary,
    colorHover: t.bgSecondary,
    colorFocus: t.bgSecondary,
    colorDisabled: t.bgTertiary,
    border: `1px solid ${t.border}`,
    borderHover: `1px solid ${t.borderDark}`,
    borderFocus: `1px solid ${t.brandText}`,
    borderDisabled: `1px solid ${t.border}`,
    boxShadowFocus: 'none',
    caretColor: t.brandText,
    iconColor: t.textTertiary,
    iconColorHover: t.textPrimary,
    iconColorPressed: t.textSecondary,
    clearColor: t.textMuted,
    clearColorHover: t.textPrimary,
    clearColorPressed: t.textPrimary,
    loadingColor: t.brandText,
    countTextColor: t.textMuted,
    suffixTextColor: t.textTertiary,
  }

  // 下拉选择器（Select / Pagination 的每页数选择共用）
  const selectionLike = {
    heightMedium: '35.2px',
    heightSmall: '28px',
    borderRadius: '8px',
    fontSizeMedium: '14px',
    fontWeight: '400',
    textColor: t.textPrimary,
    textColorDisabled: t.textMuted,
    placeholderColor: t.textMuted,
    placeholderColorDisabled: t.textMuted,
    color: t.bgSecondary,
    colorDisabled: t.bgTertiary,
    colorActive: t.bgSecondary,
    border: `1px solid ${t.border}`,
    borderHover: `1px solid ${t.borderDark}`,
    borderActive: `1px solid ${t.brandText}`,
    borderFocus: `1px solid ${t.brandText}`,
    borderDisabled: `1px solid ${t.border}`,
    boxShadowActive: 'none',
    boxShadowFocus: 'none',
    arrowColor: t.textTertiary,
    arrowColorDisabled: t.textMuted,
    clearColor: t.textMuted,
    clearColorHover: t.textPrimary,
    clearColorPressed: t.textPrimary,
    paddingSingle: '0 26px 0 12px',
    paddingMultiple: '0 26px 0 8px',
  }

  // 下拉菜单本体（Select 的选项列表）
  const menuLike = {
    borderRadius: '8px',
    color: t.bgCard,
    optionHeightMedium: '32px',
    optionFontSizeMedium: '13px',
    optionTextColor: t.textSecondary,
    optionTextColorHover: t.textPrimary,
    optionTextColorActive: t.brandText,
    optionColorHover: t.bgHover,
    optionColorActive: accentSoft,
    optionColorPending: t.bgHover,
    groupHeaderTextColor: t.textMuted,
    actionDividerColor: t.borderLight,
    padding: '4px',
  }

  // 按钮（默认 / primary / error / success 四个 type；quaternary 等变体沿用这些颜色）
  const buttonLike = {
    heightMedium: '35.2px',
    heightSmall: '28px',
    heightTiny: '22px',
    heightLarge: '40px',
    borderRadiusMedium: '6px',
    borderRadiusSmall: '4px',
    borderRadiusLarge: '8px',
    fontSizeMedium: '14px',
    fontSizeSmall: '12px',
    fontSizeTiny: '12px',
    fontWeight: '500',
    fontWeightStrong: '500',
    paddingMedium: '0 16px',
    paddingSmall: '0 12px',
    paddingTiny: '0 8px',
    // 默认档：透明底 + 次要文字色，hover 才浮出底色（等价于原来的 btn-ghost）
    color: 'transparent',
    colorHover: t.bgHover,
    colorPressed: t.bgTertiary,
    colorFocus: t.bgHover,
    colorDisabled: 'transparent',
    textColor: t.textSecondary,
    textColorHover: t.textPrimary,
    textColorPressed: t.textPrimary,
    textColorFocus: t.textPrimary,
    textColorDisabled: t.textMuted,
    // 文字按钮（text 类型）与 ghost 按钮：主色文字用对比安全的 brandText
    textColorText: t.textSecondary,
    textColorTextHover: t.textPrimary,
    textColorTextPressed: t.textPrimary,
    textColorTextFocus: t.textPrimary,
    textColorTextPrimary: t.brandText,
    textColorTextHoverPrimary: t.brandText,
    textColorTextPressedPrimary: t.brandText,
    textColorTextFocusPrimary: t.brandText,
    textColorGhostPrimary: t.brandText,
    textColorGhostHoverPrimary: t.brandText,
    textColorGhostPressedPrimary: t.brandText,
    textColorGhostFocusPrimary: t.brandText,
    border: '0 solid transparent',
    borderHover: '0 solid transparent',
    borderPressed: '0 solid transparent',
    borderFocus: '0 solid transparent',
    iconColor: t.textSecondary,
    // primary：品牌实心（等价于原来的 btn-brand）
    colorPrimary: t.primary,
    colorHoverPrimary: t.primaryHover,
    colorPressedPrimary: t.primaryActive,
    colorFocusPrimary: t.primaryHover,
    textColorPrimary: t.textOnBrand,
    textColorHoverPrimary: t.textOnBrand,
    textColorPressedPrimary: t.textOnBrand,
    textColorFocusPrimary: t.textOnBrand,
    borderPrimary: '0 solid transparent',
    // error：破坏性按钮
    colorError: t.danger,
    colorHoverError: hoverOf(t.danger),
    colorPressedError: pressOf(t.danger),
    colorFocusError: hoverOf(t.danger),
    textColorError: t.textOnDanger,
    textColorHoverError: t.textOnDanger,
    textColorPressedError: t.textOnDanger,
    textColorFocusError: t.textOnDanger,
    borderError: '0 solid transparent',
    // success：确认类按钮（很少用，但与语义色保持一致）
    colorSuccess: t.success,
    colorHoverSuccess: hoverOf(t.success),
    colorPressedSuccess: pressOf(t.success),
    textColorSuccess: t.textOnSuccess,
    textColorHoverSuccess: t.textOnSuccess,
    textColorPressedSuccess: t.textOnSuccess,
    textColorFocusSuccess: t.textOnSuccess,
    borderSuccess: '0 solid transparent',
    opacityDisabled: '0.5',
  }

  return {
    common: {
      // 品牌与语义色（hover/pressed 必须同时给，否则 naive 会保留它自己的默认派生色）
      primaryColor: t.primary,
      primaryColorHover: t.primaryHover,
      primaryColorPressed: t.primaryActive,
      primaryColorSuppl: t.primaryHover,
      infoColor: t.info,
      infoColorHover: hoverOf(t.info),
      infoColorPressed: pressOf(t.info),
      infoColorSuppl: hoverOf(t.info),
      successColor: t.success,
      successColorHover: hoverOf(t.success),
      successColorPressed: pressOf(t.success),
      successColorSuppl: hoverOf(t.success),
      warningColor: t.warning,
      warningColorHover: hoverOf(t.warning),
      warningColorPressed: pressOf(t.warning),
      warningColorSuppl: hoverOf(t.warning),
      errorColor: t.danger,
      errorColorHover: hoverOf(t.danger),
      errorColorPressed: pressOf(t.danger),
      errorColorSuppl: hoverOf(t.danger),

      // 文字
      baseColor: t.textInverse,
      textColorBase: t.textPrimary,
      textColor1: t.textPrimary,
      textColor2: t.textSecondary,
      textColor3: t.textTertiary,
      textColorDisabled: t.textMuted,
      placeholderColor: t.textMuted,
      placeholderColorDisabled: t.textMuted,
      iconColor: t.textTertiary,
      iconColorHover: t.textPrimary,
      iconColorPressed: t.textPrimary,
      iconColorDisabled: t.textMuted,
      closeIconColor: t.textTertiary,
      closeIconColorHover: t.textPrimary,
      closeIconColorPressed: t.textPrimary,
      closeColorHover: t.bgHover,
      closeColorPressed: t.bgTertiary,
      clearColor: t.textMuted,
      clearColorHover: t.textPrimary,
      clearColorPressed: t.textPrimary,

      // 面 / 边
      borderColor: t.border,
      dividerColor: t.borderLight,
      popoverColor: t.bgCard,
      cardColor: t.bgCard,
      modalColor: t.bgCard,
      bodyColor: t.bgPrimary,
      inputColor: t.bgSecondary,
      inputColorDisabled: t.bgTertiary,
      tableColor: t.bgCard,
      tableHeaderColor: t.bgSecondary,
      tableColorHover: t.bgHover,
      tableColorStriped: t.bgSecondarySoft,
      tagColor: t.bgSecondary,
      tabColor: t.bgPrimary,
      actionColor: t.bgSecondary,
      hoverColor: t.bgHover,
      pressedColor: t.bgTertiary,
      codeColor: t.bgSecondary,
      railColor: t.border,
      progressRailColor: t.borderLight,
      avatarColor: t.bgSecondary,
      invertedColor: t.textPrimary,
      buttonColor2: neutralSoft,
      buttonColor2Hover: neutralSoftHover,
      buttonColor2Pressed: withAlpha(t.textPrimary, 0.14),

      // 滚动条（naive 的 Scrollbar 内部用，尺寸与项目滚动条一致）
      scrollbarColor: t.scrollbar,
      scrollbarColorHover: t.scrollbarHover,
      scrollbarWidth: '8px',
      scrollbarHeight: '8px',
      scrollbarBorderRadius: '4px',

      // 浮层阴影：Flat 取向，只用 token 里那两级
      boxShadow1: t.shadow,
      boxShadow2: t.shadowHeavy,
      boxShadow3: t.shadowHeavy,

      // 形状 / 排版
      borderRadius: '6px',
      borderRadiusSmall: '4px',
      fontSize: '14px',
      fontSizeMini: '11px',
      fontSizeTiny: '12px',
      fontSizeSmall: '12px',
      fontSizeMedium: '14px',
      fontSizeLarge: '16px',
      fontSizeHuge: '18px',
      lineHeight: '1.6',
      heightMini: '22px',
      heightTiny: '22px',
      heightSmall: '28px',
      heightMedium: '35.2px',
      heightLarge: '40px',
      heightHuge: '44px',
      fontFamily: t.fontFamily,
      fontFamilyMono: 'ui-monospace, SFMono-Regular, "Cascadia Mono", Consolas, monospace',
      fontWeight: '400',
      fontWeightStrong: '600',
      opacityDisabled: '0.5',
      cubicBezierEaseInOut: 'cubic-bezier(0.4, 0, 0.2, 1)',
      cubicBezierEaseOut: 'cubic-bezier(0.16, 1, 0.3, 1)',
      cubicBezierEaseIn: 'cubic-bezier(0.4, 0, 1, 1)',
    },

    Button: buttonLike,

    Card: {
      color: t.bgCard,
      colorModal: t.bgCard,
      colorPopover: t.bgCard,
      textColor: t.textPrimary,
      titleTextColor: t.textPrimary,
      borderColor: t.border,
      actionColor: t.bgSecondary,
      borderRadius: '0',
      paddingMedium: '16px',
      paddingSmall: '12px',
      titleFontSizeMedium: '15px',
      titleFontWeight: '600',
      boxShadow: 'none',
    },

    // 侧边栏导航（Sidebar.vue 的 n-menu，2026-11-25 由自绘 item 改成 naive）
    // 只在这里给「颜色 / 高度 / 圆角 / 字号」：选中态 = accentSoft 底 + 品牌文字色，
    // 悬停 = bg-hover，折叠态图标用 tertiary（比展开态暗一档，压住存在感）。
    // ⚠️ 菜单本体不铺底（color: transparent）：侧栏底色来自 aside 的 --sidebar-bg。
    Menu: {
      color: 'transparent',
      itemHeight: '40px',
      borderRadius: '8px',
      fontSize: '14px',
      itemTextColor: t.textSecondary,
      itemTextColorHover: t.textPrimary,
      itemTextColorActive: t.brandText,
      itemTextColorActiveHover: t.brandText,
      itemTextColorChildActive: t.brandText,
      itemTextColorChildActiveHover: t.brandText,
      itemIconColor: t.textTertiary,
      itemIconColorHover: t.textPrimary,
      itemIconColorActive: t.brandText,
      itemIconColorActiveHover: t.brandText,
      itemIconColorChildActive: t.brandText,
      itemIconColorChildActiveHover: t.brandText,
      itemIconColorCollapsed: t.textTertiary,
      itemColorHover: t.bgHover,
      itemColorActive: accentSoft,
      itemColorActiveHover: withAlpha(t.primary, 0.2),
      // 折叠选中态：naive 会把它当成独立变量（!important 覆盖），颜色必须与展开态同源
      itemColorActiveCollapsed: accentSoft,
      arrowColor: t.textTertiary,
      arrowColorHover: t.textPrimary,
      arrowColorActive: t.brandText,
      arrowColorActiveHover: t.brandText,
      groupTextColor: t.textMuted,
      dividerColor: t.borderLight,
      borderColorHorizontal: t.borderLight,
    },

    Input: inputLike,

    InputNumber: {
      Input: inputLike,
      Button: buttonLike,
      iconColorDisabled: t.textMuted,
    },

    Select: {
      menuBoxShadow: t.shadowHeavy,
      InternalSelection: selectionLike,
      InternalSelectMenu: menuLike,
    },

    Tag: {
      borderRadius: '6px',
      heightSmall: '22px',
      heightMedium: '26px',
      fontSizeSmall: '12px',
      fontSizeMedium: '12px',
      padding: '0 8px',
      // 默认 / 可点选两种形态：透明底 + 1px 描边（延续原 chip 的外观）
      color: 'transparent',
      textColor: t.textSecondary,
      border: `1px solid ${t.borderLight}`,
      colorCheckable: 'transparent',
      colorHoverCheckable: t.bgHover,
      colorPressedCheckable: t.bgTertiary,
      textColorCheckable: t.textSecondary,
      textColorHoverCheckable: t.textPrimary,
      textColorPressedCheckable: t.textPrimary,
      colorChecked: accentSoft,
      colorCheckedHover: withAlpha(t.primary, 0.18),
      colorCheckedPressed: withAlpha(t.primary, 0.24),
      textColorChecked: t.brandText,
      closeIconColor: t.textTertiary,
      closeIconColorHover: t.danger,
      closeIconColorPressed: t.danger,
      closeColorHover: t.bgHover,
      closeColorPressed: t.bgTertiary,
      opacityDisabled: '0.4',
    },

    Modal: {
      color: t.bgCard,
      textColor: t.textPrimary,
      boxShadow: t.shadowHeavy,
    },

    Drawer: {
      color: t.bgCard,
      textColor: t.textPrimary,
      titleTextColor: t.textPrimary,
      boxShadow: t.shadowHeavy,
      headerBorderBottom: `1px solid ${t.borderLight}`,
      footerBorderTop: `1px solid ${t.borderLight}`,
      closeIconColor: t.textTertiary,
      closeIconColorHover: t.textPrimary,
      closeColorHover: t.bgHover,
    },

    Popover: {
      color: t.bgCard,
      textColor: t.textPrimary,
      borderRadius: '10px',
      boxShadow: t.shadowHeavy,
      dividerColor: t.borderLight,
      padding: '12px',
      fontSize: '13px',
    },

    Tooltip: {
      color: t.textPrimary,
      textColor: t.textInverse,
      borderRadius: '6px',
      boxShadow: t.shadowHeavy,
      padding: '5px 9px',
    },

    Dropdown: {
      color: t.bgCard,
      borderRadius: '10px',
      optionHeightMedium: '32px',
      fontSizeMedium: '13px',
      optionTextColor: t.textSecondary,
      optionTextColorHover: t.textPrimary,
      optionTextColorActive: t.brandText,
      optionTextColorChildActive: t.brandText,
      optionColorHover: t.bgHover,
      optionColorActive: accentSoft,
      groupHeaderTextColor: t.textMuted,
      dividerColor: t.borderLight,
      suffixColor: t.textMuted,
      prefixColor: t.textTertiary,
      padding: '4px',
      optionOpacityDisabled: '0.4',
    },

    Message: {
      color: t.bgCard,
      colorInfo: t.bgCard,
      colorSuccess: t.bgCard,
      colorError: t.bgCard,
      colorWarning: t.bgCard,
      colorLoading: t.bgCard,
      textColor: t.textPrimary,
      textColorInfo: t.textPrimary,
      textColorSuccess: t.textPrimary,
      textColorError: t.textPrimary,
      textColorWarning: t.textPrimary,
      textColorLoading: t.textPrimary,
      iconColor: t.textSecondary,
      iconColorInfo: t.info,
      iconColorSuccess: t.success,
      iconColorError: t.danger,
      iconColorWarning: t.warning,
      iconColorLoading: t.brandText,
      boxShadow: t.shadowHeavy,
      boxShadowInfo: t.shadowHeavy,
      boxShadowSuccess: t.shadowHeavy,
      boxShadowError: t.shadowHeavy,
      boxShadowWarning: t.shadowHeavy,
      boxShadowLoading: t.shadowHeavy,
      border: `1px solid ${t.border}`,
      borderRadius: '10px',
      padding: '10px 14px',
      fontSize: '13px',
      maxWidth: '380px',
      minWidth: '220px',
      iconSize: '18px',
      loadingColor: t.brandText,
      closeIconColor: t.textTertiary,
      closeIconColorHover: t.textPrimary,
      closeColorHover: t.bgHover,
      closeColorPressed: t.bgTertiary,
    },

    Alert: {
      borderRadius: '8px',
      border: `1px solid ${t.border}`,
      color: t.bgSecondary,
      titleTextColor: t.textPrimary,
      contentTextColor: t.textSecondary,
      iconColor: t.textSecondary,
      fontSize: '13px',
      padding: '8px 12px',
      iconSize: '18px',
      borderInfo: `1px solid ${t.info}`,
      colorInfo: t.bgSecondary,
      titleTextColorInfo: t.textPrimary,
      contentTextColorInfo: t.textSecondary,
      iconColorInfo: t.info,
      borderSuccess: `1px solid ${t.success}`,
      colorSuccess: t.bgSecondary,
      titleTextColorSuccess: t.textPrimary,
      contentTextColorSuccess: t.textSecondary,
      iconColorSuccess: t.success,
      borderWarning: `1px solid ${t.warning}`,
      colorWarning: t.bgSecondary,
      titleTextColorWarning: t.textPrimary,
      contentTextColorWarning: t.textSecondary,
      iconColorWarning: t.warning,
      borderError: `1px solid ${t.danger}`,
      colorError: t.bgSecondary,
      titleTextColorError: t.textPrimary,
      contentTextColorError: t.textSecondary,
      iconColorError: t.danger,
      closeIconColor: t.textTertiary,
      closeIconColorHover: t.textPrimary,
      closeColorHover: t.bgHover,
      closeColorPressed: t.bgTertiary,
    },

    Empty: {
      textColor: t.textTertiary,
      iconColor: t.textTertiary,
      extraTextColor: t.textMuted,
      iconSizeMedium: '36px',
      fontSizeMedium: '13px',
    },

    Spin: {
      color: t.brandText,
      textColor: t.textSecondary,
    },

    Pagination: {
      itemTextColor: t.textSecondary,
      itemTextColorHover: t.textPrimary,
      itemTextColorPressed: t.textPrimary,
      itemTextColorActive: t.brandText,
      itemTextColorDisabled: t.textMuted,
      itemColor: 'transparent',
      itemColorHover: t.bgHover,
      itemColorPressed: t.bgHover,
      itemColorActive: accentSoft,
      itemColorActiveHover: withAlpha(t.primary, 0.18),
      itemColorDisabled: 'transparent',
      itemBorder: '1px solid transparent',
      itemBorderHover: `1px solid ${t.borderLight}`,
      itemBorderPressed: `1px solid ${t.borderLight}`,
      itemBorderActive: `1px solid ${t.brandText}`,
      itemBorderDisabled: '1px solid transparent',
      itemBorderRadius: '6px',
      itemSizeMedium: '28px',
      itemFontSizeMedium: '12px',
      itemPaddingMedium: '0 6px',
      itemMarginMedium: '2px',
      jumperTextColor: t.textSecondary,
      jumperTextColorDisabled: t.textMuted,
      buttonIconColor: t.textSecondary,
      buttonIconColorHover: t.textPrimary,
      buttonIconColorPressed: t.textPrimary,
      buttonColor: 'transparent',
      buttonColorHover: t.bgHover,
      buttonColorPressed: t.bgTertiary,
      buttonBorder: '1px solid transparent',
      Select: selectionLike,
      Input: inputLike,
    },

    Checkbox: {
      borderRadius: '4px',
      sizeMedium: '16px',
      sizeSmall: '14px',
      color: 'transparent',
      colorChecked: t.primary,
      colorDisabled: t.bgTertiary,
      colorDisabledChecked: withAlpha(t.primary, 0.4),
      checkMarkColor: t.checkMark,
      checkMarkColorDisabled: t.textMuted,
      checkMarkColorDisabledChecked: t.textInverse,
      border: `1px solid ${t.borderDark}`,
      borderChecked: `1px solid ${t.primary}`,
      borderFocus: `1px solid ${t.brandText}`,
      borderDisabled: `1px solid ${t.border}`,
      borderDisabledChecked: `1px solid ${withAlpha(t.primary, 0.4)}`,
      boxShadowFocus: 'none',
      textColor: t.textPrimary,
      textColorDisabled: t.textMuted,
      labelPadding: '0 0 0 8px',
      labelFontWeight: '400',
    },

    Radio: {
      color: 'transparent',
      colorActive: t.primary,
      colorDisabled: t.bgTertiary,
      textColor: t.textSecondary,
      textColorDisabled: t.textMuted,
      dotColorActive: t.checkMark,
      dotColorDisabled: t.textMuted,
      buttonHeightMedium: '28px',
      buttonHeightSmall: '24px',
      buttonBorderRadius: '6px',
      buttonColor: 'transparent',
      buttonColorActive: t.primary,
      buttonTextColor: t.textSecondary,
      buttonTextColorActive: t.textOnBrand,
      buttonTextColorHover: t.textPrimary,
      buttonBorderColor: '1px solid transparent',
      buttonBorderColorActive: `1px solid ${t.primary}`,
      buttonBorderColorHover: `1px solid ${t.borderLight}`,
      buttonBoxShadow: 'none',
      buttonBoxShadowHover: 'none',
      buttonBoxShadowActive: 'none',
      buttonBoxShadowFocus: 'none',
      opacityDisabled: '0.4',
    },

    Switch: {
      railColor: t.border,
      railColorActive: t.primary,
      buttonColor: t.bgCard,
      iconColor: t.textInverse,
      loadingColor: t.textOnBrand,
      boxShadowFocus: 'none',
      opacityDisabled: '0.4',
    },

    Descriptions: {
      borderRadius: '8px',
      thColor: t.bgSecondary,
      tdColor: t.bgCard,
      borderColor: t.border,
      thTextColor: t.textSecondary,
      tdTextColor: t.textPrimary,
      titleTextColor: t.textPrimary,
      fontSizeMedium: '13px',
      thFontWeight: '500',
      lineHeight: '1.5',
      thPaddingMedium: '7px 12px',
      tdPaddingMedium: '7px 12px',
    },

    // tab（/works 的「作品信息 / 游戏统计 / 启动配置 / 编辑 / 评价」）：
    // 2026-11-24 用户给了 Steam 游戏页截图，要「亮字 + 下划线」的选中样式 ——
    // 未选中走 tertiary（压暗），选中走 textPrimary（提亮），下划线用品牌色。
    Tabs: {
      tabTextColorLine: t.textTertiary,
      tabTextColorActiveLine: t.textPrimary,
      tabTextColorHoverLine: t.textSecondary,
      tabTextColorDisabledLine: t.textMuted,
      barColor: t.brandText,
      tabFontWeight: '400',
      tabFontWeightActive: '500',
      tabFontSizeMedium: '13px',
      tabGapMediumLine: '24px',
      tabPaddingMediumLine: '0 0 10px',
      tabBorderColor: t.borderLight,
      paneTextColor: t.textPrimary,
      tabColor: t.bgPrimary,
      colorSegment: t.bgSecondary,
      tabColorSegment: t.bgCard,
      tabTextColorSegment: t.textSecondary,
      tabTextColorActiveSegment: t.textPrimary,
      tabTextColorHoverSegment: t.textPrimary,
      tabBorderRadius: '6px',
    },

    Badge: {
      color: t.primary,
      colorInfo: t.info,
      colorSuccess: t.success,
      colorError: t.danger,
      colorWarning: t.warning,
      fontSize: '11px',
      fontFamily: t.fontFamily,
    },

    Divider: {
      color: t.borderLight,
      textColor: t.textTertiary,
      fontWeight: '500',
    },

    Statistic: {
      labelTextColor: t.textTertiary,
      valueTextColor: t.textPrimary,
      valuePrefixTextColor: t.textPrimary,
      valueSuffixTextColor: t.textSecondary,
      labelFontSize: '11px',
      valueFontSize: '20px',
      labelFontWeight: '400',
      valueFontWeight: '600',
    },

    Progress: {
      railColor: t.borderLight,
      fillColor: t.primary,
      fillColorInfo: t.info,
      fillColorSuccess: t.success,
      fillColorWarning: t.warning,
      fillColorError: t.danger,
      iconColor: t.textInverse,
      iconColorInfo: t.textInverse,
      iconColorSuccess: t.textOnSuccess,
      iconColorWarning: t.textInverse,
      iconColorError: t.textOnDanger,
      textColorLineInner: t.textInverse,
      textColorLineOuter: t.textSecondary,
      textColorCircle: t.textPrimary,
    },

    Skeleton: {
      color: t.bgSecondary,
      colorEnd: t.bgHover,
    },

    Popconfirm: {
      fontSize: '13px',
      iconColor: t.warning,
      iconSize: '18px',
      Popover: {
        color: t.bgCard,
        textColor: t.textPrimary,
        borderRadius: '10px',
        boxShadow: t.shadowHeavy,
        padding: '12px',
      },
      Button: buttonLike,
    },

    // 分栏（测试页的 n-split）：分隔条只有 4px，静息用边框色、hover 提亮到品牌文字色
    Split: {
      resizableTriggerColor: t.borderDark,
      resizableTriggerColorHover: t.brandText,
    },
  }
}
