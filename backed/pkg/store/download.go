package store

import (
	"database/sql"
	"time"
)

// InsertDownload 创建新的下载记录
func (s *Store) InsertDownload(d *DownloadRecord) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT INTO downloads (gid, url, dir, filename, total_length, completed_length,
		 download_speed, status, error_code, error_message, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.GID, d.URL, d.Dir, d.Filename,
		d.TotalLength, d.CompletedLength, d.DownloadSpeed,
		d.Status, d.ErrorCode, d.ErrorMessage,
		now, now,
	)
	return err
}

// UpdateDownloadStatus 更新下载任务的状态、进度和错误信息
func (s *Store) UpdateDownloadStatus(gid, status string, completedLength, totalLength, downloadSpeed int64, errorCode int, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var completedAt interface{}
	if status == "complete" {
		completedAt = now
	}
	_, err := s.db.Exec(
		`UPDATE downloads SET status=?, completed_length=?, total_length=?,
		 download_speed=?, error_code=?, error_message=?,
		 updated_at=?, completed_at=COALESCE(?, completed_at)
		 WHERE gid=?`,
		status, completedLength, totalLength, downloadSpeed,
		errorCode, errorMessage, now, completedAt, gid,
	)
	return err
}

// UpdateDownloadProgress 更新下载任务的进度（不改变状态）
func (s *Store) UpdateDownloadProgress(gid string, completedLength, totalLength, downloadSpeed int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`UPDATE downloads SET completed_length=?, total_length=?, download_speed=?, updated_at=? WHERE gid=?`,
		completedLength, totalLength, downloadSpeed, now, gid,
	)
	return err
}

// DeleteDownload 删除下载记录
func (s *Store) DeleteDownload(gid string) error {
	_, err := s.db.Exec(`DELETE FROM downloads WHERE gid=?`, gid)
	return err
}

// GetDownload 获取单条下载记录
func (s *Store) GetDownload(gid string) (*DownloadRecord, error) {
	row := s.db.QueryRow(
		`SELECT gid, url, dir, filename, total_length, completed_length, download_speed,
		 status, error_code, error_message, created_at, updated_at, completed_at
		 FROM downloads WHERE gid=?`, gid,
	)
	d := &DownloadRecord{}
	err := row.Scan(&d.GID, &d.URL, &d.Dir, &d.Filename,
		&d.TotalLength, &d.CompletedLength, &d.DownloadSpeed,
		&d.Status, &d.ErrorCode, &d.ErrorMessage,
		&d.CreatedAt, &d.UpdatedAt, &d.CompletedAt)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// ListDownloads 分页查询下载记录，status为空时返回全部
func (s *Store) ListDownloads(status string, offset, limit int) ([]DownloadRecord, int, error) {
	var total int
	var rows *sql.Rows
	var err error

	if status == "" {
		err = s.db.QueryRow(`SELECT COUNT(*) FROM downloads`).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = s.db.Query(
			`SELECT gid, url, dir, filename, total_length, completed_length, download_speed,
			 status, error_code, error_message, created_at, updated_at, completed_at
			 FROM downloads ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			limit, offset,
		)
	} else {
		err = s.db.QueryRow(`SELECT COUNT(*) FROM downloads WHERE status=?`, status).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = s.db.Query(
			`SELECT gid, url, dir, filename, total_length, completed_length, download_speed,
			 status, error_code, error_message, created_at, updated_at, completed_at
			 FROM downloads WHERE status=? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			status, limit, offset,
		)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []DownloadRecord
	for rows.Next() {
		var d DownloadRecord
		if err := rows.Scan(&d.GID, &d.URL, &d.Dir, &d.Filename,
			&d.TotalLength, &d.CompletedLength, &d.DownloadSpeed,
			&d.Status, &d.ErrorCode, &d.ErrorMessage,
			&d.CreatedAt, &d.UpdatedAt, &d.CompletedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, d)
	}
	if list == nil {
		list = []DownloadRecord{}
	}
	return list, total, rows.Err()
}
