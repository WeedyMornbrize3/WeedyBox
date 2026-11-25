// 走查构建产物 CSS：确认关键类真的生成了规则、布局链未被破坏、无非法声明。
// 需先执行一次 `vite build`（产出 dist/）。
// 用法（在 frontend/ 下）：node scripts/check-build-css.mjs
//
// 2026-11-24 引入 naive-ui 后口径变了：
//   · 控件（按钮/输入框/下拉/标签/弹窗/分页…）的长相改由 naive 的 CSS-in-JS 在**运行时**注入，
//     不会出现在 dist 的 CSS 文件里 —— 所以这里不再断言 btn-*/input-theme/chip 这些 shortcut，
//     改为断言「项目的布局类 + 色值 token + 图标」以及 naive 主题桥接的源码存在。
//   · 控件外观的唯一来源是 src/styles/naiveTheme.ts（token → themeOverrides）。
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const assets = join(process.cwd(), 'dist', 'assets')
const cssFiles = readdirSync(assets).filter((f) => f.endsWith('.css'))
const css = cssFiles.map((f) => readFileSync(join(assets, f), 'utf8')).join('\n')

console.log(`CSS files: ${cssFiles.join(', ')}  (total ${css.length} chars)`)

const must = [
  // 图标尺寸五档 + 项目常用图标（presetIcons 构建期内联，缺了说明没被扫到）
  'icon-xs', 'icon-sm', 'icon-md', 'icon-lg', 'icon-xl',
  // 布局 / 结构类
  'todo-desc-block', 'card', 'filter-bar', 'filter-group', 'filter-label',
  'sidebar', 'sidebar-nav-top', 'sidebar-nav-bottom', 'logo-text', 'settings-page', 'settings-item',
  'text-priority-low', 'text-priority-medium', 'text-priority-high',
  'border-theme-light', 'bg-brand', 'text-brand', 'text-danger', 'text-success',
  'bg-secondary-soft', 'tabular-nums', 'flex-col-center',
  'sr-only', 'scrollbar-theme', 'prefers-reduced-motion',
  // toast 顶端的进度条（2026-11-25）
  'app-toast__bar', 'app-toast-countdown',
  // 滚动边缘渐隐（2026-11-25，参考 shadcn 的 scroll-fade）
  'scroll-fade', '--scroll-fade-top', '--scroll-fade-bottom',
  // 游戏库封面：点击水纹 / 名称浮层 / 批量模式样式（2026-11-25）
  'lib-ripple', 'lib-card-caption', 'lib-batch-mode', 'lib-check',
  // /works：页面背景/tab 区预模糊 hero 遮罩 + 页面背景 + 启动方式箭头旋转（2026-11-25）
  'work-glass', 'work-page-ambient', 'work-tabs-wrap', 'launch-mode-chevron',
  // 描述块 hover 只把左竖线换成品牌色（单侧规则，不能退化成方框）
  'group-hover\\:border-l-brand',
  // 动效 token
  '--motion-slow', '--ease-enter',
  // 图标（抽查几个高频的）
  'i-lucide-trash-2', 'i-lucide-list-todo', 'i-lucide-signal-high',
  'i-lucide-sun', 'i-lucide-palette', 'i-lucide-plus', 'i-lucide-refresh-cw',
  'i-lucide-search', 'i-lucide-folder-open', 'i-lucide-braces',
]

let bad = 0
for (const token of must) {
  if (!css.includes(token)) { bad++; console.log(`  MISSING  ${token}`) }
}
console.log(bad === 0 ? `\nOK: 所有 ${must.length} 个关键类都已生成规则` : `\n${bad} 个类缺失`)

// 非法 CSS 检测：border-color 收到整条简写值
const invalid = css.match(/border-color:1px solid/g)
console.log(invalid ? `\n❌ 仍有非法 border-color 简写值: ${invalid.length} 处` : '\nOK: 无 border-color 简写值污染')

