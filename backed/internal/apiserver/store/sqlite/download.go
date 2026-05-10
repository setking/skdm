package sqlite

import (
	dv1 "changeme/backed/api/apiserver/v1"
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

type download struct {
	db *sqlx.DB
}

func newDownload(ds *datastore) *download {
	return &download{ds.db}
}

// InsertDownload 创建新的下载记录
func (d *download) InsertDownload(ctx context.Context, dw *dv1.DownloadRecord) error {
	now := time.Now().UTC().Format(time.RFC3339)

	dw.CreatedAt = now
	dw.UpdatedAt = now

	_, err := d.db.NamedExecContext(ctx, `
		INSERT INTO downloads (
			gid, url, dir, filename,
			total_length, completed_length, download_speed,
			status, error_code, error_message,
			created_at, updated_at, completed_at
		) VALUES (
			:gid, :url, :dir, :filename,
			:total_length, :completed_length, :download_speed,
			:status, :error_code, :error_message,
			:created_at, :updated_at, :completed_at
		)
	`, dw)

	return err
}

// UpdateDownloadStatus 更新下载任务的状态、进度和错误信息
func (d *download) UpdateDownloadStatus(gid, status string, completedLength, totalLength, downloadSpeed int64, errorCode int, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var completedAt *string
	if status == "complete" {
		completedAt = &now
	}
	_, err := d.db.Exec(
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
func (d *download) UpdateDownloadProgress(gid string, completedLength, totalLength, downloadSpeed int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.db.Exec(
		`UPDATE downloads SET completed_length=?, total_length=?, download_speed=?, updated_at=? WHERE gid=?`,
		completedLength, totalLength, downloadSpeed, now, gid,
	)
	return err
}

// DeleteDownload 删除下载记录
func (d *download) DeleteDownload(gid string) error {
	_, err := d.db.Exec(`DELETE FROM downloads WHERE gid=?`, gid)
	return err
}

// FindDownloadByURL 根据 URL 查找下载记录（URL 完全匹配）
// 返回最近一条匹配的记录，若无匹配则返回 nil
func (d *download) FindDownloadByURL(url string) (*dv1.DownloadRecord, error) {
	row := d.db.QueryRow(
		`SELECT gid, url, dir, filename, total_length, completed_length, download_speed,
		 status, error_code, error_message, created_at, updated_at, completed_at
		 FROM downloads WHERE url=? ORDER BY created_at DESC LIMIT 1`, url,
	)
	dw := &dv1.DownloadRecord{}
	err := row.Scan(&dw.GID, &dw.URL, &dw.Dir, &dw.Filename,
		&dw.TotalLength, &dw.CompletedLength, &dw.DownloadSpeed,
		&dw.Status, &dw.ErrorCode, &dw.ErrorMessage,
		&dw.CreatedAt, &dw.UpdatedAt, &dw.CompletedAt)
	if err != nil {
		return nil, err
	}
	return dw, nil
}

// GetDownload 获取单条下载记录
func (d *download) GetDownload(gid string) (*dv1.DownloadRecord, error) {
	row := d.db.QueryRow(
		`SELECT gid, url, dir, filename, total_length, completed_length, download_speed,
		 status, error_code, error_message, created_at, updated_at, completed_at
		 FROM downloads WHERE gid=?`, gid,
	)
	dw := &dv1.DownloadRecord{}
	err := row.Scan(&dw.GID, &dw.URL, &dw.Dir, &dw.Filename,
		&dw.TotalLength, &dw.CompletedLength, &dw.DownloadSpeed,
		&dw.Status, &dw.ErrorCode, &dw.ErrorMessage,
		&dw.CreatedAt, &dw.UpdatedAt, &dw.CompletedAt)
	if err != nil {
		return nil, err
	}
	return dw, nil
}

// ListDownloads 分页查询下载记录，status为空时返回全部
func (d *download) ListDownloads(status string, offset, limit int) ([]dv1.DownloadRecord, int, error) {
	var total int
	var rows *sql.Rows
	var err error

	if status == "" {
		err = d.db.QueryRow(`SELECT COUNT(*) FROM downloads`).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = d.db.Query(
			`SELECT gid, url, dir, filename, total_length, completed_length, download_speed,
			 status, error_code, error_message, created_at, updated_at, completed_at
			 FROM downloads ORDER BY created_at DESC LIMIT ? OFFSET ?`,
			limit, offset,
		)
	} else {
		err = d.db.QueryRow(`SELECT COUNT(*) FROM downloads WHERE status=?`, status).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = d.db.Query(
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

	var list []dv1.DownloadRecord
	for rows.Next() {
		var d dv1.DownloadRecord
		if err := rows.Scan(&d.GID, &d.URL, &d.Dir, &d.Filename,
			&d.TotalLength, &d.CompletedLength, &d.DownloadSpeed,
			&d.Status, &d.ErrorCode, &d.ErrorMessage,
			&d.CreatedAt, &d.UpdatedAt, &d.CompletedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, d)
	}
	if list == nil {
		list = []dv1.DownloadRecord{}
	}
	return list, total, rows.Err()
}
