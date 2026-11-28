// frontend/src/utils/echartsSetup.ts
//
// ECharts 的**按需注册**入口（2026-11-28 新建分析页时加）。
//
// 为什么不用 `import * as echarts from 'echarts'`：那会把全部图表类型（地图/桑基/
// 矩形树图/GL…）一起打进产物。分析页只用到柱状 + 折线 + 提示框 + 网格，
// 按需注册能省掉几百 KB（Wails 把整个 dist 嵌进 exe，体积是实打实的）。
//
// 只注册一次、且必须在使用 <VChart> 之前完成 —— 所以 main.ts 与图表组件都从这里导入，
// 谁先加载都安全（模块本身幂等）。
import { use } from 'echarts/core'
import { BarChart, LineChart, PieChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  MarkLineComponent,
  DataZoomComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { ComposeOption } from 'echarts/core'
import type { BarSeriesOption, LineSeriesOption, PieSeriesOption } from 'echarts/charts'
import type {
  GridComponentOption,
  TooltipComponentOption,
  LegendComponentOption,
  MarkLineComponentOption,
  DataZoomComponentOption,
} from 'echarts/components'

use([
  BarChart,
  LineChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  MarkLineComponent,
  DataZoomComponent,
  CanvasRenderer,
])

/**
 * 本页用到的 option 类型。
 *
 * 为什么不用 `import type { EChartsOption } from 'echarts'`：
 * 那是**全量**类型（包含地图/桑基/树图等本次没注册的图表），
 * 用它写配置会让 TS 允许写下运行时根本没注册的 series 类型 —— 编译过、运行空白。
 * ComposeOption 只组合真正注册过的部分，写错图表类型会当场报错。
 */
export type StatsChartOption = ComposeOption<
  | BarSeriesOption
  | LineSeriesOption
  | PieSeriesOption
  | GridComponentOption
  | TooltipComponentOption
  | LegendComponentOption
  | MarkLineComponentOption
  | DataZoomComponentOption
>

export {}
