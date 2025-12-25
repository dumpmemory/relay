<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { removeToken, systemApi, type VersionInfo } from '../api'

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)
const isMobile = ref(false)
const version = ref<VersionInfo | null>(null)

const menuItems = [
  { path: '/', icon: 'Grid', label: '转发规则' },
  { path: '/monitor', icon: 'Monitor', label: '实时监控' },
  { path: '/settings', icon: 'Setting', label: '系统设置' }
]

const activeMenu = computed(() => route.path)

const handleResize = () => {
  isMobile.value = window.innerWidth < 768
  if (isMobile.value) {
    collapsed.value = true
  }
}

const logout = async () => {
  await systemApi.logout()
  removeToken()
  router.push('/login')
}

const loadVersion = async () => {
  const res = await systemApi.version()
  if (res.code === 0) {
    version.value = res.data
  }
}

onMounted(() => {
  handleResize()
  window.addEventListener('resize', handleResize)
  loadVersion()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<template>
  <el-container class="layout-container">
    <el-aside :width="collapsed ? '64px' : '200px'" class="layout-aside">
      <div class="logo">
        <span v-if="!collapsed">Relay</span>
        <span v-else>R</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        :collapse="collapsed"
        :collapse-transition="false"
        router
        class="aside-menu"
      >
        <el-menu-item v-for="item in menuItems" :key="item.path" :index="item.path">
          <el-icon>
            <component :is="item.icon" />
          </el-icon>
          <template #title>{{ item.label }}</template>
        </el-menu-item>
      </el-menu>
      <div class="version-info" v-if="version">
        <template v-if="!collapsed">
          <span class="version-text">{{ version.version }}</span>
        </template>
        <template v-else>
          <el-tooltip :content="version.version" placement="right">
            <span class="version-dot">v</span>
          </el-tooltip>
        </template>
      </div>
    </el-aside>

    <el-container>
      <el-header class="layout-header">
        <el-icon class="toggle-btn" @click="collapsed = !collapsed">
          <Fold v-if="!collapsed" />
          <Expand v-else />
        </el-icon>
        <div class="header-right">
          <el-dropdown @command="logout">
            <span class="user-dropdown">
              <el-icon><User /></el-icon>
              <span class="username">管理员</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="layout-main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.layout-container {
  height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
}

.layout-aside {
  position: relative;
  background: rgba(30, 41, 59, 0.95);
  backdrop-filter: blur(20px);
  transition: width 0.2s;
  overflow: hidden;
  border-right: 1px solid rgba(255, 255, 255, 0.1);
}

.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 20px;
  font-weight: 700;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(5, 150, 105, 0.1));
}

.aside-menu {
  border-right: none;
  background: transparent;
  padding: 8px 0;
}

.aside-menu:not(.el-menu--collapse) {
  width: 200px;
}

.aside-menu .el-menu-item {
  color: rgba(255, 255, 255, 0.6);
  margin: 4px 8px;
  border-radius: 8px;
  transition: all 0.3s ease;
}

.aside-menu .el-menu-item:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.aside-menu .el-menu-item.is-active {
  background: linear-gradient(135deg, #10b981, #059669);
  color: #fff;
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.3);
}

.version-info {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 12px;
  text-align: center;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(0, 0, 0, 0.2);
}

.version-text {
  color: rgba(255, 255, 255, 0.4);
  font-size: 12px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.version-dot {
  color: rgba(255, 255, 255, 0.4);
  font-size: 12px;
  cursor: default;
}

.layout-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  padding: 0 20px;
  height: 56px;
}

.toggle-btn {
  font-size: 20px;
  cursor: pointer;
  color: rgba(255, 255, 255, 0.7);
  transition: all 0.3s ease;
}

.toggle-btn:hover {
  color: #10b981;
}

.header-right {
  display: flex;
  align-items: center;
}

.user-dropdown {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  color: rgba(255, 255, 255, 0.7);
  font-size: 14px;
  padding: 8px 16px;
  border-radius: 8px;
  transition: all 0.3s ease;
}

.user-dropdown:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.08);
}

.username {
  margin: 0 4px;
}

.layout-main {
  background: transparent;
  padding: 20px;
  overflow-y: auto;
}

/* 滚动条样式 */
.layout-main::-webkit-scrollbar {
  width: 6px;
}

.layout-main::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 3px;
}

.layout-main::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 3px;
}

.layout-main::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.3);
}

@media (max-width: 767px) {
  .layout-header {
    padding: 0 16px;
  }

  .layout-main {
    padding: 16px;
  }

  .username {
    display: none;
  }

  .logo {
    font-size: 18px;
  }
}
</style>
