// API 响应类型
interface ApiResponse<T = unknown> {
  code: number
  msg: string
  data: T
}

// 获取 token
export function getToken(): string {
  return localStorage.getItem('token') || ''
}

// 设置 token
export function setToken(token: string) {
  localStorage.setItem('token', token)
}

// 清除 token
export function removeToken() {
  localStorage.removeItem('token')
}

// API 基础 URL
const BASE_URL = import.meta.env.VITE_API_BASE_URL || ''

// 统一 API 调用
export async function api<T = unknown>(action: string, data: Record<string, unknown> = {}): Promise<ApiResponse<T>> {
  try {
    const response = await fetch(`${BASE_URL}/api`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': getToken()
      },
      body: JSON.stringify({ action, data })
    })

    // 检查 HTTP 状态码
    if (!response.ok) {
      return {
        code: response.status,
        msg: `HTTP ${response.status}: ${response.statusText}`,
        data: null as T
      }
    }

    return response.json()
  } catch (error) {
    // 网络错误或其他异常
    const message = error instanceof Error ? error.message : '网络连接失败'
    return {
      code: -1,
      msg: message,
      data: null as T
    }
  }
}

// Setup API
export const setupApi = {
  status: () => api<{ need_setup: boolean }>('setup.status'),
  init: (password: string) => api('setup.init', { password })
}

// 版本信息类型
export interface VersionInfo {
  version: string
  build_time: string
  git_commit: string
}

// System API
export const systemApi = {
  login: (password: string) => api<{ token: string }>('system.login', { password }),
  logout: () => api('system.logout'),
  getSettings: () => api<Record<string, string>>('system.get_settings'),
  updateSettings: (key: string, value: string) => api('system.update_settings', { key, value }),
  changePassword: (oldPassword: string, newPassword: string) =>
    api('system.change_password', { old_password: oldPassword, new_password: newPassword }),
  geoipStatus: () => api<{ enabled: boolean; path: string }>('system.geoip_status'),
  deleteGeoip: () => api('system.delete_geoip'),
  version: () => api<VersionInfo>('system.version')
}

// Relay 类型
export interface RelayRule {
  id: string
  name: string
  src: string
  dst: string
  protocol: string
  enabled: boolean
  running: boolean
  connections: number
  bytes_in: number
  bytes_out: number
  created_at: string
}

// Relay API
export const relayApi = {
  list: () => api<RelayRule[]>('relay.list'),
  create: (name: string, src: string, dst: string, protocol: string) =>
    api<RelayRule>('relay.create', { name, src, dst, protocol }),
  update: (id: string, name: string, src: string, dst: string, protocol: string) =>
    api('relay.update', { id, name, src, dst, protocol }),
  delete: (id: string) => api('relay.delete', { id }),
  start: (id: string) => api('relay.start', { id }),
  stop: (id: string) => api('relay.stop', { id }),
  startAll: () => api('relay.start_all'),
  stopAll: () => api('relay.stop_all'),
  setEnabled: (id: string, enabled: boolean) => api('relay.set_enabled', { id, enabled }),
  status: (id?: string) => api('relay.status', id ? { id } : {}),
  exportRules: () => api<RelayRule[]>('relay.export'),
  importRules: (rules: unknown[]) => api('relay.import', { rules })
}

// Stats 类型
export interface RelayStat {
  id: number
  relay_id: string
  bytes_in: number
  bytes_out: number
  connections: number
  recorded_at: string
}

export interface AccessLog {
  id: number
  relay_id: string
  client_ip: string
  action: string
  bytes_in: number
  bytes_out: number
  duration: number
  created_at: string
}

// Stats API
export const statsApi = {
  overview: () => api<{
    total_bytes_in: number
    total_bytes_out: number
    total_connections: number
    active_relays: number
  }>('stats.overview'),
  relay: (id: string, range: string = '24h') => api<RelayStat[]>('stats.relay', { id, range }),
  logs: (relayId: string, page: number, size: number) =>
    api<{ list: AccessLog[]; total: number }>('stats.logs', { relay_id: relayId, page, size }),
  clear: (relayId?: string) => api('stats.clear', relayId ? { relay_id: relayId } : {})
}
