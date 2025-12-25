<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { relayApi, type RelayRule } from '../api'
import { useWebSocket, type TrafficData } from '../composables/useWebSocket'

const router = useRouter()
const loading = ref(false)
const rules = ref<RelayRule[]>([])
const importInput = ref<HTMLInputElement | null>(null)

// WebSocket 实时数据
const { traffic, subscribe, unsubscribe } = useWebSocket()

// 网速计算
interface SpeedData {
  bytesInSpeed: number
  bytesOutSpeed: number
  connections: number
}
const speedData = ref<Map<string, SpeedData>>(new Map())
const lastTraffic = ref<Map<string, TrafficData>>(new Map())
let speedTimer: number | null = null

// 计算网速
const calculateSpeed = () => {
  rules.value.forEach(rule => {
    if (!rule.running) return
    const current = traffic.value.get(rule.id)
    const last = lastTraffic.value.get(rule.id)

    if (current && last) {
      const bytesInSpeed = Math.max(0, current.bytes_in - last.bytes_in)
      const bytesOutSpeed = Math.max(0, current.bytes_out - last.bytes_out)
      speedData.value.set(rule.id, {
        bytesInSpeed,
        bytesOutSpeed,
        connections: current.connections
      })
    } else if (current) {
      speedData.value.set(rule.id, {
        bytesInSpeed: 0,
        bytesOutSpeed: 0,
        connections: current.connections
      })
    }

    if (current) {
      lastTraffic.value.set(rule.id, { ...current })
    }
  })
}

