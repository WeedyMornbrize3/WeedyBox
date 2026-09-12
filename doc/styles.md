# 样式系统

三层结构：**CSS 变量定义颜色** → **UnoCSS rules 映射变量** → **shortcuts 组合复用**。

## 0. 盒模型基准（`box-sizing: border-box`）⚠️

`global.css` 开头显式声明了全局 `border-box`。**这是必须的**——`presetUno` 不带 border-box reset。

缺了它的后果（实测于 9245 真实页面）：

| | 缺 border-box（content-box） | 加了 border-box |
| --- | --- | --- |
| `h-screen` 根元素实际高度 | 1018 + 30(padding) = **1048** | **1018** ✓ |
| 内容区高度 | 1072 | **1018** ✓ |
| 列表卡片底边 | 超出视口 **+36px** | 收回视口内 **−18px** |
| 最后一条 TODO 可达 | ❌ 滚不到 | **✅** |

根因：`content-box` 下 `height: 100vh` 只约束**内容盒**，padding 会额外撑高，
于是「定了高度又带 padding」的每一层都溢出，逐级累积。
**任何"最后一条/底部够不到"类问题，先查这一条。**

## 1. CSS 变量 `src/styles/global.css`

`:root` 浅色（:13），`[data-theme="dark"]` 深色（:60）。`data-theme` 由 `stores/Theme.ts: applyTheme()` 写入 `<html>`。

| 类别 | 变量 |
| --- | --- |
| 背景 | `--color-bg-base #f5f5f7` / `-primary #fff` / `-secondary #f0f0f2` / `-secondary-soft`（60% 透明）/ `-tertiary` / `-hover` / `-card` |
| 文本 | `--color-text-primary #1a1a22` / `-secondary` / `-tertiary` / `-muted` / `-inverse` |
| 品牌 | `--color-primary #5b7a8a` / `-hover` / `-active` / `-light`（12% 透明） |
| 边框 | `--color-border` / `-light` / `-dark` |
| 功能 | `--color-success` `--color-warning` `--color-danger` `--color-info` |
| 阴影 | `--color-shadow` `--color-shadow-heavy` |
| 侧边栏 | `--sidebar-bg` `--sidebar-border` |

全局隐藏滚动条（:107）。**新增颜色必须同时补 `:root` 与 `[data-theme="dark"]` 两处。**

## 2. UnoCSS 规则 `frontend/uno.config.ts`

`presetUno` + `presetAttributify` + **`presetIcons`**。

### 图标（presetIcons）

`presetIcons` 已接入，数据源是已安装的 `@iconify/json`（**236 个集合**，含 lucide / mdi / tabler / ph 等）。

```vue
<span class="i-lucide-align-left text-sm" />
```

- 命名规则 **`i-<集合>-<图标名>`**。图标在**构建时内联成 `data:image/svg+xml` URI**，运行时零请求、离线可用。
- 已用图标：`i-lucide-align-left`（TodoInput 描述开关）。
- 想减小体积可按集合单独安装（`npm i -D @iconify-json/lucide`），装后 presetIcons 会优先用单集合包。
- 查图标名：`node_modules/@iconify/json/collections.json` 看集合、`json/<集合>.json` 看图标名；或去 icones.js.org 搜。
- **不要再手写重复的 `<svg>` 字面量**——能复用就复用。`ThemeToggle.vue` 里手绘的两个 svg 属历史遗留，可择机替换。

### 规则表

| 前缀 | 类名 |
| --- | --- |
| 背景 | `bg-base` `bg-primary` `bg-secondary` `bg-secondary-soft` `bg-tertiary` `bg-hover` `bg-card` `bg-sidebar` |
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
| `filter-bar` `filter-group` `filter-label` `filter-chips` | 筛选栏骨架（横向排列 + 横向滚动） |
| `chip` / `chip-active` | 药丸按钮：未选中 / 选中（品牌色） |
| `theme-options` `theme-option` `theme-option-active` `theme-toggle` `version-text` | 主题与版本控件 |

完整定义见 `uno.config.ts` rules :14-66、shortcuts :68-161。

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

### 无效工具类会静默失效 ⚠️（最容易踩的坑）

**组件模板里写错的工具类不会报错、不会警告，只是那条 CSS 不生成，样式悄悄丢失。**
（只有 `uno.config.ts` 的 shortcuts 内部写了无效类才会打印 `unmatched utility`。）

