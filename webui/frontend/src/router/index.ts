import { createRouter, createWebHistory } from 'vue-router'
import { checkHealth } from '../api'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/setup',
      name: 'Setup',
      component: () => import('../views/Setup.vue')
    },
    {
      path: '/login',
      name: 'Login',
      component: () => import('../views/Login.vue')
    },
    {
      path: '/',
      component: () => import('../views/Layout.vue'),
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('../views/Dashboard.vue')
        },
        {
          path: 'monitor',
          name: 'Monitor',
          component: () => import('../views/Monitor.vue')
        },
        {
          path: 'settings',
          name: 'Settings',
          component: () => import('../views/Settings.vue')
        }
      ]
    }
  ]
})

// 应用状态
let needSetup = false
let initialized = false

// 初始化函数（只调用一次）
export async function initApp() {
  if (initialized) return
  const health = await checkHealth()
  needSetup = health?.need_setup ?? false
  initialized = true
}

// 路由守卫（同步检查）
router.beforeEach((to) => {
  const token = localStorage.getItem('token')

  // 未初始化时不拦截（等 initApp 完成后再导航）
  if (!initialized) {
    return
  }

  // 访问 setup 页面
  if (to.path === '/setup') {
    if (!needSetup) {
      return token ? '/' : '/login'
    }
    return
  }

  // 访问登录页
  if (to.path === '/login') {
    if (needSetup) {
      return '/setup'
    }
    if (token) {
      return '/'
    }
    return
  }

  // 访问其他页面
  if (needSetup) {
    return '/setup'
  }
  if (!token) {
    return '/login'
  }
})

export default router
