package v1

// Settings 表示用户可配置的设置项
type Settings struct {
	DefaultDownloadDir     string `json:"default_download_dir"`
	MaxConcurrentDownloads int    `json:"max_concurrent_downloads"`
	MaxConnectionPerServer int    `json:"max_connection_per_server"`
	Split                  int    `json:"split"`
	MaxDownloadLimit       int64  `json:"max_download_limit"` // bytes/s, 0=不限速
	Continue               bool   `json:"continue_download"`
	AllowOverwrite         bool   `json:"allow_overwrite"`
	AutoFileRenaming       bool   `json:"auto_file_renaming"`
	AutoCheckUpdate        bool   `json:"auto_check_update"`
	AutoStartUnfinished    bool   `json:"auto_start_unfinished"`
}

// DefaultSettings 返回默认设置
func DefaultSettings() *Settings {
	return &Settings{
		DefaultDownloadDir:     "./download",
		MaxConcurrentDownloads: 5,
		MaxConnectionPerServer: 1,
		Split:                  5,
		MaxDownloadLimit:       0, // 不限速
		Continue:               true,
		AllowOverwrite:         true,
		AutoFileRenaming:       true,
		AutoCheckUpdate:        true,
		AutoStartUnfinished:    true,
	}
}
