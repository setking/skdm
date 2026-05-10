package v1

import (
	"changeme/backed/internal/apiserver/store"
)

type Service interface {
	Downloads() DownloadSrv
	Events() EventSrv
	Settings() SettingsSrv
}

type service struct {
	store store.Factory
}

func NewService(store store.Factory) Service {
	return &service{
		store: store,
	}
}

func (s *service) Downloads() DownloadSrv {
	return newDownload(s)
}
func (s *service) Events() EventSrv { return newEvent(s) }
func (s *service) Settings() SettingsSrv {
	return newSettings(s)
}
