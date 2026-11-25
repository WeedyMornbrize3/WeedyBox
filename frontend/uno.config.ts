// uno.config.ts
// 设计系统：Flat Design（2D / 无装饰阴影 / 排版驱动 / 图标语义化）
// 配色：Teal 主色（浅色 #0F766E / 深色 #2DD4BF）+ 状态色，全部按 WCAG AA 4.5:1 核验。
// 所有颜色只以 CSS 变量形式引用（见 src/styles/global.css），此处不出现硬编码色值。
//
// 2026-11-24 引入 naive-ui 后分工变了：
//   · 控件（按钮 / 输入框 / 选择器 / 标签 / 弹窗 / 分页 / 下拉…）的长相与状态
//     全部由 naive 的主题覆盖提供 —— 见 src/styles/naiveTheme.ts（token → themeOverrides）。
//   · 本文件只剩下**布局类**与**色值 token → 类名**的映射，以及项目自己的导航/筛选/卡片布局。
//   所以不要再在这里新增 btn-* / input-* 这类控件 shortcut：控件只有一个来源（naive 主题）。
import { defineConfig, presetWind4, presetAttributify, presetIcons } from 'unocss'

export default defineConfig({
  presets: [
    presetWind4(),
    presetAttributify({
      prefix: 'un-',
      prefixedOnly: false,
    }),
    // 图标：i-<集合>-<名字>，如 i-lucide-trash-2
    // 数据源 @iconify/json（本地 236 集合），构建期内联成 data URI，运行时零请求、离线可用。
    // 统一用 lucide 单一样式（线性 / 2px stroke），不再混用 emoji 或手绘 svg。
    presetIcons({
      scale: 1,
      extraProperties: {
        'display': 'inline-block',
        'vertical-align': 'middle',
        'flex-shrink': '0',
      },
    }),
  ],

  // ===== 自定义规则（映射 CSS 变量）=====
  // 注意：这里是精确字符串匹配，不支持 `bg-secondary/50` 这类透明度后缀
  // （会被判 unmatched 静默丢弃）。需要半透明请用已定义的具体类名。
  rules: [
    // 背景
    ['bg-base', { 'background-color': 'var(--color-bg-base)' }],
    ['bg-primary', { 'background-color': 'var(--color-bg-primary)' }],
    ['bg-secondary', { 'background-color': 'var(--color-bg-secondary)' }],
    ['bg-secondary-soft', { 'background-color': 'var(--color-bg-secondary-soft)' }],
    ['bg-tertiary', { 'background-color': 'var(--color-bg-tertiary)' }],
    ['bg-hover', { 'background-color': 'var(--color-bg-hover)' }],
    ['bg-card', { 'background-color': 'var(--color-bg-card)' }],
    ['bg-sidebar', { 'background-color': 'var(--sidebar-bg)' }],

    // 文本
    ['text-primary', { 'color': 'var(--color-text-primary)' }],
    ['text-secondary', { 'color': 'var(--color-text-secondary)' }],
    ['text-tertiary', { 'color': 'var(--color-text-tertiary)' }],
    ['text-muted', { 'color': 'var(--color-text-muted)' }],
    ['text-inverse', { 'color': 'var(--color-text-inverse)' }],

    // 品牌色：填充用 --color-primary（四色原值），文字/图标用 --color-brand-text
    // （theme.ts 会按当前底色把它调到 4.5:1，保证深色主题下的导航/链接也看得清）
    ['text-brand', { 'color': 'var(--color-brand-text)' }],
    ['bg-brand', { 'background-color': 'var(--color-primary)' }],
    ['bg-brand-hover', { 'background-color': 'var(--color-primary-hover)' }],
    ['bg-brand-light', { 'background-color': 'var(--color-primary-light)' }],

    // ===== 边框 =====
    // ⚠️ 两条血泪教训，改这里的规则前务必先读：
    //
    // 1) 绝不能把简写值塞进 `border-color`。
    //    曾写成 { 'border-color': '1px solid …' }，产出
    //    `.border-theme-light{border-color:1px solid …}` —— 非法声明，整条作废。
    //
    // 2) 绝不能只给 `border-color`（即使值是合法颜色）。
    //    presetUno 的 preflight 会把所有元素重置为
    //      *,::before,::after { border-width: 0; border-style: solid }
    //    于是「只设颜色」的规则拿不到任何宽度 → 计算值 border-width: 0px、样式 none，
    //    边框完全不显示；更糟的是，当同一元素另有非零 `border-width` 时，
    //    border-style 会退回浏览器初始值 `outset`，渲染出 1.6px 的 3D 伪立体边框。
    //
    // 结论：**每条边框规则都必须自带 width + style + color 三件套**，
    // 即一律用 `border` 简写，绝不单独用 `border-color`。
    ['border-theme', { 'border': '1px solid var(--color-border)' }],
    ['border-theme-light', { 'border': '1px solid var(--color-border-light)' }],
    ['border-theme-dark', { 'border': '1px solid var(--color-border-dark)' }],
    ['border-brand', { 'border': '1px solid var(--color-primary)' }],

    // 优先级语义色（0 低 / 1 中 / 2 高）
    ['text-priority-low', { 'color': 'var(--color-text-tertiary)' }],
    ['text-priority-medium', { 'color': 'var(--color-warning)' }],
    ['text-priority-high', { 'color': 'var(--color-danger)' }],
    ['bg-priority-low', { 'background-color': 'var(--color-text-tertiary)' }],
    ['bg-priority-medium', { 'background-color': 'var(--color-warning)' }],
    ['bg-priority-high', { 'background-color': 'var(--color-danger)' }],

    // ===== 单侧边框（显式设定四边，避免被整框简写重置）=====
    // 为什么必须显式写全四边：一条 `border: 1px solid X` 的整框规则会把四边宽度都重置，
    // 于是「border-b + 整框颜色类」会变成四边都有框。
    // 这几条按 CSS 出现顺序排在本文件更后面 → 覆盖任何整框规则，
    // 真正实现「只有某一边有线」；其余边明确归零，不依赖声明顺序的巧合。
    ['border-b-theme-light', { 'border-bottom': '1px solid var(--color-border-light)', 'border-top': '0', 'border-right': '0', 'border-left': '0' }],
    ['border-r-sidebar', { 'border-right': '2px solid var(--sidebar-border)', 'border-top': '0', 'border-bottom': '0', 'border-left': '0' }],
    // 描述块 hover 时只把左竖线换成品牌色。
    // ⚠️ 这里必须是单侧规则，不能用整框的 border-brand。
    // ⚠️ 也不要再造一条 border-l-theme-light 与之共存：两者都改 border-left，
    //    UnoCSS 对同一属性只保留一条规则。基础竖线因此定义在 TodoItem.vue 的 scoped 样式里。
    ['border-l-brand', { 'border-left': '2px solid var(--color-primary)', 'border-top': '0', 'border-right': '0', 'border-bottom': '0' }],

    // 功能色
    ['text-success', { 'color': 'var(--color-success)' }],
    ['text-warning', { 'color': 'var(--color-warning)' }],
    ['text-danger', { 'color': 'var(--color-danger)' }],
    ['text-info', { 'color': 'var(--color-info)' }],
    ['bg-success', { 'background-color': 'var(--color-success)' }],
    ['bg-warning', { 'background-color': 'var(--color-warning)' }],
    ['bg-danger', { 'background-color': 'var(--color-danger)' }],
    ['bg-info', { 'background-color': 'var(--color-info)' }],

    // 阴影：Flat Design 默认不用装饰性阴影，仅保留浮层/焦点两种用途
    ['shadow-theme', { 'box-shadow': 'var(--color-shadow)' }],
    ['shadow-theme-heavy', { 'box-shadow': 'var(--color-shadow-heavy)' }],

    // 动效
    // presetUno 的 will-change 规则是 /^will-change-(.+)/，捕获组拿到带方括号的字面量，
    // `will-change-[width]` 会 unmatched 静默丢弃，只能用无方括号写法。
    ['will-change-width', { 'will-change': 'width' }],
  ],

  // ===== 快捷方式（只剩布局/组合；控件外观在 naive 主题里）=====
  shortcuts: {
    // ---------- 通用布局 ----------
    'flex-center': 'flex items-center justify-center',
    'flex-col-center': 'flex flex-col items-center justify-center',

    // ---------- 图标尺寸（统一 token，禁止在模板里混用任意字号）----------
    // 图标颜色跟随 currentColor，字号/颜色由使用处决定。
    'icon-xs': 'w-3.5 h-3.5 text-[14px] leading-none',
    'icon-sm': 'w-4 h-4 text-[16px] leading-none',
    'icon-md': 'w-5 h-5 text-[20px] leading-none',
    'icon-lg': 'w-6 h-6 text-[24px] leading-none',
    'icon-xl': 'w-8 h-8 text-[32px] leading-none',

    // ---------- 卡片（留给没有换成 n-card 的容器，如 TODO 列表滚动区）----------
    // Flat：用 1px 边框 + 直角区分层级，不用投影（2026-11-25 用户要求全站卡片方形）。
    'card': 'bg-card rounded-none border-theme p-5',

    // ---------- 顶部栏 ----------
    'titlebar': 'fixed top-0 left-0 right-0 h-[30px] flex items-center justify-between px-3 bg-base text-secondary z-[1100] select-none border-b-theme-light',

    // ---------- 侧边栏 ----------
    // 2026-11-25 起导航项本身是 naive 的 n-menu（用户要求「改用 naive ui 样式和逻辑」）：
    // 原来的 nav-item / nav-item-active / nav-icon / nav-text / nav-badge 已删除，
    // 菜单项的高度/圆角/选中色/折叠 tooltip 全部由 styles/naiveTheme.ts 的 Menu 段驱动。
    // 这里只剩侧栏外壳（宽度过渡、底色、右侧分隔线）与上下两个菜单容器。
    'sidebar': 'fixed left-0 top-0 h-screen bg-sidebar text-primary transition-[width] duration-300 ease-[cubic-bezier(0.4,0,0.2,1)] flex flex-col z-[1000] overflow-hidden select-none border-r-sidebar will-change-width',
    'sidebar-header': 'flex items-center px-5 gap-3 cursor-pointer min-h-[72px] flex-shrink-0 transition-colors duration-150 hover:bg-hover',
    'logo-icon': 'flex-shrink-0 text-brand flex-center',
    'logo-text': 'text-lg font-semibold text-primary whitespace-nowrap transition-opacity duration-250 ease-in overflow-hidden tracking-[0.5px]',
    'sidebar-nav-top': 'flex-1 flex flex-col overflow-y-auto overflow-x-hidden p-2 mt-2.5',
    'sidebar-nav-bottom': 'flex-shrink-0 p-2 pt-0',

    // ---------- 筛选栏 ----------
    // 横向排列 + 横向滚动（不换行）。各分组必须 flex-shrink-0，否则会被压扁而不是产生滚动条。
    // scrollbar-theme 是 global.css 里的普通类（非 UnoCSS utility），必须在模板 class 里单独写。
    'filter-bar': 'flex flex-nowrap items-center gap-x-4 px-3 py-2 rounded-none bg-secondary-soft border-theme-light overflow-x-auto overflow-y-hidden',
    'filter-group': 'flex items-center gap-2 flex-shrink-0',
    'filter-label': 'text-xs text-tertiary whitespace-nowrap',

    // ---------- TODO 列表 ----------
    // 刻意只保留「单一职责」的 shortcut：过长的 shortcut 链会被 UnoCSS 静默截断。
    // 注意：这里不含边框颜色类。基础色（border-l-theme-light）与悬停色
    // （group-hover:border-l-brand）都写在模板上。
    'todo-desc-block': 'mt-1 text-xs leading-relaxed text-tertiary whitespace-pre-wrap break-words pl-3 transition-colors duration-200',

    // ---------- 设置页 ----------
    'settings-page': 'max-w-[700px] mx-auto w-full',
    'settings-title': 'text-2xl font-semibold text-primary mb-5 tracking-[0.5px]',
    'settings-container': 'flex flex-col',
    'settings-item': 'flex items-center justify-between py-2.5 border-b-theme-light last:border-none last:pb-0',
    'item-info': 'flex-1 flex flex-col gap-0.5 min-w-0',
    'item-label': 'text-sm font-medium text-primary',
    'item-description': 'text-xs text-tertiary',
    'item-control': 'flex items-center gap-3 flex-shrink-0 ml-4',
  },

  // ===== 主题映射 =====
  // 与上面 rules 并存：rules 提供精确类名，theme.colors 让 preset 的语义族
  // （bg-*/text-*/border-*）也能解析到同一批变量，避免两套色值分叉。
  theme: {
    colors: {
      bg: {
        base: 'var(--color-bg-base)',
        primary: 'var(--color-bg-primary)',
        secondary: 'var(--color-bg-secondary)',
        tertiary: 'var(--color-bg-tertiary)',
        hover: 'var(--color-bg-hover)',
        card: 'var(--color-bg-card)',
        sidebar: 'var(--sidebar-bg)',
      },
      text: {
        primary: 'var(--color-text-primary)',
        secondary: 'var(--color-text-secondary)',
        tertiary: 'var(--color-text-tertiary)',
        muted: 'var(--color-text-muted)',
        inverse: 'var(--color-text-inverse)',
      },
      brand: {
        DEFAULT: 'var(--color-primary)',
        hover: 'var(--color-primary-hover)',
        active: 'var(--color-primary-active)',
        light: 'var(--color-primary-light)',
      },
      border: {
        DEFAULT: 'var(--color-border)',
        light: 'var(--color-border-light)',
        dark: 'var(--color-border-dark)',
      },
      success: 'var(--color-success)',
      warning: 'var(--color-warning)',
      danger: 'var(--color-danger)',
      info: 'var(--color-info)',
    },
  },
})
