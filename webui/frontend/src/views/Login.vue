<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { systemApi, setToken } from '../api'

const router = useRouter()
const loading = ref(false)
const password = ref('')
const cardVisible = ref(false)

// 视图模式: login | reset | reset_success
const viewMode = ref<'login' | 'reset' | 'reset_success'>('login')

// 重置密码表单
const resetForm = ref({
  recoveryKey: '',
  newPassword: '',
  confirmPassword: ''
})

// 新生成的恢复密钥
const newRecoveryKey = ref('')

const login = async () => {
  if (!password.value) {
    ElMessage.error('请输入密码')
    return
  }

  loading.value = true
  try {
    const res = await systemApi.login(password.value)
    if (res.code === 0) {
      setToken(res.data.token)
      router.replace('/')
    } else {
      ElMessage.error(res.msg)
      loading.value = false
    }
  } catch (e) {
    console.error('登录失败:', e)
    ElMessage.error('网络请求失败')
    loading.value = false
  }
}

const resetPassword = async () => {
  if (!resetForm.value.recoveryKey) {
    ElMessage.error('请输入恢复密钥')
    return
  }
  if (!resetForm.value.newPassword) {
    ElMessage.error('请输入新密码')
    return
  }
  if (resetForm.value.newPassword.length < 6) {
    ElMessage.error('密码长度至少 6 位')
    return
  }
  if (resetForm.value.newPassword !== resetForm.value.confirmPassword) {
    ElMessage.error('两次密码输入不一致')
    return
  }

  loading.value = true
  try {
    const res = await systemApi.resetPassword(
      resetForm.value.recoveryKey,
      resetForm.value.newPassword
    )
    if (res.code === 0) {
      newRecoveryKey.value = res.data.recovery_key
      viewMode.value = 'reset_success'
      ElMessage.success('密码重置成功')
    } else {
      ElMessage.error(res.msg)
    }
  } catch {
    ElMessage.error('网络请求失败')
  } finally {
    loading.value = false
  }
}

