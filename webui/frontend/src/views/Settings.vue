<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { systemApi, getToken } from '../api'

const loading = ref(false)
const geoipEnabled = ref(false)
const geoipFile = ref('')
const uploading = ref(false)
const corsOrigin = ref('*')

const passwordForm = ref({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const fetchSettings = async () => {
  loading.value = true
  try {
    const res = await systemApi.getSettings()
    if (res.code === 0) {
      geoipEnabled.value = res.data.geoip_enabled === 'true'
      geoipFile.value = res.data.geoip_file || ''
      corsOrigin.value = res.data.cors_origin || '*'
    }
  } finally {
    loading.value = false
  }
}

const saveCorsOrigin = async () => {
  const res = await systemApi.updateSettings('cors_origin', corsOrigin.value)
  if (res.code === 0) {
    ElMessage.success('CORS 设置已保存')
  } else {
    ElMessage.error(res.msg)
  }
}

const updateGeoIP = async (enabled: boolean) => {
  const res = await systemApi.updateSettings('geoip_enabled', String(enabled))
  if (res.code === 0) {
    ElMessage.success('设置已更新')
  } else {
    ElMessage.error(res.msg)
    geoipEnabled.value = !enabled
  }
}

const handleFileChange = async (uploadFile: { raw: File; name: string }) => {
  if (!uploadFile.name.endsWith('.mmdb')) {
    ElMessage.error('请上传 .mmdb 格式的 GeoIP 数据库文件')
    return false
  }

  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('file', uploadFile.raw)

    const response = await fetch('/api/upload/geoip', {
      method: 'POST',
      headers: {
        'Authorization': getToken()
      },
      body: formData
    })

    const res = await response.json()
    if (res.code === 0) {
      ElMessage.success('GeoIP 数据库上传成功')
      geoipFile.value = uploadFile.name
      fetchSettings()
    } else {
      ElMessage.error(res.msg || '上传失败')
    }
  } catch {
    ElMessage.error('上传失败')
  } finally {
    uploading.value = false
  }
  return false
}

const changePassword = async () => {
  if (!passwordForm.value.oldPassword || !passwordForm.value.newPassword) {
    ElMessage.error('请填写完整信息')
    return
  }
  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    ElMessage.error('两次密码输入不一致')
    return
  }
  if (passwordForm.value.newPassword.length < 6) {
    ElMessage.error('密码长度至少 6 位')
    return
  }

  const res = await systemApi.changePassword(
    passwordForm.value.oldPassword,
    passwordForm.value.newPassword
  )
  if (res.code === 0) {
    ElMessage.success('密码修改成功')
    passwordForm.value = { oldPassword: '', newPassword: '', confirmPassword: '' }
  } else {
    ElMessage.error(res.msg)
  }
}

onMounted(() => {
  fetchSettings()
})
</script>

<template>
  <div class="settings-page" v-loading="loading">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h3>系统设置</h3>
        <span class="header-subtitle">管理系统配置和安全选项</span>
      </div>
    </div>

    <!-- 设置网格 -->
    <div class="settings-grid">
      <!-- CORS 设置 -->
      <div class="setting-card">
        <div class="card-header">
          <div class="card-icon">
            <el-icon><Link /></el-icon>
          </div>
          <div class="card-title">
            <h4>跨域设置</h4>
            <p>CORS</p>
          </div>
        </div>
        <div class="card-body">
          <div class="form-group">
            <label class="form-label">允许的来源</label>
            <el-input
              v-model="corsOrigin"
              placeholder="* 表示允许所有，多个来源用逗号分隔"
              class="dark-input"
            />
            <div class="form-tip">
              <el-icon><InfoFilled /></el-icon>
              <span>示例：* 或 http://localhost:5173,https://example.com</span>
            </div>
          </div>
          <div class="form-actions">
            <el-button type="primary" @click="saveCorsOrigin">
              <el-icon><Check /></el-icon>
              保存设置
            </el-button>
          </div>
        </div>
      </div>

      <!-- GeoIP 设置 -->
      <div class="setting-card">
        <div class="card-header">
          <div class="card-icon">
            <el-icon><Location /></el-icon>
          </div>
          <div class="card-title">
            <h4>GeoIP 设置</h4>
            <p>IP 地理位置</p>
          </div>
        </div>
        <div class="card-body">
          <div class="form-group">
            <label class="form-label">启用 GeoIP</label>
            <div class="switch-row">
              <el-switch
                v-model="geoipEnabled"
                @change="updateGeoIP"
                :disabled="!geoipFile"
                class="green-switch"
              />
              <span v-if="!geoipFile" class="switch-tip">需要先上传 GeoIP 数据库</span>
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">GeoIP 数据库</label>
            <div v-if="geoipFile" class="current-file">
              <el-icon><Document /></el-icon>
              <span>{{ geoipFile }}</span>
              <el-icon class="file-status"><SuccessFilled /></el-icon>
            </div>
            <el-upload
              :auto-upload="false"
              :show-file-list="false"
              accept=".mmdb"
              :on-change="handleFileChange"
            >
              <el-button :loading="uploading">
                <el-icon><Upload /></el-icon>
                {{ geoipFile ? '更换文件' : '上传 MMDB 文件' }}
              </el-button>
            </el-upload>
            <div class="form-tip">
              <el-icon><InfoFilled /></el-icon>
              <span>支持 MaxMind GeoLite2 或 GeoIP2 格式的 .mmdb 文件</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 修改密码 -->
      <div class="setting-card">
        <div class="card-header">
          <div class="card-icon">
            <el-icon><Lock /></el-icon>
          </div>
          <div class="card-title">
            <h4>修改密码</h4>
            <p>安全管理</p>
          </div>
        </div>
        <div class="card-body">
          <div class="form-group">
            <label class="form-label">当前密码</label>
            <el-input
              v-model="passwordForm.oldPassword"
              type="password"
              placeholder="请输入当前密码"
              show-password
              class="dark-input"
            >
              <template #prefix>
                <el-icon><Lock /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="form-group">
            <label class="form-label">新密码</label>
            <el-input
              v-model="passwordForm.newPassword"
              type="password"
              placeholder="请输入新密码 (至少 6 位)"
              show-password
              class="dark-input"
            >
              <template #prefix>
                <el-icon><Unlock /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="form-group">
            <label class="form-label">确认新密码</label>
            <el-input
              v-model="passwordForm.confirmPassword"
              type="password"
              placeholder="请再次输入新密码"
              show-password
              class="dark-input"
            >
              <template #prefix>
                <el-icon><Unlock /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="form-actions">
            <el-button type="primary" @click="changePassword">
              <el-icon><Key /></el-icon>
              修改密码
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 页面容器 */
.settings-page {
  width: 100%;
}

