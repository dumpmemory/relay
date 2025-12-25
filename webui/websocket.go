package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage WebSocket 消息
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// WSHub WebSocket 连接管理
type WSHub struct {
	clients    map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
}

// WSClient WebSocket 客户端
type WSClient struct {
	hub     *WSHub
	conn    *websocket.Conn
	send    chan []byte
	topics  map[string]bool
	relayID string // 订阅特定 relay
	mu      sync.RWMutex
}

// NewWSHub 创建 Hub
func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
	}
}

// Run 运行 Hub
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// 收集需要移除的客户端
			var toRemove []*WSClient
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					toRemove = append(toRemove, client)
				}
			}
			h.mu.RUnlock()

			// 使用写锁移除失效客户端
			if len(toRemove) > 0 {
				h.mu.Lock()
				for _, client := range toRemove {
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						close(client.send)
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

// Broadcast 广播消息
func (h *WSHub) Broadcast(msgType string, data interface{}) {
	msg := WSMessage{Type: msgType, Data: data}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.mu.RLock()
		subscribed := client.topics[msgType]
		client.mu.RUnlock()

		if subscribed {
			select {
			case client.send <- jsonData:
			default:
			}
		}
	}
}

// BroadcastToRelay 广播到订阅特定 relay 的客户端
func (h *WSHub) BroadcastToRelay(relayID, msgType string, data interface{}) {
	msg := WSMessage{Type: msgType, Data: data}
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.mu.RLock()
		subscribed := client.topics[msgType]
		matchRelay := client.relayID == "" || client.relayID == relayID
		client.mu.RUnlock()

		if subscribed && matchRelay {
			select {
			case client.send <- jsonData:
			default:
			}
		}
	}
}

// readPump 读取消息
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			// 过滤掉正常的断开连接错误：1000(正常关闭)、1001(离开)、1005(无状态码)
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived,
				websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket 错误: %v", err)
			}
			break
		}

		// 处理订阅消息
		var req struct {
			Action  string   `json:"action"`
			Topics  []string `json:"topics"`
			RelayID string   `json:"relay_id"`
		}
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		if req.Action == "subscribe" {
			c.mu.Lock()
			for _, topic := range req.Topics {
				c.topics[topic] = true
			}
			c.relayID = req.RelayID
			c.mu.Unlock()
		} else if req.Action == "unsubscribe" {
			c.mu.Lock()
			for _, topic := range req.Topics {
				delete(c.topics, topic)
			}
			c.mu.Unlock()
		}
	}
}

// writePump 发送消息
func (c *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
