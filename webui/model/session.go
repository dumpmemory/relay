package model

import (
	"sync"
	"time"
)

// Session 会话数据
type Session struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// 内存缓存
var (
	sessionCache     sync.Map // token -> *Session (有效会话)
	invalidTokens    sync.Map // token -> time.Time (无效 token 及其过期时间)
	invalidTokenTTL  = 5 * time.Minute // 无效 token 缓存 5 分钟
)

// CreateSession 创建会话
func CreateSession(token string, ttl time.Duration) error {
	now := time.Now()
	expiresAt := now.Add(ttl)
	_, err := DB.Exec(`
		INSERT INTO sessions (token, created_at, expires_at) VALUES (?, ?, ?)
	`, token, now, expiresAt)
	if err != nil {
		return err
	}
	// 写入缓存
	sessionCache.Store(token, &Session{
		Token:     token,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	})
	// 从黑名单移除（如果有）
	invalidTokens.Delete(token)
	return nil
}

// GetSession 获取会话（仅返回未过期的）
func GetSession(token string) (*Session, error) {
	now := time.Now()

	// 检查黑名单缓存
	if v, ok := invalidTokens.Load(token); ok {
		expireTime := v.(time.Time)
		if now.Before(expireTime) {
			// 在黑名单中且未过期，直接返回错误
			return nil, ErrSessionNotFound
		}
		// 黑名单缓存过期，删除
		invalidTokens.Delete(token)
	}

	// 检查有效缓存
	if v, ok := sessionCache.Load(token); ok {
		s := v.(*Session)
		if now.Before(s.ExpiresAt) {
			return s, nil
		}
		// 缓存中的会话已过期，删除
		sessionCache.Delete(token)
	}

	// 缓存未命中，查询数据库
	var s Session
	err := DB.QueryRow(`
		SELECT token, created_at, expires_at FROM sessions
		WHERE token = ? AND expires_at > ?
	`, token, now).Scan(&s.Token, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		// 数据库中不存在，加入黑名单
		invalidTokens.Store(token, now.Add(invalidTokenTTL))
		return nil, ErrSessionNotFound
	}

	// 写入缓存
	sessionCache.Store(token, &s)
	return &s, nil
}

// DeleteSession 删除会话
func DeleteSession(token string) error {
	// 从缓存删除
	sessionCache.Delete(token)
	// 加入黑名单
	invalidTokens.Store(token, time.Now().Add(invalidTokenTTL))
	// 从数据库删除
	_, err := DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// DeleteAllSessions 删除所有会话
func DeleteAllSessions() error {
	// 清空缓存
	sessionCache.Range(func(key, value interface{}) bool {
		sessionCache.Delete(key)
		return true
	})
	// 从数据库删除
	_, err := DB.Exec(`DELETE FROM sessions`)
	return err
}

// CleanExpiredSessions 清理过期会话
func CleanExpiredSessions() error {
	now := time.Now()
	// 清理缓存中过期的会话
	sessionCache.Range(func(key, value interface{}) bool {
		s := value.(*Session)
		if now.After(s.ExpiresAt) {
			sessionCache.Delete(key)
		}
		return true
	})
	// 清理过期的黑名单
	invalidTokens.Range(func(key, value interface{}) bool {
		expireTime := value.(time.Time)
		if now.After(expireTime) {
			invalidTokens.Delete(key)
		}
		return true
	})
	// 从数据库清理
	_, err := DB.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now)
	return err
}

// ErrSessionNotFound 会话不存在错误
var ErrSessionNotFound = &sessionError{"session not found"}

type sessionError struct {
	msg string
}

func (e *sessionError) Error() string {
	return e.msg
}