/* 页面头部 */
.page-header {
  margin-bottom: 24px;
}

.header-left h3 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  background: linear-gradient(135deg, #fff, #d1fae5);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.header-subtitle {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
  margin-left: 12px;
}

/* 设置网格 */
.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(380px, 1fr));
  gap: 20px;
}

/* 设置卡片 */
.setting-card {
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* 卡片头部 */
.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  background: linear-gradient(to right, rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0.04));
}

.card-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #10b981, #059669);
  border-radius: 12px;
  color: #fff;
  font-size: 20px;
  flex-shrink: 0;
}

.card-title h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.card-title p {
  margin: 2px 0 0;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
}

/* 卡片主体 */
.card-body {
  padding: 20px;
  flex: 1;
}

/* 表单组 */
.form-group {
  margin-bottom: 20px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.8);
  margin-bottom: 8px;
}

/* 输入框样式 */
.form-group :deep(.dark-input) {
  height: 44px;
}

.form-group :deep(.dark-input .el-input__wrapper) {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  box-shadow: none;
  transition: all 0.3s ease;
}

.form-group :deep(.dark-input .el-input__wrapper:hover) {
  border-color: rgba(255, 255, 255, 0.15);
}

.form-group :deep(.dark-input .el-input__wrapper.is-focus) {
  background: rgba(255, 255, 255, 0.08);
  border-color: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

.form-group :deep(.dark-input .el-input__inner) {
  color: #fff;
}

.form-group :deep(.dark-input .el-input__inner::placeholder) {
  color: rgba(255, 255, 255, 0.4);
}

.form-group :deep(.dark-input .el-input__prefix) {
  color: rgba(255, 255, 255, 0.5);
}

/* 提示信息 */
.form-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
}

.form-tip .el-icon {
  font-size: 14px;
}

/* 开关行 */
.switch-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.switch-tip {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
}

/* 绿色开关 */
:deep(.green-switch .el-switch__input:checked + .el-switch__core) {
  background-color: #10b981;
  border-color: #10b981;
}

:deep(.green-switch .el-switch.is-disabled .el-switch__core) {
  opacity: 0.5;
}

/* 当前文件显示 */
.current-file {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.2);
  border-radius: 8px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.9);
  margin-bottom: 12px;
}

.current-file .el-icon {
  color: #10b981;
}

.file-status {
  margin-left: auto;
  color: #10b981;
}

/* 表单操作区 */
.form-actions {
  display: flex;
  gap: 12px;
  padding-top: 8px;
}

.form-actions .el-button {
  display: flex;
  align-items: center;
  gap: 6px;
}

.form-actions .el-button.el-button--primary {
  background: linear-gradient(135deg, #10b981, #059669);
  border: none;
}

.form-actions .el-button.el-button--primary:hover {
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
}

/* 响应式 */
@media (max-width: 767px) {
  .page-header {
    text-align: left;
  }

  .header-subtitle {
    margin-left: 0;
    display: block;
    margin-top: 4px;
  }

  .settings-grid {
    grid-template-columns: 1fr;
  }

  .form-actions {
    flex-direction: column;
  }

  .form-actions .el-button {
    width: 100%;
  }
}
</style>
