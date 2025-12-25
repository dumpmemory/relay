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

// 缓存 setup 状态
let setupChecked = false
let needSetup = false

// 路由守卫
router.beforeEach(async (to) => {
  const token = localStorage.getItem('token')

  // 检查健康状态（仅一次）
  if (!setupChecked) {
    const health = await checkHealth()
    if (health) {
      needSetup = health.need_setup
      setupChecked = true
    }
  }

  // 访问 setup 页面
  if (to.path === '/setup') {
    // 已完成初始化，重定向到首页或登录页
    if (setupChecked && !needSetup) {
      return token ? '/' : '/login'
    }
    return
  }

  // 访问登录页
  if (to.path === '/login') {
    // 需要初始化则跳转 setup
    if (needSetup) {
      return '/setup'
    }
    // 已登录则跳转首页
    if (token) {
      return '/'
    }
    return
  }

  // 访问其他页面
  // 需要初始化
  if (needSetup) {
    return '/setup'
  }

  // 未登录则跳转登录页
  if (!token) {
    return '/login'
  }
})

export default router
