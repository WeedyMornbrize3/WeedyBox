// frontend/src/stores/autoHeightStore.ts
//
// 高度自适应的展开/收起过渡（原 composables/useAutoHeight.ts，按项目约定搬进 stores）。
//
// 为什么放在 store 而不是 composables：
// 本项目用 Pinia 管状态，约定「composables 目录不承载这类共享逻辑」，统一从 stores 取。
// 这里没有业务状态，所以 store 只提供 duration（读 global.css 的 --motion-slow）
// 与四个 <Transition> 钩子；它们被多处复用，写成一个 store 就只解析一次全局时长。
//
// CSS 无法过渡到 `height: auto`，所以必须在进入/离开时把高度显式测出来再动画。
// 用法（配 Vue 内置 <Transition>）：
//
//   <Transition name="collapse" @enter="collapse.enter" @after-enter="collapse.afterEnter"
//               @leave="collapse.leave" @after-leave="collapse.afterLeave">
//     <div v-show="open">...</div>
//   </Transition>
//
// 无障碍：`prefers-reduced-motion: reduce` 时由 global.css 把过渡时长压到 0.01ms，
// 这里不做额外分支 —— 测出的高度在无过渡时也会被正确清除。

import { defineStore } from 'pinia'
import type { CSSProperties } from 'vue'

/** 读全局动效时长；读不到时退回 280ms（不在这里写死第二个数字之外的值） */
function readCollapseDuration(): number {
  if (typeof window === 'undefined') return 280
  const raw = getComputedStyle(document.documentElement).getPropertyValue('--motion-slow').trim()
  const n = Number.parseInt(raw, 10)
  return Number.isFinite(n) && n > 0 ? n : 280
}

const EASING = 'var(--ease-enter, cubic-bezier(0.16, 1, 0.3, 1))'

function baseStyle(duration: number): CSSProperties {
  return {
    overflow: 'hidden',
    transition: [`height ${duration}ms ${EASING}`, `opacity ${duration}ms ${EASING}`].join(', '),
  }
}

export const useAutoHeightStore = defineStore('autoHeight', () => {
  // ===== State =====
  /** 展开/收起时长（ms）；首次取用时读 global.css 的 --motion-slow */
  const duration = readCollapseDuration()

  // ===== Actions =====

  /** 进入前：高度 0，内容淡入但不位移（位移会与外层 flex 打架，这里只用高度+透明度） */
  function enter(el: Element, done: () => void) {
    const target = el as HTMLElement
    // 先挂上过渡与监听，再测量，避免高度变化早于监听器就绪
    Object.assign(target.style, baseStyle(duration))
    const onEnd = (e: TransitionEvent) => {
      if (e.propertyName !== 'height') return
      target.removeEventListener('transitionend', onEnd)
      done()
    }
    target.addEventListener('transitionend', onEnd)
    target.style.height = '0px'
    // 强制一次同步布局，确保后续高度从 0 开始过渡
    void target.offsetHeight
    target.style.height = `${target.scrollHeight}px`
  }

  /** 进入后：把固定高度放开，交给内容自适应（否则内部换行会溢出） */
  function afterEnter(el: Element) {
    const target = el as HTMLElement
    target.style.height = ''
    target.style.overflow = ''
    target.style.transition = ''
  }

  /** 离开前：从当前实际高度过渡到 0（不能直接写 0，否则浏览器无法算出起点） */
  function leave(el: Element, done: () => void) {
    const target = el as HTMLElement
    Object.assign(target.style, baseStyle(duration))
    target.style.height = `${target.scrollHeight}px`
    void target.offsetHeight
    target.style.height = '0px'
    const onEnd = (e: TransitionEvent) => {
      if (e.propertyName !== 'height') return
      target.removeEventListener('transitionend', onEnd)
      done()
    }
    target.addEventListener('transitionend', onEnd)
  }

  /** 离开后：清空内联样式，回到初始状态 */
  function afterLeave(el: Element) {
    const target = el as HTMLElement
    target.style.height = ''
    target.style.overflow = ''
    target.style.transition = ''
  }

  return { duration, enter, afterEnter, leave, afterLeave }
})
