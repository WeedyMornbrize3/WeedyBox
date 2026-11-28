// frontend/src/config/brand.ts
// 品牌标识的唯一配置点（用户 2026-11-28：「logo 图标可能会更换，先用现在的 svg 图形，
// 留下接口」）。
//
// 换 logo 时**只改这一个文件**，侧边栏（components/SidebarBrand.vue）会自动跟着变：
//   · LOGO_SRC  —— 传一张图片（相对 src 的路径、或 import 进来的 URL、或 data URI）。
//                  一旦非空，优先于图标类名渲染，用来替换成自定义 svg/png。
//   · LOGO_ICON —— UnoCSS 图标类名（构建期内联的 lucide svg）。LOGO_SRC 为空时用它，
//                  也就是现在侧边栏上显示的那个「包裹」图形。
export const APP_NAME = 'WeedyBox'

/** 自定义 logo 图（留空 = 用 LOGO_ICON 的图标图形） */
export const LOGO_SRC = ''

/** 默认 logo：lucide 的 package（线性 2px，与全站图标同一族） */
export const LOGO_ICON = 'i-lucide-package'

/**
 * logo 盒子的边长（px）。20 = 与卡片标题图标（.icon-md）同尺寸，
 * 也和 n-menu 的 iconSize 一致 —— 品牌区与下面每一行菜单图标一样大
 * （用户 2026-11-28：「侧边栏展开项与其他元素大小 ui 一致」「整体调大 与 router 大小一致」）。
 */
export const LOGO_SIZE = 20
