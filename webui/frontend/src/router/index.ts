import { createRouter, createWebHistory } from 'vue-router'
import { setupApi } from '../api'

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

  // 访问 setup 页面时检查状态
  if (to.path === '/setup') {
    if (!setupChecked) {
      try {
        const res = await setupApi.status()
        if (res.code === 0) {
          needSetup = res.data.need_setup
          setupChecked = true
        }
      } catch {
        // 网络错误，允许访问
      }
    }
    // 已完成初始化，重定向到首页或登录页
    if (setupChecked && !needSetup) {
      return token ? '/' : '/login'
    }
    return
  }

  // 访问登录页
  if (to.path === '/login') {
    // 已登录则跳转首页
    if (token) {
      return '/'
    }
    return
  }

  // 访问其他页面，先检查 setup 状态
  if (!setupChecked) {
    try {
      const res = await setupApi.status()
      if (res.code === 0) {
        needSetup = res.data.need_setup
        setupChecked = true
      }
    } catch {
      // 网络错误，继续检查 token
    }
  }

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
