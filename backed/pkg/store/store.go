package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store 封装数据库连接，提供数据访问方法
type Store struct {
	db *sql.DB
}

// Open 初始化SQLite数据库，自动创建目录和表
func Open(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("打开SQLite失败: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return &Store{db: db}, nil
}

// Close 关闭数据库连接
func (s *Store) Close() error {
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

const schema = `
CREATE TABLE IF NOT EXISTS downloads (
    gid TEXT PRIMARY KEY,
    url TEXT NOT NULL DEFAULT '',
    dir TEXT NOT NULL DEFAULT '',
    filename TEXT NOT NULL DEFAULT '',
    total_length INTEGER NOT NULL DEFAULT 0,
    completed_length INTEGER NOT NULL DEFAULT 0,
    download_speed INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    error_code INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    completed_at TEXT
);

CREATE TABLE IF NOT EXISTS download_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    gid TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_data TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    FOREIGN KEY (gid) REFERENCES downloads(gid)
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_download_events_gid ON download_events(gid);
CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);
`
