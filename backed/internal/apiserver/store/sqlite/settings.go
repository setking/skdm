package sqlite

import (
	"fmt"
	"strconv"

	dv1 "changeme/backed/api/apiserver/v1"

	"github.com/jmoiron/sqlx"
)

type settings struct {
	db *sqlx.DB
}

func newSettings(ds *datastore) *settings {
	return &settings{ds.db}
}

// SetSetting 保存键值设置项
func (s *settings) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

// GetSetting 获取设置项，不存在时返回空字符串
func (s *settings) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(
		`SELECT value FROM settings WHERE key=?`, key,
	).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// ==================== 类型化的设置存取 ====================

// SaveSettings 将 Settings 结构体保存到 SQLite（事务保证原子性）
func (s *settings) SaveSettings(st *dv1.Settings) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开启数据库事务失败: %w", err)
	}
	defer tx.Rollback()

	pairs := []struct{ key, value string }{
		{"default_download_dir", st.DefaultDownloadDir},
		{"max_concurrent_downloads", strconv.Itoa(st.MaxConcurrentDownloads)},
		{"max_connection_per_server", strconv.Itoa(st.MaxConnectionPerServer)},
		{"split", strconv.Itoa(st.Split)},
		{"max_download_limit", strconv.FormatInt(st.MaxDownloadLimit, 10)},
		{"continue_download", strconv.FormatBool(st.Continue)},
		{"allow_overwrite", strconv.FormatBool(st.AllowOverwrite)},
		{"auto_file_renaming", strconv.FormatBool(st.AutoFileRenaming)},
		{"auto_check_update", strconv.FormatBool(st.AutoCheckUpdate)},
		{"auto_start_unfinished", strconv.FormatBool(st.AutoStartUnfinished)},
	}

	for _, p := range pairs {
		if _, err := tx.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			p.key, p.value,
		); err != nil {
			return fmt.Errorf("保存设置项 %s 失败: %w", p.key, err)
		}
	}

	return tx.Commit()
}

// GetSettings 从 SQLite 读取设置，未设置的键使用默认值
func (s *settings) GetSettings() (*dv1.Settings, error) {
	settings := dv1.DefaultSettings()

	if v, err := s.GetSetting("default_download_dir"); err == nil && v != "" {
		settings.DefaultDownloadDir = v
	}
	if v, err := s.GetSetting("max_concurrent_downloads"); err == nil && v != "" {
		if n, err2 := strconv.Atoi(v); err2 == nil {
			settings.MaxConcurrentDownloads = n
		}
	}
	if v, err := s.GetSetting("max_connection_per_server"); err == nil && v != "" {
		if n, err2 := strconv.Atoi(v); err2 == nil {
			settings.MaxConnectionPerServer = n
		}
	}
	if v, err := s.GetSetting("split"); err == nil && v != "" {
		if n, err2 := strconv.Atoi(v); err2 == nil {
			settings.Split = n
		}
	}
	if v, err := s.GetSetting("max_download_limit"); err == nil && v != "" {
		if n, err2 := strconv.ParseInt(v, 10, 64); err2 == nil {
			settings.MaxDownloadLimit = n
		}
	}
	if v, err := s.GetSetting("continue_download"); err == nil && v != "" {
		if b, err2 := strconv.ParseBool(v); err2 == nil {
			settings.Continue = b
		}
	}
	if v, err := s.GetSetting("allow_overwrite"); err == nil && v != "" {
		if b, err2 := strconv.ParseBool(v); err2 == nil {
			settings.AllowOverwrite = b
		}
	}
	if v, err := s.GetSetting("auto_file_renaming"); err == nil && v != "" {
		if b, err2 := strconv.ParseBool(v); err2 == nil {
			settings.AutoFileRenaming = b
		}
	}
	if v, err := s.GetSetting("auto_check_update"); err == nil && v != "" {
		if b, err2 := strconv.ParseBool(v); err2 == nil {
			settings.AutoCheckUpdate = b
		}
	}
	if v, err := s.GetSetting("auto_start_unfinished"); err == nil && v != "" {
		if b, err2 := strconv.ParseBool(v); err2 == nil {
			settings.AutoStartUnfinished = b
		}
	}
	return settings, nil
}

// GetDefaultDownloadDir 获取默认下载目录
func (s *settings) GetDefaultDownloadDir() string {
	v, err := s.GetSetting("default_download_dir")
	if err != nil || v == "" {
		return dv1.DefaultSettings().DefaultDownloadDir
	}
	return v
}

//setFloat 保存浮点数设置（辅助函数）
func (s *settings) SetFloat(key string, val float64) error {
	return s.SetSetting(key, fmt.Sprintf("%.1f", val))
}

// getFloat 读取浮点数设置（辅助函数）
func (s *settings) GetFloat(key string, defaultVal float64) float64 {
	v, err := s.GetSetting(key)
	if err != nil || v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultVal
	}
	return f
}
