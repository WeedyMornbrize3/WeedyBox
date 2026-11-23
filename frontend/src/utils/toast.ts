// frontend/src/utils/toast.ts
// 全局「一次性操作反馈」的唯一入口（底层是 vue-sonner，2026-11-23 引入）。
//
// 为什么再包一层而不是各组件直接 import { toast }：
//   - 组件/store 只描述"发生了什么"，停留多久、描述文字怎么截断、用哪个类型在这里统一；
//   - 换组件库或改外观时，只需改这里 + components/AppToaster.vue，调用点不动；
//   - 提示文案与图标一一对应（success/error/info），不会各写各的。
//
// 边界：这里只放"一闪而过、不需要回头对照"的结果。
//   长期停留的状态（列表加载失败、入库详情、exe 已被移动等）仍然走页面内的
//   MetaMessage / store.error —— 那些信息用户要能盯着看，不能 2 秒就消失。
import type { CSSProperties } from 'vue'
import { toast } from 'vue-sonner'

/** 成功提示停留时长：短，避免一直挡在界面上 */
const SUCCESS_DURATION = 2400
/** 失败提示要多留一会儿：用户往往需要看清原因 */
const ERROR_DURATION = 3800

/** 提示 id（loading 与结果各是一条，loading 需要手动 dismiss） */
export type ToastId = string | number

export interface NotifyOptions {
  /** 第二行小字（补充说明 / 错误原因），过长会自动截断 */
  description?: string
  /** 覆盖默认停留时长（ms） */
  duration?: number
}

/** 描述文字上限：toast 只有一行的位置，超出部分截断（完整原因仍在页面内的提示条里） */
const DESCRIPTION_MAX = 120

function clampDescription(text?: string): string | undefined {
  const value = text?.trim()
  if (!value) return undefined
  return value.length > DESCRIPTION_MAX ? `${value.slice(0, DESCRIPTION_MAX - 1)}…` : value
}

/**
 * 顶部定时条（components/AppToaster.vue 里的 ::before）靠这个 CSS 变量决定走完动画的时间。
 * 必须与 toast 自身的 duration 完全一致，否则会出现「条走完了提示还挂着」或反之。
 * 时长不是有限正数时不写变量 —— 样式里 `var(--toast-duration, 0ms)` 会让条直接收成 0（不可见）。
 */
function countdownStyle(durationMs: number): CSSProperties {
  if (!Number.isFinite(durationMs) || durationMs <= 0) return {}
  return { '--toast-duration': `${durationMs}ms` }
}

/** 成功（绿勾）：入库、删除、复制这类"做成了"的结果 */
export function toastSuccess(message: string, options: NotifyOptions = {}) {
  const duration = options.duration ?? SUCCESS_DURATION
  return toast.success(message, {
    description: clampDescription(options.description),
    duration,
    style: countdownStyle(duration),
  })
}

/** 失败（红色感叹号）：请求报错、部分失败 */
export function toastError(message: string, options: NotifyOptions = {}) {
  const duration = options.duration ?? ERROR_DURATION
  return toast.error(message, {
    description: clampDescription(options.description),
    duration,
    style: countdownStyle(duration),
  })
}

/** 中性提示（蓝色 i）：没有成败之分、只是告知 */
export function toastInfo(message: string, options: NotifyOptions = {}) {
  const duration = options.duration ?? SUCCESS_DURATION
  return toast.info(message, {
    description: clampDescription(options.description),
    duration,
    style: countdownStyle(duration),
  })
}

/**
 * 正在进行的动作（转圈、不自动消失）。
 *
 * ⚠️ 两个坑（2026-11-23 线上踩过）：
 *   1. sonner 在 `type='loading'` 时**根本不启动自动关闭计时器**，所以它天然要留在屏幕上；
 *   2. 更不能用同一个 id 把它更新成结果：Toast 组件挂载时把剩余时间记成 loading 的初值，
 *      更新 type 后计时器不会重启 —— 表现就是「移除游戏的提示永远不消失」。
 * 正确用法：结束时先 `dismissToast(id)` 关掉它，再另弹一条新的 toastSuccess / toastError。
 * 新 toast 会正常自动消失，顶端定时条也从头开始走（进度与停留时长因此始终一致）。
 */
export function toastLoading(message: string, options: NotifyOptions = {}) {
  return toast.loading(message, { description: clampDescription(options.description) })
}

/** 手动关掉某条（不传 id = 关掉全部） */
export function dismissToast(id?: ToastId) {
  toast.dismiss(id)
}
