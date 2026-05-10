package v1

import (
	dv1 "changeme/backed/api/apiserver/v1"
	"changeme/backed/internal/apiserver/store"
)

// SettingsSrv defines functions used to handle user request.
type SettingsSrv interface {
	SetSetting(key, value string) error
	GetSetting(key string) (string, error)
	SaveSettings(st *dv1.Settings) error
	GetSettings() (*dv1.Settings, error)
	GetDefaultDownloadDir() string
	GetFloat(key string, defaultVal float64) float64
	SetFloat(key string, val float64) error
}

type settingsService struct {
	store store.Factory
}

func (s settingsService) SetSetting(key, value string) error {
	return s.store.Settings().SetSetting(key, value)
}

func (s settingsService) GetSetting(key string) (string, error) {
	return s.store.Settings().GetSetting(key)
}

func (s settingsService) SaveSettings(st *dv1.Settings) error {
	return s.store.Settings().SaveSettings(st)
}

func (s settingsService) GetSettings() (*dv1.Settings, error) {
	return s.store.Settings().GetSettings()
}

func (s settingsService) GetDefaultDownloadDir() string {
	return s.store.Settings().GetDefaultDownloadDir()
}

func (s settingsService) GetFloat(key string, defaultVal float64) float64 {
	return s.store.Settings().GetFloat(key, defaultVal)
}

func (s settingsService) SetFloat(key string, val float64) error {
	return s.store.Settings().SetFloat(key, val)
}

var _ SettingsSrv = (*settingsService)(nil)

func newSettings(srv *service) *settingsService {
	return &settingsService{store: srv.store}
}
