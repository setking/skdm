package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// UserDataDir 返回用户级数据目录。
// Windows: %LOCALAPPDATA%\SKDM
// macOS:   ~/Library/Application Support/SKDM
// Linux:   ~/.local/share/SKDM
func UserDataDir() string {
	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = os.Getenv("APPDATA")
		}
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, "AppData", "Local")
		}
	} else {
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				home = "."
			}
			if runtime.GOOS == "darwin" {
				base = filepath.Join(home, "Library", "Application Support")
			} else {
				base = filepath.Join(home, ".local", "share")
			}
		}
	}
	return filepath.Join(base, "SKDM")
}

type Options struct {
	// SQLite 使用文件路径
	Database              string // 比如: ./test.db 或 :memory:
	MaxIdleConnections    int
	MaxOpenConnections    int
	MaxConnectionLifeTime time.Duration
}

// New create a new sqlx db instance with the given options.
func New(opts *Options) (*sqlx.DB, error) {
	dbPath := opts.Database

	// 对于非内存数据库，解析绝对路径并确保目录和文件可写
	if dbPath != ":memory:" && dbPath != "" {
		dbPath = resolveDBPath(dbPath)
		if err := ensureWritable(dbPath); err != nil {
			// 如果指定路径不可写（例如安装到受保护的系统目录），
			// 自动回退到用户级数据目录，避免程序无法运行。
			fallbackPath := filepath.Join(UserDataDir(), filepath.Base(dbPath))
			log.Printf("[DB] 数据库目录不可写 (%v)，回退到 %s", err, fallbackPath)
			dbPath = fallbackPath
			if err := ensureWritable(dbPath); err != nil {
				return nil, fmt.Errorf("数据库回退目录也不可写 (%s): %w", dbPath, err)
			}
		}
	}

	// 不使用 DSN 查询参数（modernc.org/sqlite 不支持 _busy_timeout 等参数），
	// PRAGMA 通过连接后执行来设置。
	db, err := sqlx.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 数据库失败: %w", err)
	}

	// 确认连接可用
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接 SQLite 数据库失败: %w", err)
	}

	// 设置 pragma（modernc.org/sqlite 需要在连接后手动执行）
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 busy_timeout 失败: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = DELETE"); err != nil {
		db.Close()
		return nil, fmt.Errorf("设置 journal_mode 失败: %w", err)
	}

	// 自动建表（CREATE TABLE IF NOT EXISTS，首次运行或表不存在时创建）
	if err := migrateDatabase(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库表结构失败: %w", err)
	}

	// 连接池配置
	db.SetMaxOpenConns(opts.MaxOpenConnections)
	db.SetMaxIdleConns(opts.MaxIdleConnections)
	db.SetConnMaxLifetime(opts.MaxConnectionLifeTime)

	return db, nil
}

// resolveDBPath 将相对路径转为绝对路径，确保路径一致性。
func resolveDBPath(dbPath string) string {
	if filepath.IsAbs(dbPath) {
		return dbPath
	}
	if abs, err := filepath.Abs(dbPath); err == nil {
		return abs
	}
	return dbPath
}

// ensureWritable 确保数据库文件所在目录存在并可写。
// 在 Windows 上移除文件的只读属性，并验证目录写权限。
func ensureWritable(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录 %s 失败: %w", dir, err)
	}

	// 验证目录可写：尝试创建一个临时文件
	testFile := filepath.Join(dir, ".skdm_write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("数据库目录不可写 (%s): %w — 请检查目录权限或以管理员身份运行", dir, err)
	}
	f.Close()
	os.Remove(testFile)

	// 如果数据库文件已存在，移除 Windows 只读属性
	if info, err := os.Stat(dbPath); err == nil && runtime.GOOS == "windows" {
		perm := info.Mode()
		if perm&0200 == 0 {
			if err := os.Chmod(dbPath, perm|0200); err != nil {
				return fmt.Errorf("修改数据库文件权限失败(%s): %w", dbPath, err)
			}
		}
	}
	return nil
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
