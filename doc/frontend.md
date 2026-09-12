# 前端 (Vue 3 + Pinia)

## 启动 `src/main.ts`

| 代码 | 行 | 说明 |
| --- | --- | --- |
| `createPinia()` :10 | | 注册 Pinia |
| `themeStore.restoreState()` :17 | | 恢复主题（localStorage `theme-mode`） |
| `sidebarStore.restoreState()` :19 | | 恢复侧边栏（localStorage `sidebar-collapsed`） |
| `themeStore.setupSystemListener()` :21 | | 监听系统深色模式变化 |

`src/App.vue` 仅渲染 `MainLayout`。

## 路由 `src/router/index.ts`

Hash 模式。父级 `/` → `MainLayout`，子路由：

| path | name | 组件 | 标题 |
| --- | --- | --- | --- |
| `/` | Home | `views/HomePage.vue` | 首页 |
| `/settings` | Settings | `views/SettingPage.vue` | 设置 |
| `/todo` | Todo | `views/TodoPage.vue` | TODO |
| `/test` | test | `views/test.vue` | 测试 |

## Store

### `stores/todoStore.ts` — `useTodoStore()`

State：`todos` `loading` `error` `stats` `curPriority`（当前新建优先级，默认 1）`filters` :51

| Getter | 行 | 说明 |
| --- | --- | --- |
| `completedTodos` | 62 | 已完成列表 |
| `pendingTodos` | 66 | 未完成列表 |
| `highPriorityTodos` | 70 | priority === 2 |
| `completionRate` | 74 | 完成率百分比 |
| `filteredTodos` | 92 | **应用筛选后的列表（模板用这个，不再直接用 `todos`）** |
| `isFilterActive` | 103 | 是否处于非默认筛选（空状态文案用） |
| `isFilteredEmpty` | 108 | 有数据但被筛选挡住 |

| Action | 行 | 说明 |
| --- | --- | --- |
| `matchTime(createdAt, time)` | 80 | 时间筛选判定，未来时间按 0 处理 |
| `resetFilters()` | 112 | 恢复默认（未完成 / 全部优先级 / 全部时间） |
| `loadTodos()` | 119 | 并行拉取 list + stats |
| `addTodo(title, description?)` | 137 | 用 `curPriority` 创建，头部插入 |
| `toggleComplete(id)` | 154 | 切换并就地更新 |
| `updateTodo(id, updates)` | 170 | 局部更新 |
| `deleteTodo(id)` | 185 | 删除并过滤本地列表 |
| `loadStats()` | 197 | 仅刷新统计 |
| `init()` | 205 | 挂载时调用 → `loadTodos` |

#### 筛选（`filters` :51）

```ts
filters = reactive({ completed: 'pending', priority: null, time: 'all' })
```

| 维度 | 类型 | 可选值 | 默认 |
| --- | --- | --- | --- |
| `completed` | `CompletedFilter` | `'pending'` / `'completed'` / `'all'` | **`'pending'`（只看未完成）** |
| `priority` | `PriorityFilter` | `null` / `2` / `1` / `0` | `null`（全部） |
| `time` | `TimeFilter` | `'all'` / `'today'` / `'week'` / `'month'` | `'all'` |

- 选项常量由 store 导出：`COMPLETED_OPTIONS` `PRIORITY_OPTIONS` `TIME_OPTIONS` :23-40，天数映射 `TIME_RANGE_DAYS` :42。
- 三个维度是**与**关系，同时生效。
- 筛选在**前端内存**完成（数据量小、切换无需重新请求）；`stats` 仍是全局统计，不随筛选变化。
- 新增 todo 后因是 computed，筛选结果自动重算，无需手动刷新。

### `stores/Sidebar.ts` — `useSidebarStore()`

State：`isCollapsed`。常量：`barWidth=72`、`barWidthCollapsed=250`

| 成员 | 行 | 说明 |
| --- | --- | --- |
| `sidebarWidth` | 14 | 当前宽度 px |
| `mainPaddingLeft` | 15 | 主内容左内边距 |
| `isExpanded` | 16 | `!isCollapsed` |
| `toggleSidebar()` | 19 | 折叠切换（写 localStorage） |
| `setCollapsed(v)` | 24 | 显式设置 |
| `restoreState()` | 29 | 读取 localStorage |

