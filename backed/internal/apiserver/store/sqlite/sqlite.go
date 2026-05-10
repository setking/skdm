package sqlite

import (
	"changeme/backed/internal/apiserver/store"
	genericoptions "changeme/backed/internal/pkg/options"
	"changeme/backed/pkg/db"
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
)

type datastore struct {
	db *sqlx.DB
}

func (ds *datastore) Downloads() store.DownloadStore {
	return newDownload(ds)
}
func (ds *datastore) Events() store.EventStore { return newEvent(ds) }
func (ds *datastore) Settings() store.SettingsStore {
	return newSettings(ds)
}

func (ds *datastore) Close() error {
	err := ds.db.Close()
	if err != nil {
		return fmt.Errorf("closing database connection: %w", err)
	}
	return nil
}

var (
	sqliteFactory store.Factory
	once           sync.Once
)

// GetMySQLFactoryOr create mysql factory with the given config.
func GetSqliteFactoryOr(opts *genericoptions.SqliteOptions) (store.Factory, error) {
	if opts == nil && sqliteFactory == nil {
		return nil, fmt.Errorf("failed to get sqlite  store factory")
	}

	var err error
	var dbIns *sqlx.DB
	once.Do(func() {
		options := &db.Options{
			Database:              opts.Database,
			MaxIdleConnections:    opts.MaxIdleConnections,
			MaxOpenConnections:    opts.MaxOpenConnections,
			MaxConnectionLifeTime: opts.MaxConnectionLifeTime,
		}
		dbIns, err = db.New(options)

		// uncomment the following line if you need auto migration the given models
		// not suggested in production environment.
		// migrateDatabase(dbIns)

		sqliteFactory = &datastore{dbIns}
	})

	if sqliteFactory == nil || err != nil {
		return nil, fmt.Errorf("failed to get sqlite  store factory, sqliteFactory: %+v, error: %w", sqliteFactory, err)
	}

	return sqliteFactory, nil
}
