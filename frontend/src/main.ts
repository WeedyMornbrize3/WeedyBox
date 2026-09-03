import './styles/global.css'
import 'uno.css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useThemeStore } from './stores/Theme'
import { useSidebarStore } from './stores/Sidebar'

const pinia = createPinia()
const app = createApp(App)

app.use(router)
app.use(pinia)

const themeStore = useThemeStore()
themeStore.restoreState()
const sidebarStore = useSidebarStore()
sidebarStore.restoreState()

themeStore.setupSystemListener()

app.mount('#app')
