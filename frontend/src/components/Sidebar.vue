<template>
  <aside class="sidebar" :style="{ width: sidebarStore.sidebarWidth + 'px' }">
    <!-- 顶部：品牌区（仅展开态）+ 折叠开关
         用户 2026-11-28：「侧边栏的 ui 图标不再是特殊的正方形按钮，改用正常元素折叠 ui 替代；
         在侧边栏展开时才显示 logo」「sideBar css 折叠栏与其他 ui 保持统一」
         「去掉折叠栏 ui 底部 border，hover 样式与 router 排列 ui 一致，大小一致」
         「加入折叠渐变 Collapse Transition」——
         · 折叠控件 = router 内容区里那些图标/文字按钮的同一套口径：n-button quaternary
           + size="small"（28px 高、12px 字、transparent 底、hover 浮出 --n-color-hover）、
           **方角**（与全站卡片/按钮 0–4px 圆角一致），不再是特殊的大圆钮；
         · 与品牌区共用菜单项那一行：左内边距 12px（+ nav 的 8px = 20px 起点，
           与菜单图标同一条竖线）、行高 40px；
         · 不再画底部 border（用户要求去掉），靠间距与菜单分节，不额外加线；
         · logo/名称的显示/隐藏走 naive 的 n-collapse-transition（高度+透明度渐变），
           折叠不再是硬切。 -->
    <div class="sidebar-top" :class="{ 'sidebar-top--collapsed': sidebarStore.isCollapsed }">
      <!-- naive 的折叠渐变（用户 2026-11-28：「加入折叠渐变 Collapse Transition」）：
           高度 + 透明度一起过渡，比 v-if 硬切自然；里面的品牌区宽度跟着容器走。 -->
      <n-collapse-transition :show="!sidebarStore.isCollapsed" class="sidebar-brand-slot">
        <SidebarBrand />
      </n-collapse-transition>

      <!-- 折叠按钮（2026-11-28）：
           · 去掉折叠态 hover 提示栏（原外套 n-tooltip，已移除；aria-label 仍在）；
           · 变体 = **quaternary**：这是站内**图标按钮**的统一口径（标题栏的最小化/
             最大化/关闭、设置页的主题切换、动作栏右侧那两颗图标钮都是它），
             悬停浮出 --n-color-hover（= bg-hover token）的圆角底色。
             （用户 2026-11-28「hover 背景」——之前用 text 变体是无底色，
              那是卡片 header 里**文字按钮**的口径，图标按钮不该跟它一致。） -->
      <n-button
        quaternary
        size="small"
        class="sidebar-toggle"
        :aria-label="sidebarStore.isCollapsed ? '展开侧边栏' : '折叠侧边栏'"
        :aria-expanded="!sidebarStore.isCollapsed"
        @click="sidebarStore.toggleSidebar"
      >
        <template #icon>
          <!-- 图标尺寸跟 router 内容区同档：卡片 header 的标题图标是 .icon-md(20px)，
               菜单图标也是 20px —— 折叠按钮原来用 icon-xs(14px) 明显偏小
               （2026-11-28 用户「hover css 与 ui 大小检查」实测发现）。 -->
          <span
            :class="sidebarStore.isCollapsed ? 'i-lucide-panel-left-open' : 'i-lucide-panel-left-close'"
            class="icon-md"
            aria-hidden="true"
          />
        </template>
      </n-button>
    </div>

    <!-- 上导航
         2026-11-25 起交给 naive 的 n-menu（用户要求「改用 naive ui 样式和逻辑」）：
         选中态 = 当前路由、折叠态只留图标并在 hover 时弹 tooltip、项间距/圆角/配色
         全部由 naive 的 Menu 主题驱动（见 styles/naiveTheme.ts 的 Menu 段）。 -->
    <nav class="sidebar-nav-top" aria-label="主导航">
      <n-menu v-bind="menuProps" :options="menuItemsUp" />
    </nav>

    <!-- 下导航（设置）：与上导航共用同一份 props，选中态同样跟着路由走 -->
    <nav class="sidebar-nav-bottom" aria-label="辅助导航">
      <n-menu v-bind="menuProps" :options="menuItemsDown" />
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NCollapseTransition, NButton, NMenu, type MenuNodeProps, type MenuOption } from 'naive-ui'
import { useSidebarStore } from '../stores/Sidebar'
import SidebarBrand from './SidebarBrand.vue'

const sidebarStore = useSidebarStore()
const router = useRouter()
const route = useRoute()

interface MenuItem {
  /** 路由路径直接当菜单 key：选中态 = 当前路由，不需要额外的映射表 */
  path: string
  label: string
  /** UnoCSS 图标类名（presetIcons 构建期内联）；不用 emoji——跨平台渲染不一致，且没法跟主题色 */
  icon: string
}

