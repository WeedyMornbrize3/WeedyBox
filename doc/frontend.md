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

State：`todos` `loading` `error` `stats` `curPriority`（当前新建优先级，默认 1）

| Getter | 行 | 说明 |
| --- | --- | --- |
| `completedTodos` | 17 | 已完成列表 |
| `pendingTodos` | 21 | 未完成列表 |
| `highPriorityTodos` | 25 | priority === 2 |
| `completionRate` | 29 | 完成率百分比 |

| Action | 行 | 说明 |
| --- | --- | --- |
| `loadTodos()` | 35 | 并行拉取 list + stats |
| `addTodo(title, description?)` | 53 | 用 `curPriority` 创建，头部插入 |
| `toggleComplete(id)` | 70 | 切换并就地更新 |
| `updateTodo(id, updates)` | 86 | 局部更新 |
| `deleteTodo(id)` | 101 | 删除并过滤本地列表 |
| `loadStats()` | 113 | 仅刷新统计 |
| `init()` | 121 | 挂载时调用 → `loadTodos` |

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
| `layouts/MainLayout.vue` | `TitleBar` + `Sidebar` + `router-view`，按 store 设置左内边距 |
| `components/TitleBar.vue` | 无边框标题栏；拖拽区 `--wails-draggable: drag`；按钮调 `WindowService`（:33 最小化 / :41 最大化 / :50 关闭） |
| `components/Sidebar.vue` | 导航；`menuItemsUp` :87、`menuItemsDown` :94、`navigateTo(path)` :108 |

### TODO `components/todo/`

| 文件 | 关键成员 | 说明 |
| --- | --- | --- |
| `TodoList.vue` | `handleToggle` :56、`handleDelete` :60、`onMounted → init()` :66 | 列表容器，组合统计/输入/列表 |
| `TodoInput.vue` | `toggleDesc()` :62、`handleSubmit()` :73 | 标题输入 + 📝 描述开关（展开第二输入框）+ 添加按钮 |
| `TodoItem.vue` | `priorityColor` :81、`priorityLabel` :90、`formatDate` :100、`handleDelete` :117 | 标题主行 + 描述块（缩进 + 左竖线）；emits `toggle` `delete` `error`；props `{ todo }` |
| `TodoStats.vue` | `completionRate` :27 | props `{ stats }` |
| `TodoPrioritySelector.vue` | `priorityOptions` :23（value 0/1/2） | 直接写 `todoStore.curPriority` |

优先级色：0 绿 / 1 黄 / 2 红。

#### TodoInput 描述输入交互

| 成员 | 行 | 说明 |
| --- | --- | --- |
| `newTodo` / `desc` / `showDesc` | 56-58 | 标题、描述、描述框可见性 |
| `descInput` | 60 | 展开后自动聚焦的 input ref |
| `toggleDesc()` | 62 | 切换显示；收起时清空 `desc` |
| `handleSubmit()` | 73 | 调 `addTodo(title, desc.trim())`，成功后清空两个输入 |

- 描述框回车同样提交，Esc 收起。
- 描述框右留 `w-[8.25rem]` 占位，与上方「📝 + 添加」总宽对齐。
- 样式类 `desc-toggle` / `desc-toggle-active` 定义在组件 `<style scoped>`。

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
