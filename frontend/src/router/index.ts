// frontend/src/router/index.ts
import { createRouter, createWebHashHistory } from 'vue-router'
import MainLayout from '../layouts/MainLayout.vue'

const routes = [
  {
    path: '/',
    component: MainLayout,
    children: [
      {
        path: '',
        name: 'Home',
        component: () => import('../views/HomePage.vue'),
        meta: { title: '首页' }
      },
      {
        path: '/settings',
        name: 'Settings',
        component: () => import('../views/SettingPage.vue'),
        meta: { title: '设置' }
      },
      {
        path: '/todo',
        name: 'Todo',
        component: () => import('../views/TodoPage.vue'),
        meta: { title: 'TODO' }
      },
      {
        path: '/library',
        name: 'Library',
        component: () => import('../views/LibraryPage.vue'),
        meta: { title: '游戏库' }
      },
      {
        // 作品库：游戏库卡片点击后进这里看详情（views/WorksPage.vue 复用）
        path: '/works',
        name: 'Works',
        component: () => import('../views/WorksPage.vue'),
        meta: { title: '作品库' }
      },
      {
        path: '/testing',
        name: 'Testing',
        component: () => import('../views/test.vue'),
        meta: { title: '接口测试' }
      },
      // 旧路径保留为重定向，避免老书签 404（组件仍是同一个 test.vue）
      { path: '/test', redirect: '/testing' },
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router