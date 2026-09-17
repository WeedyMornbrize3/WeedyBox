// 走查构建产物 CSS：确认关键类真的生成了规则、布局链未被破坏、无非法声明。
// 需先执行一次 `vite build`（产出 dist/）。
// 用法（在 frontend/ 下）：node scripts/check-build-css.mjs
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const assets = join(process.cwd(), 'dist', 'assets')
const cssFiles = readdirSync(assets).filter(f => f.endsWith('.css'))
const css = cssFiles.map(f => readFileSync(join(assets, f), 'utf8')).join('\n')

console.log(`CSS files: ${cssFiles.join(', ')}  (total ${css.length} chars)`)

const must = [
  'icon-xs', 'icon-sm', 'icon-md', 'icon-lg', 'icon-xl',
  'todo-desc-block', 'card-flat', 'btn-delete', 'btn-brand', 'btn-ghost',
  'filter-bar', 'filter-group', 'filter-label', 'filter-chips', 'chip', 'chip-active',
  'nav-item', 'nav-icon', 'section-icon', 'settings-page', 'settings-section',
  'text-priority-low', 'text-priority-medium', 'text-priority-high',
  'border-theme-light', 'border-theme-dark', 'bg-brand', 'text-brand', 'text-danger', 'text-success', 'text-warning',
  'bg-secondary-soft', 'tabular-nums', 'flex-col-center',
  'sr-only', 'scrollbar-theme', 'todo-checkbox', 'prefers-reduced-motion',
  'checked\\:border-brand', 'focus-visible\\:ring-2', 'appearance-none', 'opacity-40', 'group-hover\\:border-l-brand',
  'i-lucide-trash-2', 'i-lucide-list-todo', 'i-lucide-signal-high',
  'i-lucide-inbox', 'i-lucide-loader-circle', 'i-lucide-sun', 'i-lucide-palette',
  'i-lucide-plus', 'i-lucide-rotate-ccw', 'i-lucide-search-x', 'i-lucide-layers',
]

let bad = 0
for (const token of must) {
  const hit = css.includes(token)
  if (!hit) { bad++; console.log(`  MISSING  ${token}`) }
}
console.log(bad === 0 ? `\nOK: 所有 ${must.length} 个关键类都已生成规则` : `\n${bad} 个类缺失`)

// 非法 CSS 检测：border-color 收到整条简写值
const invalid = css.match(/border-color:1px solid/g)
console.log(invalid ? `\n❌ 仍有非法 border-color 简写值: ${invalid.length} 处` : '\nOK: 无 border-color 简写值污染')

// 布局链 / 无障碍关键声明（用精确子串，避免 minify 后的选择器分组影响正则）
const layout = [
  ['box-sizing reset', 'box-sizing:border-box'],
  ['html/body/#app 定高', 'height:100%'],
  ['全局隐藏滚动条', '::-webkit-scrollbar{background:0 0!important;width:0!important'],
  ['scrollbar-theme 恢复', '.scrollbar-theme::-webkit-scrollbar{background:0 0!important;width:8px!important'],
  ['scrollbar-theme Firefox', 'scrollbar-width:thin'],
  ['h-screen', 'height:100vh'],
  ['min-h-0', 'min-height:0'],
  ['flex-1', 'flex:1'],
  ['flex-col', 'flex-direction:column'],
  ['overflow-hidden', 'overflow:hidden'],
  ['overflow-y-auto', 'overflow-y:auto'],
  ['min-h-[120px]', 'min-height:120px'],
  ['浅色主题变量', '--color-text-primary:#0f172a'],
  ['深色主题变量', '--color-text-primary:#e6edf3'],
  ['prefers-reduced-motion', 'prefers-reduced-motion:reduce'],
  ['焦点环兜底', ':focus-visible{outline:2px solid var(--color-ring)'],
  ['勾号 mask', '.todo-checkbox:checked:after'],
]
let layoutBad = 0
for (const [name, needle] of layout) {
  if (!css.includes(needle)) { layoutBad++; console.log(`  MISSING  ${name}  (${needle})`) }
}
console.log(layoutBad === 0 ? `OK: 布局链与无障碍 ${layout.length} 项断言全部通过` : `${layoutBad} 项布局断言失败`)

