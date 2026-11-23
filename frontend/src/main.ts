import './styles/global.css'
import 'uno.css'
// vue-sonner 的基础样式（全局唯一一份 toast 容器，外观在 components/AppToaster.vue 里覆盖）
import 'vue-sonner/style.css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/Theme'
import { useSidebarStore } from './stores/Sidebar'
import { useAppStore } from './stores/App'

const pinia = createPinia()
const app = createApp(App)

app.use(router)
app.use(pinia)

const themeStore = useThemeStore()
themeStore.restoreState()
const sidebarStore = useSidebarStore()
sidebarStore.restoreState()
const appStore = useAppStore()
appStore.restoreState()

themeStore.setupSystemListener()

app.mount('#app')