/** 图标尺寸：同时喂给 n-menu 的 icon-size / collapsed-icon-size（折叠态靠它居中） */
const ICON_SIZE = 20
/**
 * 折叠态菜单宽度 = 侧栏 72px − 上下容器的 8px 内边距 ×2。
 * naive 折叠时用 `collapsedWidth / 2 - iconSize / 2` 当左内边距，宽度给不准图标就不会居中。
 */
const MENU_COLLAPSED_WIDTH = 56

/** 展开态项左内边距：8px 容器内边距 + 12px，与换之前自绘 item 的 px-4 对齐 */
const MENU_INDENT = 12

// 上导航菜单项（从上到下排列）
// 注意：**作品库（/works）刻意不出现在侧栏** —— 它只从游戏库点封面进入
// （见 views/LibraryPage.vue 的 openWork），免得导航里出现两个"库"让人分不清。
const menuItemsUp: MenuOption[] = [
  { path: '/', label: '首页', icon: 'i-lucide-house' },
  { path: '/library', label: '游戏库', icon: 'i-lucide-gamepad-2' },
  // 数据分析（2026-11-28）：柱状图图标，与「游戏库」图标同档（20px icon-md）
  { path: '/stats', label: '数据分析', icon: 'i-lucide-chart-column' },
  { path: '/todo', label: 'TODO', icon: 'i-lucide-list-todo' },
  { path: '/testing', label: '测试', icon: 'i-lucide-flask-conical' },
].map(toMenuOption)

// 下导航菜单项（沉在底部）
const menuItemsDown: MenuOption[] = [
  { path: '/settings', label: '设置', icon: 'i-lucide-settings' },
].map(toMenuOption)

/**
 * MenuItem → n-menu 的 option。
 *
 * ⚠️ **刻意不设 `label`**（2026-11-28 用户要求「折叠状态下的 hover 提示栏去掉」）：
 *    naive 的菜单折叠 tooltip 是这么来的（node_modules/naive-ui/es/menu/src/）：
 *      utils.mjs:  title = rawNode.title || rawNode[labelField]   ← label 会自动变成 title
 *      MenuOption: <n-tooltip :disabled="!dropdownEnabled || title === undefined">
 *    也就是说：只要给了 `label`，折叠态就必然弹一个显示标题的 tooltip。
 *    不设 label → title 为 undefined → tooltip 自然 disabled，浮层就没了。
 *
 *    文字改由 **menuProps 上的 `renderLabel`** 渲染（注意：这是 n-menu 的 props，
 *    不是 option 的字段；写在 option 里不会生效，实测菜单会变成纯图标）。
 *
 * icon 用渲染函数而不是内置图标组件：项目图标是 UnoCSS 的 `i-lucide-*` 类（构建期内联成
 * data URI），图标盒子（宽高 / 字号 / 右外边距）由 naive 的 `.n-menu-item-content__icon` 负责。
 */
function toMenuOption(item: MenuItem): MenuOption {
  return {
    key: item.path,
    // 不写 label（写了就会触发折叠态 tooltip）；显示文字由 renderLabel 统一渲染
    icon: () => h('span', { class: item.icon, 'aria-hidden': 'true' }),
  }
}

const navigateTo = (path: string) => {
  if (path !== route.path) void router.push(path)
}

/**
 * naive 的菜单项只给了 role="menuitem"，本身不可聚焦；这里用官方扩展点 `nodeProps`
 * 补 tabindex 与回车/空格，键盘也能切页（焦点环由 global.css 的 :focus-visible 兜底）。
 *
 * ⚠️ naive 把 nodeProps 的返回类型定成 `HTMLAttributes & Record<string, string | number | undefined>`
 *    ——函数类型的 onKeydown 与那个 Record 天然互斥（库自身的类型缺口），所以这里做一次显式转换。
 */
const menuNodeProps = ((option: MenuOption) => ({
  tabindex: 0,
  onKeydown: (event: KeyboardEvent) => {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      navigateTo(String(option.key))
    }
  },
})) as unknown as MenuNodeProps

/**
 * 菜单文字渲染器（**必须挂在 n-menu 的 props 上**，不是 option 字段）。
 * naive 的 `renderLabel(rawNode)` 拿到的是 option 原对象，所以直接用 `key → 文案` 映射查表：
 * 这样 option 里就能彻底不写 `label`（写了会触发折叠态 tooltip），文字仍然正常显示。
 */
const MENU_LABELS = new Map<string, string>([
  ['/', '首页'],
  ['/library', '游戏库'],
  ['/stats', '数据分析'],
  ['/todo', 'TODO'],
  ['/testing', '测试'],
  ['/settings', '设置'],
])

const renderMenuLabel = (option: MenuOption) => MENU_LABELS.get(String(option.key)) ?? ''