// ===== 布局链 / 无障碍关键声明 =====
const layout = [
  ['box-sizing reset', 'box-sizing:border-box'],
  ['html/body/#app 定高', 'height:100%'],
  ['全局隐藏滚动条', '::-webkit-scrollbar{background:0 0!important;width:0!important'],
  ['scrollbar-theme 恢复', '.scrollbar-theme::-webkit-scrollbar{background:0 0!important;width:8px!important'],
  ['scrollbar-theme Firefox', 'scrollbar-width:thin'],
  ['h-screen', 'height:100vh'],
  // presetWind4 下 `min-h-0` 的输出是 calc(var(--spacing) * 0)，不是字面量 0
  ['min-h-0', 'min-height:calc(var(--spacing) * 0)'],
  ['flex-1', 'flex:1'],
  ['flex-col', 'flex-direction:column'],
  ['overflow-hidden', 'overflow:hidden'],
  ['overflow-y-auto', 'overflow-y:auto'],
  ['min-h-[120px]', 'min-height:120px'],
  // 主题色已从 global.css 移到 styles/theme.ts（四色运行时推导），这里只断言
  // 与主题无关的登录底色兜底与 toast 进度条动画仍然存在
  ['启动底色兜底', 'background:var(--color-bg-primary,#101418)'],
  ['toast 倒计时动画', 'animation-name:app-toast-countdown'],
  ['prefers-reduced-motion', 'prefers-reduced-motion:reduce'],
  ['焦点环兜底', ':focus-visible{outline:2px solid var(--color-ring'],
  ['提示条两行样式', '.app-toast__title'],
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
  ['border-brand', 'border:1px solid var(--color-primary)'],
]
let borderBad = 0
for (const [name, needle] of borderRules) {
  if (!css.includes(needle)) { borderBad++; console.log(`  MISSING  ${name}  -> 期望 ${needle}`) }
}
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
    if (!css.includes(z)) { sideBad++; console.log(`  MISSING  ${name} 缺少 ${z}（会退化成四边框）`) }
  }
}
console.log(sideBad === 0 ? `OK: 单侧边框 ${sideRules.length} 条均显式置零其余边` : `${sideBad} 项单侧边框断言失败`)

// ===== naive-ui 主题桥接（源码级断言）=====
// 回归背景：themeOverrides 里如果给 `var(--color-primary)` 这类未解析值，seemly 解析会失败，
// hover/pressed 退化成 naive 默认色且**不会有任何报错**。所以必须存在「运行时读 CSS 变量」
// 的桥接层，并且 App 用 n-config-provider 把同一份配置注入整棵树。
{
  const root = process.cwd()
  const themeFile = join(root, 'src', 'styles', 'naiveTheme.ts')
  const naiveFile = join(root, 'src', 'styles', 'naive.ts')
  const appFile = join(root, 'src', 'App.vue')
  const presetsFile = join(root, 'src', 'styles', 'themePresets.ts')
  const engineFile = join(root, 'src', 'styles', 'theme.ts')
  const presetsSource = existsSync(presetsFile) ? readFileSync(presetsFile, 'utf8') : ''
  const engineSource = existsSync(engineFile) ? readFileSync(engineFile, 'utf8') : ''
  const checks = [
    ['naiveTheme.ts 存在', existsSync(themeFile)],
    ['naive.ts（discrete message）存在', existsSync(naiveFile)],
    ['App.vue 使用 n-config-provider', existsSync(appFile) && readFileSync(appFile, 'utf8').includes('n-config-provider')],
    ['主题从 CSS 变量读取', existsSync(themeFile) && readFileSync(themeFile, 'utf8').includes('getComputedStyle')],
    ['toast 已走 naive message', readFileSync(join(root, 'src', 'utils', 'toast.ts'), 'utf8').includes("from '../styles/naive'")],
    // 2026-11-25 四色主题：主题清单只写 4 个颜色，其余由 theme.ts 推导
    ['四色主题清单存在', presetsSource.toLowerCase().includes('#404258') && presetsSource.toLowerCase().includes('#6b728e')],
    ['主题引擎按对比度推导', engineSource.includes('ensureContrast') && engineSource.includes('buildThemeVars')],
    ['主题写入 CSS 变量', engineSource.includes('style.setProperty')],
    // 2026-11-25 侧边栏改用 naive 的 n-menu：菜单外观必须只在 naiveTheme 的 Menu 段里改
    ['侧边栏走 n-menu', readFileSync(join(root, 'src', 'components', 'Sidebar.vue'), 'utf8').includes('n-menu')],
    ['Menu 主题已接桥', existsSync(themeFile) && readFileSync(themeFile, 'utf8').includes('Menu:')],
  ]
  let naiveBad = 0
  for (const [name, ok] of checks) {
    if (!ok) { naiveBad++; console.log(`  MISSING  ${name}`) }
  }
  console.log(naiveBad === 0 ? 'OK: naive-ui 主题桥接与 provider 接线完整' : `${naiveBad} 项 naive 桥接断言失败`)
}