const copyKey = async () => {
  try {
    await navigator.clipboard.writeText(newRecoveryKey.value)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

const backToLogin = () => {
  viewMode.value = 'login'
  password.value = ''
  resetForm.value = { recoveryKey: '', newPassword: '', confirmPassword: '' }
  newRecoveryKey.value = ''
}

onMounted(() => {
  setTimeout(() => {
    cardVisible.value = true
  }, 100)
})
</script>

<template>
  <div class="login-page">
    <!-- 背景装饰 -->
    <div class="bg-decoration">
      <div class="circle circle-1"></div>
      <div class="circle circle-2"></div>
      <div class="circle circle-3"></div>
      <div class="grid-pattern"></div>
    </div>

    <!-- 登录卡片 -->
    <div class="login-card" :class="{ 'card-visible': cardVisible }">

      <!-- ========= 登录视图 ========= -->
      <template v-if="viewMode === 'login'">
        <div class="logo-section">
          <div class="logo-icon">
            <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M24 4L4 14V34L24 44L44 34V14L24 4Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M4 14L24 24L44 14" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M24 24V44" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <circle cx="24" cy="24" r="6" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <h1 class="title">Relay</h1>
          <p class="subtitle">TCP/UDP 端口转发管理平台</p>
        </div>

        <div class="form-section">
          <el-form label-position="top">
            <el-form-item label="管理员密码">
              <el-input
                v-model="password"
                type="password"
                placeholder="请输入密码"
                size="large"
                show-password
                @keyup.enter="login"
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
              :disabled="loading"
              @click="login"
              class="login-btn"
            >
              <span v-if="!loading">登 录</span>
              <span v-else>登录中...</span>
            </el-button>
          </el-form>
        </div>

        <div class="forgot-link">
          <a @click="viewMode = 'reset'">忘记密码？</a>
        </div>
      </template>

      <!-- ========= 重置密码视图 ========= -->
      <template v-else-if="viewMode === 'reset'">
        <div class="logo-section">
          <div class="logo-icon reset-icon">
            <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M20 16H12V8" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M12 16C14.5-11.5 19-8 24-8C32.8-8 40 -0.8 40 8V8C40 16.8 32.8 24 24 24C19 24 14.5 21.5 12 18" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" transform="translate(0 16)"/>
            </svg>
          </div>
          <h1 class="title">重置密码</h1>
          <p class="subtitle">输入恢复密钥以重置管理员密码</p>
        </div>

        <div class="form-section">
          <el-form label-position="top">
            <el-form-item label="恢复密钥">
              <el-input
                v-model="resetForm.recoveryKey"
                placeholder="请输入恢复密钥"
                size="large"
                class="mono-input"
              >
                <template #prefix>
                  <el-icon><Key /></el-icon>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item label="新密码">
              <el-input
                v-model="resetForm.newPassword"
                type="password"
                placeholder="请输入新密码 (至少 6 位)"
                size="large"
                show-password
              >
                <template #prefix>
                  <el-icon><Lock /></el-icon>
                </template>
              </el-input>
            </el-form-item>

            <el-form-item label="确认新密码">
              <el-input
                v-model="resetForm.confirmPassword"
                type="password"
                placeholder="请再次输入新密码"
                size="large"
                show-password
                @keyup.enter="resetPassword"
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
              :disabled="loading"
              @click="resetPassword"
              class="login-btn"
            >
              <span v-if="!loading">重置密码</span>
              <span v-else>重置中...</span>
            </el-button>
          </el-form>
        </div>

        <div class="forgot-link">
          <a @click="backToLogin">返回登录</a>
        </div>
      </template>

      <!-- ========= 重置成功视图 ========= -->
      <template v-else-if="viewMode === 'reset_success'">
        <div class="logo-section">
          <div class="logo-icon success-icon">
            <svg viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M20 25L23 28L30 20" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
              <circle cx="24" cy="24" r="18" stroke="currentColor" stroke-width="2"/>
            </svg>
          </div>
          <h1 class="title">密码已重置</h1>
          <p class="subtitle">请保存新的恢复密钥，旧密钥已失效</p>
        </div>

        <div class="recovery-section">
          <div class="warning-box">
            <svg class="warning-icon" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd"/>
            </svg>
            <span>此密钥仅显示一次，关闭后无法再次查看！</span>
          </div>

          <div class="key-display">
            <code>{{ newRecoveryKey }}</code>
          </div>

          <el-button type="primary" class="copy-btn" @click="copyKey">
            <el-icon><DocumentCopy /></el-icon>
            复制密钥
          </el-button>

          <el-button
            type="primary"
            size="large"
            @click="backToLogin"
            class="login-btn"
          >
            返回登录
          </el-button>
        </div>
      </template>

      <!-- 底部信息 -->
      <div class="footer-info">
        <span>Secure Access Gateway</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 页面容器 */
.login-page {
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

/* 登录卡片 - 玻璃拟态 */
.login-card {
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

.login-card.card-visible {
  opacity: 1;
  transform: translateY(0);
}

/* Logo 区域 */
.logo-section {
  text-align: center;
  margin-bottom: 40px;
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

.logo-icon.reset-icon {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  box-shadow: 0 8px 24px rgba(59, 130, 246, 0.3);
}

.logo-icon.success-icon {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  box-shadow: 0 8px 24px rgba(245, 158, 11, 0.3);
}

.logo-icon svg {
  width: 36px;
  height: 36px;
}

.title {
  margin: 0 0 8px;
  font-size: 32px;
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
  margin-bottom: 16px;
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

/* 恢复密钥输入框等宽字体 */
.mono-input :deep(.el-input__inner) {
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  letter-spacing: 0.5px;
}

/* 登录按钮 */
.login-btn {
  width: 100%;
  height: 48px;
  margin-top: 8px;
  font-size: 16px;
  font-weight: 600;
  background: linear-gradient(135deg, #10b981, #059669);
  border: none;
  transition: all 0.3s ease;
}

.login-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(16, 185, 129, 0.4);
}

.login-btn:active {
  transform: translateY(0);
}

/* 忘记密码链接 */
.forgot-link {
  text-align: center;
  margin-bottom: 24px;
}

.forgot-link a {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
  cursor: pointer;
  transition: color 0.2s;
  text-decoration: none;
}

.forgot-link a:hover {
  color: #10b981;
}

/* 恢复密钥展示区域 */
.recovery-section {
  margin-bottom: 16px;
}

.warning-box {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  border-radius: 10px;
  margin-bottom: 20px;
  font-size: 13px;
  color: #fbbf24;
}

.warning-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.key-display {
  padding: 16px;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  margin-bottom: 16px;
  text-align: center;
  word-break: break-all;
}

.key-display code {
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 15px;
  color: #10b981;
  letter-spacing: 1px;
  line-height: 1.8;
}

.copy-btn {
  width: 100%;
  height: 40px;
  margin-bottom: 12px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #fff;
}

.copy-btn:hover {
  background: rgba(255, 255, 255, 0.15);
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
  .login-card {
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
    font-size: 28px;
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
