package model

import (
	"time"
)

// RelayStat 流量统计
type RelayStat struct {
	ID         int64     `json:"id"`
	RelayID    string    `json:"relay_id"`
	BytesIn    int64     `json:"bytes_in"`
	BytesOut   int64     `json:"bytes_out"`
	Connections int64    `json:"connections"`
	RecordedAt time.Time `json:"recorded_at"`
}

// AccessLog 访问日志
type AccessLog struct {
	ID        int64     `json:"id"`
	RelayID   string    `json:"relay_id"`
	ClientIP  string    `json:"client_ip"`
	Action    string    `json:"action"` // connect, disconnect
	BytesIn   int64     `json:"bytes_in"`
	BytesOut  int64     `json:"bytes_out"`
	Duration  int64     `json:"duration"` // 秒
	CreatedAt time.Time `json:"created_at"`
}

// SaveRelayStat 保存统计数据
func SaveRelayStat(relayID string, bytesIn, bytesOut, connections int64) error {
	// 按小时聚合
	now := time.Now().Truncate(time.Hour)

	// 使用 INSERT OR REPLACE 避免竞态条件
	// SQLite 支持 UPSERT 语法 (INSERT ... ON CONFLICT)
	_, err := DB.Exec(`
		INSERT INTO relay_stats (relay_id, bytes_in, bytes_out, connections, recorded_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(relay_id, recorded_at) DO UPDATE SET
			bytes_in = bytes_in + excluded.bytes_in,
			bytes_out = bytes_out + excluded.bytes_out,
			connections = connections + excluded.connections
	`, relayID, bytesIn, bytesOut, connections, now)
	return err
}

// GetRelayStats 获取统计数据
func GetRelayStats(relayID string, hours int) ([]*RelayStat, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	rows, err := DB.Query(`
		SELECT id, relay_id, bytes_in, bytes_out, connections, recorded_at
		FROM relay_stats WHERE relay_id = ? AND recorded_at >= ? ORDER BY recorded_at ASC
	`, relayID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*RelayStat
	for rows.Next() {
		s := &RelayStat{}
		if err := rows.Scan(&s.ID, &s.RelayID, &s.BytesIn, &s.BytesOut, &s.Connections, &s.RecordedAt); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// GetOverviewStats 获取总览统计
func GetOverviewStats() (totalBytesIn, totalBytesOut, totalConnections int64, err error) {
	err = DB.QueryRow(`
		SELECT COALESCE(SUM(bytes_in), 0), COALESCE(SUM(bytes_out), 0), COALESCE(SUM(connections), 0)
		FROM relay_stats
	`).Scan(&totalBytesIn, &totalBytesOut, &totalConnections)
	return
}

// SaveAccessLog 保存访问日志
func SaveAccessLog(relayID, clientIP, action string, bytesIn, bytesOut, duration int64) error {
	_, err := DB.Exec(`
		INSERT INTO access_logs (relay_id, client_ip, action, bytes_in, bytes_out, duration)
		VALUES (?, ?, ?, ?, ?, ?)
	`, relayID, clientIP, action, bytesIn, bytesOut, duration)
	return err
}

// GetAccessLogs 获取访问日志
func GetAccessLogs(relayID string, page, size int) ([]*AccessLog, int, error) {
	// 获取总数
	var total int
	query := "SELECT COUNT(*) FROM access_logs"
	args := []interface{}{}
	if relayID != "" {
		query += " WHERE relay_id = ?"
		args = append(args, relayID)
	}
	if err := DB.QueryRow(query, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// 获取数据
	query = "SELECT id, relay_id, client_ip, action, bytes_in, bytes_out, duration, created_at FROM access_logs"
	if relayID != "" {
		query += " WHERE relay_id = ?"
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"

	if relayID != "" {
		args = []interface{}{relayID, size, (page - 1) * size}
	} else {
		args = []interface{}{size, (page - 1) * size}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*AccessLog
	for rows.Next() {
		l := &AccessLog{}
		if err := rows.Scan(&l.ID, &l.RelayID, &l.ClientIP, &l.Action, &l.BytesIn, &l.BytesOut, &l.Duration, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

// ClearStats 清除统计数据
func ClearStats(relayID string) error {
	if relayID != "" {
		_, err := DB.Exec("DELETE FROM relay_stats WHERE relay_id = ?", relayID)
		if err != nil {
			return err
		}
		_, err = DB.Exec("DELETE FROM access_logs WHERE relay_id = ?", relayID)
		return err
	}
	_, err := DB.Exec("DELETE FROM relay_stats")
	if err != nil {
		return err
	}
	_, err = DB.Exec("DELETE FROM access_logs")
	return err
}

// CleanOldStats 清理旧数据 (保留30天)
func CleanOldStats() error {
	threshold := time.Now().AddDate(0, 0, -30)
	_, err := DB.Exec("DELETE FROM relay_stats WHERE recorded_at < ?", threshold)
	if err != nil {
		return err
	}
	_, err = DB.Exec("DELETE FROM access_logs WHERE created_at < ?", threshold)
	return err
}
