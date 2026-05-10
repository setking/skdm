package store

import (
	dv1 "changeme/backed/api/apiserver/v1"
)

// SettingsStore defines the user storage interface.
type SettingsStore interface {
	SetSetting(key, value string) error
	GetSetting(key string) (string, error)
	SaveSettings(st *dv1.Settings) error
	GetSettings() (*dv1.Settings, error)
	GetDefaultDownloadDir() string
	GetFloat(key string, defaultVal float64) float64
	SetFloat(key string, val float64) error
}
