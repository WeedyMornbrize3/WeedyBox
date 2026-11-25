# WeedyBox — 跨工作区项目上下文（精简/机器可读）

> 用途：给**另一个工作区 / 另一个 agent 会话**快速加载本项目现状，避免重新通读源码。
> 本文件是「现状 + 决策 + 坑」的单一入口；`doc/*.md` 是逐函数行号速查（部分已过期，见 §9）。
> 记录时点：2026-11-23（工作区有较多未提交改动：游戏库批量操作 UI、`/works` Steam 式排版、DLsite 预览图解析、Steam hero 背景图）。所有数字均为**实测**，不是估计。

> ⚠️ **2026-11-24 起全站控件改用 naive-ui**：`vue-sonner` 已移除、`btn-*` / `input-theme` / `chip-*` / `control-h` 等控件 shortcut 已从 `uno.config.ts` 删除、`MetaLaunchModeMenu.vue` 已删除。
> §1.1 的「轻提示」行、§4.7（vue-sonner）、§6.5（按钮体系 shortcut）、§7.2 里「两套输入框必须成对改」等描述**已过时**，以本节 §0 为准。

---

## 0. 2026-11-24：naive-ui 迁移（最新口径）

### 0.1 依赖与文件

- 新增 **`naive-ui@^2.45.3`**（`frontend/package.json` 的 dependencies）；**移除 `vue-sonner`**。
- 新增 `frontend/src/styles/naiveTheme.ts`：**token → naive themeOverrides 的唯一桥接层**。
  在主题切换时用 `getComputedStyle(documentElement)` 读 `--color-*` 的真实色值再推算（naive 要用可解析色值；写 `var(--x)` 会让 seemly 解析失败、hover 静默退化成 naive 默认色）。
- 新增 `frontend/src/styles/naive.ts`：`naiveConfig`（响应式 theme / themeOverrides）+ `syncNaiveTheme(isDark)` + `createDiscreteApi(['message','notification'])`。树内的 `<n-config-provider>` 与 toast 用的 discrete message **共用同一个 `naiveConfig`**。
- `App.vue`：`<n-config-provider :theme :theme-overrides :locale="zhCN" :date-locale>` 包住 `MainLayout`；`watch(() => \`${themeStore.effectiveTheme}:${themeStore.themeId}\`, …, { flush:'post' })` 负责换肤（2026-11-25 起四色主题变化也会触发）。
- `stores/Theme.ts`：新增 `systemDark` ref（原来 computed 直接读 `matchMedia`，系统换肤不会触发重算）；2026-11-25 再加 `themeId / activePreset / setThemeId`（四色主题，见 §0.6）。
- `utils/toast.ts`：**对外 API 不变**（toastSuccess / toastWarning / toastError / toastInfo / toastLoading / dismissToast），底层是 naive message（标题 + 描述两行渲染函数，样式在 `global.css` 的 `.app-toast__*`）；2026-11-25 每条消息顶端加了 n-progress 进度条。`components/AppToaster.vue` 已删除。
- 控件外观集中点：`Button / Input / InputNumber / Select / Tag / Checkbox / Radio / Switch / Card / Modal / Drawer / Popover / Tooltip / Dropdown / Menu / Message / Alert / Empty / Spin / Pagination / Descriptions / Tabs / Badge / Divider / Statistic / Progress / Popconfirm` 都有一份 override；统一口径 **控件高 35.2px（2.2rem）、按钮圆角 6px、容器 10–12px、面=bg-card / 输入=bg-secondary / 悬停=bg-hover**。

### 0.2 哪些换成了 naive、哪些仍自绘

| 区域 | 现状 |
| --- | --- |
| 标题栏 | 外壳、拖动区、关闭浮层结构保留；最小化/最大化/关闭三颗按钮 = `n-button`，关闭浮层 = `n-popover`（点外部关闭由 popover 负责）+ `n-checkbox` |
| 侧边栏 | **2026-11-25 起改为 naive 的 `n-menu`**（用户要求「改用 naive ui 样式和逻辑」）：`Sidebar.vue` 只保留外壳（宽度过渡 / 底色 / 右侧 2px 分隔线 / logo 折叠开关），菜单项的高度、圆角、选中态、折叠态 tooltip、键盘语义全交给 naive；外观在 `naiveTheme.ts` 的 `Menu` 段，见 §0.7 |
| TODO | 输入区 `n-input-group`（标题 + 描述 textarea + 添加按钮）、优先级/筛选 `n-radio-group` / checkable `n-tag`、行选择 `n-checkbox`、行删除 `n-popconfirm`、批量删除 `n-popconfirm`、空态/错误 `n-empty` / `n-alert`、加载 `n-spin` |
| 游戏库 | 顶栏动作 `n-button`、添加游戏弹窗 `n-modal preset=card`（点遮罩不关、Esc 关）、扫描面板 `n-radio-group` / `n-input` / `n-button` / `n-checkbox` / `n-tag` / `n-spin`、卡片删除 `n-popconfirm`、空态 `n-empty` |
| 作品详情 | 动作条/标签/简介容器 `n-card`、启动按钮与启动方式下拉 = `n-button` + `n-dropdown`、标签编辑 `n-dynamic-tags`、键值表 `n-descriptions`、Steam 动态 `n-statistic`、缓存徽标 `n-tag`；2026-11-24（第二次重排）起还有 `MetaWorkTopBar`（动作栏）、`n-tabs` 下划线 tab、`MetaWorkEditorForm`（`n-date-picker` / `n-dynamic-tags` / `n-input` 编辑全部元数据），见 §0.5 |
| 设置 | 卡片 `n-card`、主题模式/关闭行为 `n-radio-group`、语言 `n-select`、标签数量 `n-input-number`、清理缓存 `n-button`、版本 `n-tag` |
| 调试页 | `MetaApiTestCard` 的卡片/搜索/提示/分页/原始 JSON 都随共享积木一起换；`pre` 仍自绘 |
| 保留自绘 | hero 大图区（`MetaWorkHero`，图片+渐变遮罩+图上白字）、游戏库封面网格卡（2:3 裁切 + 选中遮罩）、右侧 56px 批量浮条（位置/尺寸/不参与布局的约束不变，内部改成 n-button + n-tooltip）、封面缩略图 `MetaCover`、来源徽标 `.src-tag`、行内破坏性按钮的描边色语言（`--color-danger` 描边 → hover 填红），确认动作交给 `n-popconfirm` |

### 0.3 交互变更（用户 2026-11-24 批准「可以改用 naive 的交互组件」）

- 删除确认从「连点两下 + 移开撤销」统一改成 **`n-popconfirm` 气泡确认**：TODO 单条、TODO 批量、游戏库批量移除。**不额外加外描边强调**（用户明确要求，勿改）。
- 启动方式菜单：原自绘 `MetaLaunchModeMenu.vue`（role=menu / 键盘导航 / fixed 摆位）**已删除**，改成「游戏启动」右侧的 `n-dropdown`（`trigger="click"`、当前方式前打勾）；分体按钮的两段圆角用内联 style 拼接，分隔线仍用 `--color-divider-on-brand`。
- `n-message` 的 loading 仍是「不能原地改成结果」：先 `dismissToast(id)` 再另弹结果（`utils/toast.ts` 里有说明）。

### 0.4 校验口径

- `npm run check:build` 已按新体系重写：不再断言 `btn-*` / `input-theme` / `chip-*` / `todo-checkbox`（这些类已不存在），改为断言布局类、色值 token、图标、单侧边框、**naive 主题桥接的源码存在性**（`naiveTheme.ts` 用 `getComputedStyle`、`App.vue` 用 `n-config-provider`、toast 走 `../styles/naive`）。产物 JS 里的 `⚠️` 仍是 `@wailsio/runtime` 的控制台提示，属已知噪音。
- 尚未做（按用户要求由用户手动测）：浏览器/实机视觉走查、深浅主题下的 naive 控件观感、Wails webview 里的弹层层级。

### 0.5 2026-11-24（第二次重排）：/works 动作栏 + 下划线 tab + 元数据可编辑

> §3.3 / §3.7 / §4 里关于 `/works` 排版的描述（左信息栏 + 右主区）**已被本节取代**，以本节为准。

- **动作栏**（用户给的第一张 Steam 截图）：hero 下方新增 `components/meta/MetaWorkTopBar.vue` —— 左「入库 / 游戏启动 + 启动方式下拉」、中三组「小图标 + 灰标签 + 亮值」（库状态 / 入库时间 / 启动方式）、右两颗方形图标按钮（重新抓取 / 打开原始页面）。状态值仍来自 `metaStore`，组件只转发事件。
- **下划线 tab**（第二张截图）：`n-tabs type="line"`，五个页面 —— 作品信息（简介 / Outline / 元数据表 + 编号 + 入库时间 + 原始页面）· 游戏统计（评分/评价数/发售日/入库 + `MetaSteamStats`；DLsite 没有在线人数与公告）· 启动配置（启动方式/启动地址或程序位置/文件状态/运行状态 + Steam 源切换方式 + 更换游戏程序）· 编辑（元数据表单 + 标签）· 评价（评分/评价数 + 商店页与 Steam 社区评价入口）。tab 外观只改 `naiveTheme.ts` 的 Tabs 段（未选中 tertiary、选中 textPrimary、2px 品牌色下划线），整条底线写在 `WorksPage.vue` 的 scoped 样式。
- **hero 小字**（`MetaWorkHero` 的 `kickerParts`）= 源 · 社团/开发商 · 发售日 · 作品形式 · 评分（哪项没有就少一个分隔点）。
- **左信息栏 `MetaWorkSidebar.vue` 已删除**（用户要求「删去 work 界面的左侧信息」）：作品信息进 hero 与「作品信息」tab，入库状态进动作栏，「游戏」进「启动配置」，原始链接进动作栏图标与各 tab 的按钮。
- **元数据可编辑**（用户：「编辑界面中 所有数据库数据都可以改变」「日期选择等信息可以用组件中的日历等组件修改」）：`components/meta/MetaWorkEditorForm.vue` 覆盖 name / maker / makerId / series / seriesId / workType / ageRating / fileFormats / fileSize / releaseDate / price / currency / priceUntil / rating / ratingCount / summary / sourceUrl / coverUrl / outline。日期用 `n-date-picker`（`value-format="yyyy-MM-dd"`；原值不是标准日期时只要不动它就不提交该字段）；文件格式用 `n-dynamic-tags`；Outline 用「每行 `标签 | 值`」的 textarea。表单是草稿 + 「保存 / 还原」，保存只提交改过的字段。
- ⚠️ **踩坑（2026-11-24 实机复现）**：`n-date-picker` 的 `v-model:formatted-value` 传**空串 `''`** 时，naive 在 setup 里 `format(new Date(NaN))` 抛 `RangeError: Invalid time value`，编辑 pane 直接白屏且后续 update 反复报错，表现为「点编辑 tab 后下面空白、路由也点不动」（用户报的「卡死」）。修法：**日期字段空值用 `null`**；`toDateValue()` 还必须校验成真实日期（`2020年99月99日` 这种脏值同样会抛）。同一次踩坑：`MetaTagEditor.vue` 漏 `import { NButton }`，标签「保存」按钮渲染成未知元素（`Failed to resolve component: n-button`）。
- 校验：`go build ./...` / `go test ./internal/service/...` / `npm run build` / `npm run check:build` 全部通过；**2026-11-24 已用 `WAILS_MCP=1 wails3 dev`（MCP http://127.0.0.1:9099/mcp）在真实应用里实测**：编辑 tab 渲染、日期日历、改名→保存→落库（series 列与 raw_json 同步）、清空→保存（raw_json 删键）、还原、标签保存按钮、tab 来回切与侧栏路由均正常。
- 注意：`wails3 dev` 的文件监听覆盖整个项目目录 —— 往项目里写任何文件（包括 `.task/*.mjs` 这种测试脚本）都会触发一次「重新生成 bindings + go build + 重启应用」。写临时脚本请放 `%TEMP%` 下，否则测试过程中应用会被反复重启。
- **后端 `LibraryService.UpdateMeta(UpdateMetaRequest{workId, source, fields map[string]any})`**：打补丁语义 —— 只改传来的键，空串/空数组 = 删掉这一项（raw_json 删键 + 公共列一起清空）；编号不可改、名称不可清空、只对**已入库**作品生效。写入复用 `UpsertLibraryWork`（其 SET 不含 cover_* / game_path / launch_mode），所以改元数据不会冲掉本地封面与游戏位置。Fetch 只读不写库 ⇒ 动作栏的「重新抓取」不会覆盖用户改过的元数据。

### 0.6 2026-11-25：四色主题 + toast 顶栏 + 测试页方形卡片

- **四色主题（用户口径：「以后主题包括现在，通过四色决定主题，其他次要颜色通过全局计算得出」）**：
  - `styles/themePresets.ts` = **主题清单 + 唯一色值来源**。每套主题 `{ light: 四色, dark: 四色 }`：`bgPrimary` 主背景 / `bgSecondary` 次一级 / `bgTertiary` 再次一级 / `brand` 品牌色（最次一级）。默认 `nightfall` 夜岚 = `#404258 / #474E68 / #50577A / #6B728E`；`ember` 暖砂保留迁移前的暖米色 + 青绿，作为回退选项。加主题只需往数组里追加一项。
  - `styles/theme.ts` = 四色 → 全站 `--color-*` 的推导引擎。文字按对比度目标自动调亮/调暗（深色：主 9:1 / 次级 6.5:1 / 三级 4.8:1 / muted 4.5:1；浅色：12 / 8 / 5.2 / 4.5），控件描边按 3:1，品牌作文字时用 `--color-brand-text` 调到 4.5:1 —— **`--color-primary` 仍是四色原值（填充），`--color-brand-text` 是对比安全版（文字/图标/焦点）**。`applyThemeVars()` 把所有变量写到 `<html>` 的 inline style，naive 再从 `getComputedStyle` 读走。
  - `global.css` 不再手写主题色（只保留动效/排版/滚动条/启动兜底 + 图片上的白字/遮罩）；`[data-theme]` 只负责 `color-scheme`。
  - `stores/Theme.ts`：新增 `themeId / activePreset / setThemeId`，`theme-id` 存 localStorage；`App.vue` 的 watch 同时盯 `effectiveTheme + themeId`，再 `syncNaiveTheme`。
  - 设置页「外观 → 主题配色」= `ThemePresetSelector`，选项直接来自主题清单（未来主题的扩展口）。`ThemeModeSelector`（浅色/深色/系统）保留。
  - `naiveTheme.ts` 的 token 增加 `brandText`：文字按钮、下拉选中项、分页当前页、tab 下划线、输入框聚焦描边、Spin 等改用 `--color-brand-text`；实心按钮/勾选/进度条仍用 `--color-primary`。
- **toast 顶端进度条**（用户要求）：`utils/toast.ts` 每条消息顶端渲染 naive `NProgress`（line、3px）—— 结果类按停留时长倒计时消退（CSS `@keyframes app-toast-countdown` + 内联 `--toast-duration`），loading 类走 `processing` 条纹且不消失；新增 `toastWarning`（3.2s）。进度条用负外边距贴到提示条最左端（`.n-message` 的 `overflow:hidden` 负责裁圆角）。
- **MetaMessage 全部改 toast**（用户要求）：`components/meta/MetaMessage.vue` **已删除**；`metaStore` 在赋值处直接弹（`fail/note/done` 辅助函数，错误标题带上下文）覆盖 `error/status/tagError/gameWarning/searchNotice`，`scanStore.error/status`、`libraryStore.error` 同样处理；页面里的提示条与重复的显式 toast 一并移除。翻页的「正在加载第 N 页…」改成 `LibraryPage` 里的常驻 loading toast（开始时弹、`pageLoading` 变 false 时 dismiss）。
- **全站方形卡片 + 边框间隔（2026-11-25 用户要求）**：用户先要求「卡片容器都改为方形 且紧凑排布 卡片容器之间只使用border间隔」，再补充「游戏库的cover之间改为方形 但间隔不改变」「todo item之间有间隔」「无间隔的卡片之间border大小相对大」。落地：
  - `naiveTheme.ts` 的 `Card.borderRadius` = `0`（所有 n-card 直角）；`card` shortcut / `filter-bar` / `TodoStats` / `TodoItem` / `lib-card` / 候选面板 / `MetaWorkTopBar` / `MetaWorkHero` 与 hero 封面全部直角；批量浮条也改成方形；弹窗/浮层（modal/popover/dropdown/toast）仍保留圆角，和内容卡片区分。
  - `global.css` 新增 `.card-stack`：`gap: 0` + 后卡 `border-top-width: 0 !important`（只留前卡下边框一条线）+ 内部共享线 **2px + `--color-border`**（比 1px 外框更重，满足用户「加粗紧凑卡片之间的 border」）。**不用负 margin 叠边框**（DPR 1.25 的 WebView 会叠成粗线）。已用于设置页、`/works` 各 tab 的卡片堆栈、`/testing` 左栏。
  - TODO 行恢复 `gap-2`（8px 间隔，不做贴边堆叠）；游戏库 cover 网格保留 `gap-3`（用户明确「间隔不改变」）。
  - 测试页仍用 n-split 左右分栏（默认 62%，可拖 40%–78%）：左栏请求/命中/候选，右栏原始 JSON；`MetaSection` 的 `square` prop 已删除（全局直角），卡片之间的分隔由 `.card-stack` 统一。
- 校验：`npm run build` / `npm run check:build` 全绿；用 vite dev（9245）核对深/浅两套主题、设置页主题配色、全站方形卡片与 `.card-stack` 单线分隔、toast 顶部进度条、n-split 左右分栏。

### 0.8 2026-11-25（第四轮）：toast 收敛 / scroll-fade / 封面动效 / hero 模糊遮罩

> §0.6 的「MetaMessage 全部改 toast」仍然成立，但**弹出频率**在 2026-11-25 又被用户收紧了，以本节为准。
>
> §0.7 的「tab 条遮罩」只罩了 tab 条、pane 保持干净；本节按用户后续要求把 tab 内容卡片也纳入遮罩，并给 tab 条加 12px 内边距；随后又收敛为**全页唯一一张遮罩**，**该 bullet 以本节为准**。

- **toast 只在四种情况弹**（用户原话：「只有在删除成功后 编辑保存后 出现错误后 刷新完成后 才弹出土司提醒」）：
  - 删除成功：TODO 单条/批量、游戏库批量移除（`LibraryPage` / `TodoList`）。
  - 编辑保存：`metaStore.saveMeta`（元数据）与 `saveTags`（标签整份保存）。
  - 出错：各 store 的 `fail` / `toastError`（搜索、扫描、入库、启动、删除部分失败等）保持弹。
  - 刷新完成：`libraryStore.refresh()`（游戏库顶栏刷新）与 `/works` 的「重新抓取元数据」。
  - **不再弹**：`metaStore.note/done`（按编号命中、从缓存读取、入库成功、启动成功、换程序、切启动方式）、搜索信息、扫描跳过提示、入库汇总成功、编辑页「没有改动」（保存按钮本来就是禁用态、直接 return）。
  - **唯一例外**：TODO 复制成功（用户 2026-11-25 明确要求「TODO复制成功后添加的土司提醒」）—— `TodoList.handleCopy` 里弹「已复制到剪贴板」，失败仍弹错误。
  - 加载态也回到页面内：游戏库翻页的常驻 loading toast 改成翻页条上方 2px 的 `n-progress`；游戏库删除的 loading toast 去掉（删除按钮自己转圈）。`utils/toast.ts` 接口不变，只是调用点变少。
- **scroll-fade（参考 shadcn 的 scroll-fade）**：新增 `utils/scrollFade.ts` 的 `vScrollFade` 指令（`main.ts` 注册为 `v-scroll-fade`）+ `global.css` 的 `.scroll-fade`。指令在 mount / scroll / ResizeObserver / MutationObserver / 图片 load 时按真实滚动位置写 `--scroll-fade-top` / `--scroll-fade-bottom`，CSS 用 `mask-image` 拼渐变 —— **到顶/到底的那一侧不淡**，不是静态遮罩。已接入：游戏库网格、添加游戏弹窗候选列表、`/works` 页面、TODO 列表卡片、设置页、`/testing` 左右两栏、原始 JSON 的 `pre`（`/works` 曾在「遮罩压内容」方案里移除过，遮罩改回内容下方后已恢复）。注意 mask 会让元素成为 fixed 后代的包含块，所以不要把 fixed 浮层（批量操作条）放进滚动容器。
- **启动方式箭头旋转**：`MetaDownloadButton` 的分体按钮右段用 `v-model:show` 接住 `n-dropdown` 的展开态，`launch-mode-chevron--open` 时 `rotate(180deg)`（并补 `aria-expanded`）。
- **`/works` 模糊遮罩（2026-11-25 最终版：全页只有一张）**：
  - 用户明确「**不模糊 hero 画面**」「**不做玻璃反光效果**」，后来进一步要求「**只使用一张模糊遮罩 … 不使用多张遮罩**」。
  - **最终层序（用户 2026-11-25：「tab 和启动所在卡片没能成功遮罩」）**：hero（不透明图）> 动作栏（65% 底，80% 时那 12% 透光几乎看不出来）与 tab 条 / tab 内容卡片（75% 底）> 遮罩 `.work-page-ambient`（`z-index:-1`、`opacity .6`）> 页面背景。按钮、输入等 UI 控件自身保持不透明；只有这一层遮罩，不再有 `.work-ambient` / `.work-tabs-ambient` / `::before` / `::after` 局部遮罩。
  - 遮罩回到内容下方后，`/works` 的滚动容器重新挂 `v-scroll-fade`（层叠上下文不再妨碍层序）。
  - **遮罩盒子 = hero 下方、同宽同高**（2026-11-25 用户要求「只在 hero 图下方」「等宽度」「等大小」「和 hero 图大小一致」）：`WorksPage` 用 `ResizeObserver` 量 hero 的 bottom / left / right / height，写成 `--mask-top/left/right/height`。滚动时只写 `transform: translate3d(0, -scrollTop, 0)` —— **滚轮向下 N，模糊图向上 N；向上同理，绝对位置不变**（用户原话「遮罩随滚轮平移 …（绝对位置不变）」）。不做倍率视差，`--work-scroll` 已删除。
  - 布局：`.work-glass` gap 8px、滚动容器 gap-2、`.n-tab-pane` 上内边距 4px（**紧凑排布**）；tab 条左右各 12px（**第一个 tab 标签有左间隔**）。
- **游戏库封面动效（仿 Steam 库卡片）**：
  - 点击：从指针落点扩散水纹（`.lib-ripple`，直径 = 卡片对角线 × 2，`animationend` + 800ms 兜底移除；命令式 `document.createElement`，不进出响应式状态）。
  - hover：卡片上浮 2px + 阴影、封面 `scale(1.045)`、底部名称浮层 `.lib-card-caption` 淡入（图片专用白字 token + 黑色渐变，不用页面底色）。
  - 批量模式（**「进入批量操作 卡片样式变化 提示」**）：网格加 `.lib-batch-mode` → 未选中的卡片 `opacity .6 + saturate(.55)`、hover 恢复、不显示名称浮层；每张卡左上角出现方形勾选框（选中填品牌色）；顶栏「已入库 N 款」换成「点击封面选择 · 已选 N 款」（同一行替换，不挤压布局）。
- **横向滚动条（2026-11-25 用户反馈）**：`/works` 的滚动容器加 `overflow-x-hidden` —— ambient 的负 inset + `scale` 把它撑出 27px 横向溢出，`.scrollbar-theme` 就会显示 8px 横条，显式裁掉。
- **滚动性能（同日用户反馈「上下滑动有一定卡顿」，确认是渲染原因）**：rAF 帧间隔 A/B 实测定位到两类开销 —— ① 每个 n-card / tab 条 / 动作栏的 `backdrop-filter`（滚动时逐帧重采样背景）；② 大面积高半径 `filter: blur()`（64/42/34px）。先去掉全部 backdrop-filter、把 blur 压到 22/18/18px，掉帧从 10/138 降到 0–1/160；随后按用户要求改成**后端预模糊**（见下一条），CSS 里已无任何 `filter: blur`，实测 0/173 帧超 16.7ms、p90 7ms。
- **预模糊 hero（2026-11-25 用户要求「只用一张模糊hero图片对所有内容进行遮罩 … 缓存到 cover 文件夹后直接使用模糊后的图片 避免渲染算力浪费」）**：新增 `LibraryService.HeroBlur(workID, source, rawURL)`（[internal/service/hero_blur.go](internal/service/hero_blur.go)）—— 取原图（data URL 或 http）→ 缩到 480px 宽 → 两遍可分离盒式模糊（不引第三方库）→ **上下镜像**（用户 2026-11-25：「模糊后的图片要作上下镜像处理」）→ JPEG 80 写入 `covers/hero_blur_v3_<source>_<id>_<urlhash>.jpg`，返回 data URL；写新文件后 `pruneHeroBlurCache` 会清掉同作品旧版本，**盘上始终只留一张**。缩图按图像真实类型取样（JPEG→YCbCr、RGBA/NRGBA/Gray 直读），避开每像素 `color.NRGBAModel.Convert` —— 1920→480 的缩图从几百 ms 降到几 ms（冷启动实测 540ms → 227ms）。前端 `metaStore.heroBlurred` 在 backgroundUrl 变化时**去抖 30ms** 再取；Steam 会先远程预览、后本地 hero，所以 `heroImageLoading` 期间先跳过、等本机 hero 落定再请求（避免先算一张远程图）。只给唯一遮罩层用（`--work-hero-image`），CSS 里不做 `filter: blur`；作品被移除时 `Remove` 清 v3/v2/旧版缓存。webp 源解码失败 → 返回错误、前端退化为不做遮罩（不崩页面）。
- 校验：`npm run build` / `npm run check:build` 全绿（断言新增 `.scroll-fade`、`.lib-ripple`、`.lib-card-caption`、`.work-page-ambient` 等）；`WAILS_MCP=1 wails3 dev -port 9245` 实机用 MCP 核对：滚动渐隐变量、批量卡片样式与提示、点击水纹、刷新完成 toast、`/works` 各层遮罩的 computed filter/mask、箭头 `rotate(180deg)`。



