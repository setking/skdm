package store

// DownloadRecord 表示一条下载记录
type DownloadRecord struct {
	GID             string  `json:"gid"`
	URL             string  `json:"url"`
	Dir             string  `json:"dir"`
	Filename        string  `json:"filename"`
	TotalLength     int64   `json:"total_length"`
	CompletedLength int64   `json:"completed_length"`
	DownloadSpeed   int64   `json:"download_speed"`
	Status          string  `json:"status"`
	ErrorCode       int     `json:"error_code"`
	ErrorMessage    string  `json:"error_message"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	CompletedAt     *string `json:"completed_at,omitempty"`
}

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
	}
}

// EventRecord 表示一条下载生命周期事件
type EventRecord struct {
	ID        int64  `json:"id"`
	GID       string `json:"gid"`
	EventType string `json:"event_type"`
	EventData string `json:"event_data"`
	CreatedAt string `json:"created_at"`
}
