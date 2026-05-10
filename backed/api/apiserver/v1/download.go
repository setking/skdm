package v1

import "time"

// DownloadRecord 表示一条下载记录
type DownloadRecord struct {
	GID             string     `db:"gid" json:"gid"`
	URL             string     `db:"url" json:"url"`
	Dir             string     `db:"dir" json:"dir"`
	Filename        string     `db:"filename" json:"filename"`
	TotalLength     int64      `db:"total_length" json:"total_length"`
	CompletedLength int64      `db:"completed_length" json:"completed_length"`
	DownloadSpeed   int64      `db:"download_speed" json:"download_speed"`
	Status          string     `db:"status" json:"status"`
	ErrorCode       int        `db:"error_code" json:"error_code"`
	ErrorMessage    string     `db:"error_message" json:"error_message"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
	CompletedAt     *time.Time `db:"completed_at" json:"completed_at,omitempty"`
}