### 0.7 2026-11-25（第三次）：侧栏改 `n-menu` + `/works` 遮罩（tab / 视差）+ 修掉两份 MainLayout

- **侧边栏改用 naive 的 `n-menu`**（用户原话：「sideBar改用naive ui样式和逻辑 menu」）：
  - `components/Sidebar.vue` 只剩外壳：`aside.sidebar`（固定宽度过渡 72 ↔ 250）+ `sidebar-header`（logo + 点击折叠）+ 上下两个 `<nav>` 容器；导航项全部由 `n-menu` 渲染。
  - **菜单项 = 路由路径当 key**：`:value="route.path"`，`@update:value` → `router.push`（同路径不重复 push）；选中态由 naive 画（`::before` 圆角块），不再手写 `nav-item-active` / 左侧 2px 竖条。
  - **折叠态交给 naive**：`:collapsed` + `collapsed-width`（= 侧栏 72 − 容器 16 = **56**，naive 用 `width/2 − iconSize/2` 居中图标）+ `icon-size = collapsed-icon-size = 20`；折叠时 naive 自动把 label `opacity: 0` 并在 hover 弹 `n-tooltip`（实测 tooltip 文案 = 菜单名）。
  - 图标仍是 UnoCSS 的 `i-lucide-*`，用 option 的 `icon: () => h('span', { class: item.icon })` 传渲染函数；图标盒子（宽高/字号/右外边距）由 naive 的 `.n-menu-item-content__icon` 负责。
  - 键盘可达性：naive 的菜单项只有 `role="menuitem"`、默认不可聚焦，用官方扩展点 `nodeProps` 补 `tabindex: 0` + 回车/空格切页。⚠️ naive 把 `nodeProps` 返回类型写成 `HTMLAttributes & Record<string, string | number | undefined>`，函数类型的 `onKeydown` 与那个 Record 互斥（库的类型缺口），所以源码里有一次 `as unknown as MenuNodeProps` 的显式转换。
  - 外观只在 `naiveTheme.ts` 的 **`Menu`** 段：`itemHeight 40px` / `borderRadius 8px` / 选中 `itemColorActive = accentSoft` + `itemTextColorActive = brandText` / 悬停 `bg-hover` / `color: 'transparent'`（底仍由 aside 的 `--sidebar-bg` 提供）。`uno.config.ts` 里 `nav-item / nav-item-active / nav-icon / nav-text / nav-text-active / nav-badge` **已删除**，只留 `sidebar / sidebar-header / logo-icon / logo-text / sidebar-nav-top / sidebar-nav-bottom`。
  - `stores/Sidebar.ts` 的常量改名（`COLLAPSED_WIDTH = 72` / `EXPANDED_WIDTH = 250`，原来 72 那个叫 `barWidth`、250 叫 `barWidthCollapsed`，名字是反的）。
  - `check-build-css.mjs`：`nav-item/nav-icon` 的断言换成 `sidebar/sidebar-nav-top/sidebar-nav-bottom/logo-text`，并新增「侧栏走 n-menu」「Menu 主题已接桥」两条源码级断言。
- **`/works` tab 区遮罩**（用户原话：「tab也做遮罩」，随后用户自己把这一块细化过）：现在由 `WorksPage.vue` 的 `.work-tabs-wrap` + `.work-tabs-ambient`（同图模糊拷贝、向下淡出）垫在整块 tab 区后面，tab 条 `.n-tabs-nav` 半透明 `bg-card 88%` + `backdrop-filter blur(14px)`、左右 12px 内边距（第一个标签不贴边），pane 里的 `.n-card` 也是 88% 半透明 + `blur(12px)` —— 模糊 hero 从卡片下面透上来。⚠️ 我最初那版给 `.n-tabs-nav` 加 `::after` 的做法已被这一版取代，改这块前先读当前源码。
- **遮罩视差（已取消）**（用户原话：「模糊后的hero遮罩跟随滚轮相对移动」；2026-11-25 后来要求「取消视差」「固定模糊图片位置」，`handleScroll` / `--work-scroll` 已删除）：`WorksPage` 的滚动容器 `@scroll.passive` → `handleScroll()`（rAF 合并）每帧只把 `scrollTop` 写成一个 CSS 变量 `--work-scroll` 到 `.works-page` 上，**不触发 Vue 重渲染、不动布局**；三层遮罩在样式末尾的 `@media (prefers-reduced-motion: no-preference)` 里按不同倍率平移：`.work-page-ambient` **−0.22**（远景，往上）、`.work-ambient` **+0.15 封顶 24px**（净 0.85，比内容慢）、`.work-tabs-ambient` **+0.12 封顶 18px**。`.work-page-ambient` 的 `inset` 底部多留 260px，往上位移时不露硬边。⚠️ 视差块必须放在样式最后：这几层前面各有自己的 `transform` 基础值，同优先级靠顺序覆盖（踩过：`.work-ambient` 的视差被后面的 `scale(1.04)` 盖掉，位移恒为 0）。
- **修掉「两份 MainLayout」**（用户原话：「顺便修改重合mainLayout问题」）：`App.vue` 原来直接 `<MainLayout />`，而 router 的 `/` 也 `component: MainLayout`，于是 DOM 里两份 MainLayout + 两套 Sidebar/TitleBar（完全重合看不出来），页面还被套了两层 `padding-left`。现在 `App.vue` 只放 `<router-view />`；`MainLayout` 的 `router-view` 内边距从 `mainPaddingLeft`（展开时 178 = 250−72，会让内容钻到 250px 侧栏底下）改成 **`sidebarWidth`**（72 ↔ 250 + `p-6` 的 24px 间距），`stores/Sidebar.ts` 的 `mainPaddingLeft` 已删。实测：`aside.sidebar` / `header.titlebar` 各 1 个，展开时内容从侧栏右边 24px 开始。
- 校验：`npm run build` / `npm run check:build` 全绿；`WAILS_MCP=1 wails3 dev`（MCP 9099）实机验证 —— 折叠 72 ↔ 展开 250、折叠 tooltip 文案、点菜单切路由（`#/todo`）、选中态颜色（`accentSoft` 底 + `brandText` 字）、单实例布局（aside/titlebar 各 1）、视差（滚动 400px 时 `.work-page-ambient` 的 `matrix(... 0,-88)`、`.work-ambient` 封顶 24px），以及 `PrintWindow` 抓图看像素效果。

---

## 1. 项目一句话 + 当前状态

Wails v3 (Go) + Vue 3 + Pinia + UnoCSS 的无边框 Windows 桌面应用。功能域：**TODO 管理**（批量删除、优先级/时间筛选、按优先级统计、双主题、系统托盘；2026-11-23 起不再有"完成"功能）、**游戏库**（DLsite / Steam 双源、扫描添加、批量操作、从库里启动游戏——Steam 源默认走 `steam://rungameid/<appid>`，也可切本地 exe）、**作品详情**（`/works`，侧栏无入口）、**设置**、**调试页**（`/testing`）。

### 1.1 技术栈与实测版本

| 层 | 技术 | 版本（实测） |
| --- | --- | --- |
| 桌面框架 | Wails v3 | `go.mod` 声明 `v3.0.0-beta.10`；**本机 CLI 实为 `v3.0.0-beta.20`** |
| 后端 | Go | 1.25.0（go.mod）/ 本机 toolchain 1.26.2 windows/amd64 |
| 数据库 | SQLite | 经 **Wails 内置 `pkg/services/sqlite`**（内部用 modernc.org/sqlite，纯 Go，无 CGO）；`modernc.org/sqlite v1.44.3` 在 go.mod 里是 **indirect** |
| 前端 | Vue / Pinia / vue-router | 3.5.41 / 2.3.1 / 4.6.4 |
| 构建 | Vite / vue-tsc | vite **`~8.2.2`（必须锁死，见 §7.1）** / typescript `~5.4.0` |
| 样式 | UnoCSS | `^66.8.1`（`presetUno` + `presetAttributify`(prefix `un-`) + `presetIcons`） |
| 组件库 | naive-ui | **2.45.3**（2026-11-24 引入，全站控件；主题桥接见 §0.1） |
| 轻提示 | naive-ui `n-message` | 2026-11-24 起（vue-sonner 已移除；入口仍是 `utils/toast.ts`，见 §0.1） |
| 图标 | lucide via `@iconify/json` | 构建期内联 data URI，**运行时零请求 / 离线可用** |
| 运行时 | Node / npm | v24.15.0 / 11.12.1 |

### 1.2 仓库与工作区状态

- 远程：`origin = https://github.com/WeedyMornbrize3/WeedyBox.git`，分支 `main`。
- 最新提交：`b621c2c 托盘点击改为「只唤回不隐藏」；移除关闭行为里的最小化选项`。
- **工作区有大量未提交改动**（2026-11-23：TODO 完成功能删除、游戏库批量操作 UI、`/works` Steam 式排版、DLsite 预览图解析、Steam hero 背景图、Steam 启动方式等），接手前先 `git status` 看实际状态，不要按旧提交理解代码。
- **`frontend/bindings/` 与 `go.sum` 都被 `.gitignore` 忽略且不在版本控制内**（`.gitignore:18` 是 `go.sum`）。
  → 换机器 / 重新 clone 后**必须先**：`wails3 generate bindings -clean=true -ts -i`，否则前端找不到 `HideWindow` 等新方法。
  该命令恒输出 1 条无害警告：`package WeedyBox/internal/database: function types are not supported by encoding/json`。
  实测输出（2026-11-23，本次改动后）：`Processed: 282 Packages, 8 Services, 68 Methods, 0 Enums, 24 Models, 1 Event`（新增 `LibraryService.SteamHeroImage`）。
- 本机环境坑：未装 Docker；`gh` CLI 已装但**未登录**；`task` CLI **未安装**（用 `wails3 task <name>` 替代）；系统代理 `127.0.0.1:7890` 会偶发拦截 TLS（`SELF_SIGNED_CERT_IN_CHAIN`），重试即可。
- 本机 `devicePixelRatio = 1.25`：**源码里的 `1px`/`2px` 是正确写法**，DevTools 显示 `0.8px`/`1.6px` 只是缩放后的计算值。
- 浏览器里跑 `npm run dev` **没有 Wails 后端**：所有写操作必然失败，只能验证「是否发请求 / 是否回滚 / UI 状态」，**不能验证落库**。

---

## 2. 目录结构（只列实际存在且重要的）

```
main.go                          应用入口：服务注册 / 窗口 / 系统托盘
internal/model/todo.go           Todo · CreateTodoDTO · UpdateTodoDTO · TodoStats
internal/database/sql_service.go SQLiteService：Execute / Query / QueryRow / GetLastInsertID / Transaction + initAllTables（**只建 todos**，游戏库表惰性创建）
internal/database/library_repo.go 游戏库持久层：ensureLibraryTables（惰性建表 + 给旧库补 game_path / launch_mode 列）/ Find / List / Upsert / Delete / RecordCoverFile / GetGamePath / SetGamePath / GetLaunchMode / SetLaunchMode
internal/database/library_repo_test.go 游戏库仓储测试（惰性建表、先读先删、并发首用、来源隔离、封面指针、game_path 与 launch_mode 同表及旧库迁移）
internal/service/todo_service.go TodoService：全部 CRUD + 批量删除 + 统计
internal/service/library_service.go LibraryService：**缓存优先**取作品 / 列游戏库 / 删除 / 本地封面转 data URL
internal/service/dialog_service.go DialogService：系统「选择文件夹」/「选择游戏程序（.exe）」对话框（Wails v3 `app.Dialog.OpenFile()`）
internal/service/scan/            本地扫描（package `scan`，独立子包；**已注册为 Wails 服务 `ScanService`**）
internal/service/scan/scan.go     ScanService：一级子文件夹内的 .exe（通用）/ 一级子文件夹名+路径（特定）/ 从文件夹挑主程序 PickGameExecutable
internal/service/scan/steam*.go   Steam 游戏目录：steam.go 多库汇总 + steam_windows.go 注册表 / steam_other.go 非 Windows 空实现（build tag）
internal/service/scan/*_test.go   扫描边界（只读一层/不递归/坏输入/取消）、vdf 解析、多库汇总去重
internal/service/game/           游戏启动（package `game`，独立子包；**已注册为 Wails 服务 `GameService`**）
internal/service/game/game_service.go  GameService：Status（位置/exe 是否存在/是否运行/生效启动方式 + 运行缓存）/ SetPath（校验并记录 exe 位置）/ SetLaunchMode（steam ↔ exe）/ Launch（steam 方式跑 steam://rungameid/<appid>，exe 方式起独立进程）
internal/service/game/launch_windows.go / launch_other.go  起进程的平台实现（DETACHED_PROCESS，工作目录=exe 目录，返回 PID）
internal/service/game/process_windows.go / process_other.go  运行状态检测（Toolhelp 快照 + QueryFullProcessImageName 比对完整路径；非 Windows 恒 false）
internal/service/getMeta/getFromDlsite.go  DLsite 元数据解析（MetaService，package getmeta）
internal/service/getMeta/meta_service.go   Wails 包装层 MetaApiService（搜索/抓取/封面下载：入库图落 covers/（优先 webp）、搜索缩略图落 cache/，下载后记账到库）
internal/service/getMeta/cover_store.go    封面存储：covers/ 目录、preImg→covers 启动迁移（目录+库路径）、优先探测同名 .webp 下载
internal/service/getMeta/testdata/        两个真实作品页 fixture（离线回归用）
internal/service/window_service.go WindowService：最小化 / 最大化 / 关闭 / 隐藏 / 显示 / 可见性
frontend/src/main.ts             Pinia + router + store 的 restoreState()
frontend/src/router/index.ts     hash 路由，MainLayout 下挂 7 个子页（/ · /library · /works · /todo · /testing · /settings + /test 重定向）
frontend/src/stores/             todoStore.ts · metaStore.ts · libraryStore.ts · scanStore.ts · autoHeightStore.ts · Sidebar.ts · Theme.ts · App.ts
frontend/src/views/WorksPage.vue 作品库页面（**侧栏无入口**，只从游戏库点封面进入；见 §3.3）
frontend/src/services/todoService.ts  Wails 绑定封装：todoApi（含 ensure() 兜底 null）
frontend/src/services/metaService.ts  Wails 绑定封装：metaApi（**永远联网**：搜索/抓取 + 缩略图缓存 CacheCover）
frontend/src/services/libraryService.ts Wails 绑定封装：libraryApi（fetch **只读**缓存优先 · **storeIn 才入库** · list/remove/coverDataUrl · listTags/addTag/removeTag 自定义标签）
frontend/src/services/scanService.ts  Wails 绑定封装：scanApi（steamFolders / executables / subfolders / pickDirectory）
frontend/src/services/gameService.ts  Wails 绑定封装：gameApi（setPath / launch / pickExecutable，游戏启动）
frontend/src/utils/metaFormat.ts 元数据展示纯函数：toMessage / formatBytes / fileUrl / buildSummaryRows（+ MetaRow 类型）
frontend/src/utils/clipboard.ts 复制统一入口 copyText()：Clipboard API → execCommand 兜底，失败抛错（调用方提示）
frontend/src/utils/toast.ts     **全站一次性反馈的唯一入口**（toastSuccess / toastWarning / toastError / toastInfo / toastLoading / dismissToast；底层 naive message，顶端带 n-progress 进度条）：文案策略、停留时长、描述截断都在这里，业务代码不直接 import naive 的 message
frontend/src/components/AppToaster.vue **已删除**（2026-11-24 由 naive 的 n-message 取代，见 §0.1）
frontend/src/styles/themePresets.ts **四色主题清单（唯一色值来源）**：每套主题 = 主背景 / 次一级 / 再次一级 / 品牌色，浅色+深色各一组；加主题只改这里
frontend/src/styles/theme.ts        四色 → 全站 `--color-*` 的推导引擎（对比度目标 + `applyThemeVars` 写入 `<html>`）
frontend/src/styles/naiveTheme.ts    **token → naive themeOverrides** 的唯一桥接（控件外观只改这里）
frontend/src/styles/naive.ts        naiveConfig（主题/中文文案）+ createDiscreteApi 的 message
frontend/src/components/meta/    作品库与调试页共用的高粒度积木（MetaSection/搜索行/封面/键值表/明细/标签…，见 §4.6）· MetaTagEditor.vue（作品详情「标签」：抓取标签与自定义标签合并编辑，hover 出 × 删除 + 末尾「+」原地输入 + 「保存」整份写库）· MetaWorkTopBar.vue（/works 动作栏，见 §0.5）· MetaWorkEditorForm.vue（/works「编辑」页的元数据表单：日期用 n-date-picker，格式用 n-dynamic-tags；**MetaWorkSidebar.vue 已于 2026-11-24 删除**）
frontend/src/components/library/ ScanPanel.vue（游戏库「扫描添加」面板：来源/扫描方式/自动搜索进度/结果行勾选/全部入库·入库所选）
frontend/src/components/todo/    TodoList · TodoItem · TodoInput · TodoStats · FilterBar · FilterChipGroup · TodoPrioritySelector
frontend/src/components/settings/ CloseActionSelect · ThemeModeSelector · ThemeToggle · LanguageSelect(占位) · SettingsSection/Item/Page · VersionInfo · CacheCleanup（设置页「清理缓存」：进页面扫 cache/ 占用 + 一键清空） · TagLimitInput（标签显示数量 4–64）
frontend/src/components/         Sidebar.vue · TitleBar.vue
frontend/src/styles/global.css    **非颜色 token**（动效/排版/滚动条/启动兜底）+ toast 两行与进度条样式；主题色已移到 themePresets.ts / theme.ts
frontend/uno.config.ts           变量→类名映射 rules + shortcuts + theme.colors
frontend/scripts/                check-unocss-tokens.mjs · check-build-css.mjs（两个回归闸门）
frontend/bindings/               ⚠️ 自动生成、被 gitignore、勿手改
doc/                             README/backend/frontend/styles/conventions + need-fix.md/need-change.md（后者两个已清空为 0 字节但保留在版本控制）
```

---

## 3. 后端 API 契约（Go → TS）

服务注册在 `main.go:54-60`：`application.NewService(windowService / todoService / dbService / metaService / libraryService)`。

### 3.1 TODO（`TodoService`，全部方法内建 3s `context.WithTimeout`）

| 方法 | 签名 | 说明 |
| --- | --- | --- |
| `Create` | `(ctx, dto CreateTodoDTO) (*Todo, error)` | INSERT 后 `GetLastInsertID` 再 `GetByID` 回读 |
| `GetByID` | `(ctx, id int64) (*Todo, error)` | |
| `GetAll` | `(ctx) ([]Todo, error)` | `ORDER BY completed ASC, priority DESC, created_at DESC` |
| `GetByStatus` | `(ctx, completed bool) ([]Todo, error)` | |
| `Update` | `(ctx, id int64, dto UpdateTodoDTO) (*Todo, error)` | 动态拼 SET；空 updates 直接回读；自动写 `updated_at` |
| `ToggleComplete` | `(ctx, id int64) (*Todo, error)` | 单条 SQL：`CASE WHEN completed = 1 THEN 0 ELSE 1 END` |
| `Delete` | `(ctx, id int64) error` | |
| `DeleteByIDs` | `(ctx, ids []int64) error` | 动态拼 `IN (?,?,…)`，**一次 SQL**；空数组直接返回 |
| `DeleteAll` | `(ctx) error` | |
| `GetStats` | `(ctx) (TodoStats, error)` | `COUNT(*)` + 两个 `SUM(CASE…)`。**`:315` 有一行 `// bug need-fix` 注释但逻辑本身正确** |
| `GetPriorityStats` | `(ctx) (map[int]int64, error)` | 前端当前未使用 |

错误处理统一风格：`errors.Is(err, context.Canceled)` → 「…被取消」、`DeadlineExceeded` → 「…超时」、否则「…失败」。

### 3.2 窗口（`WindowService`，7 个方法，均已生成绑定）

`MinimizeWindow()` · `ToggleMaximizeWindow()` · `CloseWindow()` · `IsWindowMaximised() bool` · `HideWindow()` · `ShowWindow()` · `IsWindowVisible() bool`

> `HideWindow` 与 `MinimizeWindow` 的区别：最小化**仍留任务栏**，隐藏**完全不可见**、只能靠托盘唤回 —— 所以托盘必须在 `main.go` 里创建。

### 3.3 接口测试服务（`internal/service/getMeta/meta_service.go`，`MetaApiService`）

⚠️ **不做数据加工**：返回给前端的 `work` 是 `DLsiteWork` 的 **json 编码**（走 `toRawMap` 一个 `Marshal`+`Unmarshal` 往返，字段名与 json tag 完全一致），前端直接展示原始 JSON。这是刻意的——测试页的用途就是对着 DLsite 页面核对解析结果。

已在 `main.go` 注册：`metaService := &getmeta.MetaApiService{DB: dbService}`（持有 DB 是为了下载封面后记账进游戏库），`applicationServices` 加入后**服务总数 8、方法 67**（2026-11-23 的生成输出：282 Packages, 8 Services, 67 Methods, 24 Models）。结构体直接字面量构造（不走 `NewMetaApiService()`），因此内部有 `ensure()` 惰性兜底 `meta` 字段。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `FetchWork` | `(ctx, req WorkRequest) (any, error)` | `workId` 优先，否则拿 `keyword`；关键词里含 RJ/RE/VJ 编号则按编号精确抓，否则模糊搜索后抓**第一个**候选。返回 `map[string]any` |
| `SearchWorks` | `(ctx, keyword string, page int) (*searchResponse, error)` | 关键词含编号 → `kind="id"` 且带回 `work` 详情 + 列表候选；否则 `kind="keyword"`，返回**第 page 页**候选（不自动抓详情）。DLsite 每页固定 30 条（`per_page` 传 60/120 实测也只回 30），所以分页靠 `.../per_page/30/page/N/` 逐页取；响应里带 **`total`（页面 `data-search-result-count`，精确总数）** 与 `pageCount`（优先由 total 算出，取不到才退回分页块最大页码）—— **结果不再受 30 条限制** |
| `DownloadCover` | `(ctx, workID, coverURL string) (map[string]any, error)` | 下载封面存入相对路径 `covers/`，返回 `{id, dir, path, absPath, bytes, suffix, coverUrl}`。**优先探测同名 `.webp` 档位**（DLsite 通常有，体积更小；不引图像编码库，探测失败就回落到原图），重名加 `-2/-3` 后缀，**不覆盖已有文件**。日常由 `LibraryService.Store`（入库）和调试页调用 |
| `CacheCover` | `(ctx, workID, coverURL string) (string, error)` | **搜索缩略图**：缓存到相对路径 `cache/`（不存在就建），返回 `data:<mime>;base64,…`；已缓存过就不再联网。固定文件名 `<编号>_cover.<ext>`（允许覆盖，缓存语义） |
| `CoversDir` / `CacheDir` | `() string` | `covers/` / `cache/` 的绝对路径（后者顺带创建目录） |
| `ClearCacheDir` | `() error`（包级函数，`getmeta.ClearCacheDir`） | 清空 `cache/` 下的所有文件（目录保留，不存在算成功）。`main.go` 里挂在 `app.OnShutdown`：**每次退出软件前清掉搜索缩略图** |
| `CacheInfo` | `() (map[string]any, error)` | 扫描 `cache/`（`filepath.WalkDir`），返回 `{dir, files, bytes}`。**不联网、不建目录**：目录不存在就是 0。设置页进入时调一次，把占用显示在「清理缓存」按钮左边 |
| `ClearCache` | `() (map[string]any, error)` | 手动清空 `cache/`（复用 `ClearCacheDir` 的删除逻辑），返回 `{files, bytes, rest}`（删掉的文件数/字节数、删完剩余文件数）。设置页「清理缓存」按钮调它，成功后前端重新 `CacheInfo` 以真实占用为准 |
| `SearchSteamWorks` | `(ctx, term string) (map[string]any, error)` | 按名字搜 Steam 商店（`storesearch`，**最多 10 条、无分页**），返回 `{query, total, items:[{id,name,cover,price,type}]}`。游戏库「手动搜索」切到 Steam 来源时用 |
| `MatchSteamAppByFolder` | `(ctx, folderName string) (map[string]any, error)` | 拿**文件夹名**猜 appid：清洗（去括号注释/结尾点分版本号）→ 商店搜索（长词按 0.75/0.5/0.35 逐级截短重试）→ 归一化后「完全一致 > 公共前缀占比」打分（阈值 0.75，原声/试玩/工具类降权）。返回 `{matched, appId, name, cover}`；`matched=false` 是"猜不到"，**不是错误** |
| `FetchSteamStats` | `(ctx, appID string) (map[string]any, error)` | `/works` 的「Steam 动态」：GetItems 的 `reviews.summary_filtered` + `GetNumberOfCurrentPlayers` + `ISteamNews`（最近 3 条）。三项各自独立，全挂才报错 |

