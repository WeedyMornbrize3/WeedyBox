// frontend/src/styles/naive.ts
// naive-ui 的全局配置（2026-11-24 引入）。
//
// 这里只做两件事：
//   1. 保存一份**响应式**的 ConfigProvider props（主题 / 主题覆盖 / 中文文案），
//      给 App.vue 的 <n-config-provider> 用；
//   2. 用 createDiscreteApi 建一套「脱离组件树」的 message 实例 —— utils/toast.ts
//      在 store 里被调用，拿不到 setup 上下文，discrete API 是官方给这种场景的出口。
//
// ⚠️ 两处必须共用同一个 naiveConfig 对象：树内组件（按钮/弹窗/下拉）走 provider，
//    toast 走 discrete API，配置分叉会出现「弹窗换肤了、提示还是默认绿」。
import { createDiscreteApi, darkTheme, dateZhCN, zhCN, type GlobalTheme, type GlobalThemeOverrides } from 'naive-ui'
import { reactive } from 'vue'
import { buildThemeOverrides, readDesignTokens } from './naiveTheme'

/** 全局中文文案（naive 默认英文） */
export const naiveLocale = zhCN
export const naiveDateLocale = dateZhCN

export const naiveConfig = reactive<{
  theme: GlobalTheme | null
  themeOverrides: GlobalThemeOverrides
}>({
  theme: null,
  themeOverrides: {},
})

/**
 * 主题变化时重新读一遍 token 并更新 naive 配置。
 * 必须在 `data-theme` 已写到 <html> 之后调用（App.vue 用 flush: 'post' 保证）。
 */
export function syncNaiveTheme(isDark: boolean): void {
  if (typeof document === 'undefined') return
  naiveConfig.theme = isDark ? darkTheme : null
  naiveConfig.themeOverrides = buildThemeOverrides(readDesignTokens())
}

const discrete = createDiscreteApi(['message', 'notification'], {
  configProviderProps: naiveConfig,
  messageProviderProps: {
    placement: 'top',
    max: 3,
    duration: 3000,
    keepAliveOnHover: true,
    closable: false,
  },
})

/** 全局一次性反馈（入口是 utils/toast.ts，业务代码不要直接用） */
export const message = discrete.message
/** 保留给以后确实需要「通知卡片」的场合；目前没有调用点 */
export const notification = discrete.notification