### `stores/Theme.ts` — `useThemeStore()`，`ThemeMode = 'light' | 'dark' | 'system'`

| 成员 | 行 | 说明 |
| --- | --- | --- |
| `mode` | 9 | 当前模式 |
| `effectiveTheme` | 13 | 解析 system 后的实际主题 |
| `isDark` | 23 | 是否深色 |
| `setMode(m)` | 27 | 设置并持久化 |
| `toggleTheme()` | 34 | 浅/深切换 |
| `applyTheme()` | 46 | 写 `data-theme` 到 `<html>` |
| `restoreState()` | 56 | 恢复 localStorage |
| `setupSystemListener()` | 67 | 返回取消监听的函数 |

## API 封装 `src/services/todoService.ts`

`todoApi`：`getAll()` `getByID(id)` `create(dto)` `update(id,dto)` `toggleComplete(id)` `delete(id)` `getStats()`
内部 `ensure(value, msg)` :12 将 null 转为抛错；`getAll` 空值兜底 `[]`，`getStats` 兜底全 0。

## 组件

### 布局

| 文件 | 说明 |
| --- | --- |
| `layouts/MainLayout.vue` | `TitleBar` + `Sidebar` + `router-view`（**flex 列容器**），按 store 设置左内边距 |
| `components/TitleBar.vue` | 无边框标题栏；拖拽区 `--wails-draggable: drag`；按钮调 `WindowService`（:33 最小化 / :41 最大化 / :50 关闭） |
| `components/Sidebar.vue` | 导航；`menuItemsUp` :87、`menuItemsDown` :94、`navigateTo(path)` :108 |

### TODO `components/todo/`

| 文件 | 关键成员 | 说明 |
| --- | --- | --- |
| `TodoList.vue` | `handleToggle` :72、`handleDelete` :76、`onMounted → init()` :82 | 列表容器，组合统计/输入/**筛选栏**/卡片列表 |
| `FilterBar.vue` | `storeToRefs` 取 `filters` | 筛选栏，三个 `FilterChipGroup` + 计数/重置 |
| `FilterChipGroup.vue` | `generic="T extends string \| number \| null"` | 可复用筛选组：标签 + 单选药丸，`v-model` |
| `TodoInput.vue` | `insertLineBreak()` :82、`toggleDesc()` :93、`handleSubmit()` :103 | 标题输入（**内嵌图标按钮**）+ 描述 textarea + 添加按钮 |
| `TodoItem.vue` | `priorityColor` :81、`priorityLabel` :90、`formatDate` :100 | 标题主行 + 描述块（缩进 + 左竖线）；emits `toggle` `delete` `error`；props `{ todo }` |
| `TodoStats.vue` | `completionRate` :27 | props `{ stats }` |
| `TodoPrioritySelector.vue` | `priorityOptions` :23（value 0/1/2） | 直接写 `todoStore.curPriority` |

优先级色：0 绿 / 1 黄 / 2 红。

> `FilterChipGroup` 用 Vue 3.3+ 的 `generic` 属性做泛型 props，同时适配
> `string`（状态）、`number | null`（优先级）、`string`（时间）三种取值类型。

#### 筛选栏交互

- **横向排列 + 横向滚动**：三个维度与右侧计数排在同一行，放不下时横向滚动（不换行）。
  各分组加 `flex-shrink-0`，否则会被压扁而不是产生滚动条。
- `scrollbar-theme` 必须写在模板 `class` 里——它是 `global.css` 的普通类，**不是 UnoCSS utility**，
  写进 shortcut 会被判 `unmatched`。
- 选中态用 `chip-active`（品牌色文字 + 边框 + 卡片底色）。
- 右侧常驻 `已筛选 / 总数` 计数；非默认筛选时才出现「↺ 重置」。
- 被筛选挡空时，卡片内显示不同于「还没有 TODO」的文案，并提供「恢复默认筛选」按钮。

#### 布局与高度（flex 全高链）

列表卡片用 **`flex-1` 吃掉剩余空间**，卡片高度自动适应上方内容变化（描述框展开/收起都不用管），
**不使用任何 `calc(100vh - Npx)` 魔法数字**。

