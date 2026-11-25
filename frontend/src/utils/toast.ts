// frontend/src/utils/toast.ts
// 全局「一次性操作反馈」的唯一入口（2026-11-24 从 vue-sonner 迁到 naive-ui 的 message）。
//
// 为什么再包一层而不是各组件直接 import { message }：
//   - 组件/store 只描述"发生了什么"，停留多久、描述文字怎么截断、用哪个类型在这里统一；
//   - 换组件库或改外观时，只需改这里（naive 的外观在 styles/naiveTheme.ts 的 Message 段），
//     调用点不动；
//   - 提示文案与图标一一对应（success/warning/error/info），不会各写各的。
//
// 边界（2026-11-25 更新）：页面里不再有常驻的 MetaMessage 提示条 —— 用户要求
// "MetaMessage 全部调整为 toast"。所以 store 里的 error/status 现在也统一走这里。
//
// 2026-11-25：每条 toast 顶端加一条 n-progress 进度条：
//   - 结果类提示：进度条按停留时长从左到右消退（剩余时间可视化）；
//   - loading 提示：进度条走 naive 的 processing 条纹，不自动消失。
import { h } from 'vue'
import { NProgress, type MessageReactive } from 'naive-ui'
import { message } from '../styles/naive'

/** 成功提示停留时长：短，避免一直挡在界面上 */
const SUCCESS_DURATION = 2400
/** 警告提示：比成功多留一会儿，让用户看清哪一部分失败 */
const WARNING_DURATION = 3200
/** 失败提示要多留一会儿：用户往往需要看清原因 */
const ERROR_DURATION = 3800

/** 提示句柄（loading 需要手动关；结果类会自己消失） */
export type ToastId = MessageReactive

export interface NotifyOptions {
  /** 第二行小字（补充说明 / 错误原因），过长会自动截断 */
  description?: string
  /** 覆盖默认停留时长（ms） */
  duration?: number
}

/** 描述文字上限：toast 只有一行的位置，超出部分截断（完整原因见日志/详情） */
const DESCRIPTION_MAX = 120

function clampDescription(text?: string): string | undefined {
  const value = text?.trim()
  if (!value) return undefined
  return value.length > DESCRIPTION_MAX ? `${value.slice(0, DESCRIPTION_MAX - 1)}…` : value
}

/**
 * 把「顶栏进度 + 标题 + 描述」渲染成两行。
 * naive 的 message 内容支持 render 函数，所以不必为了进度条/第二行去改全局 DOM 结构；
 * 样式在 global.css 的 `.app-toast*`（进度条倒计时靠 --toast-duration 传时长）。
 */
function renderContent(title: string, description?: string, duration = SUCCESS_DURATION) {
  const loading = duration <= 0
  return () =>
    h('div', { class: 'app-toast' }, [
      h(
        'div',
        {
          class: ['app-toast__bar', { 'app-toast__bar--loading': loading }],
          // loading 不跑倒计时；结果类由 CSS 按这个时长把进度条缩到 0
          style: loading ? undefined : { '--toast-duration': `${duration}ms` },
        },
        h(NProgress, {
          type: 'line',
          percentage: 100,
          showIndicator: false,
          height: 3,
          borderRadius: 0,
          processing: loading,
        }),
      ),
      h('div', { class: 'app-toast__title' }, title),
      description ? h('div', { class: 'app-toast__desc' }, description) : null,
    ])
}

/** 成功（绿勾）：入库、删除、复制这类"做成了"的结果 */
export function toastSuccess(title: string, options: NotifyOptions = {}) {
  const duration = options.duration ?? SUCCESS_DURATION
  return message.success(renderContent(title, clampDescription(options.description), duration), { duration })
}

/** 警告（黄色感叹号）：一部分成功、需要用户注意但不阻断 */
export function toastWarning(title: string, options: NotifyOptions = {}) {
  const duration = options.duration ?? WARNING_DURATION
  return message.warning(renderContent(title, clampDescription(options.description), duration), { duration })
}

/** 失败（红色感叹号）：请求报错、部分失败 */
export function toastError(title: string, options: NotifyOptions = {}) {
  const duration = options.duration ?? ERROR_DURATION
  return message.error(renderContent(title, clampDescription(options.description), duration), { duration })
}

/** 中性提示（蓝色 i）：没有成败之分、只是告知 */
export function toastInfo(title: string, options: NotifyOptions = {}) {
  const duration = options.duration ?? SUCCESS_DURATION
  return message.info(renderContent(title, clampDescription(options.description), duration), { duration })
}

/**
 * 正在进行的动作（转圈、不自动消失）。
 *
 * ⚠️ 用法与之前一致：结束时先 `dismissToast(id)` 关掉它，再另弹一条新的
 *   toastSuccess / toastError。**不要**指望它自己消失（duration 0 = 常驻），
 *   也**不要**试图把这条原地改成结果 —— 类型与计时器都不会跟着变。
 */
export function toastLoading(title: string, options: NotifyOptions = {}) {
  return message.loading(renderContent(title, clampDescription(options.description), 0), {
    duration: 0,
  })
}

/** 手动关掉某条（不传 = 关掉全部） */
export function dismissToast(id?: ToastId) {
  if (!id) {
    message.destroyAll()
    return
  }
  id.destroy()
}