// 格式化网速
const formatSpeed = (bytesPerSec: number): string => {
  if (bytesPerSec === 0) return '0 B/s'
  const k = 1024
  const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  const i = Math.floor(Math.log(bytesPerSec) / Math.log(k))
  return parseFloat((bytesPerSec / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

// 获取规则的速度数据
const getSpeed = (ruleId: string): SpeedData => {
  return speedData.value.get(ruleId) || { bytesInSpeed: 0, bytesOutSpeed: 0, connections: 0 }
}

// 跳转到监控页面
const goToMonitor = (ruleId: string) => {
  router.push({ path: '/monitor', query: { relay: ruleId } })
}

// 订阅运行中的 relay
watch(() => rules.value, (newRules) => {
  newRules.forEach(rule => {
    if (rule.running) {
      subscribe(rule.id)
    }
  })
}, { deep: true })
const dialogVisible = ref(false)
const dialogTitle = ref('新建规则')
const form = ref({
  id: '',
  name: '',
  src: '',
  dst: '',
  protocol: 'tcp'
})

const fetchRules = async () => {
  loading.value = true
  try {
    const res = await relayApi.list()
    if (res.code === 0) {
      rules.value = res.data || []
    } else {
      ElMessage.error(res.msg)
    }
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  dialogTitle.value = '新建规则'
  form.value = { id: '', name: '', src: '', dst: '', protocol: 'tcp' }
  dialogVisible.value = true
}

const openEdit = (row: RelayRule) => {
  dialogTitle.value = '编辑规则'
  form.value = {
    id: row.id,
    name: row.name,
    src: row.src,
    dst: row.dst,
    protocol: row.protocol
  }
  dialogVisible.value = true
}

const submitForm = async () => {
  if (!form.value.name || !form.value.src || !form.value.dst) {
    ElMessage.error('请填写完整信息')
    return
  }

  try {
    let res
    if (form.value.id) {
      res = await relayApi.update(
        form.value.id,
        form.value.name,
        form.value.src,
        form.value.dst,
        form.value.protocol
      )
    } else {
      res = await relayApi.create(
        form.value.name,
        form.value.src,
        form.value.dst,
        form.value.protocol
      )
    }

    if (res.code === 0) {
      ElMessage.success(form.value.id ? '更新成功' : '创建成功')
      dialogVisible.value = false
      fetchRules()
    } else {
      ElMessage.error(res.msg)
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

const deleteRule = async (row: RelayRule) => {
  try {
    await ElMessageBox.confirm('确定删除该规则?', '提示', {
      type: 'warning'
    })
    const res = await relayApi.delete(row.id)
    if (res.code === 0) {
      ElMessage.success('删除成功')
      fetchRules()
    } else {
      ElMessage.error(res.msg)
    }
  } catch {
    // 取消
  }
}

const startRelay = async (row: RelayRule) => {
  const res = await relayApi.start(row.id)
  if (res.code === 0) {
    ElMessage.success('启动成功')
    fetchRules()
  } else {
    ElMessage.error(res.msg)
  }
}

const stopRelay = async (row: RelayRule) => {
  const res = await relayApi.stop(row.id)
  if (res.code === 0) {
    ElMessage.success('停止成功')
    fetchRules()
  } else {
    ElMessage.error(res.msg)
  }
}

// 全部开始（仅启用的规则）
const startAll = async () => {
  const res = await relayApi.startAll()
  if (res.code === 0) {
    ElMessage.success('已启动所有启用的规则')
    fetchRules()
  } else {
    ElMessage.error(res.msg)
  }
}

// 全部停止
const stopAll = async () => {
  const res = await relayApi.stopAll()
  if (res.code === 0) {
    ElMessage.success('已停止所有规则')
    fetchRules()
  } else {
    ElMessage.error(res.msg)
  }
}

// 设置启用/禁用状态
const setEnabled = async (rule: RelayRule, enabled: boolean) => {
  const res = await relayApi.setEnabled(rule.id, enabled)
  if (res.code === 0) {
    rule.enabled = enabled
    if (!enabled && rule.running) {
      rule.running = false
    }
  } else {
    ElMessage.error(res.msg)
  }
}

// 计算统计信息
const hasRunning = () => rules.value.some(r => r.running)
const hasEnabledNotRunning = () => rules.value.some(r => r.enabled && !r.running)

const formatProtocol = (protocol: string) => {
  if (protocol === 'both') return 'TCP+UDP'
  return protocol.toUpperCase()
}

// 导出规则
const exportRules = async () => {
  try {
    const res = await relayApi.exportRules()
    if (res.code === 0 && res.data) {
      // 转换为导出格式，只保留必要字段
      const exportData = res.data.map(rule => ({
        name: rule.name,
        src: rule.src,
        dst: rule.dst,
        protocol: rule.protocol
      }))
      const jsonStr = JSON.stringify(exportData, null, 2)
      const blob = new Blob([jsonStr], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `relay-rules-${new Date().toISOString().slice(0, 10)}.json`
      a.click()
      URL.revokeObjectURL(url)
      ElMessage.success(`已导出 ${exportData.length} 条规则`)
    } else {
      ElMessage.error(res.msg || '导出失败')
    }
  } catch {
    ElMessage.error('导出失败')
  }
}

// 触发导入
const triggerImport = () => {
  importInput.value?.click()
}

// 处理导入文件
const handleImportFile = async (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  try {
    const text = await file.text()
    const rules = JSON.parse(text)

    if (!Array.isArray(rules)) {
      ElMessage.error('无效的配置文件格式')
      return
    }

    // 验证规则格式
    const validRules = rules.filter(r =>
      r && typeof r.name === 'string' &&
      typeof r.src === 'string' &&
      typeof r.dst === 'string'
    )

    if (validRules.length === 0) {
      ElMessage.error('没有找到有效的规则')
      return
    }

    await ElMessageBox.confirm(
      `即将导入 ${validRules.length} 条规则，是否继续?`,
      '确认导入',
      { type: 'info' }
    )

    const res = await relayApi.importRules(validRules)
    if (res.code === 0) {
      const result = res.data as { created?: number; skipped?: number } | null
      const created = result?.created ?? validRules.length
      const skipped = result?.skipped ?? 0
      if (skipped > 0) {
        ElMessage.success(`导入完成: 新增 ${created} 条, 跳过 ${skipped} 条 (监听地址已存在)`)
      } else {
        ElMessage.success(`成功导入 ${created} 条规则`)
      }
      fetchRules()
    } else {
      ElMessage.error(res.msg || '导入失败')
    }
  } catch (e) {
    // ElMessageBox 取消时抛出 'cancel' 或 { action: 'cancel' }
    const isCancel = e === 'cancel' || (e && typeof e === 'object' && (e as Record<string, unknown>).action === 'cancel')
    if (!isCancel) {
      console.error('导入错误:', e)
      const msg = e instanceof SyntaxError ? 'JSON 格式错误' : '导入失败'
      ElMessage.error(msg)
    }
  } finally {
    // 清空 input 以便重复选择同一文件
    target.value = ''
  }
}

onMounted(() => {
  fetchRules()
  // 每秒计算网速
  speedTimer = window.setInterval(calculateSpeed, 1000)
})

onUnmounted(() => {
  if (speedTimer) {
    clearInterval(speedTimer)
  }
  // 取消订阅
  rules.value.forEach(rule => {
    if (rule.running) {
      unsubscribe(rule.id)
    }
  })
})
</script>

<template>
  <div class="dashboard-page">
    <!-- 隐藏的导入文件 input -->
    <input
      ref="importInput"
      type="file"
      accept=".json"
      style="display: none"
      @change="handleImportFile"
    />

    <div class="page-header">
      <h3>转发规则</h3>
      <div class="header-actions">
        <el-button
          type="success"
          :disabled="!hasEnabledNotRunning()"
          @click="startAll"
        >
          <el-icon><VideoPlay /></el-icon>
          全部开始
        </el-button>
        <el-button
          type="warning"
          :disabled="!hasRunning()"
          @click="stopAll"
        >
          <el-icon><VideoPause /></el-icon>
          全部停止
        </el-button>
        <el-divider direction="vertical" />
        <el-button @click="triggerImport">
          <el-icon><Upload /></el-icon>
          导入
        </el-button>
        <el-button @click="exportRules" :disabled="rules.length === 0">
          <el-icon><Download /></el-icon>
          导出
        </el-button>
        <el-button type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>
          新建规则
        </el-button>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!loading && rules.length === 0" class="empty-state">
      <el-icon class="empty-icon"><DocumentAdd /></el-icon>
      <p>暂无转发规则</p>
      <el-button type="primary" @click="openCreate">创建第一条规则</el-button>
    </div>

    <!-- 卡片网格 -->
    <div v-else class="cards-grid" v-loading="loading">
      <div
        v-for="rule in rules"
        :key="rule.id"
        class="rule-card"
        :class="{ 'card-running': rule.running, 'card-disabled': !rule.enabled }"
      >
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="rule-name">
            <el-icon><Connection /></el-icon>
            <span>{{ rule.name }}</span>
          </div>
          <div class="header-right">
            <el-tag :type="rule.protocol === 'tcp' ? 'primary' : rule.protocol === 'udp' ? 'success' : 'warning'" size="small">
              {{ formatProtocol(rule.protocol) }}
            </el-tag>
            <el-switch
              :model-value="rule.enabled"
              size="small"
              :disabled="rule.running"
              :title="rule.running ? '运行中无法禁用' : (rule.enabled ? '点击禁用' : '点击启用')"
              @change="(val: boolean) => setEnabled(rule, val)"
            />
          </div>
        </div>

        <!-- 卡片主体 -->
        <div class="card-body">
          <div class="rule-info">
            <span class="label">监听</span>
            <span class="value">{{ rule.src }}</span>
          </div>
          <div class="rule-info">
            <span class="label">目标</span>
            <span class="value">{{ rule.dst }}</span>
          </div>
          <div class="rule-status" :class="{ clickable: rule.running }" @click="rule.running && goToMonitor(rule.id)">
            <span :class="['status-indicator', rule.running ? 'running' : 'stopped']"></span>
            <span :class="['status-text', rule.running ? 'running' : 'stopped']">
              {{ rule.running ? '运行中' : '已停止' }}
            </span>
            <!-- 运行时显示实时数据 -->
            <template v-if="rule.running">
              <span class="status-divider">|</span>
              <span class="status-stats">
                <el-icon><User /></el-icon>
                {{ getSpeed(rule.id).connections }}
              </span>
              <span class="status-stats speed-in">
                <el-icon><Download /></el-icon>
                {{ formatSpeed(getSpeed(rule.id).bytesInSpeed) }}
              </span>
              <span class="status-stats speed-out">
                <el-icon><Upload /></el-icon>
                {{ formatSpeed(getSpeed(rule.id).bytesOutSpeed) }}
              </span>
              <el-icon class="monitor-link"><Right /></el-icon>
            </template>
          </div>
        </div>

        <!-- 卡片操作栏 -->
        <div class="card-footer">
          <el-button
            v-if="!rule.running"
            type="success"
            size="small"
            :disabled="!rule.enabled"
            :title="!rule.enabled ? '请先启用此规则' : ''"
            @click="startRelay(rule)"
          >
            <el-icon><VideoPlay /></el-icon>
            启动
          </el-button>
          <el-button
            v-else
            type="warning"
            size="small"
            @click="stopRelay(rule)"
          >
            <el-icon><VideoPause /></el-icon>
            停止
          </el-button>
          <el-button type="primary" size="small" @click="openEdit(rule)">
            <el-icon><Edit /></el-icon>
            编辑
          </el-button>
          <el-button
            type="danger"
            size="small"
            :disabled="rule.running"
            @click="deleteRule(rule)"
          >
            <el-icon><Delete /></el-icon>
            删除
          </el-button>
        </div>
      </div>
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="480px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如: Web服务器" />
        </el-form-item>
        <el-form-item label="协议">
          <el-radio-group v-model="form.protocol">
            <el-radio value="tcp">仅 TCP</el-radio>
            <el-radio value="udp">仅 UDP</el-radio>
            <el-radio value="both">TCP + UDP</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="监听地址">
          <el-input v-model="form.src" placeholder="例如: 0.0.0.0:8080" />
        </el-form-item>
        <el-form-item label="目标地址">
          <el-input v-model="form.dst" placeholder="例如: 192.168.1.100:80" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
/* 页面容器 */
.dashboard-page {
  width: 100%;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-header h3 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  background: linear-gradient(135deg, #fff, #d1fae5);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.header-actions .el-button {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.empty-icon {
  font-size: 72px;
  color: rgba(255, 255, 255, 0.2);
  margin-bottom: 20px;
}

.empty-state p {
  font-size: 15px;
  color: rgba(255, 255, 255, 0.5);
  margin-bottom: 24px;
}

/* 卡片网格布局 */
.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

/* 规则卡片 - 玻璃拟态 */
.rule-card {
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  overflow: hidden;
  transition: all 0.3s ease;
  display: flex;
  flex-direction: column;
}

.rule-card:hover {
  background: rgba(255, 255, 255, 0.08);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  transform: translateY(-4px);
  border-color: rgba(16, 185, 129, 0.3);
}

.rule-card.card-running {
  border-color: rgba(16, 185, 129, 0.4);
  box-shadow: 0 0 20px rgba(16, 185, 129, 0.15);
}

.rule-card.card-disabled {
  opacity: 0.6;
  border-color: rgba(255, 255, 255, 0.05);
}

.rule-card.card-disabled:hover {
  border-color: rgba(255, 255, 255, 0.1);
}

/* 卡片头部 */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 18px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  background: linear-gradient(to right, rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0.04));
}

.rule-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
}

.rule-name .el-icon {
  font-size: 20px;
  color: #10b981;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-right .el-switch {
  --el-switch-on-color: #10b981;
}

/* 卡片主体 */
.card-body {
  padding: 20px;
  flex: 1;
}

.rule-info {
  display: flex;
  margin-bottom: 14px;
}

.rule-info .label {
  min-width: 44px;
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
}

.rule-info .value {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.9);
  font-family: 'Consolas', 'Monaco', monospace;
  word-break: break-all;
}

.rule-status {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-top: 14px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

/* 状态指示器 */
.status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  animation: pulse 2s ease-in-out infinite;
}

.status-indicator.running {
  background: #10b981;
  box-shadow: 0 0 12px rgba(16, 185, 129, 0.6);
}

.status-indicator.stopped {
  background: rgba(255, 255, 255, 0.3);
  animation: none;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    box-shadow: 0 0 12px rgba(16, 185, 129, 0.6);
  }
  50% {
    opacity: 0.7;
    box-shadow: 0 0 6px rgba(16, 185, 129, 0.3);
  }
}

.status-text {
  font-size: 13px;
}

.status-text.running {
  color: #10b981;
  font-weight: 600;
}

.status-text.stopped {
  color: rgba(255, 255, 255, 0.4);
}

.rule-status.clickable {
  cursor: pointer;
  transition: background 0.2s;
  margin: 0 -12px;
  padding: 10px 12px;
  border-radius: 8px;
}

.rule-status.clickable:hover {
  background: rgba(16, 185, 129, 0.1);
}

.status-divider {
  margin: 0 10px;
  color: rgba(255, 255, 255, 0.2);
}

.status-stats {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.6);
  margin-right: 12px;
}

.status-stats .el-icon {
  font-size: 14px;
}

.status-stats.speed-in {
  color: #10b981;
}

.status-stats.speed-out {
  color: #3b82f6;
}

.monitor-link {
  margin-left: auto;
  font-size: 16px;
  color: rgba(255, 255, 255, 0.4);
  transition: color 0.2s, transform 0.2s;
}

.rule-status.clickable:hover .monitor-link {
  color: #10b981;
  transform: translateX(2px);
}

/* 卡片操作栏 */
.card-footer {
  display: flex;
  gap: 10px;
  padding: 16px 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(0, 0, 0, 0.1);
}

.card-footer .el-button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.8);
}

.card-footer .el-button:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.2);
  color: #fff;
}