`WorkRequest{keyword, workId string}` 是前端入参——Wails 绑定不擅长可选参数，所以两个字段都给、服务端自己选。

**图片落地位置**：`const coversDir = "covers"` 是**相对路径**，实际落点取决于进程工作目录（开发态从仓库根目录跑 → `WeedyBox/covers/`）。旧版本叫 `preImg/`，`MetaApiService.ServiceStartup` 会调 `MigrateLegacyCoverDir` 自动改名并把库里 `cover_*` / `library_cover_files` 的旧路径一起替换（幂等）。返回值同时给 `path`（相对，展示用）和 `absPath`（绝对，对照用）。**格式策略**：不引图像编码库；下载时给 `jpg/jpeg/png` URL 拼出同名 `.webp` 候选并优先探测，拿到 webp 就用，服务器没有/不是 webp 就静默回落到原图（原样保存），后缀始终按**真实内容**判定（RIFF/WEBP、JPEG SOI、PNG 签名…）。

**缩略图缓存目录 `cache/`**（`const cacheDir = "cache"`，也是相对路径）与 `covers/` 分开：`covers/` 是**入库后的正式封面**（记账、不覆盖、不能丢）；`cache/` 只是搜索列表里瞄一眼用的临时图，可在退出时整体清掉（`getmeta.ClearCacheDir`），`.gitignore` 已忽略。**搜索结果每页 10 个，只给当前页下预览图**；DLsite 源数据每面 30 条，因此第 1~3 页用第 1 面、4~6 页用第 2 面……**用到哪面才取哪面**，翻到组末页（3/6/9…）时预取下一面（见 §4.3 `loadSourcePage` / `setCandidatePage`）。

**前端页面（两个）**：共用 `stores/metaStore.ts`（状态与请求）与 `components/meta/*`（积木），只是排布不同：

| 路由 | 视图 | 定位与排布 |
| --- | --- | --- |
| `/testing`（name `Testing`；旧路径 `/test` 重定向） | `views/test.vue`（薄壳，只渲染 `MetaApiTestCard`） | **调试页**：单列四卡 —— 请求 / 精确命中（含封面预览与「入库」）/ 模糊搜索候选 / **原始 JSON（可复制，只在这里出现）**。抓取入口在这一页；按钮与作品库同语义（`meta.storeIn()`，会写库） |
| `/works`（name `Works`；**侧栏无入口**） | `views/WorksPage.vue` | **作品库**：2026-11-23 按 Steam 商店页排版重做 —— 顶部 `MetaWorkHero`（大背景图 + 来源/发售日/类型 + 作品名 + 标签 chips + 右侧 2:3 竖版封面，封面优先用游戏库卡片那张 cover），下面左右两栏：左 `MetaWorkSidebar`（作品信息 / 游戏 / 快速链接，228px）、右主区（`MetaWorkActions` 动作条 + 缓存 badge + 重新抓取 → `MetaSteamStats`【Steam 源】→ `MetaTagEditor` → `MetaWorkDetails`）。**背景图规则**：Steam 源优先本机 `librarycache` 里文件名含 hero 的图（`scan.SteamHeroImage`，排除 blur/logo），没有才用商店接口的 `screenshots[0].path_full（真正的预览图，比 page_bg_raw 更亮更明显）/background_raw/background`；DLsite 源用画廊**第二张**预览图（第一张 `_img_main` 是游戏库卡片封面）。没有搜索卡/候选栏/封面保存位置卡/游戏库卡。**Steam 源作品入库后，「游戏启动」按钮右端接一个分体按钮右段 =「启动方式」下拉菜单（DropdownMenu 样式）**：通过 Steam 启动（默认，`steam://rungameid/<appid>`）/ 本地程序启动（跑 `game_path`；没有路径时先弹选 exe 框）；右段只显示一个下拉箭头（2026-11-24 用户要求「去除游戏启动旁启动方式简称」；当前方式在 `title`/`aria-label` 与下拉选项的勾选里表达），点开菜单用 `menuitemradio` + 勾选标记表达。只能从游戏库点封面进入 || `/library`（name `Library`） | `views/LibraryPage.vue` | **游戏库**：顶栏是 `MetaSection`（标题 + 右侧「已入库 N 款」+ 刷新 +「添加游戏」+「批量操作」；**原「搜索」按钮已移除**）。**卡片上没有任何操作按钮**（用户 2026-11-23 要求）：平时点封面 → `openWork()` 跳 `/works` 并 `loadById`（缓存优先）；点「批量操作」后界面右边缘滑出**窄竖条**（`.lib-batch-panel`，`position: fixed`，`top:30px→bottom:0`、**宽 56px**、`border-left`，从右整条滑入；**不参与布局、不给内容让位**，网格/卡片/滚动都不变、不挤压现有 UI；条内只放图标：标题图标 + 已选 N/总数 + 全选 / 删除 / 退出三个 36×36 图标按钮，文字走 `title`/`aria-label`；删除按钮红描边 → armed 填红 + 勾），此时点封面 = 选中 / 取消选中（**选中态 = 品牌色 2px 边框 + 品牌色半透明遮罩 + 居中勾选**），删除后自动退出；封面网格 `grid-cols-[repeat(auto-fill,minmax(150px,1fr))]`，hover 时半透明名称面板从下方淡入；**添加游戏入口在顶栏「游戏库」卡片里**（2026-11-23 起网格末尾的透明圆形加号 `.lib-add` 已删除）。点「添加游戏」弹居中「添加游戏」窗（`Teleport` 到 body，遮罩 `color-mix` + `backdrop-blur(8px)`，**只有 ✕ 与 Esc 能关，点遮罩不关**），窗内两种方式（`chip` 切换，默认「扫描添加」）：**扫描添加** = `components/library/ScanPanel.vue`（来源 Steam 库 / 选择文件夹，文件夹可选「按 exe 识别」或「按文件夹名」）扫出一批名字后**自动逐个搜索**（3 路并发，可「停止」/单行「重试」）；结果按扫描顺序一行一个（封面 + 匹配作品名 + RJ + 扫描名 + 社团·形式·发售日·容量，详情在搜索完后再补抓），行可勾选，工具条提供「全部入库」与「入库所选」（**串行**写库；已入库的行显示「已入库」且不再参与），点「详情」进 `/works` 核对；**手动搜索** = 原搜索行 + **「搜索来源」chip（DLsite / Steam）**：DLsite 支持 RJ 编号精确 / 作品名模糊（候选**每页 10 个**、分页条在候选下方、**只给当前页下 `cache/` 预览图**）；Steam 走 `storesearch`（候选 ≤10、无分页，缩略图用 `tiny_image` 直链），点「详情」按 appid 取商店详情；命中作品即关窗跳 `/works`； |

三页都**自带滚动容器**：`MainLayout.vue:6` 的 router-view 是 `overflow-hidden`，页面必须自己提供滚动；而 `global.css:207-213` 用 `!important` 全局隐藏了滚动条，只有 `.scrollbar-theme` 能恢复（`scrollbar-width: thin` + 带 `!important` 的 `::-webkit-scrollbar`）。`pr-2` 给 8px 滚动条留位。

⚠️ 但**数据不再直连 MetaApiService**：页面都走 `stores/metaStore.ts` → `libraryApi`（缓存优先、**只读**），只有 `/testing` 用 `refresh=true` 强制联网核对解析；`/testing` 的原始 JSON 因此会多出 `cached/workId/cover*` 等库侧键（后端注入，不是解析结果变化）。**现状：搜索/添加入口是 `/library` 顶栏「添加游戏」按钮（弹出居中「添加游戏」窗：默认「扫描添加」—— 扫 Steam 库或本地游戏文件夹，扫完自动逐个搜索、结果按行排列，可勾选入库或一键全部入库；也可切「手动搜索」，顶栏不再有「搜索」按钮）；搜索、扫描与点开详情都只读，只有点「入库」才会抓文本 + 下封面写库 —— 作品因此只会在入库后出现在 `/library`，点卡片进 `/works` 看详情；`/testing` 只作调试核对。**

**「下载」字样已改为「入库」；入库后按钮不再显示「已入库」，而是替换成「游戏启动」**：`MetaDownloadButton` 两个阶段 —— 未入库时是 `入库` / `入库中`（原来叫「下载封面」，无封面 URL 时禁用）；已入库时（`stores/metaStore.ts` 的 `stored` = 本次会话入库过 `saved`、后端注入的 `coverAbsPath` 非空、或 `libraryUpdatedAt` 非空）变成 `游戏启动` + play 图标，点击走 `metaStore.launchGame()` → 后端 `GameService.Launch` **按生效的启动方式分流**：`launch_mode='steam'`（Steam 源默认）跑 `steam://rungameid/<appid>`，`'exe'` 用 **`library_works.game_path`** 里记录的 exe 起独立进程。exe 方式下位置为空（老数据 / 手动搜索入库）或记录的文件已被移动时，后端返回 `missing=true`，前端立刻弹系统文件框选一次 exe（`DialogService.PickExecutable`，只筛 `.exe`），写进 `game_path` 后再启动；已设置过位置时按钮旁还有一个很轻的「更换位置」文字链接（Steam 方式下不显示）。**Steam 源入库时不再追问本机 exe**（默认 URL 启动）；想改成本地程序启动，在 `/works` 里点「游戏启动」右端的下拉段切（分体按钮，见 §6.5），届时再选 exe。真正的写入在 `LibraryService.Store`（抓文本 + 封面落 `covers/` + 写库 + 可选记录扫描到的 exe 路径），两个页面都走同一个 `metaStore.storeIn()`；前端不再有"只下图不写库"的独立入口。原来「已入库 + 禁用」的目的是挡掉重复下载（`writeFileUnique` 不覆盖同名文件，再点只会多出一份 `-2` 后缀的重复图）—— 现在已入库后根本没有入库按钮，这个目的同样达成。

### 3.7 游戏库服务（`internal/service/library_service.go`，`LibraryService`）

与 §3.3 的 `MetaApiService` **语义相反**（这是刻意的，见 §10）：`MetaApiService` 永远联网，`LibraryService` **缓存优先** —— 库里有的直接返回，没有才联网。**但读取不等于入库**：`Fetch` 只读不写，`Store` 才是"入库"（唯一会写库的方法）。所有方法都带 `source` 参数（当前唯一值 `dlsite`，为将来接别的源留位）。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `Fetch` | `(ctx, req FetchWorkRequest) (map[string]any, error)` | `FetchWorkRequest{keyword, workId, source, refresh}`。有编号且 `refresh=false` 时先查库；命中则用 `raw_json` 还原 + 注入 `cached=true` 与封面字段；未命中（或 `refresh=true`）走 `meta.FetchWork`，**只返回、不写库**（注入 `cached=false`）。`decorate` 统一挂 `gamePath` 与 **`launchMode`**（`library_works.launch_mode`；空串 = 按来源取默认），前端据此决定启动按钮的行为 |
| `Store` | `(ctx, req StoreRequest) (map[string]any, error)` | **入库**：`StoreRequest{workId, source, gamePath, gameFolder, gameFolderName}`。库里已有**完整** `raw_json` 就复用（不再抓文本），`raw_json` 残缺（id/name 或任何详情字段都没有，即"空壳"）会重新联网抓；否则联网抓一份 → **校验 `workIsComplete`（编号+名称+至少一个详情字段）** → `UpsertLibraryWork` →（有位置时）`SetGamePath` 写进 `library_works.game_path` → `DownloadCover` 把封面落到 `covers/` 并记账。**启动位置在入库时就确定**（用户 2026-11-16 要求）：直接给 `gamePath`（按 exe 名扫到/手动搜索选好）就用它；只给 `gameFolder`+`gameFolderName`（按文件夹名扫到）就调 `scan.PickGameExecutable` 进文件夹挑主程序；都空（手动搜索取消选择）则位置留空、启动时再问。空壳直接报错、不写库（扫描添加一次入库几十部，不能污染游戏库）。封面已存在时幂等返回。写库失败**直接报错**（这一步就是在"载入数据库"） |
| `List` | `(ctx, source string) ([]LibraryWork, error)` | `source` 空 = 所有来源；`ORDER BY created_at DESC, id DESC` |
| `Remove` | `(ctx, workID, source string) error` | 删除库记录，并**删除该作品入库时保存在 `covers/` 的封面图片**（按 `library_cover_files` 的全部记录 + 作品表 `cover_abs_path` 去重；文件删不掉时报错、库记录保留以便重试） |
| `CoverDataURL` | `(ctx, workID, source string) string` | 读 `cover_abs_path` → `data:<mime>;base64,…`；文件不存在返回空串（不算错误） |
| `SteamHeroImage` | `(ctx, appID string) string` | **2026-11-23**：详情页 hero 背景图。先 `scan.SteamHeroImage` 在 `appcache\librarycache\<appid>\` 递归找文件名含 hero 的图（排除 `*_blur*`/`*logo*`，`library_hero*` 优先、中文版优先），命中就读文件转 `data:<mime>;base64,…`；**找不到返回空串 + nil**，前端回落到商店接口的预览图 |
| `GetTags` | `(ctx, workID, source string) (model.TagSet, error)` | 读作品的标签集合 `{tags, edited}`。`edited=true` = 用户保存过整份列表（`work_tags` 里就是他定的那份，重抓不丢）；`false` = 只有旧版逐条加过的标签，前端还要与抓来的 `tags` 合并 |
| `SetTags` | `(ctx, workID, source string, tags []string) (model.TagSet, error)` | **唯一写标签的入口**：整份覆盖（清空 → 按顺序插入 → 记 `work_tag_edits` 标记，同一事务）。逐条清洗、丢空、去重；单条 ≤32 字符、最多 128 条，超了报错不截断；传空列表 = 用户删光（合法） |
| `UpdateMeta` | `(ctx, req UpdateMetaRequest) (map[string]any, error)` | **2026-11-24**：编辑页保存元数据。`UpdateMetaRequest{workId, source, fields map[string]any}` 打补丁：只改传来的键，空串/空数组/null = 删掉这一项；编号不可改、名称不可清空、只更新已入库作品；复用 `UpsertLibraryWork` 写库（cover_* / game_path / launch_mode 不受影响）。前端入口 `libraryApi.updateMeta()` → `metaStore.saveMeta()` |

**为什么封面用 base64 data URL 而不是 `file://`**：Wails webview 对本地文件协议的支持没实机验证过，data URL 一定可用；封面就几十~几百 KB，一次列表请求可接受。

**封面记账**：`MetaApiService.DownloadCover` 落盘后若持有 `DB` 就调 `RecordCoverFile` 写一行 `library_cover_files` 并把作品的 `cover_*` 指针指向它（`writeFileUnique` 会加 `-2/-3` 后缀，事后猜不出文件名，所以必须当时记账）。

**测试**：`internal/database/library_repo_test.go`（10 个用例）覆盖惰性建表、首写前先读/先删、**并发首次使用**（曾真实复现 `database is locked (5) (SQLITE_BUSY)`，见 §10）、来源隔离、封面指针更新。

### 3.8 本地扫描服务（`internal/service/scan/`，package `scan`，`ScanService`）

**已在 `main.go` 注册**为 Wails 服务 `ScanService`（`application.NewService(scanService)`，import `WeedyBox/internal/service/scan`）：绑定生成后前端经 `services/scanService.ts`（`scanApi`）调用，用于游戏库的「扫描添加」。**无状态**：根目录每次作为参数传入（游戏库可能挂多个根）。**放在独立子包 `scan`**（用户要求把扫描服务合并到一个文件夹），不要再往 `internal/service` 根目录塞扫描相关文件。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `ScanExecutablesInSubfolders` | `(ctx, root string) (GameScanResult, error)` | **通用**（靠 exe 认游戏，不管文件夹怎么命名）：读根目录 + 每个一级子文件夹各一层 → `GameScanResult{root, folders[], skipped[]}`；`folders[]` 每项 `{folderName, folderPath, executables[]}`，exe 为 `{fileName, name, path, size}`。**根目录自身的 .exe 不算、不递归、没有 .exe 的文件夹不进结果**；某个子文件夹读不动（权限/坏盘/被占用）只记进 `skipped` 并继续 |
| `ScanSubfolders` | `(ctx, root string) ([]Subfolder, error)` | **名字 + 绝对路径**（`Subfolder{name, path}`）：只读根目录一层，不打开子文件夹。给「按文件夹名扫描」用 —— 名字去搜索、路径留着**入库时进文件夹挑 exe** |
| `ScanSubfolderNames` | `(ctx, root string) ([]string, error)` | 同上的纯名字版本（内部复用 `ScanSubfolders`），保留给只要名单的调用方/测试 |
| `ScanSteamGameFolders` | `(ctx) ([]Subfolder, error)` | **Steam**：找本机 Steam 安装目录 → 读 `libraryfolders.vdf` 拿到全部库根 → 对每个 `<库根>\steamapps\common` **复用 `ScanSubfolders`**，返回游戏文件夹名 + 绝对路径。跨库去重（大小写不敏感）、按名排序（大小写不敏感，与资源管理器一致）。库盘没插/目录被删只跳过；一个 Steam 都找不到时返回哨兵错误 `ErrSteamNotFound`（区别于「装了但空库」） |
| `PickGameExecutable` | `(ctx, folderPath, folderName string) (string, error)` | **从游戏文件夹挑主程序**（用户 2026-11-16 规则）：先排除 `UnityCrashHandler64.exe`；有 `Game.exe` 优先；否则选**去掉扩展名的 exe 名与文件夹名最长公共子序列最长**的那个（并列取 ReadDir 顺序第一个）；没有可用 exe 返回空串 + nil（不阻断入库，启动时再让用户选） |
| `SteamHeroImage`（包级函数 `scan.SteamHeroImage`，不是 Wails 方法） | `(ctx, appID string) (string, error)` | **2026-11-23**：在 `<Steam>\appcache\librarycache\<appid>\` 里**不限深度**递归找文件名含 `hero` 的图片（`library_hero.jpg` / `library_hero_schinese.jpg`），排除 `*_blur*`（几 KB 模糊占位）与 `*logo*`；打分 `library_hero*`(100) + `schinese`(20)，同分越浅越优先；没有目录/没有图返回空串（不报错）。由 `LibraryService.SteamHeroImage` 读成 data URL 给 `MetaWorkHero` |

路径一律 `filepath.Abs` + `Clean` 成绝对路径（结果里的 `path` 将来要用来启动游戏，不能依赖进程工作目录）。`.exe` 判定大小写不敏感且用 `filepath.Ext`（`game.exe.txt` 不算）；`entry.IsDir()` 不够——Windows junction 的 `IsDir()` 是 false，要再 `Stat` 一次跟随链接。排序取 `os.ReadDir` 的按名顺序（**不按大小**，主程序不一定是最大 exe，选择交给调用方）。测试见 `scan/scan_test.go`：用真实 `t.TempDir()` 树断言「只读一层」的边界，不 mock 文件系统。

**Steam 安装目录的找法**（`scan/steam_windows.go`，build tag `windows`）：`HKCU\Software\Valve\Steam\SteamPath`（最准，值是正斜杠 `e:/steam`，要 `filepath.FromSlash`）→ `HKLM\SOFTWARE\Valve\Steam\InstallPath`（从 64 位进程读 32 位安装器写的键要带 **`WOW64_32KEY`**，所以 64/32 两个视图都试）→ `%ProgramFiles(x86)%\Steam` / `%ProgramFiles%\Steam` 兜底；注册表残留指向已删目录时由 `os.Stat` 筛掉。非 Windows 用 `scan/steam_other.go` 的空实现（本项目 GOOS 固定 windows，只是为了 `GOOS=darwin go build ./...` 不炸）。

**库路径来自 `libraryfolders.vdf`**：现役位置 `<Steam>\steamapps\libraryfolders.vdf`（块内 `"path" "D:\\SteamLibrary"`），旧客户端在 `config\libraryfolders.vdf`，两处都读；旧格式（键是数字、值直接是路径）在没有 `path` 行时兜底。**不引 VDF 第三方库**：两条行级正则足够，反斜杠/引号转义手工还原。**依赖**：`golang.org/x/sys/windows/registry` —— 该模块本来就在 `go.mod`（indirect），这次只是变成 direct，**没有新增模块**。测试见 `scan/steam_test.go`（vdf 现役/旧版样本、多库汇总去重、vdf 在 config 下、主库重复列出、无 common、取消）。真实机器已烟测：注册表 `e:/steam` → 46 个游戏文件夹。

### 3.9 系统对话框（`internal/service/dialog_service.go`，`DialogService`）

已注册（第 7 个服务）。两个方法：

- `PickDirectory(title string) (string, error)`：`app.Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true)` 打开系统「选择文件夹」框，`AttachToWindow(app.Window.Current())` 让它贴着主窗口。前端封装是 `scanApi.pickDirectory()`，只在「扫描添加 → 选择文件夹」用。
- `PickExecutable(title string) (string, error)`：`CanChooseFiles(true).CanChooseDirectories(false).AllowsOtherFileTypes(false).AddFilter("可执行程序", "*.exe")`，给「游戏启动」第一次没有位置 / 更换位置时选 `.exe`。前端封装是 `gameApi.pickExecutable()`。

**用户取消返回空串 + nil**（不是错误），只有应用实例未就绪/对话框创建失败才报错。`PromptForSingleSelection` 内部走 `InvokeSync` 切到主线程，所以从服务 goroutine 调用不会死锁。为什么不塞进 WindowService：窗口操作与"问用户要一个路径"是两类事。**注意 `go.mod` 的 beta.10 里 `application.Get()` 是包级全局实例**，服务里直接取即可。

### 3.10 游戏启动服务（`internal/service/game/`，package `game`，`GameService`）

已注册为第 8 个服务（`application.NewService(gameService)`，import `WeedyBox/internal/service/game`）。位置就是 `library_works.game_path` 一列（用户要求「游戏路径和 library_repo.go 合并在一个表内」），不单开表；`GameService` 只负责校验与起进程，DB 读写仍走 `database/library_repo.go` 的 `GetGamePath` / `SetGamePath` / `GetLaunchMode` / `SetLaunchMode`。

**启动方式（2026-11-23，用户要求）**：`library_works.launch_mode` 一列，取值 `''`（没选过）/ `'steam'` / `'exe'`。**默认按来源解析**：Steam 源 → `steam`（跑 `steam://rungameid/<appid>`，交给 Steam 客户端，不需要本地 exe）；其它源 → `exe`（跑 `game_path` 里的程序）。前端 `/works` 的启动方式下拉（「游戏启动」右端的分体按钮右段）只在 Steam 源、已入库时出现。

| 方法 | 签名 | 行为 |
| --- | --- | --- |
| `Status` | `(ctx, workID, source string) (GameStatus, error)` | 读位置 + 检测 exe 是否存在 + 是否运行 + 生效的启动方式，返回 `{workId, path, exists, running, pid, launchMode}`。缓存命中先复核 PID（`processAlive` 比对完整映像路径，防 PID 复用）；失效/未命中就全量扫进程（快照按 exe 名粗筛 → `QueryFullProcessImageName` 比对**完整路径**），找到则把路径+PID 载入运行缓存。exe 不存在时清掉缓存并返回 `exists=false` |
| `SetPath` | `(ctx, req SetGamePathRequest) (GamePathState, error)` | `SetGamePathRequest{workId, path, source}`。`normalizeGamePath`：非空 → `filepath.Abs` + `Clean` → `os.Stat` 必须存在且**不是文件夹** → `DB.SetGamePath`（只更新已在游戏库里的作品；作品不在库里时报错，不顺手造空壳行）。返回 `{workId, path, exists:true}` |
| `SetLaunchMode` | `(ctx, req SetLaunchModeRequest) (GameLaunchModeState, error)` | `{workId, source, mode}`；`mode` 只接受 `steam/exe/''`；`steam` 仅对 Steam 源开放（dlsite 没有 appid）且要求作品已在库里。返回 `{workId, mode}`（生效值）。只改「下次怎么启动」，不启动游戏 |
| `Launch` | `(ctx, workID, source string) (GameLaunchResult, error)` | 先解析生效方式：**steam** → 校验编号是纯数字 appid → `startURL("steam://rungameid/<appid>")`（`ShellExecute "open"`）→ `{started:true, mode:'steam', message:"已请求 Steam 启动游戏（…）"}`，**不看也要求 `game_path`**；**exe** → 读 `game_path`：空 → `{started:false, missing:true, message:"还没有记录…"}`；文件不存在 → `missing:true`（**不报 error**，前端据此弹选 exe 框）；否则 `startDetached` 起独立进程 → `{started:true, running:true, path, mode:'exe'}`，并把 PID 载入运行缓存。进程起不来（权限/损坏）→ error |

