// uno.config.ts
// 设计系统：Flat Design（2D / 无装饰阴影 / 排版驱动 / 图标语义化）
// 配色：Teal 主色（浅色 #0F766E / 深色 #2DD4BF）+ 状态色，全部按 WCAG AA 4.5:1 核验。
// 所有颜色只以 CSS 变量形式引用（见 src/styles/global.css），此处不出现硬编码色值。
import { defineConfig, presetUno, presetAttributify, presetIcons } from 'unocss'

export default defineConfig({
  presets: [
    presetUno(),
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

    // 品牌色
    ['text-brand', { 'color': 'var(--color-primary)' }],
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
    //    border-style 会退回浏览器初始值 `outset`，渲染出 1.6px 的 3D 伪立体边框
    //    （实测 chip-active: `1.6px outset rgb(45,212,191)`）。
    //
    // 结论：**每条边框规则都必须自带 width + style + color 三件套**，
    // 即一律用 `border` 简写，绝不单独用 `border-color`。
    // 需要「底色/主题色 + 指定线宽」时，把本规则写在后面覆盖简写（CSS 按出现顺序取胜）。
    ['border-theme', { 'border': '1px solid var(--color-border)' }],
    ['border-theme-light', { 'border': '1px solid var(--color-border-light)' }],
    ['border-theme-dark', { 'border': '1px solid var(--color-border-dark)' }],
    ['border-brand', { 'border': '1px solid var(--color-primary)' }],
    ['border-brand-l', { 'border-left': '2px solid var(--color-primary)' }],

    // 优先级语义色（0 低 / 1 中 / 2 高）
    ['text-priority-low', { 'color': 'var(--color-text-tertiary)' }],
    ['text-priority-medium', { 'color': 'var(--color-warning)' }],
    ['text-priority-high', { 'color': 'var(--color-danger)' }],
    ['bg-priority-low', { 'background-color': 'var(--color-text-tertiary)' }],
    ['bg-priority-medium', { 'background-color': 'var(--color-warning)' }],
    ['bg-priority-high', { 'background-color': 'var(--color-danger)' }],
    ['border-priority-medium', { 'border': '1px solid var(--color-warning)' }],
    ['border-priority-high', { 'border': '1px solid var(--color-danger)' }],

    // ===== 单侧边框（显式设定四边，避免被整框简写重置）=====
    // 为什么必须显式写全四边：一条 `border: 1px solid X` 的整框规则会把四边宽度都重置，
    // 于是「border-b + 整框颜色类」会变成四边都有框。
    // 这几条按 CSS 出现顺序排在本文件更后面 → 覆盖任何整框规则，
    // 真正实现「只有某一边有线」；其余边明确归零，不依赖声明顺序的巧合。
    ['border-b-theme-light', { 'border-bottom': '1px solid var(--color-border-light)', 'border-top': '0', 'border-right': '0', 'border-left': '0' }],
    ['border-r-sidebar', { 'border-right': '2px solid var(--sidebar-border)', 'border-top': '0', 'border-bottom': '0', 'border-left': '0' }],
    // 描述块 hover 时只把左竖线换成品牌色。
    // ⚠️ 这里必须是单侧规则，不能用整框的 border-brand：
    //    整框简写会给四边都加上 1px，描述块就会从「一条左竖线」变成「一个方框」。
    // ⚠️ 也不要再造一条 border-l-theme-light 与之共存：两者都改 border-left，
    //    UnoCSS 对同一属性只保留一条规则，模板里同时写会丢掉其中一个。
    //    基础竖线因此定义在 TodoItem.vue 的 scoped 样式里，各管一件事。
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

  // ===== 快捷方式 =====
  shortcuts: {
    // ---------- 通用布局 ----------
    'flex-center': 'flex items-center justify-center',
    'flex-between': 'flex items-center justify-between',
    'flex-col-center': 'flex flex-col items-center justify-center',
    'absolute-center': 'absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2',

    // ---------- 图标尺寸（统一 token，禁止在模板里混用任意字号）----------
    // 图标颜色跟随 currentColor，字号/颜色由使用处决定。
    'icon-xs': 'w-3.5 h-3.5 text-[14px] leading-none',
    'icon-sm': 'w-4 h-4 text-[16px] leading-none',
    'icon-md': 'w-5 h-5 text-[20px] leading-none',
    'icon-lg': 'w-6 h-6 text-[24px] leading-none',
    'icon-xl': 'w-8 h-8 text-[32px] leading-none',

    // ---------- 标题与文本 ----------
    'title-lg': 'text-2xl font-semibold text-primary',
    'title-md': 'text-lg font-semibold text-primary',
    'title-sm': 'text-base font-medium text-primary',
    'subtitle': 'text-sm text-tertiary',

    // ---------- 卡片 ----------
    // Flat：用 1px 边框 + 圆角区分层级，不用投影。
    // 边框一律由 `border-theme` 这类「自带 width+style+color」的规则提供，
    // 不再依赖通用 `border` 类去补宽度（那正是边框消失的成因）。
    'card': 'bg-card rounded-xl border-theme p-5',
    'card-flat': 'bg-card rounded-xl border-theme',
    'card-hover': 'card hover:border-brand transition-colors duration-200',

    // ---------- 按钮 ----------
    'btn-brand': 'px-4 py-2 bg-brand text-inverse rounded-lg font-medium hover:bg-brand-hover transition-colors duration-200 cursor-pointer border-none',
    'btn-ghost': 'px-4 py-2 text-secondary hover:text-primary hover:bg-hover rounded-lg transition-colors duration-200 cursor-pointer border-none bg-transparent',
    // 行内图标按钮（完成 / 删除共用）：只定义尺寸与布局，
    // 颜色与状态一律由调用处的动态 class 决定——shortcut 里写颜色会压掉调用处
    // （UnoCSS 把 shortcuts 输出在 rules 之前），这个坑已经踩过。
    'btn-row': '!w-7 !h-7 !rounded-lg !p-0 flex-center border transition-colors duration-200 cursor-pointer flex-shrink-0 disabled:cursor-not-allowed disabled:opacity-40',
    'input-theme': 'bg-secondary rounded-lg px-3 py-2 text-primary placeholder-tertiary border-theme focus:border-brand transition-colors duration-200',

    // ---------- 顶部栏 ----------
    'titlebar': 'fixed top-0 left-0 right-0 h-[30px] flex items-center justify-between px-3 bg-base text-secondary z-[1100] select-none border-b-theme-light',
    'titlebar-btn': 'flex-center bg-transparent border-none text-tertiary w-7 h-7 rounded cursor-pointer transition-colors duration-200 leading-none p-0 hover:bg-hover hover:text-primary',
    'titlebar-btn-close': 'flex-center bg-transparent border-none text-tertiary w-7 h-7 rounded cursor-pointer transition-colors duration-200 leading-none p-0 hover:bg-danger hover:text-white',

    // ---------- 侧边栏 ----------
    'sidebar': 'fixed left-0 top-0 h-screen bg-sidebar text-primary transition-[width] duration-300 ease-[cubic-bezier(0.4,0,0.2,1)] flex flex-col z-[1000] overflow-hidden select-none border-r-sidebar will-change-width',
    'sidebar-header': 'flex items-center px-5 gap-3 cursor-pointer min-h-[72px] flex-shrink-0 transition-colors duration-150 hover:bg-hover',
    'logo-icon': 'flex-shrink-0 text-brand flex-center',
    'logo-text': 'text-lg font-semibold text-primary whitespace-nowrap transition-opacity duration-250 ease-in overflow-hidden tracking-[0.5px]',
    'sidebar-nav-top': 'flex-1 flex flex-col overflow-y-auto overflow-x-hidden p-2 mt-2.5',
    'sidebar-nav-bottom': 'flex-shrink-0 flex flex-col-reverse overflow-y-auto overflow-x-hidden p-2 pt-2',
    'nav-item': 'flex items-center px-4 py-2.5 rounded-lg cursor-pointer transition-colors duration-200 ease-in relative gap-3 text-secondary flex-shrink-0 mb-2.5 hover:bg-hover hover:text-primary',
    'nav-item-active': '!bg-brand-light !text-brand before:content-[\'\'] before:absolute before:left-0 before:top-1/2 before:-translate-y-1/2 before:w-0.5 before:h-5 before:bg-brand before:rounded-r',
    'nav-icon': 'flex-shrink-0',
    'nav-text': 'text-sm font-[450] whitespace-nowrap transition-opacity duration-250 ease-in text-secondary overflow-hidden',
    'nav-text-active': 'text-brand',
    'nav-badge': 'ml-auto bg-brand text-inverse text-[10px] px-[7px] py-[1px] rounded-[10px] font-medium min-w-[18px] text-center',

    // ---------- 筛选栏 ----------
    // 横向排列 + 横向滚动（不换行）。各分组必须 flex-shrink-0，否则会被压扁而不是产生滚动条。
    // scrollbar-theme 是 global.css 里的普通类（非 UnoCSS utility），必须在模板 class 里单独写。
    // ⚠️ 不要把 `scrollbar-theme` 写进 shortcut：它不是 UnoCSS utility（定义在 global.css），
    //    写进来会判 unmatched。同理 `group` 也不要在 shortcut 里引用，必须在模板 class 中直接写。
    'filter-bar': 'flex flex-nowrap items-center gap-x-4 px-3 py-2 rounded-lg bg-secondary-soft border-theme-light overflow-x-auto overflow-y-hidden',
    'filter-group': 'flex items-center gap-2 flex-shrink-0',
    'filter-label': 'text-xs text-tertiary whitespace-nowrap',
    'filter-chips': 'flex flex-nowrap gap-1',
    'chip': 'px-2.5 py-1 rounded-md cursor-pointer text-xs whitespace-nowrap bg-transparent text-secondary border-theme-light hover:bg-hover hover:text-primary transition-colors duration-200 flex items-center gap-1',
    // 选中态：品牌色文字 + 边框 + 卡片底色（不靠颜色单独表意，同时有边框与字重变化）
    // ⚠️ 必须用 !border-brand（自带 1px solid）而不是 !border-color-brand：
    //    只改颜色会让 border-style 退回 outset，渲染成 1.6px 的 3D 立体边框。
    'chip-active': '!bg-brand-light !text-brand !border-brand font-medium',

    // ---------- TODO 列表 ----------
    // ⚠️ 这里刻意只保留「单一职责」的 shortcut。
    // 实测教训：把十几条 utility（含 flex-center、appearance-none、focus-visible:ring-*）
    // 串成一条 shortcut 时，UnoCSS 会丢弃链条中后段的部分声明——产物 CSS 里
    // .todo-check 只剩 :checked/:hover 两条，width/appearance/焦点环全部静默消失。
    // 因此列表行的样式直接写在组件模板的 class 里（见 TodoItem.vue），便于逐一核对产物。
    // 注意：这里刻意不含边框颜色类。基础色（border-l-theme-light）与悬停色
    // （group-hover:border-l-brand）都写在模板上——两者的选择器权重相同（0,1,0），
    // 若基础色留在 shortcut 内，它会按出现顺序压掉模板上的 border-l-brand，
    // 导致「单侧变品牌色」失效（曾实测到）。谁的颜色谁负责，避免权重巧合。
    'todo-desc-block': 'mt-1 text-xs leading-relaxed text-tertiary whitespace-pre-wrap break-words pl-3 transition-colors duration-200',

    // ---------- 设置页 ----------
    'settings-page': 'max-w-[700px] mx-auto w-full',
    'settings-title': 'text-2xl font-semibold text-primary mb-5 tracking-[0.5px]',
    'settings-container': 'flex flex-col gap-5',
    'settings-section': 'bg-card rounded-xl border-theme p-4',
    'section-header': 'flex items-center gap-2.5 mb-3 pb-2.5 border-b-theme-light',
    'section-icon': 'text-brand flex-center',
    'section-title': 'text-[15px] font-semibold text-primary m-0',
    'settings-item': 'flex items-center justify-between py-2.5 border-b-theme-light last:border-none last:pb-0',
    'item-info': 'flex-1 flex flex-col gap-0.5 min-w-0',
    'item-label': 'text-sm font-medium text-primary',
    'item-description': 'text-xs text-tertiary',
    'item-control': 'flex items-center gap-3 flex-shrink-0 ml-4',

    // ---------- 主题控件 ----------
    'theme-toggle': 'flex-center bg-transparent border-none text-secondary w-8 h-8 rounded-md cursor-pointer transition-colors duration-200 ease-in p-1 hover:bg-hover hover:text-primary',
    'theme-options': 'flex gap-1 bg-secondary p-[3px] rounded-[10px] border-theme',
    'theme-option': 'flex items-center gap-1 px-3 py-1.5 border-none rounded-lg cursor-pointer bg-transparent text-secondary text-xs transition-colors duration-200 whitespace-nowrap hover:text-primary hover:bg-hover',
    'theme-option-active': '!bg-card !text-primary font-medium',
    'version-text': 'text-xs text-tertiary bg-secondary px-3 py-1 rounded-md border-theme-light',

    // 主内容区
    'main-content': 'flex-1 p-6 overflow-y-auto bg-primary',
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
