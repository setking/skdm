package options

import (
	"time"
)

type SqliteOptions struct {
	Database              string        `json:"database"                           mapstructure:"database"`
	MaxIdleConnections    int           `json:"max-idle-connections,omitempty"     mapstructure:"max-idle-connections"`
	MaxOpenConnections    int           `json:"max-open-connections,omitempty"     mapstructure:"max-open-connections"`
	MaxConnectionLifeTime time.Duration `json:"max-connection-life-time,omitempty" mapstructure:"max-connection-life-time"`
}

func NewSqliteOptions() *SqliteOptions {
	return &SqliteOptions{
		Database:              "./app.db",
		MaxIdleConnections:    1,
		MaxOpenConnections:    1,
		MaxConnectionLifeTime: 0,
	}
}

// Validate checks validation of ServerRunOptions.
func (s *SqliteOptions) Validate() []error {
	errors := []error{}

	return errors
}