/** 上下两个 n-menu 共用的 props（选中值、折叠配置、文字渲染、跳转回调） */
const menuProps = computed(() => ({
  value: route.path,
  collapsed: sidebarStore.isCollapsed,
  collapsedWidth: MENU_COLLAPSED_WIDTH,
  iconSize: ICON_SIZE,
  collapsedIconSize: ICON_SIZE,
  indent: MENU_INDENT,
  rootIndent: MENU_INDENT,
  nodeProps: menuNodeProps,
  renderLabel: renderMenuLabel,
  'onUpdate:value': (key: string | number) => navigateTo(String(key)),
}))
</script>

<style scoped>
/* ===== 顶部条：展开态「logo + 名称 + 开关」，折叠态只有开关（居中）=====
   用户 2026-11-28：「sideBar css 折叠栏与其他 ui 保持统一」「与其他 router ui 一起排布」
   「去掉折叠栏 ui 底部 border」「hover 样式与 router 排列 ui 一致 大小一致」
   「整体调大 与 router 大小一致」「直接复用 router ui css 样式」
   「侧边栏折叠时有折叠 ui 明显卡顿 修改」。
   统一口径 = 直接照搬 router 内容区 n-card header 的数值（实测 50 行高 / 20 图标 /
   15 标题 / 10 间距 / 12 内边距），这样侧栏顶行与右侧「游戏库」卡片 header 同高同节奏。

   ⚠️ 卡顿的真正原因（2026-11-28 实测，不是掉帧：帧间隔全程 7ms、无 longtask）：
     折叠态原来用 `justify-content: center` 把按钮居中，而展开态是左对齐 padding 布局。
     侧栏宽度是 300ms 过渡的，`justify-content` 却**在折叠瞬间立刻切换** →
     按钮位置从 x≈218 直接瞬移到 x≈21（实测单帧跳 37px），看起来就是"卡一下"。
     修法：**所有会变的量都改成可过渡的布局量**，不用 justify 切换：
       · 按钮用 `margin-left: auto` 锚在右侧 → x 恒等于「侧栏宽 − 右内边距 − 28」；
       · 右内边距 12px ↔ 22px 参与过渡（72−22−28 = 22 = 折叠态的中线位置）；
       · 品牌区宽度 0 ↔ auto 参与过渡（折叠时不再占位，按钮才会真正落到中线）。
     三者都随侧栏宽度连续变化，按钮全程平滑滑动。 */
.sidebar-top {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  height: 50px;
  margin: 24px 0 0;
  padding: 0 12px;
  overflow: hidden;
  transition:
    padding-left 300ms cubic-bezier(0.4, 0, 0.2, 1),
    padding-right 300ms cubic-bezier(0.4, 0, 0.2, 1);
}

/* 折叠态：左内边距归零、右内边距 22px。
   72px − 22px(右) − 28px(按钮) = 22px，按钮正好落在 72px 的中线上。 */
.sidebar-top--collapsed {
  padding-left: 0;
  padding-right: 22px;
}

/* 折叠按钮：**图标按钮的统一口径**（用户 2026-11-28「直接复用 router ui css 样式」
   「大小一致」「hover 背景」）。
   站内有两类按钮，别混：
     · **图标按钮**（标题栏最小化/最大化/关闭、设置页主题切换、动作栏右侧两颗、
       游戏库批量面板）：`quaternary` + 28px 方/圆盒，hover 浮出 --n-color-hover 底色；
     · **文字按钮**（卡片 header 的「刷新 / 添加游戏 / 批量操作」）：`text`，
       hover 无底色、只提亮文字。
   折叠钮属于前者 → 用 quaternary 才会「hover 有背景」，与其它图标按钮一致。
   ⚠️ 不能在 class 里覆盖 --n-color-hover：naive 把 --n-* 以内联样式写在按钮元素上，
      外部 class 赢不过内联值（2026-11-28 实测无效）。要换反馈只能换变体。 */
.sidebar-toggle {
  flex-shrink: 0;
  /* 28×28：与站内其它图标按钮同尺寸；圆角交给 Button.borderRadiusSmall(4px)，
     与批量面板那几颗图标钮一致（小方角，不是圆角）。 */
  width: 28px;
  height: 28px;
  padding: 0;
  /* 锚在右侧：按钮 x 由「侧栏宽 − 右内边距 − 28」决定，随宽度过渡平滑移动 */
  margin-left: auto;
}

/* n-collapse-transition 会在品牌区外面包一层 div：展开态占满剩余宽度；
   折叠态宽度收到 0（参与过渡），按钮才会真正落到中线上。
   n-collapse-transition 自己负责高度与透明度渐变（用户要求的 Collapse Transition）。 */
.sidebar-brand-slot {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  transition:
    max-width 300ms cubic-bezier(0.4, 0, 0.2, 1),
    flex-grow 300ms cubic-bezier(0.4, 0, 0.2, 1);
}

.sidebar-top--collapsed .sidebar-brand-slot {
  flex: 0 0 0;
  max-width: 0;
}
</style>
