package service

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DGHeroin/relay/webui/model"
	"github.com/google/uuid"
)

// RelayStatus 转发状态
type RelayStatus struct {
	Running     bool  `json:"running"`
	Connections int64 `json:"connections"`
	BytesIn     int64 `json:"bytes_in"`
	BytesOut    int64 `json:"bytes_out"`
}

// Connection 连接信息
type Connection struct {
	ID        string     `json:"id"`
	ClientIP  string     `json:"client_ip"`
	Location  string     `json:"client_location,omitempty"`
	Target    string     `json:"target"`
	Protocol  string     `json:"protocol"`
	BytesIn   int64      `json:"bytes_in"`
	BytesOut  int64      `json:"bytes_out"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Duration  int64      `json:"duration"`
	Active    bool       `json:"active"`
}

// Broadcaster 广播接口
type Broadcaster interface {
	BroadcastToRelay(relayID, msgType string, data interface{})
}

// RelayInstance 单个转发实例
type RelayInstance struct {
	rule        *model.RelayRule
	stopCh      chan struct{}
	tcpListener net.Listener
	udpConn     net.PacketConn

	connections sync.Map // id -> *Connection (活跃连接)
	connCount   int64
	bytesIn     int64
	bytesOut    int64

	// 速度计算（EMA 平滑）
	lastBytesIn    int64
	lastBytesOut   int64
	smoothSpeedIn  float64 // EMA 平滑后的入站速度
	smoothSpeedOut float64 // EMA 平滑后的出站速度

	// 连接历史记录
	historyMu sync.Mutex
	history   []*Connection // 已断开的连接历史

	broadcaster Broadcaster
	geoIP       *GeoIPService
}

const maxHistorySize = 100 // 保留最多100条历史记录

// RelayManager 转发管理器
type RelayManager struct {
	instances sync.Map // id -> *RelayInstance
}

// NewRelayManager 创建管理器
func NewRelayManager() *RelayManager {
	return &RelayManager{}
}

// Start 启动转发
func (m *RelayManager) Start(rule *model.RelayRule, broadcaster Broadcaster, geoIP *GeoIPService) error {
	log.Printf("[RelayMgr] Start 调用: id=%s, name=%s, src=%s, dst=%s, protocol=%s",
		rule.ID, rule.Name, rule.Src, rule.Dst, rule.Protocol)

	if _, exists := m.instances.Load(rule.ID); exists {
		log.Printf("[RelayMgr] 规则已在运行: %s", rule.ID)
		return fmt.Errorf("规则已在运行")
	}

	instance := &RelayInstance{
		rule:        rule,
		stopCh:      make(chan struct{}),
		broadcaster: broadcaster,
		geoIP:       geoIP,
	}

	// 启动 TCP
	if rule.Protocol == "tcp" || rule.Protocol == "both" {
		log.Printf("[RelayMgr] 启动 TCP 监听: %s", rule.Src)
		if err := instance.startTCP(); err != nil {
			log.Printf("[RelayMgr] TCP 启动失败: %v", err)
			return fmt.Errorf("TCP 启动失败: %v", err)
		}
		log.Printf("[RelayMgr] TCP 监听成功: %s", rule.Src)
	}

	// 启动 UDP
	if rule.Protocol == "udp" || rule.Protocol == "both" {
		log.Printf("[RelayMgr] 启动 UDP 监听: %s", rule.Src)
		if err := instance.startUDP(); err != nil {
			if instance.tcpListener != nil {
				instance.tcpListener.Close()
			}
			log.Printf("[RelayMgr] UDP 启动失败: %v", err)
			return fmt.Errorf("UDP 启动失败: %v", err)
		}
		log.Printf("[RelayMgr] UDP 监听成功: %s", rule.Src)
	}

	m.instances.Store(rule.ID, instance)

	// 启动状态推送
	go instance.pushStatus()

	log.Printf("[RelayMgr] 转发启动完成: %s (%s -> %s)", rule.Name, rule.Src, rule.Dst)
	return nil
}

// Stop 停止转发
func (m *RelayManager) Stop(id string) {
	if v, ok := m.instances.Load(id); ok {
		instance := v.(*RelayInstance)
		close(instance.stopCh)
		if instance.tcpListener != nil {
			instance.tcpListener.Close()
		}
		if instance.udpConn != nil {
			instance.udpConn.Close()
		}
		m.instances.Delete(id)
		log.Printf("转发停止: %s", id)
	}
}

// StopAll 停止所有
func (m *RelayManager) StopAll() {
	m.instances.Range(func(key, value interface{}) bool {
		m.Stop(key.(string))
		return true
	})
}

// IsRunning 检查是否运行
func (m *RelayManager) IsRunning(id string) bool {
	_, ok := m.instances.Load(id)
	return ok
}

// GetStatus 获取状态
func (m *RelayManager) GetStatus(id string) RelayStatus {
	if v, ok := m.instances.Load(id); ok {
		instance := v.(*RelayInstance)
		return RelayStatus{
			Running:     true,
			Connections: atomic.LoadInt64(&instance.connCount),
			BytesIn:     atomic.LoadInt64(&instance.bytesIn),
			BytesOut:    atomic.LoadInt64(&instance.bytesOut),
		}
	}
	return RelayStatus{Running: false}
}

// GetAllStatus 获取所有状态
func (m *RelayManager) GetAllStatus() map[string]RelayStatus {
	result := make(map[string]RelayStatus)
	m.instances.Range(func(key, value interface{}) bool {
		id := key.(string)
		result[id] = m.GetStatus(id)
		return true
	})
	return result
}

// ActiveCount 活跃数量
func (m *RelayManager) ActiveCount() int {
	count := 0
	m.instances.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetConnections 获取连接列表
func (m *RelayManager) GetConnections(id string) []Connection {
	if v, ok := m.instances.Load(id); ok {
		instance := v.(*RelayInstance)
		var conns []Connection
		instance.connections.Range(func(key, value interface{}) bool {
			conn := value.(*Connection)
			conn.Duration = int64(time.Since(conn.StartedAt).Seconds())
			conns = append(conns, *conn)
			return true
		})
		return conns
	}
	return nil
}

// ==================== RelayInstance ====================

// countingWriter 包装 io.Writer，实时统计写入字节数
type countingWriter struct {
	w       io.Writer
	counter *int64      // 连接级别计数器
	total   *int64      // 全局计数器
	connRef *Connection // 连接引用，用于实时更新
	isIn    bool        // true=入站, false=出站
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	if n > 0 {
		atomic.AddInt64(cw.counter, int64(n))
		atomic.AddInt64(cw.total, int64(n))
		// 实时更新连接的字节数
		if cw.isIn {
			atomic.StoreInt64(&cw.connRef.BytesIn, atomic.LoadInt64(cw.counter))
		} else {
			atomic.StoreInt64(&cw.connRef.BytesOut, atomic.LoadInt64(cw.counter))
		}
	}
	return n, err
}

func (r *RelayInstance) startTCP() error {
	ln, err := net.Listen("tcp", r.rule.Src)
	if err != nil {
		return err
	}
	r.tcpListener = ln

	go func() {
		for {
			select {
			case <-r.stopCh:
				return
			default:
				conn, err := ln.Accept()
				if err != nil {
					select {
					case <-r.stopCh:
						return
					default:
						continue
					}
				}
				go r.handleTCP(conn)
			}
		}
	}()

	return nil
}

func (r *RelayInstance) handleTCP(client net.Conn) {
	defer client.Close()

	// 连接到目标
	remote, err := net.DialTimeout("tcp", r.rule.Dst, 5*time.Second)
	if err != nil {
		log.Printf("连接目标失败: %v", err)
		return
	}
	defer remote.Close()

	// 记录连接
	connID := uuid.New().String()
	clientAddr := client.RemoteAddr().String()
	clientIP, _, _ := net.SplitHostPort(clientAddr)

	location := ""
	if r.geoIP != nil {
		location = r.geoIP.Lookup(clientIP)
	}

	connInfo := &Connection{
		ID:        connID,
		ClientIP:  clientIP,
		Location:  location,
		Target:    r.rule.Dst,
		Protocol:  "tcp",
		StartedAt: time.Now(),
		Active:    true,
	}
	r.connections.Store(connID, connInfo)
	atomic.AddInt64(&r.connCount, 1)

	// 记录日志
	model.SaveAccessLog(r.rule.ID, clientIP, "connect", 0, 0, 0)

	// 双向复制（使用 countingWriter 实时统计）
	var bytesIn, bytesOut int64
	done := make(chan struct{}, 2)

	// 入站：client -> remote
	go func() {
		cw := &countingWriter{
			w:       remote,
			counter: &bytesIn,
			total:   &r.bytesIn,
			connRef: connInfo,
			isIn:    true,
		}
		io.Copy(cw, client)
		// 关闭写入方向，通知对方结束
		if tc, ok := remote.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	// 出站：remote -> client
	go func() {
		cw := &countingWriter{
			w:       client,
			counter: &bytesOut,
			total:   &r.bytesOut,
			connRef: connInfo,
			isIn:    false,
		}
		io.Copy(cw, remote)
		// 关闭写入方向，通知对方结束
		if tc, ok := client.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}()

	// 等待两个方向都完成
	<-done
	<-done

	// 更新连接信息并移入历史
	now := time.Now()
	connInfo.EndedAt = &now
	connInfo.Duration = int64(now.Sub(connInfo.StartedAt).Seconds())
	connInfo.Active = false
	// 最终字节数已通过 countingWriter 实时更新，这里确保最终值正确
	connInfo.BytesIn = atomic.LoadInt64(&bytesIn)
	connInfo.BytesOut = atomic.LoadInt64(&bytesOut)

	r.connections.Delete(connID)
	atomic.AddInt64(&r.connCount, -1)
	r.addToHistory(connInfo)

	// 保存统计
	model.SaveRelayStat(r.rule.ID, bytesIn, bytesOut, 1)
	model.SaveAccessLog(r.rule.ID, clientIP, "disconnect", bytesIn, bytesOut, connInfo.Duration)
}

// addToHistory 添加到历史记录
func (r *RelayInstance) addToHistory(conn *Connection) {
	r.historyMu.Lock()
	defer r.historyMu.Unlock()

	// 添加到开头（最新的在前）
	r.history = append([]*Connection{conn}, r.history...)

	// 保持最多 maxHistorySize 条记录
	if len(r.history) > maxHistorySize {
		r.history = r.history[:maxHistorySize]
	}
}

func (r *RelayInstance) startUDP() error {
	pc, err := net.ListenPacket("udp", r.rule.Src)
	if err != nil {
		return err
	}
	r.udpConn = pc

	go func() {
		buf := make([]byte, 65535)
		clients := make(map[string]*udpClient)
		var mu sync.Mutex

		for {
			select {
			case <-r.stopCh:
				return
			default:
				pc.SetReadDeadline(time.Now().Add(time.Second))
				n, addr, err := pc.ReadFrom(buf)
				if err != nil {
					continue
				}

				key := addr.String()
				mu.Lock()
				client, exists := clients[key]
				if !exists {
					// 新客户端
					remote, err := net.Dial("udp", r.rule.Dst)
					if err != nil {
						mu.Unlock()
						continue
					}

					clientIP, _, _ := net.SplitHostPort(key)
					location := ""
					if r.geoIP != nil {
						location = r.geoIP.Lookup(clientIP)
					}

					client = &udpClient{
						addr:      addr,
						remote:    remote,
						lastSeen:  time.Now(),
						startedAt: time.Now(),
						clientIP:  clientIP,
						location:  location,
					}
					clients[key] = client

					connID := uuid.New().String()
					connInfo := &Connection{
						ID:        connID,
						ClientIP:  clientIP,
						Location:  location,
						Target:    r.rule.Dst,
						Protocol:  "udp",
						StartedAt: time.Now(),
						Active:    true,
					}
					r.connections.Store(connID, connInfo)
					client.connID = connID
					client.connInfo = connInfo
					atomic.AddInt64(&r.connCount, 1)

					model.SaveAccessLog(r.rule.ID, clientIP, "connect", 0, 0, 0)

					// 接收远程响应
					go func(c *udpClient) {
						buf := make([]byte, 65535)
						for {
							c.remote.SetReadDeadline(time.Now().Add(30 * time.Second))
							n, err := c.remote.Read(buf)
							if err != nil {
								break
							}
							pc.WriteTo(buf[:n], c.addr)
							atomic.AddInt64(&c.bytesOut, int64(n))
							atomic.AddInt64(&r.bytesOut, int64(n))
						}

						// 清理并移入历史
						mu.Lock()
						delete(clients, c.addr.String())
						mu.Unlock()

						now := time.Now()
						c.connInfo.EndedAt = &now
						c.connInfo.Duration = int64(now.Sub(c.startedAt).Seconds())
						c.connInfo.Active = false
						c.connInfo.BytesIn = c.bytesIn
						c.connInfo.BytesOut = c.bytesOut

						r.connections.Delete(c.connID)
						atomic.AddInt64(&r.connCount, -1)
						r.addToHistory(c.connInfo)

						model.SaveRelayStat(r.rule.ID, c.bytesIn, c.bytesOut, 1)
						model.SaveAccessLog(r.rule.ID, c.clientIP, "disconnect", c.bytesIn, c.bytesOut, c.connInfo.Duration)
					}(client)
				}
				mu.Unlock()

				client.lastSeen = time.Now()
				client.remote.Write(buf[:n])
				atomic.AddInt64(&client.bytesIn, int64(n))
				atomic.AddInt64(&r.bytesIn, int64(n))
			}
		}
	}()

	return nil
}

type udpClient struct {
	addr      net.Addr
	remote    net.Conn
	lastSeen  time.Time
	startedAt time.Time
	clientIP  string
	location  string
	connID    string
	connInfo  *Connection
	bytesIn   int64
	bytesOut  int64
}

// pushStatus 定期推送状态
func (r *RelayInstance) pushStatus() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			if r.broadcaster == nil {
				continue
			}

			// 推送连接列表（活跃 + 历史）
			var conns []Connection

			// 先添加活跃连接
			r.connections.Range(func(key, value interface{}) bool {
				conn := value.(*Connection)
				c := *conn
				c.Duration = int64(time.Since(conn.StartedAt).Seconds())
				c.BytesIn = atomic.LoadInt64(&conn.BytesIn)
				c.BytesOut = atomic.LoadInt64(&conn.BytesOut)
				conns = append(conns, c)
				return true
			})

			// 再添加历史记录
			r.historyMu.Lock()
			for _, h := range r.history {
				conns = append(conns, *h)
			}
			r.historyMu.Unlock()

			r.broadcaster.BroadcastToRelay(r.rule.ID, "relay.connections", map[string]interface{}{
				"relay_id":    r.rule.ID,
				"connections": conns,
			})

			// 计算速度（使用 EMA 指数移动平均平滑）
			// EMA 公式: smoothed = alpha * current + (1 - alpha) * previous
			// alpha = 0.3 提供较好的平滑效果，同时保持响应速度
			const alpha = 0.3

			currentBytesIn := atomic.LoadInt64(&r.bytesIn)
			currentBytesOut := atomic.LoadInt64(&r.bytesOut)
			lastIn := atomic.LoadInt64(&r.lastBytesIn)
			lastOut := atomic.LoadInt64(&r.lastBytesOut)

			// 计算瞬时速度
			instantSpeedIn := float64(currentBytesIn - lastIn)
			instantSpeedOut := float64(currentBytesOut - lastOut)

			// 应用 EMA 平滑
			if r.smoothSpeedIn == 0 && instantSpeedIn > 0 {
				// 首次有数据时直接使用瞬时值
				r.smoothSpeedIn = instantSpeedIn
			} else {
				r.smoothSpeedIn = alpha*instantSpeedIn + (1-alpha)*r.smoothSpeedIn
			}

			if r.smoothSpeedOut == 0 && instantSpeedOut > 0 {
				r.smoothSpeedOut = instantSpeedOut
			} else {
				r.smoothSpeedOut = alpha*instantSpeedOut + (1-alpha)*r.smoothSpeedOut
			}

			// 速度过小时归零（避免显示 0.1 B/s 这样的值）
			if r.smoothSpeedIn < 1 {
				r.smoothSpeedIn = 0
			}
			if r.smoothSpeedOut < 1 {
				r.smoothSpeedOut = 0
			}

			// 更新上一秒的值
			atomic.StoreInt64(&r.lastBytesIn, currentBytesIn)
			atomic.StoreInt64(&r.lastBytesOut, currentBytesOut)

			// 推送流量统计（包含平滑后的速度）
			r.broadcaster.BroadcastToRelay(r.rule.ID, "relay.traffic", map[string]interface{}{
				"relay_id":        r.rule.ID,
				"bytes_in":        currentBytesIn,
				"bytes_out":       currentBytesOut,
				"bytes_in_speed":  int64(r.smoothSpeedIn),
				"bytes_out_speed": int64(r.smoothSpeedOut),
				"connections":     atomic.LoadInt64(&r.connCount),
			})
		}
	}
}