已确认**不存在**的写法（都曾在本项目里实际使用并导致 bug）：

| 写的 | 结果 | 正确写法 |
| --- | --- | --- |
| `top-50%` | ❌ 不生成（`%` 不能用在 `top-` 后） | `top-1/2` 或 `top-[50%]` |
| `right-2` | ❌ 不生成（本项目 preset 无此间距刻度） | `right-2` 改成具体值或用 flex 排布 |
| `bg-secondary/50` | ❌ 不生成（自定义 rule 是精确匹配） | `bg-secondary-soft` |
| `will-change-[width]` | ❌ 不生成（preset 正则缺陷） | `will-change-width` |

**排查方法**：怀疑某个类没生效时，直接查产物 CSS 里有没有它。

```powershell
# 在 frontend/ 下执行，把 <类名> 换成要查的类（注意转义 [ . % 等字符）
Select-String -Path .\dist\assets\*.css -Pattern 'top-1\\/2' -SimpleMatch
```

或写个脚本用 `parseToken` 批量确认（见上一节）。

**结论**：**布局不要依赖"绝对定位 + 手调 padding 预留空间"**。一旦定位类失效或内边距与实际宽度不一致，
元素就会重叠且没有任何报错。优先用 **flex 排布**（兄弟元素各占各的空间，结构上不可能重叠）。

### 滚动条：全局隐藏 + 局部恢复

`global.css:106` 用 `::-webkit-scrollbar { display: none !important }` **全局隐藏了滚动条**。
需要滚动条的容器加类名 `scrollbar-theme`（定义在 `global.css:122`，非 UnoCSS）即可恢复：

```vue
<div class="card p-2 flex-1 min-h-[120px] overflow-y-auto scrollbar-theme"> ... </div>
```

| 要点 | 说明 |
| --- | --- |
| 必须配 `overflow-y-auto` | `scrollbar-theme` 只负责滚动条外观，不产生溢出滚动 |
| 为什么是 `!important` | 全局规则带 `!important`；**带 `!important` 的声明永远胜过不带的**，与选择器权重无关 |
| 为什么靠权重而非顺序 | 覆盖规则 `.scrollbar-theme::-webkit-scrollbar`（0,1,0）> 全局 `::-webkit-scrollbar`（0,0,0），两者同为 `!important` 时按权重取胜，**不依赖 import 顺序** |
| 为什么不写在 `uno.config.ts` | UnoCSS utility 规则只能输出**单层声明**，无法生成 `::-webkit-scrollbar` 伪元素选择器——嵌套对象会被字符串化成 `[object Object]`（实测 4 种写法全失败：嵌套 `&`、`$selector`、`:::`；`meta.parent` 只用于输出分组排序，不包裹选择器） |
| 颜色 | 用 `--color-scrollbar` / `--color-scrollbar-hover`，自动跟随明暗主题 |
| Firefox | 由 `scrollbar-width: thin` + `scrollbar-color` 覆盖 |

> 若将来不再全局隐藏滚动条，`scrollbar-theme` 可退化为「只设颜色」的普通样式，其中的 `!important` 也可一并去掉。

### 自定义 rule 不支持 `/opacity` 后缀 ⚠️

`uno.config.ts` 的 `rules` 是**精确字符串匹配**，不是颜色规则，因此 `bg-secondary/50`
这类透明度后缀会被判为 `unmatched utility` 并**静默丢弃**（只出一条构建警告）。
需要半透明时用已定义的 `bg-secondary-soft`（对应 `--color-bg-secondary-soft`）。

> 同理也不支持 `text-primary/70`、`border-theme/50` 等写法——要半透明颜色就新增一个 rule + 对应 CSS 变量。

### 其他

- `bg-primary` 是**背景**、`text-primary` 是**文字色**，二者都映射 `--color-*` 变量，但语义不同（`--color-bg-primary` vs `--color-text-primary`）。
- 组件内 `<style scoped>` 中的硬编码颜色（如 `ThemeToggle.vue`）不随主题变量走，改主题时需一并检查。
- 覆盖样式用 `!` 前缀，如 `!bg-card`、`!h-[calc(100vh-30px)]`。
- `border-r-2` 只设**宽度**、不设颜色；颜色需另加 `border-sidebar` 等，两者缺一都会看不见边框。
