import './styles/global.css'
import 'uno.css'
// vue-echarts 的宿主元素样式（x-vue-echarts 要 display:block + 100% 宽高，否则图表高度为 0）
import 'vue-echarts/style.css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/Theme'
import { useSidebarStore } from './stores/Sidebar'
import { useAppStore } from './stores/App'
import { syncNaiveTheme } from './styles/naive'
import { vScrollFade } from './utils/scrollFade'
// ECharts 按需注册（柱状/折线/提示框/网格）。必须在使用 <VChart> 之前跑一遍，
// 放在入口最稳：分析页是懒加载路由，但图表组件自己也会 import 这个模块（幂等）。
import './utils/echartsSetup'

const pinia = createPinia()
const app = createApp(App)

app.use(router)
app.use(pinia)
// 滚动容器边缘渐隐（参考 shadcn 的 scroll-fade；用法见 utils/scrollFade.ts）
app.directive('scroll-fade', vScrollFade)

const themeStore = useThemeStore()
themeStore.restoreState()
const sidebarStore = useSidebarStore()
sidebarStore.restoreState()
const appStore = useAppStore()
appStore.restoreState()

themeStore.setupSystemListener()

// 首次渲染前先把设计 token 喂给 naive-ui，避免启动瞬间闪一下 naive 的默认绿。
// 之后的主题切换由 App.vue 里的 watch 负责。
syncNaiveTheme(themeStore.isDark)

app.mount('#app')
