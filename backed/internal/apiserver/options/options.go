package options

import (
	genericoptions "changeme/backed/internal/pkg/options"
	"encoding/json"
)

type Options struct {
	SqliteOptions           *genericoptions.SqliteOptions    `json:"sqlite"    mapstructure:"sqlite"`
	GenericServerRunOptions *genericoptions.ServerRunOptions `json:"server"   mapstructure:"server"`
}

func NewOptions() *Options {
	o := Options{
		SqliteOptions:           genericoptions.NewSqliteOptions(),
		GenericServerRunOptions: genericoptions.NewServerRunOptions(),
	}
	return &o
}

func (o *Options) String() string {
	data, _ := json.Marshal(o)

	return string(data)
}
