package v1

import (
	dv1 "changeme/backed/api/apiserver/v1"
	"changeme/backed/internal/apiserver/store"
	"context"
)

// DownloadSrv defines functions used to handle user request.
type DownloadSrv interface {
	InsertDownload(ctx context.Context, dw *dv1.DownloadRecord) error
	UpdateDownloadStatus(gid, status string, completedLength, totalLength, downloadSpeed int64, errorCode int, errorMessage string) error
	UpdateDownloadProgress(gid string, completedLength, totalLength, downloadSpeed int64) error
	DeleteDownload(gid string) error
	FindDownloadByURL(url string) (*dv1.DownloadRecord, error)
	GetDownload(gid string) (*dv1.DownloadRecord, error)
	ListDownloads(status string, offset, limit int) ([]dv1.DownloadRecord, int, error)
}

type downloadService struct {
	store store.Factory
}

func (d downloadService) InsertDownload(ctx context.Context, dw *dv1.DownloadRecord) error {
	return d.store.Downloads().InsertDownload(ctx, dw)
}

func (d downloadService) UpdateDownloadStatus(gid, status string, completedLength, totalLength, downloadSpeed int64, errorCode int, errorMessage string) error {
	return d.store.Downloads().UpdateDownloadStatus(gid, status, completedLength, totalLength, downloadSpeed, errorCode, errorMessage)
}

func (d downloadService) UpdateDownloadProgress(gid string, completedLength, totalLength, downloadSpeed int64) error {
	return d.store.Downloads().UpdateDownloadProgress(gid, completedLength, totalLength, downloadSpeed)
}

func (d downloadService) DeleteDownload(gid string) error {
	return d.store.Downloads().DeleteDownload(gid)
}

func (d downloadService) FindDownloadByURL(url string) (*dv1.DownloadRecord, error) {
	return d.store.Downloads().FindDownloadByURL(url)
}

func (d downloadService) GetDownload(gid string) (*dv1.DownloadRecord, error) {
	return d.store.Downloads().GetDownload(gid)
}

func (d downloadService) ListDownloads(status string, offset, limit int) ([]dv1.DownloadRecord, int, error) {
	return d.store.Downloads().ListDownloads(status, offset, limit)
}

var _ DownloadSrv = (*downloadService)(nil)

func newDownload(srv *service) *downloadService {
	return &downloadService{store: srv.store}
}
