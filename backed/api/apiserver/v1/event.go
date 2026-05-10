package v1

// EventRecord 表示一条下载生命周期事件
type EventRecord struct {
	ID        int64  `json:"id"`
	GID       string `json:"gid"`
	EventType string `json:"event_type"`
	EventData string `json:"event_data"`
	CreatedAt string `json:"created_at"`
}