// ===== 边框专项断言 =====
// 回归背景：presetUno 的 preflight 把所有元素重置为 border-width:0; border-style:solid。
// 因此「只设 border-color」的规则会让边框彻底不显示；若元素另有非零 border-width，
// 样式还会退回初始值 outset，渲染成 3D 伪立体边框。
// 所以每条 border-* 规则都必须自带 width + style + color 三件套。
const borderRules = [
  ['border-theme', 'border:1px solid var(--color-border)'],
  ['border-theme-light', 'border:1px solid var(--color-border-light)'],
  ['border-theme-dark', 'border:1px solid var(--color-border-dark)'],
  ['border-brand', 'border:1px solid var(--color-primary)'],
]
let borderBad = 0
for (const [name, needle] of borderRules) {
  if (!css.includes(needle)) { borderBad++; console.log(`  MISSING  ${name}  -> 期望 ${needle}`) }
}
// 反向断言：任何 CSS 变量色值都不应被塞进 border-color 的简写位置
const shorthandInColor = css.match(/border-color:1px\s+solid/g)
if (shorthandInColor) { borderBad++; console.log(`  ❌ 有 ${shorthandInColor.length} 处 border-color 收到简写值（非法 CSS）`) }
console.log(borderBad === 0 ? `OK: 边框规则 ${borderRules.length} 项均自带 width+style+color` : `${borderBad} 项边框断言失败`)

// 单侧边框：必须显式把其余边归零，否则会渲染成四边都有框
const sideRules = [
  ['border-b-theme-light', 'border-bottom:1px solid var(--color-border-light)', ['border-top:0', 'border-right:0', 'border-left:0']],
  ['border-r-sidebar', 'border-right:2px solid var(--sidebar-border)', ['border-top:0', 'border-bottom:0', 'border-left:0']],
  // 描述块 hover：必须是单侧规则，否则会从「一条竖线」变成「一个方框」
  ['border-l-brand', 'border-left:2px solid var(--color-primary)', ['border-top:0', 'border-right:0', 'border-bottom:0']],
]
let sideBad = 0
for (const [name, own, zeros] of sideRules) {
  if (!css.includes(own)) { sideBad++; console.log(`  MISSING  ${name} -> 期望含 ${own}`) }
  for (const z of zeros) {
    // 同一规则块内应出现归零声明（minify 后形如 border-right:0）
    if (!css.includes(z)) { sideBad++; console.log(`  MISSING  ${name} 缺少 ${z}（会退化成四边框）`) }
  }
}
console.log(sideBad === 0 ? `OK: 单侧边框 ${sideRules.length} 条均显式置零其余边` : `${sideBad} 项单侧边框断言失败`)

// ===== 同一元素上「两条规则都改 border-left / border-top …」的自查（仅告警）=====
// 背景：UnoCSS 对同一属性只保留一条规则，且 shortcut 输出在 rules 之前、
// rules 之间按字母序，因此同一元素上叠加两条改同一侧边框的类时，
// 后写的那条可能被静默丢弃（曾导致 hover 时描述块从「一条竖线」变成「一个方框」）。
// 这里扫描模板，把同一 class 属性里出现两次同侧边框类的地方列出来，供人工确认。
const SRC = join(process.cwd(), 'src')
function walk(dir) {
  const out = []
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, e.name)
    if (e.isDirectory()) out.push(...walk(p))
    else if (e.name.endsWith('.vue')) out.push(p)
  }
  return out
}
const sidePattern = /(?:^|[\s"'])!?(?:group-hover:|hover:|focus:)?border-(l|t|r|b)(?:-[a-z-]+)?(?=[\s"'])/g
const conflicts = []
for (const file of walk(SRC)) {
  const text = readFileSync(file, 'utf8')
  for (const m of text.matchAll(/class="([^"]*)"/g)) {
    const attr = m[1]
    const sides = {}
    for (const sm of attr.matchAll(sidePattern)) {
      const side = sm[1]
      // 同一侧出现两次（且不是同类不同尺寸）时记一笔
      sides[side] = (sides[side] ?? 0) + 1
    }
    const dup = Object.entries(sides).filter(([, n]) => n > 1)
    if (dup.length) {
      conflicts.push(`${file.replace(process.cwd(), '.')} :: ${dup.map(([s, n]) => `border-${s}×${n}`).join(', ')} :: ${attr.slice(0, 90)}`)
    }
  }
}
if (conflicts.length) {
  console.log(`\n⚠️  疑似同侧边框类叠加 ${conflicts.length} 处（请确认是否真的需要，UnoCSS 可能只保留其一）：`)
  for (const c of conflicts) console.log('    - ' + c)
} else {
  console.log('OK: 模板中无同侧边框类叠加')
}


// emoji 是否还残留在产物 JS 里（结构性图标应为 0）
const jsFiles = readdirSync(assets).filter(f => f.endsWith('.js'))
const js = jsFiles.map(f => readFileSync(join(assets, f), 'utf8')).join('\n')
const emoji = js.match(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{FE0F}]/gu)
console.log(emoji ? `\n⚠️ 产物 JS 中仍有 emoji: ${[...new Set(emoji)].join(' ')}` : '\nOK: 产物 JS 中无 emoji')