完整高度链（缺任一环都会退化成固定高度，描述框一展开最后一条就滚不到）：

| 层 | 类名 | 作用 |
| --- | --- | --- |
| `MainLayout` 根 | `flex h-screen` | 定高基准 |
| router-view | `flex-1 min-h-0 **flex flex-col overflow-hidden**` | 变成可伸缩的 **flex 容器**；`overflow-hidden` 交出滚动权（若保留 `overflow-y-auto`，它成为滚动容器，flex 链失效） |
| `TodoPage` | `h-full min-h-0 flex flex-col max-w-3xl mx-auto w-full` | 撑满并居中；宽度用 `w-full`+`max-w`（不用 `flex-1`，避免与 `max-w` 冲突） |
| `TodoList` 根 | `h-full min-h-0 flex flex-col` | 竖向 flex 列 |
| 标题/统计/输入/筛选栏 | 各加 `flex-shrink-0` | 固定高度，不许被压缩 |
| 列表卡片 | `flex-1 min-h-[120px] overflow-y-auto` | **吃掉剩余高度**并内部滚动 |

> ⚠️ 不要改回 `max-h-[calc(100vh-Npx)]`。实测那个 N **算不准**：`100vh-368px` 时卡片底部
> 溢出视口 36px，最后一条永远滚不到（描述框只是把问题放大，折叠时同样溢出）。
> 交给 flex 自适应后，卡片底边在描述框展开/折叠两种状态下都稳定贴住可用区底部。

> 注意：Vue 会把**路由组件的根元素与 router-view 合并成同一个元素**（class 会拼接）。
> 所以「页面包裹层」不是 router-view 的子节点，而是它本身，`overflow-hidden` 也因此作用于该元素。

> **设置页需自己滚**：router-view 改为 `overflow-hidden` 后，设置页内容略高于可用高度时会失去滚动。
> 因此 `SettingsPage.vue` 加了 `min-h-0 overflow-y-auto`。

#### TodoInput 描述输入交互

描述框是 `<textarea>`（标题仍是单行 `<input>`）。
**描述开关是嵌在标题输入框内部的图标按钮**，靠 flex 排布，不用绝对定位：

```html
<div class="flex gap-2">
  <!-- 输入框容器：自带边框/底色，内部 flex 排 [input, 按钮] -->
  <div class="flex items-center gap-1 flex-1 min-w-0 px-2 rounded-[10px] bg-secondary border-theme focus-within:border-brand">
    <input class="flex-1 min-w-0 bg-transparent border-none outline-none py-2" />
    <button class="desc-toggle flex-shrink-0 w-6 h-6">
      <span class="i-lucide-align-left" />
    </button>
  </div>
  <button class="flex-shrink-0 px-5 ...">添加</button>
</div>
```

> ⚠️ **不要改回 `absolute` + `pr-10` 预留内边距的写法。**
> 之前用 `absolute right-2 top-50%` 定位按钮、再给 input 加 `pr-10` 留位，
> 结果按钮与添加按钮重叠。根因是 **`right-2` 和 `top-50%` 都不是合法 utility**（详见 `styles.md`
> 「无效工具类会静默失效」），且 `pr-10` 是手调值——内边距与实际按钮宽度一旦不一致就重叠。
> 现在两个按钮是**同级/同容器 flex item**，结构上不可能重叠。

| 成员 | 行 | 说明 |
| --- | --- | --- |
| `newTodo` / `desc` / `showDesc` | 68-70 | 标题、描述、描述框可见性 |
| `descInput` | 72 | 展开后自动聚焦的 textarea ref |
| `insertLineBreak()` | 82 | **Ctrl/Cmd+Enter 时在光标处插入 `\n`** |
| `toggleDesc()` | 93 | 切换显示；收起时清空 `desc` |
| `handleSubmit()` | 103 | 调 `addTodo(title, desc.trim())`，成功后清空两个输入 |

| 按键 | 行为 | 实现 |
| --- | --- | --- |
| `Enter` | 提交 | `@keydown.enter.exact.prevent` |
| `Ctrl+Enter` | 换行 | `@keydown.ctrl.enter.prevent="insertLineBreak"` |
| `Cmd+Enter` | 换行 | `@keydown.meta.enter.prevent="insertLineBreak"`（macOS） |
| `Shift+Enter` | 换行 | 原生行为，不拦截 |
| `Esc` | 收起描述框 | `@keydown.esc` |

