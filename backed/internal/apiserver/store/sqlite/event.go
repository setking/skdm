package sqlite

import (
	"time"

	dv1 "changeme/backed/api/apiserver/v1"

	"github.com/jmoiron/sqlx"
)

type event struct {
	db *sqlx.DB
}

func newEvent(ds *datastore) *event {
	return &event{ds.db}
}
func (s *event) InsertEvent(gid, eventType, eventData string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO download_events (gid, event_type, event_data, created_at) VALUES (?, ?, ?, ?)`,
		gid, eventType, eventData, now,
	)
	return err
}

// ListEventsByGID 获取指定下载任务的所有事件记录
func (s *event) ListEventsByGID(gid string) ([]dv1.EventRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, gid, event_type, event_data, created_at
		 FROM download_events WHERE gid=? ORDER BY created_at ASC`, gid,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []dv1.EventRecord
	for rows.Next() {
		var e dv1.EventRecord
		if err := rows.Scan(&e.ID, &e.GID, &e.EventType, &e.EventData, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	if list == nil {
		list = []dv1.EventRecord{}
	}
	return list, rows.Err()
}
