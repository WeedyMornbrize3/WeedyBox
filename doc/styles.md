# 样式系统

三层结构：**CSS 变量定义颜色** → **UnoCSS rules 映射变量** → **shortcuts 组合复用**。

## 1. CSS 变量 `src/styles/global.css`

`:root` 浅色（:13），`[data-theme="dark"]` 深色（:60）。`data-theme` 由 `stores/Theme.ts: applyTheme()` 写入 `<html>`。

| 类别 | 变量 |
| --- | --- |
| 背景 | `--color-bg-base #f5f5f7` / `-primary #fff` / `-secondary #f0f0f2` / `-tertiary` / `-hover` / `-card` |
| 文本 | `--color-text-primary #1a1a22` / `-secondary` / `-tertiary` / `-muted` / `-inverse` |
| 品牌 | `--color-primary #5b7a8a` / `-hover` / `-active` / `-light`（12% 透明） |
| 边框 | `--color-border` / `-light` / `-dark` |
| 功能 | `--color-success` `--color-warning` `--color-danger` `--color-info` |
| 阴影 | `--color-shadow` `--color-shadow-heavy` |
| 侧边栏 | `--sidebar-bg` `--sidebar-border` |

全局隐藏滚动条（:107）。**新增颜色必须同时补 `:root` 与 `[data-theme="dark"]` 两处。**

## 2. UnoCSS 规则 `frontend/uno.config.ts`

`presetUno` + `presetAttributify`。

| 前缀 | 类名 |
| --- | --- |
| 背景 | `bg-base` `bg-primary` `bg-secondary` `bg-tertiary` `bg-hover` `bg-card` `bg-sidebar` |
| 文本 | `text-primary` `text-secondary` `text-tertiary` `text-muted` `text-inverse` `text-brand` |
| 品牌 | `bg-brand` `bg-brand-hover` `bg-brand-light` `border-brand` |
| 边框 | `border-theme`（含 1px solid）`border-theme-light` `border-theme-dark` `border-sidebar` |
| 功能 | `text-` / `bg-` + `success` `warning` `danger` `info` |
| 阴影 | `shadow-theme` `shadow-theme-heavy` |
| 动效 | `will-change-width` |

## 3. 常用 shortcuts

| 快捷类 | 用途 |
| --- | --- |
| `card` / `card-hover` | 卡片容器 / 可悬停卡片 |
| `btn-brand` / `btn-ghost` / `btn-delete` | 主按钮 / 幽灵按钮 / 圆形删除按钮 |
| `input-theme` | 输入框统一样式 |
| `title-lg/-md/-sm` `subtitle` | 标题层级 |
| `flex-center` `flex-between` `flex-col-center` `absolute-center` | 布局 |
| `main-content` | 主内容区 |
| `titlebar` `titlebar-btn` `titlebar-btn-close` | 标题栏 |
| `sidebar` `sidebar-header` `logo-icon` `logo-text` | 侧边栏框架 |
| `sidebar-nav-top` `sidebar-nav-bottom` `nav-item` `nav-item-active` `nav-icon` `nav-text` `nav-badge` | 侧边栏导航 |
| `settings-page` `settings-section` `section-header` `settings-item` `item-label` `item-description` `item-control` | 设置页 |
| `theme-options` `theme-option` `theme-option-active` `theme-toggle` `version-text` | 主题与版本控件 |

完整定义见 `uno.config.ts` rules :14-47、shortcuts :65-148。

## 4. 用法示例

```vue
<div class="card flex-between">
  <span class="title-sm">标题</span>
  <button class="btn-brand">确定</button>
</div>
```

## 5. 注意事项

### 优先写进 shortcuts

**可重复使用的样式组合必须加到 `uno.config.ts` 的 `shortcuts` 中**，不要在各组件模板里重复那一长串类名。
单值映射（如 `border-sidebar` → `border-color: var(--sidebar-border)`）写进 `rules`。

### `will-change` 不支持方括号任意值 ⚠️

`presetUno`（`preset-mini/rules.mjs`）的规则是 `/^will-change-(.+)/`，捕获组拿到的是**带方括号的字面量**，
生成的选择器 `.will-change-\[width\]` 永远不等于 token `will-change-[width]`，于是被判为 unmatched **静默丢弃**，
只在构建时输出一行 `[unocss] unmatched utility` 警告，**不会报错**。

| 写法 | 结果 |
| --- | --- |
| `will-change-width` | ✅ `will-change:width` |
| `will-change-transform` | ✅ `will-change:transform` |
| `will-change-[width]` | ❌ 失效（unmatched），**错误根源是 preset 规则，不是本项目配置** |

> 对比：`transition-[width]`、`ease-[cubic-bezier(...)]`、`shadow-[1px_0_16px_var(...)]` 都正常——
> 它们走的是 preset 中会先剥离方括号的正则。**方括号写法只对部分属性有效，不能想当然套用。**

排查方法：构建日志出现 `[unocss] unmatched utility` 时，用以下脚本确认某个类是否被识别：

```js
import { createGenerator } from 'unocss'
import config from './uno.config.ts'
const uno = await createGenerator(config)
console.log(await uno.parseToken('will-change-width'))  // undefined = 未识别
```

### 其他

- `bg-primary` 是**背景**、`text-primary` 是**文字色**，二者都映射 `--color-*` 变量，但语义不同（`--color-bg-primary` vs `--color-text-primary`）。
- 组件内 `<style scoped>` 中的硬编码颜色（如 `ThemeToggle.vue`）不随主题变量走，改主题时需一并检查。
- 覆盖样式用 `!` 前缀，如 `!bg-card`、`!h-[calc(100vh-30px)]`。
- `border-r-2` 只设**宽度**、不设颜色；颜色需另加 `border-sidebar` 等，两者缺一都会看不见边框。
