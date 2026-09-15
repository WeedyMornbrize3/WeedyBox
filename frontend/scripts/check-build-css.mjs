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
  'checked\\:border-brand', 'focus-visible\\:ring-2', 'appearance-none', 'opacity-40', 'group-hover\\:border-brand',
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


// emoji 是否还残留在产物 JS 里（结构性图标应为 0）
const jsFiles = readdirSync(assets).filter(f => f.endsWith('.js'))
const js = jsFiles.map(f => readFileSync(join(assets, f), 'utf8')).join('\n')
const emoji = js.match(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{FE0F}]/gu)
console.log(emoji ? `\n⚠️ 产物 JS 中仍有 emoji: ${[...new Set(emoji)].join(' ')}` : '\nOK: 产物 JS 中无 emoji')
