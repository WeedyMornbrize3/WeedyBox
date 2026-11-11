// frontend/src/utils/clipboard.ts
//
// 复制的统一入口。
//
// 为什么不在每个组件里直接 `navigator.clipboard.writeText`：
//   - Clipboard API 只在「安全上下文」可用，且 WebView2 里偶尔会抛权限错误；
//   - 旧 WebView / 自定义协议下 navigator.clipboard 可能整个不存在。
// 所以这里做一次降级：优先 Clipboard API，失败退回 textarea + execCommand('copy')。
//
// 注意：调用方必须在**用户手势**（点击/keydown）的同步调用链里调用它，
//      否则浏览器会以「缺少用户激活」为由拒绝写入。

/** 复制一段文本。成功 resolve，失败 reject（错误里带原因，调用方可直接提示）。 */
export async function copyText(text: string): Promise<void> {
  if (!text) {
    throw new Error('没有可复制的内容')
  }

  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // 落到下面的兜底实现（权限/非安全上下文）
    }
  }

  if (!legacyCopy(text)) {
    throw new Error('剪贴板不可用')
  }
}

/**
 * 兜底实现：造一个不可见的 textarea，选中后 execCommand('copy')。
 * execCommand 已废弃，但在老 WebView 里是唯一可靠路径；失败返回 false 而不是抛错。
 */
function legacyCopy(text: string): boolean {
  if (typeof document === 'undefined') return false

  const holder = document.createElement('textarea')
  holder.value = text
  // 放到视口外 + 透明：避免滚动跳动与视觉闪烁
  holder.setAttribute('readonly', '')
  holder.style.position = 'fixed'
  holder.style.top = '0'
  holder.style.left = '0'
  holder.style.opacity = '0'
  holder.style.pointerEvents = 'none'

  document.body.appendChild(holder)
  try {
    holder.select()
    holder.setSelectionRange(0, holder.value.length)
    return document.execCommand('copy')
  } catch {
    return false
  } finally {
    document.body.removeChild(holder)
  }
}
