<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { relayApi, type RelayRule } from '../api'
import { useWebSocket, type TrafficData } from '../composables/useWebSocket'

const route = useRoute()
const rules = ref<RelayRule[]>([])
const selectedRelay = ref<string>('')
const loading = ref(false)

const { connections, traffic, subscribe, unsubscribe, isConnected } = useWebSocket()

const currentConnections = computed(() => {
  if (!selectedRelay.value) return []
  const conns = connections.value.get(selectedRelay.value) || []
  // 排序：活跃连接优先，然后按开始时间倒序
  return [...conns].sort((a, b) => {
    if (a.active !== b.active) return a.active ? -1 : 1
    return new Date(b.started_at).getTime() - new Date(a.started_at).getTime()
  })
})

const defaultTraffic: TrafficData = { relay_id: '', bytes_in: 0, bytes_out: 0, connections: 0 }

const currentTraffic = computed(() => {
  if (!selectedRelay.value) return defaultTraffic
  return traffic.value.get(selectedRelay.value) || defaultTraffic
})

const selectedRule = computed(() => {
  return rules.value.find(r => r.id === selectedRelay.value)
})

const formatBytes = (bytes: number | undefined): string => {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const formatTime = (timestamp: string): string => {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleTimeString()
}

const formatDuration = (seconds: number | undefined): string => {
  if (!seconds || seconds === 0) return '0s'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${mins}m`
}

const tableRowClassName = ({ row }: { row: { active: boolean } }) => {
  return row.active ? 'row-active' : 'row-inactive'
}

const fetchRules = async () => {
  loading.value = true
  try {
    const res = await relayApi.list()
    if (res.code === 0) {
      rules.value = (res.data || []).filter((r) => r.running)

      // 优先从 URL 参数读取要选中的 relay
      const queryRelayId = route.query.relay as string
      if (queryRelayId && rules.value.find(r => r.id === queryRelayId)) {
        selectRelay(queryRelayId)
      } else if (rules.value[0] && !selectedRelay.value) {
        selectRelay(rules.value[0].id)
      }
    } else {
      ElMessage.error(res.msg)
    }
  } finally {
    loading.value = false
  }
}

const selectRelay = (id: string) => {
  if (selectedRelay.value) {
    unsubscribe(selectedRelay.value)
  }
  selectedRelay.value = id
  if (id) {
    subscribe(id)
  }
}

onMounted(() => {
  fetchRules()
})

onUnmounted(() => {
  if (selectedRelay.value) {
    unsubscribe(selectedRelay.value)
  }
})
</script>

<template>
  <div class="monitor-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h3>实时监控</h3>
        <span class="header-subtitle">WebSocket 实时数据流</span>
      </div>
      <div class="header-right">
        <div :class="['ws-status', { connected: isConnected }]">
          <span class="status-dot"></span>
          <span class="status-text">{{ isConnected ? '已连接' : '未连接' }}</span>
        </div>
      </div>
    </div>

    <!-- 主布局 -->
    <div class="monitor-layout" v-loading="loading">
      <!-- 左侧规则列表 -->
      <div class="relay-sidebar">
        <div class="sidebar-header">
          <el-icon><Connection /></el-icon>
          <span>运行中的规则</span>
        </div>
        <div class="relay-list">
          <div v-if="rules.length === 0" class="empty-list">
            <el-icon><VideoPause /></el-icon>
            <p>暂无运行中的规则</p>
          </div>
          <div
            v-for="rule in rules"
            :key="rule.id"
            :class="['relay-item', { active: selectedRelay === rule.id }]"
            @click="selectRelay(rule.id)"
          >
            <div class="relay-icon">
              <el-icon><Connection /></el-icon>
            </div>
            <div class="relay-info">
              <div class="relay-name">{{ rule.name }}</div>
              <div class="relay-addr">{{ rule.src }}</div>
            </div>
            <div v-if="selectedRelay === rule.id" class="relay-active">
              <el-icon><Check /></el-icon>
            </div>
          </div>
        </div>
      </div>

      <!-- 右侧监控内容 -->
      <div class="monitor-content">
        <template v-if="selectedRule">
          <!-- 统计卡片 -->
          <div class="stats-grid">
            <div class="stat-card stat-connections">
              <div class="stat-icon">
                <el-icon><Link /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">当前连接</div>
                <div class="stat-value">{{ currentTraffic.connections }}</div>
              </div>
            </div>
            <div class="stat-card stat-in">
              <div class="stat-icon">
                <el-icon><Download /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">入站流量</div>
                <div class="stat-value">{{ formatBytes(currentTraffic.bytes_in) }}</div>
              </div>
            </div>
            <div class="stat-card stat-out">
              <div class="stat-icon">
                <el-icon><Upload /></el-icon>
              </div>
              <div class="stat-info">
                <div class="stat-label">出站流量</div>
                <div class="stat-value">{{ formatBytes(currentTraffic.bytes_out) }}</div>
              </div>
            </div>
          </div>

          <!-- 连接表格 -->
          <div class="connections-panel">
            <div class="panel-header">
              <div class="panel-title">
                <el-icon><List /></el-icon>
                <span>连接记录</span>
              </div>
              <div class="panel-count">{{ currentConnections.length }} 条</div>
            </div>
            <div class="table-container">
              <el-table
                :data="currentConnections"
                :row-class-name="tableRowClassName"
                :header-cell-class-name="'table-header-cell'"
                class="monitor-table"
                max-height="400"
              >
                <el-table-column prop="active" label="状态" width="70">
                  <template #default="{ row }">
                    <span :class="['status-tag', row.active ? 'active' : 'inactive']">
                      {{ row.active ? '活跃' : '断开' }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column prop="protocol" label="协议" width="60">
                  <template #default="{ row }">
                    <span :class="['protocol-tag', row.protocol || 'tcp']">
                      {{ (row.protocol || 'tcp').toUpperCase() }}
                    </span>
                  </template>
                </el-table-column>
                <el-table-column prop="client_ip" label="客户端 IP" min-width="140" show-overflow-tooltip />
                <el-table-column prop="client_location" label="位置" width="100" show-overflow-tooltip>
                  <template #default="{ row }">
                    <span class="location-text">{{ row.client_location || '-' }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="started_at" label="开始时间" width="90">
                  <template #default="{ row }">
                    {{ formatTime(row.started_at) }}
                  </template>
                </el-table-column>
                <el-table-column prop="duration" label="时长" width="80" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ formatDuration(row.duration) }}
                  </template>
                </el-table-column>
                <el-table-column prop="bytes_in" label="入站" width="100" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ formatBytes(row.bytes_in) }}
                  </template>
                </el-table-column>
                <el-table-column prop="bytes_out" label="出站" width="100" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ formatBytes(row.bytes_out) }}
                  </template>
                </el-table-column>
              </el-table>
              <div v-if="currentConnections.length === 0" class="empty-table">
                <el-icon><DocumentDelete /></el-icon>
                <p>暂无连接记录</p>
              </div>
            </div>
          </div>
        </template>

        <!-- 空状态 -->
        <div v-else class="empty-state">
          <div class="empty-icon">
            <el-icon><Monitor /></el-icon>
          </div>
          <p>请从左侧选择一个运行中的规则</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 页面容器 */
.monitor-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
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

.ws-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.ws-status .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.3);
}

.ws-status .status-text {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
}

.ws-status.connected {
  border-color: rgba(16, 185, 129, 0.3);
  background: rgba(16, 185, 129, 0.1);
}

.ws-status.connected .status-dot {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.6);
}

.ws-status.connected .status-text {
  color: #10b981;
  font-weight: 500;
}

/* 主布局 */
.monitor-layout {
  display: flex;
  gap: 20px;
  flex: 1;
  min-height: 0;
}

/* 左侧边栏 */
.relay-sidebar {
  width: 280px;
  flex-shrink: 0;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.8);
  font-weight: 600;
  font-size: 14px;
}

.sidebar-header .el-icon {
  color: #10b981;
}

.relay-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

/* 滚动条样式 */
.relay-list::-webkit-scrollbar {
  width: 4px;
}

.relay-list::-webkit-scrollbar-track {
  background: transparent;
}

.relay-list::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.2);
  border-radius: 2px;
}

.empty-list {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: rgba(255, 255, 255, 0.3);
}

.empty-list .el-icon {
  font-size: 40px;
  margin-bottom: 12px;
}

.empty-list p {
  font-size: 13px;
  margin: 0;
}

/* 规则项 */
.relay-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.3s ease;
  margin-bottom: 4px;
}

.relay-item:hover {
  background: rgba(255, 255, 255, 0.08);
}

.relay-item.active {
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.relay-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(16, 185, 129, 0.1);
  border-radius: 8px;
  color: #10b981;
  flex-shrink: 0;
}

.relay-info {
  flex: 1;
  min-width: 0;
}

.relay-name {
  font-size: 14px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.9);
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.relay-addr {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  font-family: 'Consolas', 'Monaco', monospace;
}

.relay-active {
  color: #10b981;
}

/* 右侧内容区 */
.monitor-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

/* 统计卡片网格 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: all 0.3s ease;
}

.stat-card:hover {
  background: rgba(255, 255, 255, 0.08);
  transform: translateY(-2px);
}

.stat-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  font-size: 20px;
}

.stat-connections .stat-icon {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: #fff;
}

.stat-in .stat-icon {
  background: linear-gradient(135deg, #10b981, #059669);
  color: #fff;
}

.stat-out .stat-icon {
  background: linear-gradient(135deg, #f59e0b, #d97706);
  color: #fff;
}

.stat-info {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.5);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 22px;
  font-weight: 700;
  color: #fff;
}

/* 连接面板 */
.connections-panel {
  flex: 1;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 300px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.8);
}

.panel-title .el-icon {
  color: #10b981;
}

.panel-count {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.4);
  background: rgba(255, 255, 255, 0.05);
  padding: 4px 10px;
  border-radius: 12px;
}

.table-container {
  flex: 1;
  overflow: hidden;
}

/* 表格样式 */
.monitor-table {
  background: transparent !important;
  font-size: 13px;
}

/* 表头样式 - 多层覆盖 */
.monitor-table :deep(.el-table__header-wrapper),
.monitor-table :deep(.el-table__header),
.monitor-table :deep(.el-table__header thead),
.monitor-table :deep(.el-table__header tr),
.monitor-table :deep(.el-table__header th) {
  background: transparent !important;
}

.monitor-table :deep(.table-header-cell) {
  background: rgba(255, 255, 255, 0.05) !important;
  border-color: rgba(255, 255, 255, 0.1) !important;
  color: rgba(255, 255, 255, 0.7) !important;
  font-weight: 600;
  padding: 14px 0 !important;
}

.monitor-table :deep(.table-header-cell .cell) {
  padding: 0 12px;
}

.monitor-table :deep(.el-table__header th) {
  background: rgba(255, 255, 255, 0.05) !important;
  border-color: rgba(255, 255, 255, 0.1) !important;
  color: rgba(255, 255, 255, 0.7) !important;
  font-weight: 600;
  padding: 14px 0 !important;
}

.monitor-table :deep(.el-table__body tr) {
  background: transparent !important;
  transition: all 0.2s ease;
}

.monitor-table :deep(.el-table__body tr:hover > td) {
  background: rgba(255, 255, 255, 0.03) !important;
}

.monitor-table :deep(.el-table__body td) {
  border-color: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.8);
  padding: 10px 0;
}

.monitor-table :deep(.el-table__empty-block) {
  background: transparent;
}

.monitor-table :deep(.el-table__empty-text) {
  color: rgba(255, 255, 255, 0.3);
}

/* 活跃行 */
:deep(.row-active) {
  background: rgba(16, 185, 129, 0.03) !important;
}

:deep(.row-active > td) {
  color: rgba(255, 255, 255, 0.95);
}

/* 非活跃行 */
:deep(.row-inactive) {
  opacity: 0.5;
}

:deep(.row-inactive > td) {
  color: rgba(255, 255, 255, 0.4);
}

/* 状态标签 */
.status-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}

.status-tag.active {
  background: rgba(16, 185, 129, 0.2);
  color: #10b981;
}

.status-tag.inactive {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.4);
}

/* 协议标签 */
.protocol-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 600;
}

.protocol-tag.tcp {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.protocol-tag.udp {
  background: rgba(245, 158, 11, 0.2);
  color: #fbbf24;
}

.location-text {
  color: rgba(255, 255, 255, 0.5);
}

/* 空表格状态 */
.empty-table {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: rgba(255, 255, 255, 0.3);
}

.empty-table .el-icon {
  font-size: 48px;
  margin-bottom: 12px;
}

.empty-table p {
  font-size: 14px;
  margin: 0;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  background: rgba(255, 255, 255, 0.05);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.empty-icon {
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 20px;
  color: rgba(255, 255, 255, 0.3);
  margin-bottom: 20px;
}

.empty-icon .el-icon {
  font-size: 40px;
}

.empty-state p {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.4);
  margin: 0;
}

/* 响应式 */
@media (max-width: 1024px) {
  .monitor-layout {
    flex-direction: column;
  }

  .relay-sidebar {
    width: 100%;
    max-height: 200px;
  }

  .stats-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 767px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .header-subtitle {
    margin-left: 0;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .stat-card {
    padding: 16px;
  }

  .stat-icon {
    width: 40px;
    height: 40px;
  }

  .stat-value {
    font-size: 18px;
  }
}
</style>
