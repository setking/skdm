package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Options struct {
	// SQLite 使用文件路径
	Database              string // 比如: ./test.db 或 :memory:
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
}

// New create a new sqlx db instance with the given options.
func New(opts *Options) (*sqlx.DB, error) {
	// 确保数据库文件所在目录存在（排除 :memory: 等特殊名称）
	if opts.Database != ":memory:" && opts.Database != "" {
		dir := filepath.Dir(opts.Database)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建数据库目录 %s 失败: %w", dir, err)
		}
	}

	// 使用 DELETE 日志模式（而非 WAL），避免因上次异常退出遗留的 WAL
	// 文件导致数据库打开失败。桌面应用场景下 DELETE 模式性能完全足够。
	dsn := opts.Database + "?_busy_timeout=5000&_journal_mode=DELETE"

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 数据库失败: %w", err)
	}

	// 确认连接可用
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接 SQLite 数据库失败: %w", err)
	}

	// 自动建表（CREATE TABLE IF NOT EXISTS，首次运行或表不存在时创建）
	if err := migrateDatabase(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库表结构失败: %w", err)
	}

	// 连接池配置（SQLite 其实不太需要高并发连接池）
	db.SetMaxOpenConns(opts.MaxOpenConnections)
	db.SetMaxIdleConns(opts.MaxIdleConnections)
	db.SetConnMaxLifetime(opts.MaxConnectionLifeTime)

	return db, nil
}

// migrateDatabase 检查并创建数据库表结构（幂等操作，已存在的表不重复创建）。
func migrateDatabase(db *sqlx.DB) error {
	schema := `
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
	_, err := db.Exec(schema)
	return err
}
