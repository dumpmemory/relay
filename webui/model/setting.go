package model

import (
	"time"
)

// Setting 系统设置
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetSetting 获取设置
func GetSetting(key string) (string, error) {
	var value string
	err := DB.QueryRow("SELECT value FROM system_settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetSetting 设置值
func SetSetting(key, value string) error {
	_, err := DB.Exec(`
		INSERT INTO system_settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP
	`, key, value, value)
	return err
}

// GetAllSettings 获取所有设置
func GetAllSettings() (map[string]string, error) {
	rows, err := DB.Query("SELECT key, value FROM system_settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}

// IsSetupCompleted 检查是否完成初始化
func IsSetupCompleted() bool {
	value, err := GetSetting("setup_completed")
	if err != nil {
		return false
	}
	return value == "true"
}

// SetSetupCompleted 设置初始化完成
func SetSetupCompleted() error {
	return SetSetting("setup_completed", "true")
}