启动实现（`launch_windows.go`）：`exec.Command(exe)` + `cmd.Dir = filepath.Dir(exe)`（很多游戏按相对路径找资源）+ `SysProcAttr.CreationFlags = windows.DETACHED_PROCESS`（独立进程、关掉 WeedyBox 不带走游戏、不闪控制台）+ `Process.Release()`，返回 PID；**steam URL 走 `windows.ShellExecute(0, "open", url, …)`**（由系统转交 Steam 客户端，不闪控制台、不产生子进程 PID）。非 Windows 用 `launch_other.go` 空平台实现（build tag），只为 `GOOS=darwin` 检查不炸。

**运行检测**（`process_windows.go`）：**不引 gopsutil**（项目约定不新增依赖；`golang.org/x/sys` 本来就在 go.mod）——`CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS)` 枚举进程，先用 `ExeFile` 基名粗筛，再 `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` + `QueryFullProcessImageName` 比对完整路径（同名 exe 不会误判）。`GameService` 持 `mu sync.Mutex` + `running map[归一化小写路径]pid` 运行缓存：查询命中先 `processAlive` 复核，游戏退出/文件删除就从缓存移除。前端进入作品详情调一次 `Status`，`WorksPage` 每 5 秒轮询 + 窗口 focus 时刷新，按钮在「游戏启动 ↔ 运行中」之间实时切换（Steam 方式启动后由 Steam 拉起游戏进程，几秒内同样能被检测到）。测试见 `game/game_service_test.go`（路径校验、SetPath 写入、Upsert 不丢位置、未入库拒绝、Launch `missing` 分支、Status 无位置/文件被删清缓存、运行缓存、**启动方式默认解析 / 切换与写入 / 非数字 appid 拒绝 / exe 方式 missing**）与 `process_windows_test.go`（真起一个 `cmd.exe`/ping 进程，验证按完整路径能认出运行中，查完即杀）。

### 3.11 Steam 元数据源（2026-11-21）