> ⚠️ **Ctrl+Enter 必须自己插入换行，不能指望原生行为。**
> Chromium/WebKit 的 textarea 原生只在 **Enter** 和 **Shift+Enter** 下换行；
> **带 Ctrl 或 Meta 的 Enter 被直接丢弃**，不会插入换行。
> 曾经的错误做法是只写 `@keydown.enter.exact.prevent`，指望 Ctrl+Enter 落到"原生换行"——
> `.exact` 确实让它不触发提交，但原生也不会换行，结果按键完全无响应。
>
> `insertLineBreak` 手动改写 `desc.value` 并 `nextTick` 后复位光标到 `start + 1`；
> **注意 textarea 自行改 `value` 不会触发 `input` 事件**，所以必须同步 `v-model`。

- 描述框 `h-[4.5rem]` + `resize-none`（固定高度，不用 `resize-y`——避免右下角出现拉动提示），
  内容超出时由 textarea 自身滚动。
- **快捷键提示 `desc-hint` 是 textarea 的 flex 同级元素、排在下方右对齐**，
  绝不与输入文字或 placeholder 重叠（早前用 `absolute bottom` 定位导致与 placeholder 撞在一起）。
- **placeholder 只说「添加描述（可选）...」**，快捷键说明交给 `desc-hint`——
  两处都写「Enter 添加 / Ctrl+Enter 换行」会显示成重复文本。
- 样式类 `desc-toggle` / `desc-toggle-active` / `desc-field` / `desc-hint` 定义在组件 `<style scoped>`。
- 描述多行显示依赖 `TodoItem.vue` 描述块的 `white-space: pre-wrap`。

#### 描述区展开过渡

用 Vue 内置 `<Transition name="desc-slide">` 包住描述容器（`v-show` 放在**容器 div** 上，
不要直接放在 textarea 上，否则过渡类会落到组件根以外而失效）：

| 过渡类 | 作用 |
| --- | --- |
| `desc-slide-enter-active` / `leave-active` | `max-height .28s` + `opacity .22s` + `margin-top .28s`，`overflow: hidden` |
| `desc-slide-enter-from` / `leave-to` | `max-height: 0` + `opacity: 0` + `margin-top: 0` |
| `desc-slide-enter-to` / `leave-from` | `max-height: 14rem` + `opacity: 1` |

> 用 `max-height` 而非 `height: auto` 过渡——`auto` 无法参与 transition；
> `14rem` 是张开后的上限，内容超出时由 textarea 自身滚动。
> `leave-to` 里的 `margin-top: 0` 需要 `!important` 才能压过 `.desc-field` 的 `margin-top: .5rem`。

`desc-hint`（右下角「Enter 添加 · Ctrl+Enter 换行」）平时 `opacity: 0`，
`:focus-within` 或 `:hover` 时淡入，避免常驻干扰。

### 设置 `components/settings/`

| 文件 | Props / emits | 说明 |
| --- | --- | --- |
| `SettingsPage.vue` | slot `title`、默认 slot | 页面容器 |
| `SettingsSection.vue` | `icon` `title` | 分组卡片 |
| `SettingsItem.vue` | `label` `description?` | 单行设置项 |
| `ThemeToggle.vue` | — | 深浅切换按钮（`themeStore.toggleTheme`） |
| `ThemeModeSelector.vue` | `v-model` 三态 | 浅色/深色/系统 |
| `LanguageSelect.vue` | `v-model` | **占位，未实现** |
| `VersionInfo.vue` | `version?`（默认 `beta`） | 版本徽章 |

`views/SettingPage.vue` :42 用 computed 双向绑定 `themeMode` ↔ `themeStore.mode`。

## 新增 Todo 字段时的改动顺序

1. `internal/model/todo.go` 加字段 → 2. `internal/database/sql_service.go` `initAllTables` 加列 → 3. `internal/service/todo_service.go` 的 SQL 与 `rowToTodo` → 4. 重新生成 bindings → 5. `stores/todoStore.ts` → 6. 组件模板。
