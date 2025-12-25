package model

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

// InitDB 初始化数据库
func InitDB(dataDir string) error {
	// 使用更严格的权限，仅所有者可读写执行
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}

	dbPath := filepath.Join(dataDir, "relay.db")
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	// 创建表
	if err := createTables(); err != nil {
		return err
	}

	log.Printf("数据库初始化完成: %s", dbPath)
	return nil
}

func createTables() error {
	// system_settings 表
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS system_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// relay_rules 表
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS relay_rules (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			src TEXT NOT NULL,
			dst TEXT NOT NULL,
			protocol TEXT NOT NULL DEFAULT 'both',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// relay_stats 表
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS relay_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			relay_id TEXT NOT NULL,
			bytes_in INTEGER NOT NULL DEFAULT 0,
			bytes_out INTEGER NOT NULL DEFAULT 0,
			connections INTEGER NOT NULL DEFAULT 0,
			recorded_at DATETIME NOT NULL,
			FOREIGN KEY (relay_id) REFERENCES relay_rules(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_relay_stats_relay_id ON relay_stats(relay_id)`)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_relay_stats_recorded_at ON relay_stats(recorded_at)`)
	if err != nil {
		return err
	}
	// 唯一索引用于 UPSERT 操作
	_, err = DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_relay_stats_unique ON relay_stats(relay_id, recorded_at)`)
	if err != nil {
		return err
	}

	// access_logs 表
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS access_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			relay_id TEXT NOT NULL,
			client_ip TEXT NOT NULL,
			action TEXT NOT NULL,
			bytes_in INTEGER NOT NULL DEFAULT 0,
			bytes_out INTEGER NOT NULL DEFAULT 0,
			duration INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (relay_id) REFERENCES relay_rules(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_access_logs_relay_id ON access_logs(relay_id)`)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_access_logs_created_at ON access_logs(created_at)`)
	if err != nil {
		return err
	}

	return nil
}

// CloseDB 关闭数据库
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