**需求原话**：「steam源添加(meta)根据游戏id( ...\Steam\steamapps目录下有appmanifest_id相关文件 扫描id 并通过 https://store.steampowered.com/api/appdetails?appids=id 获取json数据 优先复用已有逻辑)进入相关文件夹 并无限递归查找含有 600x900 或 capsule 字样的图片文件 用作封面(游戏库封面展示图像格式也全部变为2:3等比图样 包括steam源和 DLsite源 和以后所有可能添加源) 另外 作品信息显示界面未来要作修改 暂时不进行改动」；补充：「查找封面图片文件夹目录在...\Steam\appcache\librarycache内 通过已有逻辑查找 目录下含有游戏id名称的文件夹 进入查找相关图像 没有文件夹则不适用图像」。

- **来源**：`database.SourceSteam = "steam"`，编号 = appid（纯数字）。与 dlsite 同表（`library_works` 不变结构），`UNIQUE(source, work_id)` 天然隔离。
- **扫描**（`internal/service/scan/steam_apps.go`）：`ScanSteamApps()` 读每个库 `steamapps\appmanifest_<appid>.acf`，解析 appid / name / installdir / StateFlags / SizeOnDisk。**StateFlags 按位判 4**（实测 R.E.P.O. 是 6 = 已安装|待更新，写 `==4` 会漏）；跨库按 appid 去重、按名称排序。库根来自 `steamLibraryRoots()`（主库 + `libraryfolders.vdf`，现役与旧版位置都读）。
- **元数据**（`internal/service/getMeta/getFromSteam.go`）：`MetaService.FetchSteamApp` → `store.steampowered.com/api/appdetails?appids=<id>&l=schinese&cc=CN`；`parseSteamAppDetails` 解析外层 `{"<appid>":{success,data}}`，`buildSteamWork` 映射成作品 map：`id/name/maker`(开发商)`/workType`(类型中文)`/releaseDate/summary/coverUrl`(header_image)`/tags`(类型+分类+开发商+发行商)`/outline`(开发商/发行商/平台/评测/成就/Metacritic)`/sourceUrl/fetchedAt`，外加 `steamType`（原始 type，前端据此跳过 tool/dlc）与 `steam`（完整 appdetails，作品界面以后重做时直接读）。**genres.id 是字符串、categories.id 是数字**，两个 id 都不声明，只用 description。
- **分流**（`internal/service/library_service.go`）：`Fetch` / `Store` 先看 `source`；steam 走 `fetchSteamWork` / `storeSteamWork`，**缓存优先、Fetch 只读、Store 才写库**的语义与 dlsite 完全一致。位置仍走 `resolveStoreGamePath`（扫描给的安装目录 → `scan.PickGameExecutable`）。
- **封面**（`scan.SteamCoverImage` + `getmeta.ImportCoverFile`）：在 `<Steam>\appcache\librarycache` 下找名字含 appid 的目录（精确同名优先）→ **不限深度**递归找文件名含 `600x900` 或 `capsule` 的图片，打分 `600x900`(100) > `capsule`(40) > `_schinese`(+20)，同分越浅越优先；**没有对应目录 / 没有候选图就返回空串**（用户要求：没有文件夹就不用图像，不联网兜底）。找到后复制进 `covers/`（沿用不覆盖、重名 `-2/-3`）并记账，`library_cover_files.source='steam'`。
- **前端**：`scanStore` 的 Steam 来源改用 `scanApi.steamApps()`，行内带 `source:'steam'` + `appId`，直接 `libraryApi.fetch('', appid, false, 'steam')` 取详情（不再拿文件夹名去 DLsite 搜）；`steamType !== 'game'` 的行标「已跳过」、不给入库。`metaStore` 增加 `source` 状态，`loadById(id, refresh, fromSource?)` 把来源贯穿到入库/启动/标签/封面调用；`stored` 判定增加 `libraryUpdatedAt`（Steam 可能没有封面，只看 `coverAbsPath` 会误判未入库）。
- **封面比例**：`LibraryPage.vue` 网格封面 `aspect-[4/3]` → **`aspect-[2/3]`**（`object-cover`），Steam / DLsite / 以后所有来源统一。`/works` 作品信息界面本次**未改**（用户明确以后单独改）。
- **实测**（本机 `E:\steam`）：19 份 appmanifest → 19 条；17 条在 librarycache 找到封面（`Beat Banger` 与本机 `Steamworks Common Redistributables` 没有对应图），嵌套 hash 目录里的 `library_capsule.jpg` 也能递归命中。测试：`scan/steam_apps_test.go`（manifest 解析/位判断/多库去重/封面递归与打分）、`getMeta/getFromSteam_test.go`（字段映射/失败分支/封面导入记账）。
- **标签改走 GetItems（2026-11-22，用户要求「tag 的获取尝试使用别的接口」）**：`getFromSteam.go` 抓到 appdetails 后调 `enrichSteamWork`（`steam_browse.go`）——① 用 `IStoreBrowseService/GetItems/v1`（`include_tag_count:20` + `include_reviews`）拿**玩家标签**（tagid + 权重），经 `IStoreService/GetMostPopularTags`（进程内缓存 24h）换成中文名，**按权重降序排在前面**，原来 appdetails 的「类型/分类/开发商/发行商」接在后面去重（`mergeSteamTags`，上限 `steamTagLimit=64`）；② 评价汇总写进 `work["steamReviews"]` 并补一行 outline「Steam 评价：好评如潮 97%（N 篇）」；③ GetItems 原始数据留在 `work["steamBrowse"]`。**这一步任何失败都只是"少一点信息"**，不报错、不影响入库。
- **手动搜索一次同时查 DLsite 与 Steam（2026-11-23，取代原来的「搜索来源」chip）**：用户原话「手动搜索只显示DLsite数据源 应该也显示steam数据源」。`metaStore.runMultiSearch()` 用 `Promise.allSettled` 并发调 `SearchWorks`（DLsite，30 条/页、前端 10 条一页）与 `SearchSteamWorks`（storesearch，10 条、无分页），弹窗里用**源切换器**分开显示（2026-11-24 起不再把两源归并到同一列表：按钮带条数，默认 DLsite、无 DLsite 结果时自动落到 Steam），分页只属于 DLsite（Steam ≤10 无分页，切到 Steam 时整条分页不出现）；一路失败只写 `searchNotice`（`MetaMessage tone="warning"`），另一路的结果照常展示。编号输入（RJ/RE/VJ）仍只走 DLsite 精确抓取。Steam 候选缩略图用 `tiny_image` 直链（公网 CDN，不需要 `cache/`；`CacheCover` 只认 DLsite 编号）。
- **扫描结果的「多数据来源」（2026-11-23，用户原话「自动扫描游戏功能 自动搜索时应该显示多数据来源」）**：`scanStore.searchEntry` 对**每个本地条目（exe / 文件夹）同时搜两个来源** —— Steam 先 `MatchSteamAppByFolder`（清洗 → storesearch → 逐级截短 → 相似度打分），猜得到 appid 就出一行 Steam；然后拿同一个名字 `SearchWorks`，取第一个候选出一行 DLsite。一个条目因此最多两行（Steam 行 + DLsite 行），**各自勾选、各自入库**（同一游戏在两个来源里是两条库记录），行内用 `.src-tag` 徽标标出来源。**Steam 库来源的扫描仍只出 Steam 行**（appid 已确定，再按英文名去 DLsite 搜只会引入误配；要对照用「手动搜索」）。**打分口径同日改为编辑距离相似度**（`steamNameScore` = 1 − Levenshtein/较长串长度，<0.6 算不像）：旧的前缀占比法对「demon root」vs「Demons Roots」这种差一个字母的名字给 0，导致扫描永远猜不到 Steam 源。联网烟测（2026-11-23，`demon root`）：Steam `Demons Roots` / appid `1933660`、DLsite `Demons Roots` / `RJ349517`，两边详情都能取到；此前的匹配样例（`黑神话悟空→2358720`、`艾尔登法环黄金树幽影→2778580`、`ULTRAKILL v1.0→1229490`、`Hades→1145360`、`Stardew_Valley→413150`）仍成立，`hades`→`hadesii`（0.714 < 0.75 阈值）仍判不匹配。
- **/works「Steam 动态」（2026-11-22）**：`components/meta/MetaSteamStats.vue`，只在 `meta.source === 'steam'` 时渲染：评价（label + 好评率 + 篇数）、当前在线、最近 3 条公告（标题外链 + 日期）。数据 `MetaApiService.FetchSteamStats`，`applyWork` 时自动取一次，头部有「刷新」。
- **启动方式可选 Steam URL / 本地 exe（2026-11-23，用户要求「通过 steam 源添加的游戏 游戏启动可选通过 steam 启动（通过运行 steam://rungameid/<AppID> URL 运行 steam 游戏）也可选通过本地 exe 直接启动 默认使用 URL 启动」）**：`library_works.launch_mode` 存用户选择（空 = 默认），`Fetch`/`Store` 经 `decorate` 带回 `launchMode`；`GameService.SetLaunchMode` 切换、`Launch` 按生效方式分流（steam → `ShellExecute` 打开 `steam://rungameid/<appid>`；exe → 原有 `startDetached`）。Steam 源**默认 `steam`**，入库时不再追问本机 exe；切到「本地程序启动」且没有 `game_path` 时先弹系统文件框选一次 exe。

### 3.12 Steam 公开接口清单（2026-11-22 调研，**已部分接入**）

用户要求「在网上搜索相关api信息 把可能可以扩展当前软件的api保留」。逐条实测（本机 PowerShell / `curl.exe` 都在 schannel 层失败，改用 Go `net/http` 探测），结论写进 [`doc/steam-api.md`](./doc/steam-api.md)，入口常量在 `internal/service/getMeta/steam_endpoints.go`。

- **已接入（2026-11-22 当天接着做完的三条 + 标签源）**：`GetItems`（Steam 标签 + 评价汇总，见 §3.11）、`GetMostPopularTags`（tagid → 中文名）、`storesearch`（手动搜索的 Steam 来源 + 扫描「按文件夹名」猜 appid）、`GetNewsForApp` + `GetNumberOfCurrentPlayers`（`/works` 的 Steam 动态）。代码在 `steam_browse.go`，测试 `steam_browse_test.go`。
- **免 key 可用但暂未用**：`GetStoreCategories`（categoryid → 中文名，比 appdetails 的 type 更细）、`search/suggest`（HTML 片段）、`featured/featuredcategories`（商店推荐位，与本软件无关）。
- **不能忘的两个实测坑**：`appdetails` **一次只能一个 appid**（逗号拼多个 → `400 null`）、`filters=` 参数被忽略；`storesearch` **最多 10 条且按词匹配**（`黑神话悟空` 0 条、`黑神话` 9 条 → 所以文件夹名匹配要"逐级截短"）。
- **不可用**：`appreviews`（匿名请求一律 `302` 到商店首页；换区、加 cookie、加 `X-Requested-With`/`Referer`、改 POST 全都不行 → 评价改用 GetItems 的 `reviews.summary_filtered`）；`ISteamApps/GetAppList/v2` 已 `404` 下线；SteamSpy 返回 Cloudflare 挑战页。
- **要 Web API key（默认不接）**：`IStoreService/GetAppList`、`IPlayerService/GetOwnedGames`、`GetDLCForApps`、`GetSchemaForGame`、`GetGlobalAchievementPercentagesForApp`。
- **还没做、也仍需用户点头的**：联网封面兜底（`assets.library_capsule` 现成 600×900）—— 与用户「librarycache 没有就不要封面」的决定冲突，未经同意不做。

### 3.4 获取外部元数据（`internal/service/getMeta/getFromDlsite.go`，package `getmeta`）

抓 DLsite 作品页。**不依赖第三方 DLsite 库**（`go.felesatra.moe/dlsite` 已从 `go.mod` 移除，原因见 §10），只用 `net/http` + `goquery`。`MetaApiService`（§3.3）是它的 Wails 包装层。

| 方法 | 签名 |
| --- | --- |
| `NewMetaService(options ...MetaOption)` | 默认 `http.Client{Timeout: 20s}`、`tagLimit=24` |
| `WithHTTPClient(*http.Client)` / `WithTagLimit(int)` | 注入客户端 / 限制标签数 |
| `GetWorkByID` | `(ctx, id string) (*DLsiteWork, error)` —— 按 RJ/RE/VJ 取单个作品 |
| `SearchWorkByName` | `(ctx, keyword string) ([]DLsiteSearchItem, error)` —— 第一页候选（`SearchResultPage(ctx, keyword, page)` 返回该页 30 条 + 总页数 + 总数）；每项只有 `ID`/`Name`/`Cover`。⚠️ 缩略图在页面里是 Vue 组件 `<thumb-with-ng-filter-block :thumb-candidates="['//img…webp','//img…jpg']">`，**不是** `<img>` 的 `src`/`data-src`；块内那个 `<img>` 只是弹层占位图（`src` 是内联 gif data URL）。`extractSearchItemCover` 因此优先读 `:thumb-candidates` 的第一项并补成 `https://`，退路才看传统图片属性且跳过 `data:` 占位 —— 曾经的"搜索结果不出图、点详情才有图"就是这个坑 |
| `GetWorkByName` | `(ctx, keyword string) (*DLsiteWork, error)` —— 搜索取首个再抓详情 |
| `NormalizeDLsiteID` | `(raw string) (string, bool)` —— 抽编号，转大写、**保留前导零** |

`DLsiteWork` 字段（json tag 全 camelCase）：`ID` `Name` `AlternateID`(拼音 slug) `Maker` `MakerID` `Series` `SeriesID` `WorkType` `AgeRating` `FileFormats[]` `FileSize` `ReleaseDate` `Price` `Currency` `PriceUntil` `Rating` `RatingCount` `CoverURL` `Summary` `PreviewImages[]` `Tags[]` `Outline[]` `SourceURL` `FetchedAt`

**`PreviewImages[]`（2026-11-23）**：作品页画廊的大图，**保持页面顺序** —— 第一张是封面 `_img_main.jpg`（游戏库卡片用的就是它），之后是 `_img_smp1`、`_img_smp2`…（含 `<source>` 的 webp 变体时优先取 jpg；`_100x100`/`_240x240` 缩略图排除）。用途：`/works` 的 hero 背景取**第二张**（`previewImages[1]`），见 §3.3 的 `/works` 行与 §10 决策。

两个真实作品的实测值（**旧模板**时代的离线记录；旧 fixture 已随工作区裁剪删除，现役新模板见下方「九个坑」与测试说明）：

| 字段 | RJ01721474 | RJ01682332 |
| --- | --- | --- |
| 作品名 | 【简体中文版】ELF PARADISE～… | 【簡体中文版】乙女の花園…【CV.涼本あきほ】 |
| 社团 / ID | 青春×フェティシズム / `RG51931` | RaRo / `RG49556` |
| 系列 / 系列ID | `ELF PARADISE` / `SRI0000043774` | 空 / 空 |
| 作品形式 / 年龄指定 | ボイス・ASMR / `R18` | ボイス・ASMR / `全年齢` |
| 文件格式 / 容量 | `[WAV mp3 mp4同梱]` / `16.83GB` | `[WAV MP3同梱]` / 空 |
| 发售日 | `2026-09-26` | `2026-10-07` |
| 价格 / 币种 / 有效期 | `1540` / `JPY` / `2026-10-24T00:00:00+09:00` | `1650` / `JPY` / 空 |
| 评分 / 评价数 | `5` / `16` | `4.81` / `43` |
| 简介长度 / 标签数 | 4799 字符 / 10 个 | 2417 字符 / 10 个 |
| 封面 | `…RJ01721474_img_main_240x240.jpg` | `…RJ01664616_img_main.webp`（页面自带的值） |

`Outline[]` 额外原样带出：`作者` `シナリオ` `イラスト` `声優` `音楽` `その他` `対応言語` `ジャンル` —— 页面新增字段不用改后端。

**拿不到的**：曲目表与试听音频（`ol.work_tracklist_list` 在当前页面已不存在，改由 Vue 渲染）、销量/愿望单/评论数/折扣率、社团与系列的详情页、多语言版本对应关系、评论正文。

**标签（`Tags`）的拼接规则（2026-11-12，用户要求）**：来源 = 分类（`dd a[href*="/genre/"]`，旧页面 `.main_genre a`）+ 作品形式（按 `dd` 里的 a/span 逐个取）+ 社团名 + 作者 + 剧情 + 插画 + 声优（按元素取，一个声优一个元素）+ 年龄指定/販売形式。全部经 `splitTagText` 按**空白字符和 `/`** 切分（`双声道立体声/人头麦` → 两个、`みたかりん 野上菜月` → 两个），再去重、按 `defaultTagLimit=64` 截断。**多值字段必须按元素取**：中文页面里相邻元素之间没有空白，取整段 `dd.Text()` 会粘成 `异种X造孽命令…` / `みたかりん野上菜月`。前端设置页的「标签显示数量」（4–64，默认 24，`stores/App.ts` 的 `tagLimit`）只控制**显示**多少个，超出折成 `+N`。

**九个坑（改之前必读，2026-11-12 复核）**：① 必须 HTTPS；② 前导零丢了必 404；③ 必须发 `Cookie: adultchecked=1; locale=zh-cn`（代码里写死中文；R18 不退化，且页面标签变成中文 —— **只认日文标签会全部取空**）；④ 标题只取 `#work_name` 的文本（现代码里没有 `<a>`）；⑤ **页面模板 2026-11 换新版**：`#work_outline` 表格、`#work_maker`、`<meta itemprop="price/ratingValue">` 全部消失，改用 `div.c-productInfo` 的 `dt.c-productInfo__key → dd.c-productInfo__value`（信息表）+ `<script type="application/ld+json">` 的 Product（价格/评分/社团/拼音/简介）。**只认旧选择器时解析结果就是"只有名字+封面"的空壳**（`raw_json` 约 280B）—— 用户报"扫描入库只有封面和作品名"的根因就在这里，旧库里已有空壳也会被 `workIsComplete` 判为不完整、重新抓取；⑥ 标签文字随 locale 变（`社团名/サークル名`、`发售日/販売日`、`文件容量/ファイル容量`…），取值一律走 `outlineValueAny` 多语言候选；genre 用 `dd a[href*="/genre/"]` 识别，不认标签；⑦ `SeriesID` 旧页面取内嵌 JSON，新页面取 `title_id/SRI…` 链接；`MakerID` 取 `maker_id/RG…` 链接或 JSON-LD `brand.url`；⑧ 必须判 `FindStringSubmatch` 长度（否则单作 panic）；⑨ `CoverURL` 已改为 `og:image` 优先。

**图片可以直接爬（实测）**：`img.dlsite.jp` DNS 干净（Cloudflare `104.18.24.157` / `104.18.25.157`），**不需要 Cookie、不需要 Referer**，`curl --noproxy '*'` 直连即可（PowerShell `Invoke-WebRequest -OutFile` 会因 .NET 默认代理报 `NonInteractive mode`，不要用它）。同一封面 4 个档位：`_img_smp1.jpg` 1470×1000 / 808 KB > `_img_main.jpg` 560×420 / 222 KB（`og:image` 就是它）> `_img_main.webp` 38.9 KB > `_img_main_240x240.jpg` 45.9 KB。

**`CoverURL` 取大图的优先级（已修）**：旧实现扫 `img, source` 时先命中 `#work_thumb img` 的 `_img_main_240x240`，于是永远拿最小那张。现在 `extractCoverURL` 的顺序是 `meta[property="og:image"]` → `meta[itemprop="image"]` → `extractCoverURLWidest`（扫 `img,source`，用 `coverWidthScore` 按后缀宽度挑最大的：显式 `_WxH` 取 W，其次 `_smp`=1470 > `_img_main`=560）。实测两个 fixture 都从 240×240 升到 560×420：`RJ01721474_img_main.jpg` / `RJ01664616_img_main.jpg`。

**测试**：`new_template_test.go` 用两个 fixture —— `work_new_RJ01232863.html`（日文标签的精简页面）与 `work_new_zh_RJ01232863.html`（**真实中文页面**，去掉 style/script 后约 81KB），分别锁住「日文标签」与「中文标签（应用实际发的 locale）」两条路径，断言具体值；另有 `TestParseWorkRejectsPageWithoutName`（结构变化必须报错而非返回空壳）。旧的 `work_RJ01721474.html` / `work_RJ01682332.html` 已随工作区裁剪删除。`go test ./internal/service/getMeta/` 全绿。实测（2026-11-12 联网）：RJ01232863 的社团/系列/形式/分级/容量/发售日/价格/评分/简介/标签全部解析成功。

### 3.5 数据模型（`internal/model/todo.go`）

```go
type Todo struct {
    ID int64 `json:"id"`; Title string `json:"title"`
    Description string `json:"description,omitempty"`
    Priority int `json:"priority"`   // 0 低 / 1 中 / 2 高
    Completed bool `json:"completed"`
    CreatedAt, UpdatedAt time.Time
}
type CreateTodoDTO struct { Title string; Description string; Priority int }
type UpdateTodoDTO struct { Title, Description *string; Priority *int; Completed *bool }  // 全指针 = 局部更新
type TodoStats struct { Total, Completed, Pending int64 }
```

> `rowToTodo` / `rowToStats` 依赖 SQLite 返回 **`int64` / `time.Time`**，改列类型会 panic。

`internal/model/library.go`：

```go
type LibraryWork struct {   // 来源无关：公共列 + 来源专有字段整份进 RawJSON
    Source string           // "dlsite"（当前唯一值）
    WorkID string           // RJ01721474
    Name, Maker, MakerID, Series, SeriesID, WorkType, AgeRating, ReleaseDate,
    CoverURL, SourceURL, RawJSON string
    FetchedAt, CreatedAt, UpdatedAt time.Time
    CoverDir, CoverPath, CoverAbsPath, CoverSuffix string
    CoverBytes int64
}
type CoverFile struct {     // 每次下载一行，作品表的 cover_* 指向最新一行
    ID int64; Source, WorkID, Dir, Path, AbsPath, Suffix, CoverURL string
    Bytes int64; CreatedAt time.Time
}
```

### 3.6 表结构

**`todos`**（`internal/database/sql_service.go:45-53`）：`todos(id INTEGER PK AUTOINCREMENT, title TEXT NOT NULL, description TEXT, priority INTEGER DEFAULT 1, completed INTEGER DEFAULT 0, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP)`。**2026-11-23 起前端不再读写 `completed`**（完成功能从界面移除）；列与 `TodoService.ToggleComplete/GetStats` 保留不动，删列要另外做数据迁移。

**游戏库两张表**：**不在 `initAllTables` 里建**，改由 `internal/database/library_repo.go` 的 `ensureLibraryTables(ctx)` 在**每次读写前** `CREATE TABLE IF NOT EXISTS`（幂等）——这是用户要求的「当需要保存到库中时 即是需要下载数据才新建表 节约空间」。

- `library_works`：`id PK / source / work_id / name / maker / maker_id / series / series_id / work_type / age_rating / release_date / cover_url / source_url / raw_json / fetched_at / cover_dir / cover_path / cover_abs_path / cover_bytes / cover_suffix / **game_path** / **launch_mode** / created_at / updated_at`，唯一键 `UNIQUE(source, work_id)`。`game_path` = 这个作品在本机上的游戏程序（.exe）绝对路径；`launch_mode` = 启动方式（`''` 默认 / `'steam'` / `'exe'`，2026-11-23 加）。两列都与元数据同表但**不被 Upsert 覆盖**（ON CONFLICT SET 里不含它们）。旧库文件没有这两列 —— `ensureLibraryTables` 末尾用 `ensureLibraryWorkColumn`（`PRAGMA table_info` 检测后 `ALTER TABLE … ADD COLUMN`）分别补 `game_path` / `launch_mode`（并发重复 ALTER 按已存在忽略）。
- `library_cover_files`：`id PK / source / work_id / dir / path / abs_path / bytes / suffix / cover_url / created_at`，索引 `idx_library_cover_files_work(source, work_id)`
- `work_tags`（作品标签，**惰性建表**，由 `tag_repo_test.go` 覆盖）：`id PK / source / work_id / tag / created_at`，`UNIQUE(source, work_id, tag)`，索引 `idx_work_tags_work(source, work_id)`。存的是"用户整理后的整份列表"（抓取标签与自定义标签合并的结果），顺序 = `id` 递增。
- `work_tag_edits`（标签已改写标记，同一处惰性建表）：`source + work_id` 主键 + `updated_at`，有行 = 这个作品的标签被用户整份改写过。为什么需要它：用户把标签**全删光**时 `work_tags` 也是空的，光看行数分不清「从没整理过」与「整理成空列表」。
- 两张标签表都刻意不挂在 `library_works` 上：作品库详情页可以打开还没入库的作品，那里的标签也要能存。

`UpsertLibraryWork` 用 `INSERT … ON CONFLICT(source, work_id) DO UPDATE SET …`，且**不覆盖 `cover_*` / `game_path` / `launch_mode` 列**（封面由 `RecordCoverFile` 维护，启动位置由 `SetGamePath` / 扫描入库维护，启动方式由 `SetLaunchMode` 维护；重新抓元数据不该把它们清空）。

DB 文件 `WeedyBoxDB.db` 建在**可执行文件同目录**（`sqlite.Config{DBSource: "WeedyBoxDB.db"}`）。

---

## 4. 前端架构

### 4.1 数据流

```
组件 → stores/todoStore.ts → services/todoService.ts → bindings/…/todoservice.ts
     → Go service.TodoService → database.SQLiteService → SQLite

组件 → stores/metaStore.ts → services/libraryService.ts → bindings/…/libraryservice.ts
     → Go service.LibraryService ─┬→ database/library_repo.go → SQLite（缓存命中）
                                  └→ getmeta.MetaApiService → DLsite（未命中/强制刷新）

组件 → stores/metaStore.ts → services/gameService.ts → bindings/…/gameservice.ts
     → Go game.GameService → database/library_repo.go（读/写 library_works.game_path）→ 启动 exe
```

调试页（`/testing`）是唯一例外：它用 `refresh=true` 让 `LibraryService` 跳过缓存，等价于直连解析器。

`services/todoService.ts` 导出一个 `todoApi`：`getAll · getByID · create · update · delete · deleteMany(ids)`，用本地 `ensure<T>(value, message)` 把绑定层的 `T | null` 收成 `T`。
（2026-11-23 起「完成」功能从界面移除：`toggleComplete` / `getStats` 不再从前端调用；后端 `TodoService.ToggleComplete/GetStats` 与 `todos.completed` 列仍在，只是没有入口。）

### 4.2 `stores/todoStore.ts`（Pinia setup store，id `todo`）

- **State**：`todos` · `loading` · `error` · `curPriority`(默认 1) · `filters{priority,time}`
- **Getters**：`highPriorityTodos` · **`filteredTodos`（模板渲染只用这一个）** · `isFilterActive` · `isFilteredEmpty`
- **筛选类型**：`PriorityFilter=number|null`、`TimeFilter='all'|'today'|'week'|'month'`；选项常量 `PRIORITY_OPTIONS` / `TIME_OPTIONS` / `TIME_RANGE_DAYS{today:0,week:7,month:30}`（`CompletedFilter/COMPLETED_OPTIONS` 已随完成功能删除）
- **Actions**：
  - `loadTodos()`：`todoApi.getAll()`（不再取 stats）
  - `addTodo(title, description?)`：`unshift`
  - `updateTodo(id, updates)`：就地替换返回对象
  - `deleteTodo(id)`：本地 filter
  - **`deleteMany(ids)`**：去重 → 记录 `{index,todo}` 快照 → 乐观移除 → 失败时**按原索引升序插回**恢复顺序
  - `init()` · `resetFilters()`

### 4.3 其余 store

| store | id | 关键点 |
| --- | --- | --- |
| `stores/Sidebar.ts` | `sidebar` | `barWidth=72`、`barWidthCollapsed=250`；**命名反直觉**：`isCollapsed=true` → 宽度 250。localStorage 键 `sidebar-collapsed` |
| `stores/Theme.ts` | `theme` | `ThemeMode='light'|'dark'|'system'`(默认 system)；`effectiveTheme`；`applyTheme()` 用 `documentElement[data-theme="dark"]`；localStorage `theme-mode`；`setupSystemListener()` 监听 `prefers-color-scheme` |
| `stores/App.ts` | `app` | `CloseAction='ask'|'hide'|'close'`（默认 `ask`）；localStorage `close-action`；`VALID` 数组判定，**非法值/旧值 `minimise` 一律回落 `ask`**；导出 `CLOSE_ACTION_OPTIONS`；`setCloseAction` + `restoreState` + `watch` 兜底落盘。**另有 `tagLimit`**（作品标签显示数量，默认 24，范围 4–64，localStorage `tag-limit`，导出 `TAG_LIMIT_MIN/MAX/DEFAULT`，`setTagLimit` 会自动夹取；注：2026-11-22 起 `/works` 的标签由 `MetaTagEditor` 全量显示可编辑，这个数量只作用于调试页的标签云） |
| `stores/metaStore.ts` | `meta` | 元数据（DLsite / Steam 双源）：State `keyword·loading·downloading·launching·error·status·work·saved·coversPath·candidateCovers(id→dataURL)·sourcePages(DLsite页号→30条)·sourcePageCount·searchTotal·pageLoading·candidatePage·raw·cached·libraryUpdatedAt·coverAbsPath·gamePath·**launchMode（''|'steam'|'exe'，库里存的启动方式）**·gamePathExists·gameRunning·gameWarning·**tagList（编辑稿：抓取标签 + 自定义标签合并）·tagSaved（上次写库的那份）·tagsEdited·tagsSaving·tagError**·**searchNotice（部分来源失败时的 warning 提示）·steamCandidates·steamSearchTotal·steamStats·steamStatsLoading·heroImage（Steam 本机 librarycache 的 hero 图 data URL；空 = 本机没有）**`；Getters `rawJson·hasResult·stored·launchBlocked（exe 方式下记了路径但 exe 不存在）·**effectiveLaunchMode（显式选择优先，否则 Steam 源='steam'、其它='exe'）**·**backgroundUrl（/works 的 hero 背景图：Steam 本机 hero → 商店 screenshots[0].path_full（真正的预览图，比 page_bg_raw 更亮更明显）/background_raw/background → 封面；DLsite 取 previewImages[1] 即画廊第二张，旧缓存从 coverUrl 推 `_img_smp1`）**·tagsDirty（编辑稿与已保存不同，决定「保存」按钮显不显示）·candidates·candidateTotal·candidatePageCount·pagedCandidates`；Actions `runSearch（仅 DLsite；调试页核对解析用，可 refresh）·**runMultiSearch（手动搜索：DLsite + Steam 用 Promise.allSettled 同时搜，结果分组显示；编号输入仍走 DLsite 精确抓取）**·**loadSteamStats()（Steam 动态：评价/在线/公告；applyWork 自动取一次）**·**loadSteamHeroImage(appId)（applyWork 对 Steam 源自动取一次本机 hero 图）**·loadById·searchBySample·loadCandidateCovers·loadSourcePage·setCandidatePage·storeIn()（**入库**；Steam 源不再追问本机 exe）·launchGame()（**游戏启动**；exe 方式无位置时先弹选 exe 框，Steam 方式直接跑 URL）·changeGamePath()（更换 exe）·**setGameLaunchMode('steam'\|'exe')（切换启动方式；切到 exe 且无路径时先选程序）**·refreshGameStatus()（查 exe 是否存在 + 是否运行 + 生效启动方式，页面轮询用）·**loadTags（载入时：edited 就用库里那份，否则抓取 tags 打底 + 旧自定义标签追加）·addTag / removeTag（只改编辑稿）·saveTags（整份写库）**·refreshCoversDir·ensureCoversDir·clearResult`。导出 `META_SAMPLES`。**缓存优先**（走 `libraryApi`）；调试页传 `refresh=true` 强制联网。`keyword` 不参与"打开作品"（`loadById` 不写回），搜索弹窗开/关都会清空它（见 §10）。**DLsite 候选分页每页 10 条**（30/10=3，第 1~3 页对应 DLsite 第 1 面…用到哪面才取哪面）；**Steam 候选最多 10 条、无分页**（缩略图用 `tiny_image` 直链，不进 `cache/`） |
| `stores/libraryStore.ts` | `library` | 游戏库列表：State `works·loading·error·covers(id→dataURL)·**removing（批量删除中）**·loaded`；Getters `count`；Actions `load(force=false)·loadCovers()·coverFor(work)·**removeMany(works)（批量删除：逐条走后端、成功的移出列表与封面缓存、失败的汇总进 error）**·refreshCover(workId, source)·has(workId, source?)`。**列表不按来源过滤**（`libraryApi.list()`，dlsite/steam 一起显示），封面 key 用 `来源:编号` |
| `stores/scanStore.ts` | `scan` | 游戏库「扫描添加」：State `source('steam'\|'folder')·mode('exe'\|'folder')·folderPath·scanning·picking·error·status·entries({name,detail,origin,path?,folderPath?,source?,appId?})·done·searching·searched（按条目计数）·entryResults(每条目 1~2 行) **+ `matches` = entryResults 摊平的 computed**·detailing·detailed·detailTotal·storingAll·storeProgress·storeTotal·storeError`；Getters `count·matchedCount（按条目去重，同一条目 Steam+DLsite 双命中算一条）·storableCount·selectedCount·allSelected`；Actions `setSource·setMode·pickFolder()（原生选目录，取消返回空串不动路径）·scan()·searchAll()·stopSearch()·retryMatch()·toggleSelect/All·storeAll/storeSelected（返回 {stored,failed}）·isStored()`。**扫描只读磁盘；扫完立刻自动搜索（3 路并发）**。**扫描结果记录来源**：`origin='exe'` 带 exe 绝对路径（入库直接记）、`origin='folder'` 带文件夹绝对路径（入库时后端进文件夹挑主程序）；批量入库时随 `libraryApi.storeIn(workId, {…}, match.source)` 写进 `library_works.game_path`。**2026-11-23 起每个本地条目同时搜两个来源**（`searchEntry` → `steamRow` + `dlsiteRow`）：Steam 行靠 `matchSteamApp(name)` 猜 appid（阈值 0.75 的编辑距离相似度），DLsite 行按原名搜第一个候选；**一行 = 一个来源的一份元数据**，各自 `source` / `entryIndex` / 勾选 / 入库（`ScanMatch` 因此多了 `source` 与 `entryIndex`）。Steam 库来源的条目只出 Steam 行（appid 已确定，不再按英文名去 DLsite 搜） |
| `stores/autoHeightStore.ts` | `autoHeight` | `<Transition>` 的高度自适应钩子（原 `composables/useAutoHeight.ts`，已按约定搬进 stores）：`duration`（读 `--motion-slow`）+ `enter·afterEnter·leave·afterLeave`；用法 `@enter="collapse.enter"`，`const collapse = useAutoHeightStore()` |

> ⚠️ 命名不一致：`Sidebar.ts`/`Theme.ts` 大写开头，`todoStore.ts`/`App.ts`/`metaStore.ts`/`libraryStore.ts` 混用 —— 新增 store 时 import 大小写要**手写对**。

### 4.4 services/ 封装层（不是 store，是所有后端调用的唯一入口）

| 文件 | 导出 | 说明 |
| --- | --- | --- |
| `frontend/src/services/todoService.ts` | `todoApi` | `getAll · getByID · create · update · delete · deleteMany(ids)`；用本地 `ensure<T>(value, message)` 把绑定层的 `T | null` 收成 `T`（`toggleComplete/getStats` 已于 2026-11-23 移除，后端接口仍在） |
| `frontend/src/services/metaService.ts` | `metaApi` | **永远联网**：`searchWorks(keyword, page=1)`（返回该页 30 条 + `pageCount`）` · fetchWork(keyword, workId?) · coversDir() · cacheCover(workId, coverUrl)`（后者返回 `cache/` 里的缩略图 data URL）` · searchSteam(term)`（Steam 商店候选 ≤10 条）` · matchSteamApp(folderName)`（文件夹名 → appid，`matched=false` 不是错误）` · steamStats(appId)`（评价/在线/公告）；同样用 `ensure()` 兜底 null |
| `frontend/src/services/libraryService.ts` | `libraryApi` | `fetch(keyword, workId?, refresh=false, source='dlsite')`（**只读**缓存优先；`source='steam'` 时 `workId` 是 appid） · `storeIn(workId, options?, source='dlsite')`（**入库，唯一写库入口**；`options` = `{gamePath?, gameFolder?, gameFolderName?}`） · `list(source?)`（空 = **所有来源**） `· remove(workId, source?) · coverDataUrl(workId, source?)` · `getTags(workId, source?) · setTags(workId, tags, source?)`（**标签整份读写**：`TagSet{tags, edited}`，编辑过程不写库）；导出 `LibrarySource='dlsite'|'steam'`、`LIBRARY_SOURCE_DLSITE/STEAM` 与 `LibraryMeta`（= `WorkMeta` + `cached/workId/cover*/gamePath`） |
| `frontend/src/services/scanService.ts` | `scanApi` | `steamApps()`（**Steam 源**：`SteamApp[]` = appid/名称/安装目录/库根，来自 appmanifest） · `steamFolders()`（旧的 `common` 子文件夹列表，仍保留） · `executables(root)`（返回 `{root, folders[], skipped[]}`） · `subfolders(root)`（返回 `Subfolder[]`） · `pickDirectory(title?)`（原生文件夹选择框，取消返回空串）；导出 `ScannedGame{name,detail,origin,path?,folderPath?,source?,appId?}` 与 `Subfolder` |
| `frontend/src/services/gameService.ts` | `gameApi` | `status(workId, source?)`（返回 `{path, exists, running, pid, launchMode}`，进入详情/轮询用） · `setPath(workId, path, source?)`（校验并写入 `library_works.game_path`） · **`setLaunchMode(workId, mode, source?)`**（写 `library_works.launch_mode`：`'steam'` / `'exe'` / `''`=默认） · `launch(workId, source?)`（返回 `{started, running, path, missing, message, mode}`；steam 方式跑 `steam://rungameid/<appid>`，exe 方式起本地进程） · `pickExecutable(title?)`（原生选 `.exe` 框，取消返回空串） |

> 绑定层返回的是 `map[string]any` / `{ [k: string]?: any }`，封装层用 `as LibraryMeta` 收窄（否则 `vue-tsc` 报 TS2345）。注意 `frontend/bindings/` 是**生成物、被 gitignore**：用 `wails3 generate bindings -clean=true -ts -i` 生成（实测 beta.24 `-ts -i` 下服务与 models 产出的是 **`.ts`**，只有个别 runtime 包是 `.js`；`tsconfig` 的 `exclude` 只覆盖 `**/*.js`）。新克隆仓库后先跑一次生成命令，缺文件会直接解析失败。

### 4.5 `components/meta/*` 积木（作品库与调试页共用）

| 组件 | 作用 |
| --- | --- |
| `MetaSection` | 卡片外壳（`section-header` + 图标/标题/`#actions` 插槽 + `bodyClass` 内容区），全站唯一的"卡片长相"定义。**没有默认插槽内容时不渲染 body**，并把上下内边距交给 header（`pt-4` → `py-3`）—— 用于「只有标题 + 右侧动作」的顶栏卡片，避免空 body 把卡片撑高 |
| `MetaSearchBar` | 搜索行（`input-theme` + `btn-brand`），受控 `v-model` + `@submit`，不发请求 |
| `MetaSampleChips` / `MetaMessage` | 快捷示例 chips / 一行提示（`tone=info|error|success|warning`，字面量表映射图标与颜色；`warning` 2026-11-23 加，用于「双源搜索时某一路失败但另一路有结果」） |
| `MetaCover` / `MetaCoverFigure` | 封面缩略图（`sm` 44px / `md` 140×105，失败退占位图，`src` 变化复位）/ 封面 + URL 标注 |
| `MetaKeyValueList` / `MetaTags` / `MetaSubsection` / `MetaTextBlock` | 两列键值表（`dense`/`scrollable`）/ 不可点标签云（`limit` 控制显示个数，超出折 `+N`）/ 小标题块 / 长文本块（`whitespace-pre-wrap` + 限高） |
| `MetaTagEditor` | 作品详情「标签」：**抓取标签与自定义标签合并成一份** `chip` 列表 —— hover 标签右上角浮出 14px 圆形 ×（`title`/`aria-label` 提示删除，hover 填红），末尾固定一颗同款「+」（点开原地变输入框：先保持原宽、随输入变宽；回车/失焦提交、空输入无视、Esc 取消）；删除/新增**只改 store 的编辑稿**，`tagsDirty` 时标题右侧出现「保存」，点它才整份写库。数据 `metaStore.tagList / tagsDirty / addTag / removeTag / saveTags`（后端 `work_tags` + `work_tag_edits`） |
| `MetaWorkOverview` / `MetaWorkDetails` | 详情上半（封面+名称+关键字段+标签+入库/启动按钮（含**启动方式下拉**，见 `MetaLaunchModeMenu`）+回执，`props` 含 `stored/gamePath/gamePathExists/launchMode('steam'\|'exe')/steamSource/running/launching/showTags`，`emits: download/launch/changePath/setLaunchMode`；`showTags=false` 时不出标签云 —— 作品库把标签统一交给 `MetaTagEditor`）/ 下半（summary + outline） |
| ~~`MetaLaunchModeMenu`~~（已删除，见 §0） | 原「启动方式」下拉菜单（2026-11-23 用户要求，取代原来的 chip 组）：是**分体按钮的右段**，紧贴 `游戏启动` 右边（左段 `btn-brand-split`、右段 `btn-brand-attach`，中间 `border-l-on-brand` 分隔线）。按钮上显示当前方式简称（`Steam`/`本地`）+ chevron，点开是 `role="menu"` 浮层（`fixed` + `Teleport to=body`，按触发按钮 rect 摆位、下方放不下就翻到上方、宽度不小于按钮），选项 `menuitemradio` + `aria-checked` + 勾选标记。键盘：↓ 展开、↑↓ 移动、Esc 关闭且焦点回按钮、Tab 离开关闭；点组件外/页面滚动也关闭。启动中/运行中禁用（与「更换位置」一致），只 emit `select`，写库仍走 `metaStore.setGameLaunchMode` |
| `MetaDownloadButton` / `MetaSavedNotice` / `MetaCoversPath` | **入库 / 游戏启动按钮**：未入库 = `入库`（无封面禁用）/ `入库中`；已入库 = `游戏启动`（play 图标；`running` 时「运行中」+ activity 图标并禁用；`launching` 时「启动中」）+ **Steam 源时右端接 `MetaLaunchModeMenu` 组成分体按钮**（`props` 新增 `steamSource`，`emits` 新增 `setLaunchMode`，向上转发给 `MetaWorkActions` → 页面）+ **exe 方式**下有位置时旁边一个轻量「更换位置」文字按钮（`i-lucide-folder-search`，运行中/启动中禁用）。**exe 方式记了位置但 exe 不存在时启动按钮禁用**（页面同时给提示）；**Steam 方式（`launchMode='steam'`）不需要本地 exe、不禁用也不显示「更换位置」**；路径选择沿用原系统文件框 / 成功回执 + `file://` 打开 / 封面落地目录（失败静默留空） |
| `MetaCandidateList` / `MetaPager` / `MetaRawJson` / `MetaEmptyState` | 候选行（`@select` 出编号）/ 分页条（2026-11-24 按 shadcn-vue 的 Pagination 重做，并**忽略旧逻辑**：上一页 / 页码组（当前页 outline、其余 ghost、MoreHorizontal 省略号，首尾固定 + 当前页左右各 1 页）/ 下一页，用 `n-button` 实现；不再有「共 N 个结果」与跳页输入，接口为 `page · pageCount` → `@change`；搜索结果列表与分页合并成一张卡片；添加游戏弹窗用 flex + `max-height:86vh` 撑满内容区，只有候选列表滚动、分页固定在卡片底部，弹窗整体垂直居中，不需要滚动窗口）/ 原始 JSON + 复制 / 空态 |

拆分层级：**store = 拿数据 → 积木 = 长什么样 → views = 放哪**。加新页面只需组合积木，不用再写请求。

### 4.6 TODO 组件公开接口

`TodoItem.vue`（`props: todo, isUpdating?, isDeleting?, selected?`；`emits: delete[id], select[id]`）—— 共 2 个 emit，**没有 `toggle` / `error` emit**（「完成」按钮已移除）。

`TodoList.vue` 持有：
- `selectedIds: number[]`、`visibleIds`、`allVisibleSelected`、`toggleSelected`、`toggleSelectAll`、`clearSelection`
- `armed: 'delete' | null`、`deleting`（只剩「删除所选」一个批量动作）
- 筛选面板 `isFilterOpen`，localStorage 键 `todo-tool-panel`（值是 `'filter'` 或 `''`）；`activeFilterCount`（优先级 + 时间）
- 两个关键 `watch`：`watch(selectedIds, disarm)`；**切换任一筛选 → `clearSelection() + disarm()`**（防止「看不见的选中项被批量操作带上」）

### 4.7 一次性反馈（2026-11-24 起改为 naive message；以下 sonner 细节仅作历史参考）

```
业务代码（组件）→ utils/toast.ts（toastSuccess / toastError / toastInfo / toastLoading / dismissToast）
                → vue-sonner 的 toast() 队列 → components/AppToaster.vue 里唯一的 <Toaster>（fixed 浮层）
```

- **为什么不是各页自己写**：原来 TODO 页把反馈条当成列表容器里的一个 flex 子项，出现/消失会把下面的列表卡片上下推动；而且只有一种字号、没有图标层次。改成浮层后**完全不参与布局**，样式与图标也统一。
- **外观只走 token**：`AppToaster.vue` 用 `[data-sonner-toaster].app-toaster[data-sonner-theme]`（比库自带主题选择器高一级）把 `--normal-bg/--normal-border/--normal-text/--border-radius` 映射到 `--color-bg-card / --color-border / --color-text-primary`，图标色按 `data-type` 取 `--color-success/danger/warning/info`；容器 `top-center`、`--width: 320px`（库默认 356，用 `:style` 覆盖，因为库把 `--width` 写成内联样式）、顶部偏移 42px 避开 30px 标题栏。
- **只用图标不换底色**（`richColors=false`）：与 Flat 取向一致（只用 1px 描边区分层级）。
- **顶端定时条**（2026-11-23 用户要求「为土司动画添加定时条在土司提醒顶端」）：每条 toast 的 `::before` 是一条 2px 语义色横线，用 `--toast-duration`（由 `utils/toast.ts` 按与 `duration` 相同的值写成内联 CSS 变量）从 `scaleX(1)` 走到 `scaleX(0)`，直观表示还剩多久自动消失；悬停/键盘聚焦时 `animation-play-state: paused`（sonner 同时也暂停关闭计时器）；loading 类型没有剩余时间概念，不显示条。⚠️ 变量必须和 `duration` 同源，否则会出现「条走完了提示还挂着」。
- **loading toast 的两个坑**（2026-11-23 实测 bug，用户报「移除游戏的提示没有自动消失」）：①sonner 在 `type='loading'` 时**不启动自动关闭计时器**；②**不要用同一个 id 把 loading 原地更新成结果** —— Toast 组件挂载时把剩余时间定死成 loading 的初值，更新 type 后计时器不会重启。正确做法是 `dismissToast(id)` 关掉 loading，再另弹一条新的结果 toast（新 id → 计时器与定时条都从头开始）。`LibraryPage` 的批量移除就是这么做的，并且延迟 300ms 才弹 loading（删一两款时不会闪一下）。
- **文案边界**：toast 只放"一闪而过、不需要回头对照"的结果（入库/移除/删除/复制/错误）；需要盯着看的信息（列表加载失败、入库详情、exe 已被移动）仍留在页面里的 `MetaMessage` / `store.error`。描述文字在 `utils/toast.ts` 里截到 120 字。
- **哪些动作会弹**：游戏库「入库」（`/works` 的入库按钮 = `WorksPage.handleStoreIn`；扫描批量入库 = `ScanPanel` 的 `storeAll/storeSelected`，用 `scanStore.storeMany` 返回的 `{stored,failed}` 汇总成一条，成功与失败各弹一次）、游戏库「移除」（`LibraryPage.handleBatchDelete`）、TODO 的单条/批量删除与复制、TODO 操作报错。

---

## 5. 交互模型（当前最终形态，逐项都是用户明确要求过的）

1. **行左侧 checkbox = 「选择」**。选中行呈 `border-brand bg-brand-light`。
2. **没有「完成」按钮/完成状态**（用户 2026-11-23 要求删除）：行右侧只剩删除按钮；`todos.completed` 列仍存在于库里，但界面不读不写，也不再按完成状态筛选。
3. **单条删除 = 连点两下**：第一下进待确认态（`i-lucide-check` + 红底白字 + `scale-110`），第二下才 `emit('delete')`；`@mouseleave`/`@blur` 自动撤销；提交中显示 `i-lucide-loader-circle spin`，另有 `STUCK_TIMEOUT_MS = 4000` 兜底复位 + `watch(() => props.isDeleting)` 回落。
4. **一个批量按钮**（删除所选）也是**连点两下**，仅当 `selectedIds.length > 0` 时出现，带数量徽标。
5. **切换筛选清空所有选择**（理由见 4.4）。
6. **统计栏**（`TodoStats`）改为「总计 + 高/中/低」按优先级计数，数据由传入的 `filteredTodos` 前端算出，不再调后端 `GetStats`。
7. **行内按钮静息不透明度 `opacity-75`**（不是 0.4）：这是两套主题下同时满足「有意义图标 ≥3:1」的最小值 —— 浅色 3.49~4.01、深色 4.03~4.49；0.4 时仅 1.78:1。提亮交给 `group-hover:opacity-100` / `group-focus-within:opacity-100` / `focus-visible:opacity-100`。
8. **批量按钮 `.bulk-btn`**：默认透明底 + danger 描边/文字 → hover 填 danger + 反色文字 → 待确认 `.is-armed` 填 danger（**不额外加外描边**）。反色文字走 `--bulk-on-accent`。数量徽标 `color-mix(in srgb, currentColor 22%, transparent)`。
9. **关闭行为**：标题栏 X 按 `appStore.closeAction` 分流 —— `ask` 弹出标题栏内小面板（**刻意不是模态框**，因为这是两个不同意图的二选一），选项只有「隐藏到托盘」/「退出程序」+「记住我的选择」；`hide`/`close` 直接执行。独立的最小化按钮**不受影响**。
10. **托盘左键 = 只唤回、永不隐藏**（`main.go:102-112`）：`win.IsVisible()` 时直接 `return`；否则 `tray.PositionWindow(win, 0)` + `Show()` + `UnMinimise()` + `Focus()`。不可用 Wails 默认的 `ToggleWindow`（它的语义是「显示则隐藏、否则显示」，会导致手滑再点一下窗口又没了）。隐藏统一走标题栏 X 或设置里的「隐藏到托盘」。
11. **点击卡片 = 复制「标题 + 描述」**（用户要求）：`TodoItem` 根 `<div>` 绑 `@click`，拼成 `标题\n描述` 后 emit `copy`，由 `TodoList.handleCopy` 走 `utils/clipboard.ts` 的 `copyText()`（Clipboard API 失败退 `execCommand`）并弹 toast「已复制到剪贴板」（复制的原文作为第二行小字，超 120 字截断）。**刻意不加 `cursor-pointer`** —— 指针保持默认，整行不暗示可点；选择框加 `@click.stop`、行内按钮本来就有 `.stop`，点它们不会误复制。
12. **一次性反馈统一是浮层 toast**（vue-sonner，2026-11-23）：入库、从游戏库移除、TODO 删除/复制/报错都走 `utils/toast.ts`；位置 `top-center`、320px 宽、图标按类型取语义色、不换成整块彩色底，**顶端有按停留时长走的定时条**（loading 除外）。长时间停留的信息不进 toast（见 §4.7）。

---

## 6. 设计系统（三层 token）

层级：**`global.css` CSS 变量（唯一色值）→ `uno.config.ts` rules（变量→类名）→ shortcuts（组合）**。`uno.config.ts` 里**不允许出现硬编码色值**。

### 6.1 已核验的对比度定稿（WCAG AA 4.5:1，均为实测）

| 组合 | 值 | 对比度 |
| --- | --- | --- |
| 浅色 `text-primary` | `#0f172a` | 17.9:1 |
| 浅色 `text-secondary` | `#334155` | 10.4:1 |
| 浅色 `text-tertiary` | `#475569` | 7.6:1 |
| 浅色 `text-muted` | `#64748b` | 4.8:1（仅辅助小字） |
| 浅色品牌（文字/填充统一） | `#0f766e` | 对白 5.47:1 / 对底 5.25:1 |
| 浅色 success / warning / danger | `#15803d` / `#b45309` / `#dc2626` | 5.02 / 5.02 / 4.83:1 |
| 深色品牌 | `#2dd4bf` | 9.21:1 |
| 深色 success / warning / danger | `#4ade80` / `#fbbf24` / `#f87171` | 9.84 / 10.27 / 6.20:1 |

**关键坑（不要重犯）**：
- 设计系统推荐的 `#0D9488`(teal-600) **作文字只有 3.74:1、白字压其上也只有 3.74:1**，两者都不达标 → 文字/填充统一用 teal-700。
- **深色下 success/brand/danger 都是亮色，白字压上去只有 1.74 / 1.86 / 2.77:1** → 必须翻转成深色文字：`--color-text-on-brand:#04211f`、`--color-text-on-success:#04211f`、`--color-text-on-danger:#2a0a0a`；`--color-check-mark` 同步为 `#04211f`。
- `html,body,#app` 的 `background:#101418` 是**启动底色**（与窗口 `BackgroundColour` 一致，避免启动闪白），**不是主题色**，两套主题下都保持深色；界面底色由 `#app` 内的 `bg-primary` 覆盖。

### 6.2 动效 token（按「变化幅度」分档，不按元素类型）

`--motion-fast:160ms`（颜色/透明度）· `--motion-base:220ms`（位移/缩放）· `--motion-slow:280ms`（展开收起/进出场）；缓动 `--ease-enter:cubic-bezier(0.16,1,0.3,1)`（进入减速）/ `--ease-leave:cubic-bezier(0.4,0,1,1)`（离开加速）。`prefers-reduced-motion: reduce` 下全部压到 0.01ms。

### 6.3 图标尺寸 token（替代混用的 `text-sm`/`text-xl`）

`icon-xs 14px` · `icon-sm 16px` · `icon-md 20px` · `icon-lg 24px` · `icon-xl 32px`（每个含 `w/h` + `text-[Npx]` + `leading-none`）。

### 6.4 半透明 / 颜色类只能用**已定义的具体类名**

rules 是**精确字符串匹配**：`bg-secondary/50`、`text-primary/70`、`w-full-16px`、`will-change-[width]` 全部 **unmatched 且静默丢弃**。需要半透明请用 `bg-secondary-soft` 这类已定义类。

### 6.5 按钮体系（2026-11-23 按 shadcn-vue 重做 → 2026-11-24 已迁到 naive-ui，见 §0）

用户要求「button 默认样式修改参考 shadcn-vue（unocss 级别）」，落地方式是把 shadcn 的**形状与状态**照搬，颜色仍全部走本项目 token：

| shortcut | 内容 | 对应 shadcn |
| --- | --- | --- |
| `btn` | 基座：`inline-flex` 居中 · `rounded-md`（6px）· `text-sm font-medium` · `transition-colors duration-200` · `outline-none` + `focus-visible:ring-[3px] focus-visible:ring-brand-soft` · `disabled:pointer-events-none disabled:opacity-50` | `Button` 的 base |
| `btn-brand` | `btn` + `control-h` + `px-4` + `bg-brand text-inverse shadow-theme hover:bg-brand-hover` | `variant="default"`（实心主色 + shadow-xs） |
| `btn-ghost` | `btn` + `control-h` + `px-4` + 透明底、`hover:bg-hover` | `variant="ghost"` |
| `btn-brand-split` | `btn-brand` + `!rounded-r-none`：分体按钮**左段**（主操作） | `ButtonGroup` 左段 |
| `btn-brand-attach` | `btn-brand` + `!rounded-l-none !px-2.5 !gap-1 border-l-on-brand`：分体按钮**右段**（下拉触发） | `ButtonGroup` 右段 |

要点：①基座里**不含高度与内边距** —— 高度必须由 `control-h` 提供（与 `input-theme`、`TodoInput` 外壳同高）；②基座里**不含阴影**（shadcn 的 shadow-xs 只挂在 default 变体上）；③圆角覆盖必须用 longhand + `!important`（`rounded-md` 是简写，谁胜取决于合并顺序）；④聚焦环是 3px、50% 透明 —— 颜色来自 `--color-ring-soft`（`color-mix` 出来，两套主题各一份，自定义属性的 `var()` 替换发生在声明所在元素上，只写 `:root` 会让深色主题拿到浅色计算结果）；⑤分体按钮的分隔线是 `border-l-on-brand`（`--color-divider-on-brand` = 30% 反色），必须排在 `border-none` 之后，否则 `border-style:none` 会把左边框一起清掉。

---

## 7. 必须遵守的工程约定（踩过坑才有的）

### 7.1 依赖与版本

- **`frontend/package-lock.json` 必须入库**（`vite` 锁 `~8.2.2`）。忽略它会让 clone 重新解析到 vite **8.3.0**，触发 UnoCSS HMR 回归（上游 [unocss#5331](https://github.com/unocss/unocss/issues/5331)）：**动态导入的路由页（设置/TODO）样式不生效，首页正常**。症状判定：`node -e "console.log(require('./frontend/node_modules/vite/package.json').version)"`。
- **`go.sum` 被 `.gitignore:18` 忽略且不在版本控制内** —— 首次构建需联网由 `go mod tidy` 重新生成。
- **依赖变更（2026-11-24）**：新增 `naive-ui@^2.45.3`（用户点名引入，全站控件换它），**移除 `vue-sonner`**；除此之外没有任何新的前端依赖。控件外观的唯一来源是 `src/styles/naiveTheme.ts`（不再有 btn-*/input-theme 这类 shortcut 要三处同步）。`go.mod` 的模块集合也没变，但 `golang.org/x/sys` 因 Steam 扫描要用 `windows/registry`，从 indirect 提升为 **direct require**（仍是 v0.47.0，本地早已存在，无新下载；`go mod tidy -diff` 为空）。
- **桌面端离线优先**：不引联网字体（Plus Jakarta Sans 只在 font-family 里作为可选首选项，实际回退到系统栈），图标走 `presetIcons` 构建期内联。

### 7.2 UnoCSS 七类静默失效（本项目的核心坑，改样式前必读）

1. **`border-color` 不能塞简写值**：`{'border-color':'1px solid X'}` → `.border-x{border-color:1px solid X}` 非法 CSS、整条作废。
2. **只设 `border-color` 等于没有边框**：preflight 把 `*,::before,::after` 重置为 `border-width:0;border-style:solid`；若元素另有非零 `border-width`，`border-style` 会退回 UA 初始值 **`outset`** → 渲染成 1.6px 的 3D 伪立体边框。**结论：每条 border-* 规则必须自带 width+style+color，一律用 `border` 简写。**
3. **`border` 工具类只输出 `border-width`**：`<button>` 的 UA 默认 `border-style` 是 `outset` → 按钮上必须写 `border border-solid`。
4. **同一元素上两条改同一属性的类只保留一条**：模板同时写 `border-l-theme-light` 与 `border-l-brand` 时后者**根本不会生成** → 基础样式放组件 `<style scoped>`，UnoCSS 只负责 `group-hover:border-l-brand`。
5. **shortcuts 输出在 rules 之前；rules 之间按字母序（同权重后者胜）**：`.text-danger` 排在 `.text-white` 之后会压掉白字 → **同一属性只由一处决定**。
6. **过长的 shortcut 链会被静默截断**（十余条 utility 串成一个 shortcut 后只剩两条，`width`/`appearance`/焦点环全丢）。
7. **没有 `placeholder-<color>` 工具类**：`placeholder-tertiary` 会被静默丢弃，只在构建时留一行 `[unocss] unmatched utility "placeholder-tertiary" in shortcut "input-theme"` 警告，元素上完全看不出异常。正确写法是 `placeholder:text-tertiary` 变体（`input-theme` 已修）。

**单侧边框规则的正确写法**：必须**显式把其余三边归零**，否则整框简写会重置四边。现有三条：`border-b-theme-light`、`border-r-sidebar`、`border-l-brand`。

**输入框有两套实现，CSS 必须成对改（改一个就改另一个）**：

| | `input-theme`（`uno.config.ts:190` shortcut） | `TodoInput.vue` 复合输入框 |
| --- | --- | --- |
| 结构 | `<input class="input-theme">`，盒模型全在 shortcut 里 | 外层壳 `div` 承边框/圆角/底色，内层无边框 `input` |
| 边框 | `border-theme` + `focus:border-brand` | 壳 `border-theme` + `focus-within:border-brand` |
| 用于 | 搜索框（`components/meta/MetaSearchBar.vue`，作品库与调试页共用） | TODO 添加框（`components/todo/TodoInput.vue`） |

**控件高度统一走 `control-h`**（`uno.config.ts:161`，`h-[2.2rem]` = **35.2px**）：`btn-brand`（`uno.config.ts:164`）、`input-theme`、`TodoInput.vue` 的外壳三处都挂它，因此三者同高。⚠️ **挂了固定高度就不要再写纵向 padding** —— 总高已由 `control-h` 决定，`py-*` 不会再改变盒高，只会把内容挤出盒子（`TodoInput.vue` 内层 input 因此不带 `py-*`，垂直居中交给外壳的 `items-center`）。历史教训：输入框曾用 `py-2.5`、按钮用 `py-2`，高度由「内容高 + 纵向 padding」撑出，实测 41.6px vs 35.2px，同一行 form（`items-end`）底部对齐时顶部错开 6.4px。

两者实测盒模型**逐项相同**：`border-radius: 10px`、`padding: 10px 12px`、`font-size: 14px`、`line-height: 20px`、`border: 0.8px solid var(--color-border)`、总高 **35.2px**；聚焦时边框都变 `var(--color-brand)`。**任一项（内边距 / 圆角 / 字号 / 行高 / 边框色 / 聚焦色 / 高度）都必须在两处同步修改**，否则两个输入框的圆角与高度会分叉。

**提交按钮统一走 `btn-brand`**（`uno.config.ts`）：TODO 的「添加」与作品库/调试页的「搜索」类串同为 `btn-brand flex items-center gap-1.5 shrink-0`（搜索按钮在 `components/meta/MetaSearchBar.vue`），TODO 额外带 `disabled:opacity-50 disabled:cursor-not-allowed`（`btn-brand` 基座已含 `disabled:pointer-events-none disabled:opacity-50`，调用处的写法保留即可）。`HomePage.vue:10` 也用它 —— 即全站按钮已统一到这一套 shortcut（`btn` 基座 + `btn-brand` / `btn-ghost` / 分体两段，见 §6.5），不要再手写 `rounded-lg px-4 py-2 ...`。

### 7.3 布局高度链（缺一环就整页溢出）

`presetUno` **不带** `border-box` reset → `global.css:15-19` 手写 `*,*::before,*::after{box-sizing:border-box}`；再加 `html,body,#app{height:100%;overflow:hidden}`，沿链 `h-screen` / `min-h-0` / `flex-1` / `overflow-hidden`。实测：加 `border-box` 后 root 高度 1048→1018，列表底部从「超出视口 36px、最后一条滚不到」变为完整落在视口内。

滚条：全局 `::-webkit-scrollbar{display:none!important}` 隐藏；局部用 `.scrollbar-theme` 恢复（**必须也带 `!important`**，因为「带 `!important` 的声明永远胜过不带的」，与选择器权重无关）。滚动条伪元素只能写在纯 CSS 里 —— UnoCSS utility 无法生成 `::-webkit-scrollbar` 这种伪元素选择器。

### 7.4 代码约定

- Go 方法**首字母大写**才会导出到前端绑定。
- **`frontend/bindings/` 只读**，改 Go 签名后重新生成。
- Go 返回指针/切片 → 前端绑定类型是 `T | null`，统一在 `services/todoService.ts` 的 `ensure()` 兜底。
- 所有 `TodoService` 方法内建 3s `context.WithTimeout`，并区分 `Canceled` / `DeadlineExceeded` 文案。
- 前端 store 文件命名不统一（见 4.3），import 大小写手写。
- 用户明确要求：**`doc/need-fix.md` 与 `doc/need-change.md` 可以清空内容，但文件必须保留在版本控制中**（当前均为 0 字节）。

---

## 8. 构建 / 验证工作流

### 8.1 命令

| 命令 | 说明 |
| --- | --- |
| `wails3 dev` | 开发模式（装依赖 → 生成绑定 → 生成图标 → 构建后端 → Vite 端口 **9245** `strictPort` → 开窗口） |
| `wails3 build` | 生产构建 → `bin/weedybox.exe` |
| `wails3 package` | NSIS 安装包 → `bin/WeedyBox-amd64-installer.exe` |
| `wails3 task <name>` | 等价 `task <name>`（本机未装独立 `task` CLI） |
| `cd frontend; npm run dev` | 只跑前端（**无后端**，只能调样式） |
| `cd frontend; npm run build` | `vue-tsc && vite build --mode production` |
| `cd frontend; npm run check:tokens` | `node scripts/check-unocss-tokens.mjs` |
| `cd frontend; npm run check:build` | `node scripts/check-build-css.mjs`（**先 `vite build` 再跑**） |
| `go build ./...` · `go vet . ./internal/...` | Go 侧静态检查 |
| `npx vue-tsc` | 前端类型检查 |

### 8.2 两个回归闸门（改动样式后必跑）

- `frontend/scripts/check-unocss-tokens.mjs`：逐个 `uno.parseToken` 打印解析结果；刻意保留 `bg-secondary/50`、`will-change-[width]` 两个「已知应失效」样例作对照。
- `frontend/scripts/check-build-css.mjs`：断言构建产物 —— 关键类是否生成（约 54~60 个，**含分体按钮的 `btn-brand-split` / `btn-brand-attach`**）、无 `border-color:1px solid` 污染、17 项布局链与无障碍断言、**边框规则自带三件套**、**单侧边框置零其余边（含分体按钮分隔线 `border-l-on-brand`）**、「同侧边框类叠加」扫描、「border 缺 border-style」扫描、**按钮聚焦光环组**（3px ring 宽度、`--un-ring-color:var(--color-ring-soft)`、`--color-ring-soft` / `--color-divider-on-brand` 两个 color-mix token 真的生成）、**搜索框/表单组**（`input-theme` 底座 + 聚焦两条 border 规则、`::placeholder{color:var(--color-text-tertiary)}` 存在、反向断言无 `.placeholder-<color>` 无效类、`.input-theme` 含 border 的规则必须恰好 2 条）、**控件高度一致性组**（`control-h`/`input-theme`/`btn-brand` 三条规则都必须含 `height:2.2rem` 且**不得带纵向 padding**；扫 `TodoInput.vue` 源码确认外壳挂了 `control-h`、内层 `input` 没写 `py-*`）。
  注意：**断言压缩后的 CSS 不能拿带 `{}` 的整串去 `includes`** —— minify 会把选择器分组写成 `.input-theme:focus,[input-theme=""]:focus{…}`，必须按「规则块」切分（`css.match(/[^{}]*\.input-theme[^{}]*\{[^}]*\}/g)`）再判断。
- 已知噪音：产物 JS 里的 `⚠️` 来自 `@wailsio/runtime` 的控制台提示，**不是界面 emoji**。

### 8.3 调试手段与测试教训

- **用 chrome-devtools MCP 读 `getComputedStyle` 的计算样式**，不要只看源码；可一次确认有无 emoji、图标尺寸是否精确命中、token 是否按主题解析、间距节奏。
- **Vite dev server 会提供陈旧的 CSS/模块转换缓存**（曾服务 9528 字符 vs 实际 11942，导致浏览器一直解析旧 token、反复误判「代码没生效」）→ **删 `frontend/node_modules/.vite` 并重启 dev server**。判断手段：枚举 `document.styleSheets` 命中的规则 + 直接 `fetch` dev server 该文件比对长度。
- 注入测试数据：
  ```js
  const pinia = document.querySelector('#app').__vue_app__.config.globalProperties.$pinia
  const st = pinia._s.get('todo')
  // 测完必须清空，避免污染用户的 SQLite：
  st.$patch({ todos: [], stats: { total: 0, completed: 0, pending: 0 } })
  ```
- 反复踩到的测试坑：Vue DOM 更新异步（派发事件后必须 `await`）；合成事件测 `:checked` 会与手动赋值叠加导致计数翻倍；`:hover` 伪类无法用合成事件触发；颜色过渡中途取样会读到中间值（已填充却读到 `transparent`），需等 ≥500ms。
- **沙箱限制**：`spawn EPERM`（不允许 Node 捕获子进程输出）；PowerShell 中文输出有 mojibake（判断文件内容必须用 read 工具）；PowerShell 不支持 `\u{...}` 正则与 here-string `<<<`。

---

## 9. 已知问题与文档陈旧点（接手前先看这里）

### 9.1 未完成 / 待办

| 位置 | 状态 |
| --- | --- |
| `frontend/src/components/settings/LanguageSelect.vue` | 仅占位，i18n 未实现 |
| `frontend/src/components/todo/TodoItem.vue` | 描述**只读展示，尚不能编辑**（后端 `Update` 已支持 `description`） |
| `frontend/src/views/test.vue` | 薄壳（只渲染 `MetaApiTestCard`），DLsite **调试页**（路由 `/testing`，见 §3.3） |
| `frontend/src/views/WorksPage.vue` | 作品库：**2026-11-24 第二次重排（见 §0.5）**——`MetaWorkHero` 大背景图 → `MetaWorkTopBar` 动作栏 → 下划线 tab（作品信息 / 游戏统计 / 启动配置 / 编辑 / 评价）；`MetaWorkSidebar` 已删除；「编辑」页用 `MetaWorkEditorForm` 改全部元数据 + `MetaTagEditor` 改标签，都点保存才写库。缓存优先。**侧栏无入口**，只从游戏库点封面进入（用户要求）。搜索入口在游戏库顶栏「添加游戏」（弹窗里的「手动搜索」）。**尚未做**收藏、列表分页、批量入库、历史记录；从游戏库进来后没有显式「返回游戏库」按钮（靠侧栏点游戏库） |
| `frontend/src/views/LibraryPage.vue` | 游戏库（路由 `/library`）：网格 + 点封面跳详情 + 顶栏「添加游戏」按钮（打开「添加游戏」窗：扫描添加 / 手动搜索，候选每页 10 个）。**卡片上没有任何操作按钮**：顶栏「批量操作」进入选卡模式后，操作栏（全选 / 删除所选，连点两下确认）展开在刷新按钮所在的卡片里，删除走 `libraryStore.removeMany`。点封面进 `/works` 后可「游戏启动」（Steam 源默认 `steam://rungameid/<appid>`，也可切本地 exe）。**尚未做**游戏库列表排序/分页、在网格卡片上直接启动/打开所在文件夹、封面缺失时自动补下 |
| `internal/service/library_service.go` | 游戏库只跑了离线单元测试（`internal/database/library_repo_test.go`），**Wails 端的真实接线未实机验证** |
| `internal/service/getMeta/*_test.go` + `testdata/` | 本工作区当前是**已从磁盘删除（未提交）**状态（属用户「裁剪」清理），恢复：`git checkout -- internal/service/getMeta` |
| `stores/scanStore.ts` | **Steam 库来源的扫描只出 Steam 行**：appid 已确定，未再按英文名反查 DLsite 对应作品（避免误配；要 DLsite 数据请用「手动搜索」双源）。扫描结果按「扫描条目 × 来源」拆行，**同一文件夹出现 Steam + DLsite 两行是特性**（各存一份元数据，`UNIQUE(source, work_id)` 不冲突） |
| `internal/service/todo_service.go:315` | 一行 `// bug need-fix` 注释（`GetStats`），逻辑本身正确，属陈旧标记 |
| `main.go:62` | `Description` 仍是 `"A demo of using raw HTML & CSS"` |
| `build/config.yml` | 仍是模板（`companyName="My Company"`、`productName="My Product"`、`version 0.0.1`） |

### 9.2 需**实机**确认（无头浏览器验证不到）

- **托盘**：托盘图标是否出现在通知区域、左键是否只唤回不隐藏、窗口已显示时再点托盘应无反应、右键菜单「显示主窗口/退出程序」是否可用 —— 浏览器观察不到 Windows 通知区域。
- **DLsite 测试页（`/testing`）**：浏览器里没有 Wails 后端，`/wails/runtime` 一律 404，因此**所有真实调用都必须实机验证**（用户明确说「由于网络限制因素 不需要测试 完成后我手动测试」，故本地未跑任何联网调用）。需要确认的点：
  1. 按 `RJ01721474` 精确抓取 → 元数据是否与页面一致、`coverUrl` 是否已是 `_img_main.jpg`（560×420 那一档，不再是 240×240）；
  2. 封面在 webview 里能否直接显示（`Referer` 已设为 `https://www.dlsite.com/`，`<img referrerpolicy="no-referrer">`）；
  3. 点「入库」是否真的在**工作目录下**生成 `covers/`，文件名形如 `RJ01721474_cover.webp`（DLsite 有同名 webp 档位时应优先拿到 webp；没有 webp 时回落到 `.jpg` 等原格式）；卡片里显示的是 `absPath`，以那个为准；入库成功后按钮应立刻**变成「游戏启动」**（不再是「已入库 + 禁用」），且**退出再进 `/works` 仍是「游戏启动」**（此时靠后端注入的 `coverAbsPath`，不是会话内缓存）；
  4. 模糊搜索（如 `エルフ パラダイス`）是否返回候选列表；
  5. 「打开已保存的图片」的 `file:///` 链接在 Wails webview 里是否可用（**把握最低的一项**，不可用不影响主流程）；
  6. **搜索候选缩略图**：首次搜索会把图缓存到工作目录的 `cache/`（目录自动创建），列表里的小图应能显示；删掉 `cache/` 再搜一次仍能显示（重新下载）；结果多于 10 个时出现分页条，翻到哪页才下哪页的图（看 `cache/` 文件数增加）；
  7. **退出清缓存**：正常退出应用后 `cache/` 应为空（目录还在），`covers/` 里的入库封面不能被删；
  8. **只有点「入库」才进游戏库**：搜到的作品不应出现在 `/library`；点「入库」后 `covers/` 生成图、库里多出一张带封面的卡片，`/works` 的按钮变成「游戏启动」。**入库时就记录 exe**：扫描添加「按 exe 识别」直接记那个 exe；「按文件夹名 / Steam 库」由后端进文件夹挑主程序（排除 UnityCrashHandler64、优先 Game.exe、否则取与文件夹名匹配最多的）；手动搜索入库会先弹 `.exe` 选择框，**取消也能正常入库**（位置留空）。再点「游戏启动」：已有位置就起进程；没位置/之前取消过则这时弹选择框，选完存库并启动。**启动后按钮几秒内变「运行中」并禁用；手动关掉游戏后（≤5 秒轮询）变回「游戏启动」。把 exe 挪走/删掉后再进详情：按钮应变为禁用并显示"游戏程序已被删除或移动了位置，请点「更换位置」…"，用「更换位置」重选后可正常启动并清除提示。**
  9. **设置页「清理缓存」**：进入设置页应显示 `cache/` 当前占用（先搜几次让 `cache/` 里有图）；点「清理缓存」后占用归零、`cache/` 里文件消失、`covers/` 里的入库封面不受影响；没有缓存时按钮应禁用并显示「当前无缓存」。

### 9.3 `doc/*.md` 的已确认错误（本文件 §5/§6/§7 优先于它们）

- `doc/styles.md`「无效工具类」表**有两行是错的**：`top-50%` 与 `right-2` **实际合法生成**（旧构建产物 CSS 里也有这两条规则）。真正 unmatched 的是 `w-full-16px`、`bg-secondary/50`、`text-primary/70`、`border-theme/50`、`will-change-[width]`。
- `doc/frontend.md` 称 `desc-hint` 是 textarea 的 flex 同级元素 —— 模板里**根本不存在**（只在 `<style scoped>` 里，是死代码）。
- `doc/frontend.md` 提到 MainLayout 有 `overflow-y-auto` —— 实际 `frontend/src/layouts/MainLayout.vue:6` 只有 `overflow-hidden`。
- `doc/conventions.md` 称 `global.css:8` 的 body 背景硬编码深色 —— 实际在 `global.css:29`（且是设计上的启动底色，非缺陷）。
- `doc/conventions.md` 与 `README.md` 称数据库走 `modernc.org/sqlite` —— 实际走 **Wails 内置 `pkg/services/sqlite`**，modernc 只是传递依赖（indirect）。
- `doc/conventions.md` 的「扫描范围」列表里 `vendor/`、`frontend/build/`、`build/{android,ios,linux,docker}/`、`.wails/`、`wails-build/`、`logs/`、`.vscode/`、`.idea/`、`dist/` **本机均不存在**；实际被忽略且存在的只有 `node_modules/`、`bin/`、`.task/`、`frontend/dist/`、`frontend/node_modules/`、`frontend/bindings/`、`build/darwin/`、根 `WeedyBoxDB.db`、`go.sum`。
- `README.md` 与 `doc/conventions.md` 称 `go.sum` 被忽略 —— **正确**（早前「go.sum 已被跟踪」的判断是错的，实测 `git ls-files go.sum` 为空）。

---

## 10. 决策日志（为什么是现在这样，改动前请先读）

| 决策 | 理由 |
| --- | --- |
| 关闭行为只有 `ask/hide/close`，**没有 `minimise`** | 标题栏左侧已有独立最小化按钮，放在关闭行为里是重复入口，且让「关闭」这个决定含糊 |
| 接口测试服务**不做数据加工**，`work` 原样返回 `DLsiteWork` 的 json | 测试页的用途就是「对着 DLsite 页面核对解析结果」。任何二次加工（改字段名、拍平、只挑几个字段）都会让「后端解析错了」和「展示层挑错了」变得无法区分。格式化交给前端 `JSON.stringify(raw, null, 2)` |
| `covers` 用**相对路径**而不是 `os.UserConfigDir()` 之类的绝对路径 | 用户原话「保存在工作区目录下的 preImg 使用相对路径」（2026-11-16 目录改名 `covers`，策略不变）。代价是落点随进程工作目录变化（开发态是仓库根），所以返回值同时给 `path`（相对）与 `absPath`（绝对），界面上直接显示 `absPath`，不留"到底存哪了"的歧义 |
| 下载封面**不覆盖**已有文件，同名加 `-2/-3` 后缀 | 同一部作品可能被反复入库（库记录被删后重收、调试页重复点），覆盖会把上一次下到的图悄悄冲掉；`O_EXCL` 还能顺带防止两次并发请求写同一个文件 |
| 托盘左键**只唤回不隐藏** | Wails 默认 `ToggleWindow` 是「可见则隐藏、否则显示」，会让「唤回」后手滑再点一下窗口又消失 |
| 关闭询问用**标题栏内小面板**而非模态框 | 这是「两个不同意图的二选一」，用模态打断不值得 |
| 行左侧 checkbox 改为**选择**、完成移为行右侧独立按钮 | 原设计里「勾选」同时承担完成与选择，语义冲突；且旧版勾选框 `w-0 h-0` 完全隐藏、键盘不可操作。**2026-11-23：行右侧的完成按钮已按用户要求整体删除**，只剩删除按钮 |
| 批量按钮**连点两下确认**、且 `armed` 互斥 | 用户明确要求「不展开面板、直接执行但需连点两下」；互斥是防止「以为在确认完成、其实点的是删除」（2026-11-23 后只剩删除一个批量动作，`armed` 仍保留待确认语义） |
| 「选择范围」面板**整套移除** | 用户要求批量按钮直接执行、不展开面板 |
| 弃用 `go.felesatra.moe/dlsite`，自写 `internal/service/getMeta/getFromDlsite.go` | 该库的 `Work.Name` **恒为空**：它只找 `#work_name` 里的 `<a>`，而现代码没有 `<a>`；且它用裸 `http.Get` 无超时、不发成人确认 Cookie、`TrackList` 因选择器已失效也恒为空。自写版与直接 `net/http`+`goquery` 相比没有额外依赖，反而能删掉一整个模块树 |
| DLsite 解析器用**离线 fixture** 测试 | 本机 `www.dlsite.com` DNS 被污染（解析到 Meta 的 IP 段），真实抓取每次卡 21 秒还可能失败；把两个真实页面存进 `testdata/` 后 `go test` 0.3 秒稳定通过，且改解析代码后立刻知道有没有弄坏 |
| **切换筛选清空选择** | 筛选会换掉可见条目，残留旧选择会让「看不见的选中项被批量操作带上」（2026-11-23 后没有「已完成/取消完成」的语义反转，但"看不见的选中项"仍是误删风险） |
| 「取消完成」用 warning 橙 | 与「推进」（success 绿）在语义上区分「回退」。**2026-11-23 随完成功能一并移除** |
| 行内按钮静息 `opacity-75` 而非 0.4 | 0.4 时图标对比度仅 1.78:1；0.75 是两套主题同时 ≥3:1 的最小值 |
| 优先级溢出修复：`TodoStats` 改 `flex-1 min-w-0` | 原 `flex-shrink-0` 让统计栏（约 271px）与优先级选择器（172px）都拒绝收缩，容器 <455px 时溢出而统计栏自身的 `flex-wrap` 永远得不到触发；实测溢出区间 380~452px、越界最多 75px，修复后 320–660px 全范围零溢出 |
| 描述块基础左竖线放 scoped 样式、hover 变色放 UnoCSS | 同一元素两条改 `border-left` 的类会被 UnoCSS 只保留一条（见 §7.2.4） |
| 游戏库批量操作做成**右侧浮空长条卡片**（宽 56px，`top:112px; right:3px`） | 用户连续三轮定稿：①「浮空于界面上卡片 避免重叠 在现有右侧空处弹出 缩小 ui」②「长条形界面 在右侧显示 不遮盖现有ui」「不挤压现有ui」③「太宽了 遮挡住了ui」→「在游戏库字样下方 游戏卡片右侧显示 并且使用长条浮空设计 作为独立卡片弹出」。所以它 **`position: fixed` 完全不参与布局**（不挤压网格），宽度压到 56px 只放图标（文字走 `title`/`aria-label`），高度随内容、圆角+描边+阴影像一张独立卡片；位置对准器：`top` 取「游戏库」顶栏卡片下方，`right:3px` 贴窗口右侧留白。**不要再给它加占位 padding/挤压网格** |
| 2026-11-23：删除网格末尾的 `.lib-add` 圆形加号，「添加游戏」按钮并入顶栏「游戏库」卡片右侧（已入库 N 款 / 刷新 / 添加游戏 / 批量操作） | 用户原话：「添加游戏ui不再使用原先圆形ui 去除 添加到游戏库字样卡片中」——入口跟标题在一起更符合"在游戏库里加东西"的心智，也避免网格末尾的空格子占位 |
| `/works` 用 Steam 商店页式排版（hero 大图 + 左信息栏 + 右主区），背景图按来源取 | 用户原话：「work界面ui排版变更使用如图所示排版 背景图如果是steam源使用本地扫描函数扫描含hero字样图片 如果没有就使用api获取的预览图填充 小图使用library卡片cover图像 DLsite源使用第二张预览图作背景(第一张作为游戏卡片cover)」。落地：`MetaWorkHero`（背景 + 名称 + 标签 + 2:3 竖版封面）、`MetaWorkSidebar`（作品信息/游戏/快速链接）、右主区（动作条 + Steam 动态 + 标签 + 简介）。背景图优先级在 `metaStore.backgroundUrl`：Steam 本机 `librarycache` 的 `library_hero*`（`scan.SteamHeroImage`，排除 blur/logo）→ 商店 `screenshots[0].path_full（真正的预览图，比 page_bg_raw 更亮更明显）/background_raw/background` → 封面；DLsite `previewImages[1]`（画廊第二张）→ 旧缓存从 `coverUrl` 推 `_img_smp1`。小图用 `libraryStore.coverFor()`（游戏库卡片那张），未入库才退远程 `coverUrl` |

| emoji 全面换 lucide | emoji 跨平台渲染不一致、无法用主题变量控色；lucide 经 `presetIcons` 构建期内联，离线可用且零运行时请求 |
| 不引外部字体 | 桌面端离线优先 |
| 图标尺寸提为 5 档 token | 原代码混用 `text-sm`/`text-xl` 控图标大小，语义错位 |
| 作品库与调试页**共用** `stores/metaStore.ts` 与 `components/meta/*`，而不是各写一份 | 用户要求「MetaApiTestCard.vue 内容高度组件化，方便重新排列使用」。拆三层：**store 拿数据 / 积木定长相 / views 定排布**。两页因此只是同一批积木的两种组合：调试页单列四卡（含原始 JSON），作品库双列（详情+候选栏）。请求、错误兜底、封面下载等逻辑只有一处，改一次两页同步 |
| 状态放 **Pinia store** 而不是 composable（`composables/useMetaSearch.ts` 已删） | 用户明确要求「本项目使用了 pinia 所以状态管理相关尽量写在 stores 文件夹中」。附带好处：同一个 store 实例被两个页面共享，切页不丢搜索结果 |
| 游戏库表**来源无关**（`source` 列 + `raw_json`），而不是 `dlsite_works` 专表 | 用户要求「表存储元数据来源 来源于dlsite则为dlsite字段 剩下存储为id等作品信息」。公共列存跨来源可比的元数据（id/名称/社团/系列/发售日），来源特有字段整份塞 `raw_json`；唯一键 `UNIQUE(source, work_id)`，将来接别的源加一行常量即可，不动表结构 |
| 游戏库表**惰性创建**（`ensureLibraryTables` 每次读写前 `CREATE TABLE IF NOT EXISTS`），不在 `initAllTables` 里建 | 用户要求「当需要保存到库中时 即是需要下载数据才新建表 节约空间」。代价是每次多条 DDL 语句（幂等，代价可忽略）；换来的是从不用游戏库的用户库里不会凭空多两张空表 |
| `SQLiteService` 的 `Execute`/`Query` 加 `mu sync.Mutex` **串行化** | 惰性建表让「并发首次使用」天然撞上 DDL 竞争，实测 `go test ./internal/database/` 报 `database is locked (5) (SQLITE_BUSY)` —— 正是用户提醒的「使用表时可能会导致数据库异常」。SQLite 多连接并发写同一文件必 BUSY；桌面端串行化开销可忽略，`Query` 一次性取完行不把锁泄给调用方 |
| 游戏库封面走 **base64 `data:` URL**（`CoverDataURL`），不用 `file://` | Wails webview 对本地文件协议的支持没实机验证过（§9.2 第 5 项把握最低），data URL 一定可用；封面几十~几百 KB，一次列表请求可接受 |
| **删除库记录不删磁盘图片**（已于 2026-11-16 改为删除入库封面） | 图片是用户自己的文件，应用删库记录属于"清理索引"，顺手删文件是不可逆的越权操作。**这条已按用户新要求推翻**：现在只删本应用入库时保存的 `covers/` 文件，见下面「封面目录改名 + 删图」决策；用户自己另存/转换的图片仍不受影响 |
| 缓存优先放**新服务** `LibraryService`，不往 `MetaApiService` 加开关 | 两个语义相反：调试页要**永远联网**（否则解析改动被缓存掩盖），作品库/游戏库要**缓存优先**。混在一个方法里只能靠布尔开关，调用方永远搞不清这次到底走没走网络；两个服务各自语义单一，前端 `metaApi` / `libraryApi` 一眼可辨 |
| **`composables/` 目录取消**，唯一残留的 `useAutoHeight.ts` 改名 `stores/autoHeightStore.ts`（`useAutoHeightStore`），`TodoList.vue` 改 `collapse.enter` 等 | 用户要求「本项目 vue3 已经引入了 pinia 所以 composable 应该封装进 store 中 记得调整相关使用了该函数的库的相对路径及其名称」。项目里共享逻辑只认 stores 一个入口，避免两套目录各放一半 |
| **作品库 `/works` 在侧栏没有入口**，只能从游戏库点封面进入（路由保留） | 用户要求「保留 router 中游戏库相关前端 删去作品库（仅仅只是在 router 中 后面要复用作品库相关 component）」，随后明确「不在 sidebar 显示 而是通过游戏库进入」。于是：路由留着（游戏库 `openWork()` 要跳它）、侧栏只留「游戏库」，导航里不会出现两个"库"让人分不清；`WorksPage.vue` 与 `components/meta/*` 一行没删 |
| 游戏库卡片 = **封面网格 + hover 浮现名称 + 右上角删除**，点封面进作品库 | 用户要求「预览图横向排列、在预览图下方显示名称（hover 时半透明界面从下方淡入）、点击预览图进入原本的作品库界面、右上角添加删除按钮」。实现要点：半透明背景只能走 scoped 的 `color-mix(in srgb, var(--color-bg-base) 88%, transparent)`（UnoCSS rules 精确匹配，`bg-secondary/50` 会被静默丢弃）；名称面板用 `translate-y-full → group-hover:translate-y-0` 做"从下淡入"；删除按钮与封面按钮是**同级兄弟**（button 不能嵌套 button），用 `@click.stop` 隔离点击。**连点两下的待确认态在 `@mouseleave` / `@blur` 时自动撤销**（用户要求「在移开时取消二次确认 仿照 todoitem 的删除按钮逻辑」，`cancelArm()` 与 TodoItem 的 `cancelDelete()` 同款）。外观也与 TodoItem 对齐：形状走 `btn-row`，静息 `opacity-75`、hover/聚焦提亮到 100，颜色状态（红描边 → hover 填红 → armed 填红 + `scale-110`、文字用 `--color-text-on-danger`）写在 `.lib-del`，**与 `.row-btn-delete` 必须同步改**。**2026-11-23 已推翻其中"右上角删除"部分**：用户要求「按钮删除等操作 对游戏卡片的操作不显示在游戏卡片上 批量操作按钮改在刷新按钮所在卡片上」，卡片只剩封面按钮，删除移到批量操作栏（`.lib-op`/`.lib-op-danger`） |
| **搜索移到游戏库顶部的居中弹窗**，作品库不再有搜索卡 | 用户要求「移除上方作品库·DLsite元数据卡片（搜索功能保留 并添加在游戏库界面的上方 通过按钮显示搜索 通过悬浮窗弹出提示搜索 弹窗在中央 显示时周围模糊）」。因此 `/works` 只剩作品信息（也去掉了「封面保存位置」「游戏库」两张卡）；弹窗用 `Teleport to="body"` + `z-[1300]` 避免被页面层级/overflow 裁掉，遮罩 `color-mix(...) + backdrop-filter: blur(8px)`；**关闭只留 ✕ 与 Esc，点遮罩不关**（用户要求「取消…点击搜索卡片以外区域会退出搜索界面的逻辑」，避免手滑点旁边就把搜索关掉） |
| 「下载封面」文案改成**「入库」** | 用户要求「下载封面字样改为"入库"功能不变（下载文本数据 图片 并加入库 载入数据库相关数据）」。`MetaDownloadButton` 默认 `入库` / `入库中`。**注：随后用户又要求"搜索结果不能自动入库"，语义已升级为"点入库才写库"（见下面一行），本行只保留文案变更的历史** |
| **搜索/浏览只读，只有「入库」才写库** | 用户要求「搜索内容只有在点击入库后才会下载封面并且在游戏库中显示 在搜索内容后点开就会自动入库 去除该逻辑 文本和图片在点击入库后才下载并载入数据库中」。`LibraryService.Fetch` 去掉 upsert（搜索、点开详情都不再写库），新增 `Store` 负责"抓文本 + 下封面到 covers + 写库"，前端只有 `metaStore.storeIn()` 调它。代价：打开未入库作品每次都联网（没有缓存可命中），换来的是游戏库只装用户显式收的作品 |
| **搜索缩略图缓存在 `cache/`，与 `covers/` 分开** | 用户要求「并且显示在搜索界面的缩略图(载入缓存文件夹cache中 没有就创建一个)」。`MetaApiService.CacheCover` 把候选小图落到工作目录的相对路径 `cache/`（不存在就 `MkdirAll`）并返回 data URL；文件名固定 `<编号>_cover.<ext>`（缓存语义，允许覆盖，已缓存不重复联网）。与 `covers/` 的区别：`covers/` 是入库后的正式存档（`-2/-3` 不覆盖），`cache/` 可以整目录删掉重下 |
| 游戏库顶栏压成**单行**：标题 + 右侧「已入库 N 款」+ 搜索 + 刷新 | 用户要求「将游戏库界面顶部卡片文字高度居中 并缩减卡片高度 并在游戏库右边显示已经入库多少款游戏」。做法：`MetaSection` 在没有默认插槽内容时不渲染 body，header 上下内边距改 `py-3`（有 body 时仍是 `pt-4`），卡片高度因此只剩「按钮 35.2px + 24px」；标题/数量/按钮同处 `items-center` 的 header，天然垂直居中。数量用 `count`（`withCoverCount` 随之删除，页面上不再有"多少部已下载封面"）。之后按用户要求把「刷新」调到「搜索」左边 |
| **搜索列表直接显示预览图**（不再"点详情才有图"） | 用户要求「原本要点击详情才加载预览图 现在在搜索界面直接显示预览图」，并明确要载入 `cache/`。根因不在前端：DLsite 搜索页的缩略图是 Vue 组件 `:thumb-candidates="['//img…']"`，旧的 `parseSearchItem` 抓的是块内那个 `src="data:image/gif…"` 的弹层占位图 → `Cover` 是个内联 gif，`CacheCover` 拿到 `data:` URL 直接放弃（http 请求不认这个 scheme），所以列表永远空白、cache 里也没有文件。修复：解析 `:thumb-candidates`（`extractSearchItemCover`），`CacheCover` 显式跳过 `data:` 占位图；前端仍用 `covers[id]`（cache → data URL）优先、远程直链兜底 |
| 搜索结果**每页 10 条**，只给当前页下预览图；**退出时清空 `cache/`** | 用户要求「搜索卡片分页 每10个搜索结果为一页 只有当前页面需要加载预览图 以节约网络资源 每次退出软件前 清除cache中文件」。分页状态放 store（`candidatePage` / `pagedCandidates` / `setCandidatePage`），`setCandidatePage` 只对当前页调 `loadCandidateCovers`，已缓存过的直接命中 `cache/` 不再联网；清理挂在 Go 的 `app.OnShutdown` → `getmeta.ClearCacheDir()`（只删 `cache/` 里的文件，`covers/` 不动）。分页条本身在 `MetaPager`：前 3 页 / 当前页 / 后 3 页 + 输入页码跳转（用户后续要求「前三页和后三页可以跳转 并支持输入数字跳转到相应页面」） |
| 搜索**不再受 30 条限制**：按 DLsite 源数据页懒加载（1~3 页取第 1 面，4~6 页取第 2 面…） | 用户要求「检查搜索结果是否有限制 如果有则取消 可能是30个搜索结果加载限制」，随后明确「DLsite搜索界面每一面只有30条 能否在本页面在3页内时加载DLsite源数据页第一面 后三的倍数页加载第二面」，并要求「首先获取搜索结果个数调整页面数量 在需要加载源数据第二面内容时才重新爬取加载」。实测 `per_page` 无效（传 60/120 仍只回 30 条），只能翻页：`SearchWorks` 加 `page` 入参，返回 `pageCount` 与**精确总数** `total`（页面 `data-search-result-count`，如 1710 → 57 页）；前端 `sourcePages` 存"DLsite 页号→30 条"，总页数直接由精确总数算，`ceil(前端页/3)` 决定需要哪一面，**用到才取**；翻到第 3/6/9…页时静默预取下一面 |
| 「入库」按钮在**已入库后变 `已入库` + 禁用**（**已被下面「游戏启动」一行取代**；判定 `stored` 的规则沿用） | 用户要求「作评库界面的入库按钮 在打开界面时如果已经入库则显示已入库 并将按钮的状态改为不可按下」。判定用**库里已有封面文件**（后端注入的 `coverAbsPath` 非空，或本次会话刚下过 `saved`），而不是"在游戏库里" —— 元数据随抓取就入库、`/works` 又只从游戏库进入，按"在库里"判定会让按钮永远不可点、封面永远下不来。禁用还顺手挡掉重复下载：`writeFileUnique` 不覆盖同名文件，重名加 `-2/-3` 后缀，多点一次就多一份重复图 |
| 已入库后**不再显示「已入库」，替换为「游戏启动」** | 用户要求「已经入库游戏的入库按钮 现在是在入库后变为已入库并将按钮状态改为不可按 改为入库后不显示替换为游戏启动 具体逻辑为按钮组件添加逻辑判断 调用数据库中存储的游戏位置启动(如果没有就新建相关条目或新建表)」，随后明确「游戏路径和 library_repo.go 合并在一个表内」。实现：①位置就是 `library_works.game_path` 一列（不单开表）—— 与元数据同表，但 `UpsertLibraryWork` 的 `ON CONFLICT SET` 不含它（重抓元数据/重新入库不冲掉位置），旧库文件由 `ensureGamePathColumn` 用 `PRAGMA table_info` + `ALTER TABLE ADD COLUMN` 补列；②`MetaDownloadButton` 按 `stored` 分支：未入库 = `入库` / `入库中`，已入库 = `游戏启动`（play 图标；`launching` 时「启动中」），不再有禁用态；③点击走 `metaStore.launchGame()` → `GameService.Launch` 用记录的绝对路径起独立进程（`cmd.Dir` = exe 所在目录、`DETACHED_PROCESS`、`Process.Release()`）；④没位置/文件已不在时后端返回 `missing=true`（不报错），前端弹 `DialogService.PickExecutable` 选一次 `.exe` → `SetPath` 写库 → 再启动；已设置位置时按钮旁另有轻量「更换位置」；⑤扫描添加「按 exe 识别」的每行携带 `path`，`Store({gamePath})` 入库时写入位置，所以扫描入库的游戏第一次点「游戏启动」就能直接跑。原「已入库 + 禁用」防重复下载的目的，现在由"已入库后根本没有入库按钮"达成 |
| **启动按钮先检测 exe + 运行状态，运行中/文件丢失都禁用** | 用户要求「进入相关界面时启动按钮检测 首先检测游戏的路径文件exe文件是否存在(如果不存在提示用户游戏已经删除或变更位置 并将原本按钮改为不可按下状态 路径选择复用原本ui 不需要改变) 使用gopsutil等库(可以不是该库 自行判断使用什么库合适 用来读取特定程序运行状态) 通过数据库的路径信息 如果游戏路径正确则判断游戏是否运行(运行就将游戏路径载入缓存中 方便实时查看游戏运行状态 不再运行如果路径在缓存中 就将路径移除缓存) 如果运行便显示运行中相关字样 没有运行就显示运行相关字样」。实现：①新方法 `GameService.Status(workID)` 返回 `{path, exists, running, pid}`；进入 `/works`（`applyWork`）检测一次 + 页面每 5 秒轮询 + 窗口 focus 刷新；②exe 不存在 → 清运行缓存，前端 `gameWarning` 提示"游戏程序已被删除或移动了位置"，`launchBlocked` 让启动按钮禁用，「更换位置」仍可用（沿用原系统文件框，UI 未改）；③运行检测**不引 gopsutil**：`x/sys/windows` 的 Toolhelp 快照按 exe 名粗筛 + `QueryFullProcessImageName` 比对完整路径（同名 exe 不误判），`GameService` 用 `running map[小写绝对路径]pid` 做运行缓存，缓存命中先 `processAlive` 复核（防 PID 复用），失效/未命中再全量扫；④运行中按钮显示「运行中」并禁用（防重复启动），游戏退出后轮询自动切回「游戏启动」；`Launch` 成功后把 PID 直接载入缓存 |
| `loadById` **不把编号写回搜索框**，且搜索弹窗**每次打开都清空输入框** | 用户要求「在打开游戏库中游戏时 搜索界面会自动填入作品rj号 取消该逻辑」。两处一起收拾：①`loadById` 原本顺手 `keyword = id`；②`runSearch` 会把搜过的关键词（常是 RJ）留在 store 里，弹窗重开就带出来。现在 `openAddGame()`（原 `openSearch()`）与 `finishSearch()` 都清 `meta.keyword`，搜索框只反映"这次开窗后自己输入的"。代价：不再"沿用上次关键词方便反复调整"；失败的搜索仍留在框里，便于改错 |
| 游戏库「添加游戏」入口先放网格末尾加号（2026-11-23 又收回顶栏卡片） | 用户要求「在游戏库卡片的游戏卡片最后加一个加号button以添加游戏 加号替换为原本headbar搜索按钮功能 headbar搜索按钮去除」，随后补充「添加游戏为透明圆形ui」。实现：`.lib-add` 是 3.5rem（56px）见方、`border-radius:50%`、**全程透明底**（静息与 hover 都不填色）的圆形按钮 —— 虚线 `--color-border-dark` 描边 + 居中 `i-lucide-plus`，hover / 键盘聚焦只把描边与图标换成品牌色；它放在网格最后一位的 `flex-center` 容器里（刻意不做成 4:3 方块，圆形能把"入口"和作品卡区分开）。虚线边框整条写在 scoped 样式里：UnoCSS 的 `border-*` rules 都是 `1px solid` 简写，再叠 `border-dashed` 会互相覆盖（同属性只留一条）。空库时网格仍然渲染（`v-if="works.length \|\| !loading"`），保证唯一的添加入口始终可见，空库提示改成一行文字（原 `MetaEmptyState` 在本页不再使用） |
| **搜索栏为空时点「搜索」只提示「搜索栏不能为空」，绝不跳详情页** | 用户报 bug：「点击搜索按钮 如果搜索栏为空会弹出上次的搜索结果详细页 修改逻辑 改为点击搜索按钮时 如果搜索栏为空 提示搜索栏不能为空」。根因不是没做空值校验（`runSearch` 本来就有早退），而是**调用方拿 `meta.work` 当"搜索成功"的标志**：空关键词早退时不会清 `work`，上一次搜到的作品还留在 store 里，于是 `submitSearch` 的 `if (meta.work) finishSearch()` 照样跳 `/works`。修法两层：①`runSearch` 改成返回 `boolean`（这次是否真的发起了搜索），空值统一提示 `搜索栏不能为空` 并返回 `false`；②`LibraryPage.submitSearch` 只认返回 `true` 才看 `work`。提示复用弹窗里现成的 `MetaMessage tone="error"`，不另做 toast。**同类坑：判断"有没有结果"不能用 store 里的残留状态。** |
| 设置页新增「清理缓存」：进页扫描 `cache/` 占用显示在按钮旁，一键清空（`covers/` 不动） | 用户要求「设置界面添加清理缓存功能(将缓存cache内所有文件删去 并在进入设置界面时运行dir命令 扫描cache文件夹大小 并显示在清理缓存功能旁边)」。分两层：①后端 `MetaApiService` 加 `CacheInfo`（`filepath.WalkDir` 统计 `{dir,files,bytes}`，目录不存在算 0，不联网、不建目录）与 `ClearCache`（复用 `ClearCacheDir` 的删除逻辑，回 `{files,bytes,rest}`）；②前端新组件 `components/settings/CacheCleanup.vue`，`onMounted` 调 `CacheInfo` —— 设置页是**动态导入的路由页、没有 keep-alive**，每次进入都会重新挂载，所以 `onMounted` 就等价于"进入设置界面时扫描"，不必额外写路由钩子。按钮旁显示 `占用 X（N 个文件）`（无缓存时显示「当前无缓存」并禁用），清理后**重新统计一次**、以真实占用为准（`rest > 0` 说明有文件删不掉，能立刻看见）。按钮颜色沿用项目的破坏性语义（透明底 + danger 描边/图标 → hover 填红 + `--color-text-on-danger`），反馈是按钮下方一行 4 秒自动消失的小字；**缓存是可再生的临时图，所以刻意不做连点两下确认**（与 TODO/游戏库的删除区别开），也不建模态框。 |
| **游戏库「添加游戏」改为双模式：扫描添加（新）+ 手动搜索（原流程）** | 用户要求「将这些函数整合到前端调用 功能是添加游戏的新方法 原先搜索改为手动搜索 将获取到的游戏名称导入到搜索函数获取相关游戏信息 并让用户决定是否入库」。实现：`main.go` 注册 `ScanService`（第 6 个服务）+ 新增 `DialogService.PickDirectory`（第 7 个服务，原生选文件夹）；后端扫描逻辑不变（只读、不联网、不写库）。前端新增 `services/scanService.ts`（`scanApi`）、`stores/scanStore.ts`（来源/方式/路径/结果）、`components/library/ScanPanel.vue`；`LibraryPage` 弹窗标题改「添加游戏」，顶部 `chip` 双模式切换，默认「扫描添加」：来源 Steam（一键扫）或本地文件夹（路径输入 + 「选择…」原生框 + 「按 exe 识别 / 按文件夹名」），扫出名字后**自动逐个搜索**（3 路并发，可「停止」/单行「重试」），结果按扫描顺序一行一个（封面 + 匹配作品名 + RJ + 扫描名），行可勾选，工具条提供「全部入库」/「入库所选」（**串行**写库；已入库的行显示「已入库」且不再参与），点「详情」进 `/works` 核对；「手动搜索」保留原搜索行。扫描结果留在独立 store（关窗不丢）。**弹窗内容整体可滚动**（`max-h-[80vh]` + body `overflow-y-auto`），避免结果列表与候选同屏时撑破弹窗。 |
| 扫描结果改为**全部直接搜索 + 一键全部入库 / 选择部分入库** | 用户要求「扫描添加的点文件去搜索改为全部直接搜索 并将搜索结果排列展示用户 用户决定是否添加」「分为一键全部入库 和选择部分入库」。实现：`scanStore` 扫完自动 `searchAll()`（3 路并发、`searchToken` 代次作废旧请求、可停止、单行重试）；每行 `ScanMatch{state, workId, workName, cover, selected, stored, storeError}`，匹配到作品时用 `library.has()` 预标「已入库」；工具栏「全选 / 全部入库(N) / 入库所选(N)」，批量走 `storeMany()` **串行**调 `libraryApi.storeIn`（别让几十个"抓文本+下图"同时压网络与 SQLite），每行独立报错；成功后 `library.load(true)` 让关窗后网格立刻出现。入库返回的完整元数据回填到行，库里已有的行直接读库里的列。**详情补全**：搜索完成后再对匹配行逐条 `libraryApi.fetch('', id)`（缓存优先、只读）抓详情，`match.info = 社团 · 作品形式 · 发售日 · 容量` 显示在行里（失败显示「详情获取失败」原因，不影响入库）。封面直接用 DLsite 远程 URL（`MetaCover` 自带失败占位），不批量下 `cache/`。 |
| 入库前校验元数据**完整性**（扫描入库也要拿到作品信息） | 用户要求「扫描入库的游戏也应获取作品相关信息」。`LibraryService.Store` 本来就按编号抓详情写库；这次加一道底线校验 `workIsComplete`（`id` 与 `name` 都非空）：抓回来是空壳就报错、**不写库**；库里已有 `raw_json` 若是残缺的（旧数据/解析退化）也不复用，改为联网重抓。扫描行在入库成功后用返回的 `maker/workType/releaseDate` 回填一行摘要，让"信息已抓到并保存"肉眼可见。测试：`internal/service/library_service_test.go`（`TestWorkIsComplete` 7 例、`TestRowFromWorkCarriesWorkInfo`）。 |
| **标签：多值切分 + 更多来源 + 数量可配 + 用户自定义标签** | 用户要求「标签分割有问题 连在一起了 尝试使用空字符和/分割标签 并进行检查 并且标签含有json中的 声优 分类 插画 剧情 作者 社团名 且标签的数量可以由用户自定义 在设置界面中 且用户可在作品详细页中自行添加标签」。三块：①解析：`extractTags` 的标签来源扩充到 分类/作品形式/社团名/作者/剧情/插画/声优，多值字段**按元素取**（中文页元素间无空白，整段 `dd.Text()` 会粘成「异种X造孽…」「みたかりん野上菜月」），最后用 `splitTagText` 按空白与 `/` 切分、去重，`defaultTagLimit` 放宽到 64；②显示：设置页「标签显示数量」`TagLimitInput`（4–64，默认 24）写进 `stores/App.ts` 的 `tagLimit`，`MetaTags` 按 `limit` 截断并折 `+N`；③自定义：新表 `work_tags`（惰性建表、`UNIQUE(source,work_id,tag)`）+ `LibraryService.ListTags/AddTag/RemoveTag` + 前端 `MetaUserTags`（详情页输入/移除，存后端，与抓取标签分开、重抓不丢）。测试：`TestSplitTagText`、`TestParseWorkTagsIncludeDetailFields`、`TestWorkTagsLazyTableAndCRUD`、`TestNormalizeUserTag`。**注：③的「逐条加/删」与 `MetaUserTags` 已被 2026-11-22「标签改成合并 + 原地编辑 + 显式保存」一行取代**（标签改为整份保存，`ListTags/AddTag/RemoveTag` 换成 `GetTags/SetTags`）。 |
| **DLsite 换模板后"只有封面和名字"的根因修复** | 用户报「依然没有抓取到详细信息 只有封面和作品名 应该抓取详细页的信息 检查是否抓取错误」。排查：DB 里扫描入库的行 `raw_json` 只有 6 个键（约 280B），maker/workType/releaseDate 全空；实测页面发现 2026-11 DLsite 换新版（Vue + JSON-LD），旧的 `#work_outline` / `#work_maker` / `<meta itemprop>` 全部失效；并且代码发的是 `locale=zh-cn`，页面标签变成中文（社团名/发售日/文件容量），只认日文标签同样取空。修法：`parseWork` 双模板兼容（旧 `#work_outline` + 新 `div.c-productInfo` 的 dt/dd），价格/评分/社团/拼音/简介改读 JSON-LD Product，标签用 `outlineValueAny` 多语言候选，genre 用 `href*="/genre/"` 识别、`SeriesID` 取 `title_id/SRI…`、`MakerID` 取 `maker_id/RG…`/`brand.url`。配套：`workIsComplete` 收紧为「id+name+至少一个详情字段」，`Fetch` 与 `Store` 遇到旧空壳都当缓存未命中重抓（`/works` 立刻能显示详情，重新入库会把库里的空壳覆盖）。测试：`getMeta/new_template_test.go` + 两个 fixture（日文标签精简页、中文真实页）。 |

| **入库时就记录 exe（按扫描来源决定路径）** | 用户要求「在游戏入库时便存储相关exe 具体为记录结果通过什么文件夹名或exe搜索获得 如果通过文件夹名获取 读取该文件夹内exe记录位置(存在多个exe名则选择匹配文件夹名字字符最多的那个 首先排除含有UnityCrashHandler64.exe文件 优先选择Game.exe文件为路径) 如果通过exe名称搜索获得则直接使用改exe路径 手动搜索添加则在入库时便向用户请求路径 如果不提供便在需要启动时才请求路径」。实现：①扫描结果每行记 `origin`（exe/folder）+ `path`/`folderPath`；`ScanSubfolders` / `ScanSteamGameFolders` 改为返回名字+绝对路径（`Subfolder`）；②`StoreRequest` 增加 `gameFolder`/`gameFolderName`，`LibraryService.resolveStoreGamePath` 决定位置：直接给 exe 用之，给文件夹则调 `scan.PickGameExecutable`（排除 `UnityCrashHandler64.exe` → 优先 `Game.exe` → 否则选 exe 名与文件夹名最长公共子序列最长者）；挑不到/都没给就留空，**不阻断入库**；③手动搜索入库（`metaStore.storeIn`）先弹系统文件框选 exe，取消则照常入库但位置留空、启动时再问；已记录位置的不重复问；④扫描批量入库不再提示，路径随 `storeIn` 写入 `library_works.game_path`。 |

| **封面目录 preImg → covers；下载优先 webp；删库记录时删图** | 用户要求「preImg文件夹改名为covers(记得更改相关路径) 入库时保存的本地图像格式改为webp 在从游戏库移除游戏时 同时删除入库后保存的图像」，随后明确「不需要额外导入库 在下载资源到本地时优先检查是否可以下载webp格式图像」。实现：①目录/方法/前端全部改名（`coversDir`、`CoversDir`、`coversPath`、`MetaCoversPath`）；`MigrateLegacyCoverDir` 在 `MetaApiService.ServiceStartup` 里把旧 `preImg/` 改名（已存在则逐文件合并）并 REPLACE 库里 `cover_*` / `library_cover_files` 的旧路径，幂等；②不引编码库：`fetchImagePreferringWebP` 给 jpg/jpeg/png URL 拼同名 `.webp` 候选，单次探测，拿到 webp 就用，否则回落原图（搜索缩略图同样优先 webp）；③`LibraryService.Remove` 先按 `ListCoverFiles` + `cover_abs_path` 去重删磁盘文件（`os.IsNotExist` 忽略），再删库记录 —— 文件删不掉时保留记录可重试；④`covers/` 为空也不删目录，后续入库继续用。 |

| **新增 Steam 元数据源（appid + 商店 appdetails + 本机 librarycache 封面）** | 用户要求「steam源添加(meta)根据游戏id…扫描id 并通过 appdetails 获取json数据 优先复用已有逻辑」「查找封面图片文件夹目录在...\Steam\appcache\librarycache内…目录下含有游戏id名称的文件夹 进入查找相关图像 没有文件夹则不适用图像」。实现：`database.SourceSteam='steam'`（编号=appid）；扫描读 `appmanifest_<id>.acf`（StateFlags **按位**判 4，6 也算已安装）；详情走 `MetaApiService.FetchSteamWork`（`l=schinese&cc=CN`，归一化成与 DLsite 同一套作品 map，完整 appdetails 原样留在 `steam` 键）；`LibraryService.Fetch/Store` 按 source 分流，缓存优先/只读/只 Store 写库的语义不变；封面从 `appcache\librarycache\<appid>\` 递归（不限深度）找名字含 `600x900`/`capsule` 的图，600x900 优先、中文版其次、越浅越好，**找不到就不用封面**（不联网兜底）；`ImportCoverFile` 复制进 `covers/` 并记账。前端 Steam 行按 appid 直接取详情，`steamType !== 'game'` 的行跳过。**不复用 `steamFolders` 的"按文件夹名去 DLsite 搜"**：Steam 游戏的元数据固定来自 Steam（用户：「一个游戏数据固定一个来源」）。 |

| **游戏库封面统一 2:3** | 用户要求「游戏库封面展示图像格式也全部变为2:3等比图样 包括steam源和 DLsite源 和以后所有可能添加源」。`LibraryPage.vue` 封面容器 `aspect-[4/3]` → `aspect-[2/3]` + `object-cover`（DLsite 的 4:3 图居中裁掉两边，Steam 的 `library_600x900` 原生 2:3 不裁）；卡片 key / 封面缓存 key 都改成 `来源:编号`，为以后多来源做准备。**作品信息界面 `/works` 本次不动**（用户：「作品信息显示界面未来要作修改 暂时不进行改动」）。 |

| **Steam 接口调研只「保留入口」，不接入** |用户要求「在网上搜索相关api信息 把可能可以扩展当前软件的api保留」。做法：①`doc/steam-api.md` 逐条写清参数/返回/坑/能扩展什么，全部标 2026-11-22 实测（含反例：appreviews 302、GetAppList 404、appdetails 多 appid 400、steamspy 403）；②`internal/service/getMeta/steam_endpoints.go` 收一组免 key 常量，`steamAppDetailsFmt` 改由 `steamAppDetailsAPIURL` 派生（单一事实来源），除此之外**零改动**——不注册服务、不加绑定、不写库。理由：这些接口只是候选，先接入会带出用户没要的 UI/表结构改动；而把「哪个接口还能信」沉淀成文档，下次接的时候不用重新试错（本次最贵的一步是 PowerShell/curl 在 schannel 层不通，得用 Go 探）。 |

| **Steam 标签改走 GetItems；手动搜索 / 扫描 / 作品页补齐 Steam**（2026-11-22） | 用户要求「tag 的获取尝试使用别的接口（不是原来的 appdetails）」，并确认「改动三条 + tag 换接口」「手动搜索可以是 steam 也可以是 DLsite」。落点：①`steam_browse.go` 用 `IStoreBrowseService/GetItems`（`include_tag_count:20` + `include_reviews`）取**玩家标签**与评价汇总，`IStoreService/GetMostPopularTags` 做 tagid→中文名（进程内缓存 24h），标签按权重降序排在 appdetails 的分类标签前面、去重截断到 `steamTagLimit`；②`MetaApiService` 新增 `SearchSteamWorks` / `MatchSteamAppByFolder` / `FetchSteamStats`（bindings 已重生成，当天为 65 个方法；2026-11-23 加启动方式后为 67）；③前端弹窗加「搜索来源」chip（DLsite / Steam）、`scanStore` 对「按文件夹名」的行先猜 Steam、`/works` 加 `MetaSteamStats`。**所有 Steam 附加数据都是"能拿到就更好"**：GetItems／标签字典／公告任一失败只退化成少一块信息，不影响 appdetails 主链与入库。匹配阈值 0.75 + 查询词逐级截短，是实测「storesearch 按词匹配」（`黑神话悟空` 0 条 / `黑神话` 9 条）逼出来的。 |

| **标签改成「合并 + 原地编辑 + 显式保存」**（2026-11-22，两轮需求定稿） | 第一轮用户要求「鼠标指针hover到标签右上角显示x号提示删除 用户点击从数据库中删除标签 添加标签显示改为在标签排列位置的最后排列css格式与原标签格式相同的占位+号 点击后用户可以输入文字添加标签(位置在+号标签中) 如果输入为空则无视本次修改 如果不为空则添加输入内容为标签到占位+号左侧 删去原本的添加标签前端内容」；第二轮澄清「我的标签删除 我的标签与抓取标签合并 … 删掉后不是隐藏 而是改写数据库标签 但不是每删一个就改写数据库 有改动后添加保存按钮 点击后进行标签的重新写入 且添加标签的ui点击后应保持原有大小 更具输入动态扩展标签大小」。落点：①**合并**：删除独立的「我的标签」区块，抓取来的 `tags` 与 `work_tags` 合成**一份**编辑稿 `metaStore.tagList`（载入时：`GetTags.edited=true` 直接用库里那份，否则抓取 tags 打底 + 旧自定义标签追加、去重）；概览卡的标签云在 `/works` 关掉（`MetaWorkOverview.showTags=false`，调试页仍显示），避免同页两处标签。②**UI**：标签是 `chip`，`li` 加 `position:relative`，× 是 14px 圆形绝对定位在 `top/right:-5px`，静息 `opacity:0` + `pointer-events:none`（不占点击区、防误删），`.tag-chip:hover / :focus-within / .tag-remove:focus-visible` 才浮出，`title`+`aria-label` 提示删除，hover 填 danger 红（沿用游戏库/TODO 删除的破坏性配色）；末尾固定一颗同款 `chip` 的「+」（`i-lucide-plus icon-sm`），点它把该 `li` 换成 `chip` 外壳的输入框（`!border-brand` 表聚焦），`display:inline-grid` + 隐藏镜像 span 叠同一网格单元 → **先保持 `+` 的原宽（`min-width:2.375rem`），再随输入内容变宽**；回车/失焦提交、Esc 取消，空输入直接收起不写库。③**批量保存**：增删只改编辑稿，`tagsDirty`（与 `tagSaved` 逐项比较，顺序也算）时标题右侧出现「保存」；点它调 `SetTags` **整份覆盖**（清空 → 按顺序插入 → 记 `work_tag_edits` 标记，同一事务），返回的清洗结果作为新基准。④**持久化语义**：`edited=true` 后重新抓取元数据**不会**冲掉用户整理的标签（`work.tags` 只用于未整理过的初始合并）；`work_tag_edits` 的存在是为了区分「从没整理过」与「整理成空列表」（后者也要显示为空）。测试：`TestWorkTagsSaveAndEditFlag`（覆盖/保序/空列表/来源隔离）、`TestNormalizeTagSet`（去重保序、超长超量报错）。 |

| **游戏库批量操作 / TODO 去掉完成 / Steam 启动方式**（2026-11-23，用户三条要求） | ①游戏库：用户要求「只有批量操作游戏库时才弹出操作栏 按钮删除等操作 对游戏卡片的操作不显示在游戏卡片上 批量操作按钮改在刷新按钮所在卡片上」——卡片上的删除按钮与 `.lib-del` 整体删除，顶栏加「批量操作」，进入后操作栏（`MetaSection` 的默认插槽，就在刷新按钮那张卡里）提供全选/删除所选；选卡态 = 封面内侧 2px 品牌色内描边（`inset box-shadow`，不改尺寸），`libraryStore.removeMany` 逐条删、失败汇总。②TODO：用户要求「完成按钮 完成等 TODO 功能删除」——移除行内完成按钮、批量完成、完成状态筛选、完成率统计与 `todoStore` 的 completion 相关状态；`TodoStats` 改为按优先级计数（前端算，不再调 `GetStats`）。后端 `todos.completed` 与 `ToggleComplete/GetStats` 保留（删列需迁移）。③Steam 启动方式：用户要求「通过 steam 源添加的游戏 游戏启动可选通过 steam 启动（通过运行 steam://rungameid/<AppID> URL 运行）也可选通过本地 exe 直接启动 默认使用 URL 启动」——`library_works.launch_mode` + `GameService.SetLaunchMode`；`Launch` 按生效方式分流，Steam 默认 `steam`（`ShellExecute` 打开 URL，不要求本机 exe），exe 方式保持原逻辑；`/works` 概览加 chip，Steam 源入库不再追问 exe。 |

| **手动搜索与扫描添加改成「多数据来源」**（2026-11-23） | 用户原话：「自动扫描游戏功能 自动搜索时应该显示多数据来源 手动搜索只显示DLsite数据源 应该也显示steam数据源 以demon root这个名称进行测试」。①手动搜索去掉「搜索来源」chip，`metaStore.runMultiSearch` 用 `Promise.allSettled` 同时打 DLsite 与 Steam，弹窗按来源分开显示（2026-11-24 起改为源切换器，默认 DLsite，无 DLsite 结果时落到 Steam；分页只归 DLsite）；②扫描结果从「一条目一行」改成「一条目 × 来源」最多两行：`ScanMatch` 增加 `source` + `entryIndex`，勾选与入库都按行（同一游戏收两份也不冲突 —— `UNIQUE(source, work_id)`）；③`steamNameScore` 从前缀占比换成**编辑距离相似度**：`demon root` vs `Demons Roots` 只差一个字母，旧算法给 0，导致扫描永远只出 DLsite 一行（实测烟测见 §3.11）。Steam 库来源的扫描仍只出 Steam 行（元数据来源固定）。 |

---

## 11. 相关提交（自设计系统重构起，倒序）

`b621c2c` 托盘点击只唤回不隐藏；移除关闭行为里的最小化 ·
`81d25bc` 新增「隐藏到托盘」+ 系统托盘图标 ·
`1bd902a` 标题栏 X 首次点击询问，可在设置页修改 ·
`be098be` 切换筛选清空选择；统一三个批量按钮风格 ·
`30b6305` 修复完成按钮伪立体边框；边框宽度统一整数 px；补回归扫描 ·
`701b4e2` 完成按钮 UI 对齐删除按钮；「已完成」筛选下语义反转为取消完成 ·
`6d4793f` 交互模型调整：选择与完成分离，批量收敛为两个按钮 ·
`b8c3df8` 新增「删除范围」面板（后已移除） ·
`f159ca3` UI 排版调整：极简取向 + 统一过渡动效 + 统计/筛选按需显现 ·
`21607b0` 修复描述块 hover 出现方框 ·
`e6a42bf` 修复 need-fix / need-change ·
`23f9770` 修复边框不显示 ·
`aa8f800` 设计系统重构：Flat Design + Teal 主题，图标全面替换 emoji