// ===== 模板静态扫描 =====
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

// 「border 缺少 border-style」自查（仅告警）
// 背景：UnoCSS 的 `border` 工具类只输出 border-width，不输出 border-style。
// <button> 的 UA 默认 border-style 是 `outset`，于是计算值变成 `1px outset`。
{
  const BORDER_RE = /(^|[\s"'])border(-[trbl])?(?![-\w])/
  const offenders = []
  for (const file of walk(SRC)) {
    const text = readFileSync(file, 'utf8')
    for (const m of text.matchAll(/class="([^"]*)"/g)) {
      const attr = m[1]
      if (!BORDER_RE.test(attr)) continue
      if (/\bborder-(solid|dashed|dotted|double|none)\b/.test(attr)) continue
      offenders.push(`${file.replace(process.cwd(), '.')} :: ${attr.slice(0, 90)}`)
    }
  }
  if (offenders.length) {
    console.log(`\n⚠️  ${offenders.length} 处用了 border 但没写 border-style（<button> 上会退化成 outset 立体边）：`)
    for (const o of offenders) console.log('    - ' + o)
  } else {
    console.log('OK: 无「border 缺 border-style」的写法')
  }
}

// 同一元素上「两条规则都改 border-left / border-top …」的自查（仅告警）
const sidePattern = /(?:^|[\s"'])!?(?:group-hover:|hover:|focus:)?border-(l|t|r|b)(?:-[a-z-]+)?(?=[\s"'])/g
const conflicts = []
for (const file of walk(SRC)) {
  const text = readFileSync(file, 'utf8')
  for (const m of text.matchAll(/class="([^"]*)"/g)) {
    const attr = m[1]
    const sides = {}
    for (const sm of attr.matchAll(sidePattern)) {
      sides[sm[1]] = (sides[sm[1]] ?? 0) + 1
    }
    const dup = Object.entries(sides).filter(([, n]) => n > 1)
    if (dup.length) {
      conflicts.push(`${file.replace(process.cwd(), '.')} :: ${dup.map(([s, n]) => `border-${s}×${n}`).join(', ')} :: ${attr.slice(0, 90)}`)
    }
  }
}
if (conflicts.length) {
  console.log(`\n⚠️  疑似同侧边框类叠加 ${conflicts.length} 处：`)
  for (const c of conflicts) console.log('    - ' + c)
} else {
  console.log('OK: 模板中无同侧边框类叠加')
}

// emoji 是否还残留在产物 JS 里（结构性图标应为 0）
const jsFiles = readdirSync(assets).filter((f) => f.endsWith('.js'))
const js = jsFiles.map((f) => readFileSync(join(assets, f), 'utf8')).join('\n')
const emoji = js.match(/[\u{1F300}-\u{1FAFF}\u{2600}-\u{27BF}\u{2B00}-\u{2BFF}\u{FE0F}]/gu)
console.log(emoji ? `\n⚠️ 产物 JS 中仍有 emoji: ${[...new Set(emoji)].join(' ')}` : '\nOK: 产物 JS 中无 emoji')
