// 走查 UnoCSS 是否能识别我们写下的类名。
// 背景：组件模板里写错的工具类不会报错、不会警告，只是那条 CSS 不生成，样式悄悄丢失。
// 只在一次构建里出现 `[unocss] unmatched utility` 才提示 shortcut 内部写了无效类。
// 用法（在 frontend/ 下）：node scripts/check-unocss-tokens.mjs
import { createGenerator } from 'unocss'
import config from '../uno.config.ts'

const uno = await createGenerator(config)

const tokens = [
  // 快捷方式是否成组解析
  'icon-xs', 'icon-sm', 'icon-md', 'icon-lg', 'icon-xl',
  'card-flat', 'btn-delete', 'todo-desc-block',
  'filter-bar', 'chip', 'chip-active', 'nav-icon', 'section-icon',
  // 存疑的 preset 能力
  'checked:border-brand',
  'focus-visible:ring-2',
  'focus-visible:ring-brand',
  'appearance-none',
  'w-[18px]',
  'text-[14px]',
  'break-words',
  'whitespace-pre-wrap',
  'min-w-[120px]',
  'py-1.5',
  'py-2.5',
  'gap-2.5',
  'duration-200',
  'border-theme-light',
  'border-theme-dark',
  'bg-priority-low',
  'text-priority-high',
  'opacity-40',
  'group-hover:border-brand',
  'tabular-nums',
  'i-lucide-trash-2',
  'i-lucide-list-todo',
  'i-lucide-circle-check-big',
  'i-lucide-loader-circle',
  // 已知应失效：自定义 rule 不支持透明度后缀
  'bg-secondary/50',
  // 已知应失效：presetUno 的 will-change 不支持方括号任意值
  'will-change-[width]',
]

for (const t of tokens) {
  const parsed = await uno.parseToken(t)
  let out = 'UNMATCHED'
  if (parsed) {
    const first = Array.isArray(parsed) ? parsed[0] : parsed
    out = String(first[2] ?? first[1]).slice(0, 120)
  }
  console.log(`${t.padEnd(30)} -> ${out}`)
}
