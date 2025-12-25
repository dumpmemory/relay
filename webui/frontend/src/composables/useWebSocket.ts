import { ref } from 'vue'
import { getToken } from '../api'

export interface WSMessage {
  type: string
  data: unknown
}

export interface Connection {
  id: string
  client_ip: string
  client_location?: string
  target: string
  protocol: string
  bytes_in: number
  bytes_out: number
  started_at: string
  ended_at?: string
  duration: number
  active: boolean
}

export interface TrafficData {
  relay_id: string
  bytes_in: number
  bytes_out: number
  connections: number
}

type MessageCallback = (msg: WSMessage) => void

// 全局单例
let ws: WebSocket | null = null
const connected = ref(false)
const connections = ref<Map<string, Connection[]>>(new Map())
const traffic = ref<Map<string, TrafficData>>(new Map())
const subscribedRelayIds = new Set<string>()
const messageCallbacks = new Set<MessageCallback>()

const connect = () => {
  if (ws && ws.readyState === WebSocket.OPEN) return

  // 获取 token，未登录时不连接
  const token = getToken()
  if (!token) {
    // 延迟重试，等待登录
    setTimeout(connect, 3000)
    return
  }

  const baseUrl = import.meta.env.VITE_API_BASE_URL || ''
  let wsUrl: string

  if (baseUrl) {
    wsUrl = baseUrl.replace(/^http/, 'ws') + '/ws'
  } else {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    wsUrl = `${protocol}//${window.location.host}/ws`
  }

  // 添加 token 到查询参数
  wsUrl += `?token=${encodeURIComponent(token)}`

  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    connected.value = true
    // 订阅所有消息类型
    ws?.send(JSON.stringify({
      action: 'subscribe',
      topics: ['relay.connections', 'relay.traffic', 'relay.status']
    }))
    // 恢复之前的 relay 订阅
    subscribedRelayIds.forEach(relayId => {
      ws?.send(JSON.stringify({
        action: 'subscribe',
        topics: ['relay.connections', 'relay.traffic'],
        relay_id: relayId
      }))
    })
  }

  ws.onclose = () => {
    connected.value = false
    ws = null
    // 3秒后重连
    setTimeout(connect, 3000)
  }

  ws.onerror = () => {
    connected.value = false
  }

  ws.onmessage = (event) => {
    try {
      const msg: WSMessage = JSON.parse(event.data)
      handleMessage(msg)
      // 通知所有回调
      messageCallbacks.forEach(cb => cb(msg))
    } catch (e) {
      console.error('WebSocket 消息解析失败', e)
    }
  }
}

const handleMessage = (msg: WSMessage) => {
  switch (msg.type) {
    case 'relay.connections': {
      const data = msg.data as { relay_id: string; connections: Connection[] }
      connections.value.set(data.relay_id, data.connections || [])
      break
    }
    case 'relay.traffic': {
      const data = msg.data as TrafficData
      traffic.value.set(data.relay_id, data)
      break
    }
  }
}

const subscribeRelay = (relayId: string) => {
  subscribedRelayIds.add(relayId)
  if (ws && connected.value) {
    ws.send(JSON.stringify({
      action: 'subscribe',
      topics: ['relay.connections', 'relay.traffic'],
      relay_id: relayId
    }))
  }
}

const unsubscribeRelay = (relayId: string) => {
  subscribedRelayIds.delete(relayId)
  if (ws && connected.value) {
    ws.send(JSON.stringify({
      action: 'unsubscribe',
      relay_id: relayId
    }))
  }
  connections.value.delete(relayId)
  traffic.value.delete(relayId)
}

// 注册消息回调
const onMessage = (callback: MessageCallback) => {
  messageCallbacks.add(callback)
}

// 取消消息回调
const offMessage = (callback: MessageCallback) => {
  messageCallbacks.delete(callback)
}

// 自动连接
connect()

export function useWebSocket() {
  return {
    connected,
    isConnected: connected,
    connections,
    traffic,
    subscribe: subscribeRelay,
    unsubscribe: unsubscribeRelay,
    subscribeRelay,
    onMessage,
    offMessage
  }
}
