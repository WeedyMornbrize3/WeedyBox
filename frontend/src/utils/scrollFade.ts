// frontend/src/utils/scrollFade.ts
// 滚动边缘渐隐（参考 shadcn 的 scroll-fade，按本项目的 UnoCSS / 无 Tailwind 体系改写）。
//
// 为什么不用纯 CSS 的静态 mask：
//   静态 `mask-image: linear-gradient(transparent, #000 24px, …)` 在「还没滚动」和
//   「已经滚到底」时也会把第一条/最后一条淡掉，看起来像内容被裁了。
//   这里用指令在 mounted / scroll / 尺寸变化 / 子节点变化时，只改元素上的两个 CSS 变量：
//     --scroll-fade-top    > 0 表示上方还有内容没滚出来
//     --scroll-fade-bottom > 0 表示下方还有内容没滚出来
//   global.css 的 `.scroll-fade` 用它们拼 mask-image，因此「能不能继续滚」与实际一致。
//
// 用法：<div class="scroll-fade overflow-y-auto" v-scroll-fade>…</div>
// 注意：mask 会让该元素成为 fixed 后代的包含块，且会一起淡掉元素内的滚动条；
//       所以只加在纯列表/正文的滚动容器上，不要把 fixed 浮层（如批量操作条）放进去。

import type { Directive } from 'vue'

/** 渐隐高度：比常规留白略大一点，肉眼能看出「下面还有」但不至于吃掉首行 */
const FADE_SIZE = 28

function sync(el: HTMLElement) {
  const maxScroll = el.scrollHeight - el.clientHeight
  const scrollable = maxScroll > 1
  const top = scrollable && el.scrollTop > 1 ? FADE_SIZE : 0
  const bottom = scrollable && el.scrollTop < maxScroll - 1 ? FADE_SIZE : 0
  el.style.setProperty('--scroll-fade-top', `${top}px`)
  el.style.setProperty('--scroll-fade-bottom', `${bottom}px`)
}

/** 每个元素一份清理函数（WeakMap 避免把闭包挂在 DOM 上） */
const cleanups = new WeakMap<HTMLElement, () => void>()

export const vScrollFade: Directive<HTMLElement> = {
  mounted(el) {
    let frame = 0
    const schedule = () => {
      if (frame) return
      frame = window.requestAnimationFrame(() => {
        frame = 0
        sync(el)
      })
    }

    el.addEventListener('scroll', schedule, { passive: true })

    // 容器尺寸变化（窗口缩放 / 侧栏折叠 / tab 从 display:none 变可见）
    const resizeObserver = new ResizeObserver(schedule)
    resizeObserver.observe(el)

    // 内容变化（搜索结果、候选分页、TODO 增删）—— scrollHeight 变了但容器尺寸没变，
    // ResizeObserver 不会触发，用 MutationObserver 兜住。
    const mutationObserver = new MutationObserver(schedule)
    mutationObserver.observe(el, { childList: true, subtree: true, characterData: true })

    // 图片加载完会改变内容高度（封面 data URL 是异步填进来的）
    el.addEventListener('load', schedule, true)
    schedule()

    cleanups.set(el, () => {
      if (frame) window.cancelAnimationFrame(frame)
      el.removeEventListener('scroll', schedule)
      el.removeEventListener('load', schedule, true)
      resizeObserver.disconnect()
      mutationObserver.disconnect()
    })
  },

  unmounted(el) {
    cleanups.get(el)?.()
    cleanups.delete(el)
  },
}
