// uno.config.ts
import { defineConfig, presetUno, presetAttributify } from 'unocss'

export default defineConfig({
  presets: [
    presetUno(),
    presetAttributify({
      prefix: 'un-',
      prefixedOnly: false,
    }),
  ],

  // ===== 自定义规则（映射 CSS 变量） =====
  rules: [
    // 背景色
    ['bg-base', { 'background-color': 'var(--color-bg-base)' }],
    ['bg-primary', { 'background-color': 'var(--color-bg-primary)' }],
    ['bg-secondary', { 'background-color': 'var(--color-bg-secondary)' }],
    ['bg-tertiary', { 'background-color': 'var(--color-bg-tertiary)' }],
    ['bg-hover', { 'background-color': 'var(--color-bg-hover)' }],
    ['bg-card', { 'background-color': 'var(--color-bg-card)' }],
    ['bg-sidebar', { 'background-color': 'var(--sidebar-bg)' }],

    // 文本色
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
    ['border-brand', { 'border-color': 'var(--color-primary)' }],

    // 边框
    ['border-theme', { 'border': '1px solid var(--color-border)' }],
    ['border-theme-light', { 'border-color': 'var(--color-border-light)' }],
    ['border-theme-dark', { 'border-color': 'var(--color-border-dark)' }],

    // 功能色
    ['text-success', { 'color': 'var(--color-success)' }],
    ['text-warning', { 'color': 'var(--color-warning)' }],
    ['text-danger', { 'color': 'var(--color-danger)' }],
    ['text-info', { 'color': 'var(--color-info)' }],
    ['bg-success', { 'background-color': 'var(--color-success)' }],
    ['bg-warning', { 'background-color': 'var(--color-warning)' }],
    ['bg-danger', { 'background-color': 'var(--color-danger)' }],
    ['bg-info', { 'background-color': 'var(--color-info)' }],

    // 阴影
    ['shadow-theme', { 'box-shadow': 'var(--color-shadow)' }],
    ['shadow-theme-heavy', { 'box-shadow': 'var(--color-shadow-heavy)' }],
  ],

  // ===== 快捷方式 =====
  shortcuts: {
    // ---------- 通用布局 ----------
    'flex-center': 'flex items-center justify-center',
    'flex-between': 'flex items-center justify-between',
    'flex-col-center': 'flex flex-col items-center justify-center',
    'absolute-center': 'absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2',

    // ---------- 通用组件 ----------
    // 标题
    'title-lg': 'text-2xl font-semibold text-primary',
    'title-md': 'text-lg font-semibold text-primary',
    'title-sm': 'text-base font-medium text-primary',
    'subtitle': 'text-sm text-tertiary',

    // 卡片
    'card': 'bg-card rounded-xl border border-theme p-5',
    'card-hover': 'card hover:border-brand transition-colors duration-200',

    // 按钮
    'btn-brand': 'px-4 py-2 bg-brand text-inverse rounded-lg hover:bg-brand-hover transition-colors duration-200 cursor-pointer border-none',
    'btn-ghost': 'px-4 py-2 text-secondary hover:text-primary hover:bg-hover rounded-lg transition-colors duration-200 cursor-pointer border-none bg-transparent',
    'btn-delete': '!w-6 !h-6 !rounded-full !p-0 flex items-center justify-center text-xs border-2 border-brand bg-transparent text-brand hover:bg-brand hover:text-inverse transition-colors duration-200 cursor-pointer flex-shrink-0',
    // 输入框
    'input-theme': 'bg-secondary border border-theme rounded-lg px-3 py-2 text-primary placeholder-tertiary focus:outline-none focus:border-brand transition-colors duration-200',

    // ---------- 主内容 ----------
    'main-content': 'flex-1 p-6 overflow-y-auto bg-primary',

    // ---------- 顶部栏 TitleBar ----------
    'titlebar': 'fixed top-0 left-0 right-0 h-[30px] flex items-center justify-between px-3 bg-base text-secondary z-[1100] select-none',
    'titlebar-btn': 'bg-transparent border-none text-tertiary text-base w-7 h-7 rounded cursor-pointer transition-all duration-200 flex items-center justify-center leading-none p-0 font-light hover:bg-hover hover:text-primary',
    'titlebar-btn-close': 'bg-transparent border-none text-tertiary text-base w-7 h-7 rounded cursor-pointer transition-all duration-200 flex items-center justify-center leading-none p-0 font-light hover:bg-danger hover:text-white',

    // ---------- 侧边栏 Sidebar ----------
    'sidebar': 'fixed left-0 top-0 h-screen bg-sidebar text-primary transition-[width] duration-300 ease-[cubic-bezier(0.4,0,0.2,1)] flex flex-col z-[1000] shadow-[1px_0_16px_var(--color-shadow)] overflow-hidden select-none border-r-2 border-sidebar will-change-[width]',
    'sidebar-header': 'flex items-center px-5 gap-3 cursor-pointer min-h-[72px] flex-shrink-0 transition-colors duration-150 hover:bg-hover',
    'logo-icon': 'flex-shrink-0 text-brand flex items-center justify-center',
    'logo-text': 'text-lg font-semibold text-primary whitespace-nowrap transition-opacity duration-250 ease-in overflow-hidden tracking-[0.5px]',

    // 侧边栏导航
    'sidebar-nav-top': 'flex-1 flex flex-col overflow-y-auto overflow-x-hidden p-2 mt-2.5',
    'sidebar-nav-bottom': 'flex-shrink-0 flex flex-col-reverse overflow-y-auto overflow-x-hidden p-2 pt-2',

    // 侧边栏导航项
    'nav-item': 'flex items-center px-4 py-2.5 rounded-lg cursor-pointer transition-colors duration-200 ease-in relative gap-3 text-secondary flex-shrink-0 mb-2.5 hover:bg-hover hover:text-primary',
    'nav-item-active': '!bg-brand-light !text-brand before:content-[\'\'] before:absolute before:left-0 before:top-1/2 before:-translate-y-1/2 before:w-0.5 before:h-5 before:bg-brand before:rounded-r',
    'nav-icon': 'text-xl w-6 text-center flex-shrink-0 leading-none',
    'nav-text': 'text-sm font-[450] whitespace-nowrap transition-opacity duration-250 ease-in text-secondary overflow-hidden',
    'nav-text-active': 'text-brand',
    'nav-badge': 'ml-auto bg-brand text-inverse text-[10px] px-[7px] py-[1px] rounded-[10px] font-medium min-w-[18px] text-center',

    // ---------- 设置页面 Settings ----------
    'settings-page': 'max-w-[700px] mx-auto w-full',
    'settings-title': 'text-2xl font-semibold text-primary mb-5 tracking-[0.5px]',
    'settings-container': 'flex flex-col gap-5',

    // 设置区块
    'settings-section': 'bg-card rounded-xl border border-theme p-4 transition-colors duration-300',
    'section-header': 'flex items-center gap-2.5 mb-3 pb-2.5 border-b border-theme-light',
    'section-icon': 'text-[18px]',
    'section-title': 'text-[15px] font-semibold text-primary m-0',

    // 设置项
    'settings-item': 'flex items-center justify-between py-2.5 border-b border-theme-light last:border-none last:pb-0',
    'item-info': 'flex-1 flex flex-col gap-0.5 min-w-0',
    'item-label': 'text-sm font-medium text-primary',
    'item-description': 'text-xs text-tertiary',
    'item-control': 'flex items-center gap-3 flex-shrink-0 ml-4',

    // 主题切换按钮
    'theme-toggle': 'bg-transparent border-none text-secondary w-8 h-8 rounded-md cursor-pointer flex items-center justify-center transition-colors duration-200 ease-in p-1 hover:bg-hover hover:text-primary',
    'theme-icon': 'w-5 h-5',

    // 主题选项
    'theme-options': 'flex gap-1 bg-secondary p-[3px] rounded-[10px] border border-theme',
    'theme-option': 'flex items-center gap-1 px-3 py-1.25 border-none rounded-lg cursor-pointer bg-transparent text-secondary text-xs transition-all duration-200 whitespace-nowrap hover:text-primary hover:bg-hover',
    'theme-option-active': '!bg-card !text-primary shadow-[0_1px_4px_var(--color-shadow)]',
    'theme-option-icon': 'text-sm',
    'theme-option-label': 'font-medium',
    'theme-option-check': 'text-brand font-semibold text-xs ml-0.5',

    // 版本号
    'version-text': 'text-xs text-tertiary bg-secondary px-3 py-0.75 rounded-md border border-theme-light',
  },

  // ===== 主题配置 =====
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