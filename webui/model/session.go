package model

import (
	"time"
)

// Session 会话数据
type Session struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// CreateSession 创建会话
func CreateSession(token string, ttl time.Duration) error {
	now := time.Now()
	expiresAt := now.Add(ttl)
	_, err := DB.Exec(`
		INSERT INTO sessions (token, created_at, expires_at) VALUES (?, ?, ?)
	`, token, now, expiresAt)
	return err
}

// GetSession 获取会话（仅返回未过期的）
func GetSession(token string) (*Session, error) {
	var s Session
	err := DB.QueryRow(`
		SELECT token, created_at, expires_at FROM sessions
		WHERE token = ? AND expires_at > ?
	`, token, time.Now()).Scan(&s.Token, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteSession 删除会话
func DeleteSession(token string) error {
	_, err := DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// DeleteAllSessions 删除所有会话
func DeleteAllSessions() error {
	_, err := DB.Exec(`DELETE FROM sessions`)
	return err
}

// CleanExpiredSessions 清理过期会话
func CleanExpiredSessions() error {
	_, err := DB.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	return err
}
