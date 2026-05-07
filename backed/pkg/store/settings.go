package store

import (
	"fmt"
	"strconv"
)

// SetSetting 保存键值设置项
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

// GetSetting 获取设置项，不存在时返回空字符串
func (s *Store) GetSetting(key string) (string, error) {
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

// SaveSettings 将 Settings 结构体保存到 SQLite
func (s *Store) SaveSettings(st *Settings) error {
	_ = s.SetSetting("default_download_dir", st.DefaultDownloadDir)
	_ = s.SetSetting("max_concurrent_downloads", strconv.Itoa(st.MaxConcurrentDownloads))
	_ = s.SetSetting("max_connection_per_server", strconv.Itoa(st.MaxConnectionPerServer))
	_ = s.SetSetting("split", strconv.Itoa(st.Split))
	_ = s.SetSetting("max_download_limit", strconv.FormatInt(st.MaxDownloadLimit, 10))
	_ = s.SetSetting("continue_download", strconv.FormatBool(st.Continue))
	_ = s.SetSetting("allow_overwrite", strconv.FormatBool(st.AllowOverwrite))
	_ = s.SetSetting("auto_file_renaming", strconv.FormatBool(st.AutoFileRenaming))
	return nil
}

// GetSettings 从 SQLite 读取设置，未设置的键使用默认值
func (s *Store) GetSettings() (*Settings, error) {
	settings := DefaultSettings()

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
	return settings, nil
}

// GetDefaultDownloadDir 获取默认下载目录
func (s *Store) GetDefaultDownloadDir() string {
	v, err := s.GetSetting("default_download_dir")
	if err != nil || v == "" {
		return DefaultSettings().DefaultDownloadDir
	}
	return v
}

// setFloat 保存浮点数设置（辅助函数）
func setFloat(s *Store, key string, val float64) error {
	return s.SetSetting(key, fmt.Sprintf("%.1f", val))
}

// getFloat 读取浮点数设置（辅助函数）
func (s *Store) getFloat(key string, defaultVal float64) float64 {
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
