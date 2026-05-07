package store

import "time"

// InsertEvent 记录下载生命周期事件
func (s *Store) InsertEvent(gid, eventType, eventData string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO download_events (gid, event_type, event_data, created_at) VALUES (?, ?, ?, ?)`,
		gid, eventType, eventData, now,
	)
	return err
}

// ListEventsByGID 获取指定下载任务的所有事件记录
func (s *Store) ListEventsByGID(gid string) ([]EventRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, gid, event_type, event_data, created_at
		 FROM download_events WHERE gid=? ORDER BY created_at ASC`, gid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []EventRecord
	for rows.Next() {
		var e EventRecord
		if err := rows.Scan(&e.ID, &e.GID, &e.EventType, &e.EventData, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	if list == nil {
		list = []EventRecord{}
	}
	return list, rows.Err()
}
