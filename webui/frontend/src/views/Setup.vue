<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { setupApi, setToken, systemApi } from '../api'

const router = useRouter()
const loading = ref(false)
const cardVisible = ref(false)
const form = ref({
  password: '',
  confirmPassword: ''
})

const submit = async () => {
  if (!form.value.password) {
    ElMessage.error('请输入密码')
    return
  }
  if (form.value.password !== form.value.confirmPassword) {
    ElMessage.error('两次密码输入不一致')
    return
  }
  if (form.value.password.length < 6) {
    ElMessage.error('密码长度至少 6 位')
    return
  }

  loading.value = true
  try {
    const res = await setupApi.init(form.value.password)
    if (res.code === 0) {
      ElMessage.success('初始化完成')
      // 自动登录
      const loginRes = await systemApi.login(form.value.password)
      if (loginRes.code === 0) {
        setToken(loginRes.data.token)
      }
      await router.push('/')
    } else {
      ElMessage.error(res.msg)
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  setTimeout(() => {
    cardVisible.value = true
  }, 100)
})
</script>

<template>
  <div class="setup-page">
    <!-- 背景装饰 -->
    <div class="bg-decoration">
      <div class="circle circle-1"></div>
      <div class="circle circle-2"></div>
      <div class="circle circle-3"></div>
      <div class="grid-pattern"></div>
    </div>

    <!-- 初始化卡片 -->
    <div class="setup-card" :class="{ 'card-visible': cardVisible }">
      <!-- Logo 区域 -->
      <div class="logo-section">
        <div class="logo-icon">
          <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M24 4L4 14V34L24 44L44 34V14L24 4Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M4 14L24 24L44 14" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <path d="M24 24V44" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
            <circle cx="24" cy="24" r="6" stroke="currentColor" stroke-width="2"/>
          </svg>
        </div>
        <h1 class="title">系统初始化</h1>
        <p class="subtitle">首次使用，请设置管理员密码</p>
      </div>

      <!-- 表单区域 -->
      <div class="form-section">
        <el-form :model="form" @submit.prevent label-position="top">
          <el-form-item label="管理员密码">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="请输入密码 (至少 6 位)"
              size="large"
              show-password
            >
              <template #prefix>
                <el-icon><Lock /></el-icon>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item label="确认密码">
            <el-input
              v-model="form.confirmPassword"
              type="password"
              placeholder="请再次输入密码"
              size="large"
              show-password
              @keyup.enter="submit"
            >
              <template #prefix>
                <el-icon><Lock /></el-icon>
              </template>
            </el-input>
          </el-form-item>

          <el-button
            type="primary"
            size="large"
            :loading="loading"
            @click="submit"
            class="submit-btn"
          >
            <span v-if="!loading">完成设置</span>
            <span v-else>设置中...</span>
          </el-button>
        </el-form>
      </div>

      <!-- 底部信息 -->
      <div class="footer-info">
        <span>Initial Setup Wizard</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 页面容器 */
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
}

/* 背景装饰 */
.bg-decoration {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  overflow: hidden;
  pointer-events: none;
}

/* 浮动圆圈 */
.circle {
  position: absolute;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1), rgba(5, 150, 105, 0.1));
  animation: float 20s ease-in-out infinite;
}

.circle-1 {
  width: 400px;
  height: 400px;
  top: -200px;
  left: -100px;
  animation-delay: 0s;
}

.circle-2 {
  width: 300px;
  height: 300px;
  bottom: -150px;
  right: -80px;
  animation-delay: -5s;
}

.circle-3 {
  width: 200px;
  height: 200px;
  top: 50%;
  right: 10%;
  animation-delay: -10s;
}

@keyframes float {
  0%, 100% {
    transform: translate(0, 0) scale(1);
  }
  25% {
    transform: translate(30px, -30px) scale(1.05);
  }
  50% {
    transform: translate(-20px, 20px) scale(0.95);
  }
  75% {
    transform: translate(20px, 10px) scale(1.02);
  }
}

/* 网格纹理 */
.grid-pattern {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.03) 1px, transparent 1px);
  background-size: 50px 50px;
}

/* 初始化卡片 - 玻璃拟态 */
.setup-card {
  position: relative;
  width: 420px;
  padding: 48px 40px;
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-radius: 24px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
  opacity: 0;
  transform: translateY(30px);
  transition: all 0.6s cubic-bezier(0.16, 1, 0.3, 1);
}

.setup-card.card-visible {
  opacity: 1;
  transform: translateY(0);
}

/* Logo 区域 */
.logo-section {
  text-align: center;
  margin-bottom: 32px;
}

.logo-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  margin-bottom: 20px;
  background: linear-gradient(135deg, #10b981, #059669);
  border-radius: 16px;
  color: #fff;
  box-shadow: 0 8px 24px rgba(16, 185, 129, 0.3);
}

.logo-icon svg {
  width: 36px;
  height: 36px;
}

.title {
  margin: 0 0 8px;
  font-size: 28px;
  font-weight: 700;
  background: linear-gradient(135deg, #fff, #d1fae5);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: 1px;
}

.subtitle {
  margin: 0;
  font-size: 14px;
  color: rgba(255, 255, 255, 0.6);
}

/* 表单区域 */
.form-section {
  margin-bottom: 32px;
}

.form-section :deep(.el-form-item__label) {
  color: rgba(255, 255, 255, 0.8);
  font-size: 14px;
  font-weight: 500;
}

.form-section :deep(.el-input) {
  height: 48px;
}

.form-section :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
  transition: all 0.3s ease;
}

.form-section :deep(.el-input__wrapper:hover) {
  border-color: rgba(255, 255, 255, 0.2);
}

.form-section :deep(.el-input__wrapper.is-focus) {
  background: rgba(255, 255, 255, 0.1);
  border-color: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

.form-section :deep(.el-input__inner) {
  color: #fff;
  font-size: 15px;
}

.form-section :deep(.el-input__inner::placeholder) {
  color: rgba(255, 255, 255, 0.4);
}

.form-section :deep(.el-input__prefix) {
  color: rgba(255, 255, 255, 0.5);
}

/* 提交按钮 */
.submit-btn {
  width: 100%;
  height: 48px;
  margin-top: 8px;
  font-size: 16px;
  font-weight: 600;
  background: linear-gradient(135deg, #10b981, #059669);
  border: none;
  transition: all 0.3s ease;
}

.submit-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(16, 185, 129, 0.4);
}

.submit-btn:active {
  transform: translateY(0);
}

/* 底部信息 */
.footer-info {
  text-align: center;
  padding-top: 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.footer-info span {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  letter-spacing: 2px;
  text-transform: uppercase;
}

/* 响应式 */
@media (max-width: 480px) {
  .setup-card {
    width: calc(100% - 32px);
    margin: 16px;
    padding: 36px 24px;
  }

  .circle-1,
  .circle-2,
  .circle-3 {
    display: none;
  }

  .title {
    font-size: 24px;
  }

  .logo-icon {
    width: 56px;
    height: 56px;
  }

  .logo-icon svg {
    width: 32px;
    height: 32px;
  }
}
</style>