.card-footer .el-button.el-button--success {
  background: linear-gradient(135deg, #10b981, #059669);
  border: none;
  color: #fff;
}

.card-footer .el-button.el-button--success:hover {
  box-shadow: 0 4px 12px rgba(16, 185, 129, 0.4);
}

.card-footer .el-button.el-button--warning {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  border: none;
  color: #fff;
}

.card-footer .el-button.el-button--warning:hover {
  box-shadow: 0 4px 12px rgba(245, 158, 11, 0.4);
}

.card-footer .el-button.el-button--danger {
  background: linear-gradient(135deg, #ef4444, #dc2626);
  border: none;
  color: #fff;
}

.card-footer .el-button.el-button--danger:hover {
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
}

.card-footer .el-button.el-button--primary {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  border: none;
  color: #fff;
}

.card-footer .el-button.el-button--primary:hover {
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
}

.card-footer .el-button:disabled {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.2);
}

/* 响应式断点 */
@media (min-width: 1920px) {
  .cards-grid {
    grid-template-columns: repeat(4, 1fr);
  }
}

@media (min-width: 1200px) and (max-width: 1919px) {
  .cards-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (min-width: 768px) and (max-width: 1199px) {
  .cards-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 767px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .page-header h3 {
    font-size: 20px;
  }

  .header-actions {
    display: flex;
    gap: 8px;
  }

  .header-actions .el-button {
    flex: 1;
  }

  .cards-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .card-footer {
    flex-wrap: wrap;
  }

  .card-footer .el-button {
    flex: 1 1 calc(50% - 5px);
    min-width: 0;
  }
}
</style>
