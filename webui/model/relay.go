package model

import (
	"time"

	"github.com/google/uuid"
)

// RelayRule 转发规则
type RelayRule struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Src       string    `json:"src"`
	Dst       string    `json:"dst"`
	Protocol  string    `json:"protocol"` // tcp, udp, both
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateRelayRule 创建规则
func CreateRelayRule(name, src, dst, protocol string) (*RelayRule, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := DB.Exec(`
		INSERT INTO relay_rules (id, name, src, dst, protocol, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)
	`, id, name, src, dst, protocol, now, now)
	if err != nil {
		return nil, err
	}

	return &RelayRule{
		ID:        id,
		Name:      name,
		Src:       src,
		Dst:       dst,
		Protocol:  protocol,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetRelayRule 获取单个规则
func GetRelayRule(id string) (*RelayRule, error) {
	rule := &RelayRule{}
	var enabled int
	err := DB.QueryRow(`
		SELECT id, name, src, dst, protocol, enabled, created_at, updated_at
		FROM relay_rules WHERE id = ?
	`, id).Scan(&rule.ID, &rule.Name, &rule.Src, &rule.Dst, &rule.Protocol, &enabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	rule.Enabled = enabled == 1
	return rule, nil
}

// GetAllRelayRules 获取所有规则
func GetAllRelayRules() ([]*RelayRule, error) {
	rows, err := DB.Query(`
		SELECT id, name, src, dst, protocol, enabled, created_at, updated_at
		FROM relay_rules ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*RelayRule
	for rows.Next() {
		rule := &RelayRule{}
		var enabled int
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Src, &rule.Dst, &rule.Protocol, &enabled, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rule.Enabled = enabled == 1
		rules = append(rules, rule)
	}
	return rules, nil
}

// GetEnabledRelayRules 获取所有启用的规则
func GetEnabledRelayRules() ([]*RelayRule, error) {
	rows, err := DB.Query(`
		SELECT id, name, src, dst, protocol, enabled, created_at, updated_at
		FROM relay_rules WHERE enabled = 1 ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*RelayRule
	for rows.Next() {
		rule := &RelayRule{}
		var enabled int
		if err := rows.Scan(&rule.ID, &rule.Name, &rule.Src, &rule.Dst, &rule.Protocol, &enabled, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rule.Enabled = enabled == 1
		rules = append(rules, rule)
	}
	return rules, nil
}

// UpdateRelayRule 更新规则
func UpdateRelayRule(id, name, src, dst, protocol string) error {
	_, err := DB.Exec(`
		UPDATE relay_rules SET name = ?, src = ?, dst = ?, protocol = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, src, dst, protocol, id)
	return err
}

// DeleteRelayRule 删除规则
func DeleteRelayRule(id string) error {
	_, err := DB.Exec("DELETE FROM relay_rules WHERE id = ?", id)
	return err
}

// SetRelayEnabled 设置规则启用状态
func SetRelayEnabled(id string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := DB.Exec("UPDATE relay_rules SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", enabledInt, id)
	return err
}

// GetRelayRuleBySrc 按监听地址查询规则
func GetRelayRuleBySrc(src string) (*RelayRule, error) {
	rule := &RelayRule{}
	var enabled int
	err := DB.QueryRow(`
		SELECT id, name, src, dst, protocol, enabled, created_at, updated_at
		FROM relay_rules WHERE src = ?
	`, src).Scan(&rule.ID, &rule.Name, &rule.Src, &rule.Dst, &rule.Protocol, &enabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return nil, err
	}
	rule.Enabled = enabled == 1
	return rule, nil
}
