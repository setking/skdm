package store

import (
	dv1 "changeme/backed/api/apiserver/v1"
	"context"
)

// DownloadStore defines the user storage interface.
type DownloadStore interface {
	InsertDownload(ctx context.Context, dw *dv1.DownloadRecord) error
	UpdateDownloadStatus(gid, status string, completedLength, totalLength, downloadSpeed int64, errorCode int, errorMessage string) error
	UpdateDownloadProgress(gid string, completedLength, totalLength, downloadSpeed int64) error
	DeleteDownload(gid string) error
	SoftDeleteDownload(gid string) error
	RestoreDownload(gid string) error
	FindDownloadByURL(url string) (*dv1.DownloadRecord, error)
	GetDownload(gid string) (*dv1.DownloadRecord, error)
	ListDownloads(status string, offset, limit int) ([]dv1.DownloadRecord, int, error)
}
