package store

var client Factory

type Factory interface {
	Downloads() DownloadStore
	Events() EventStore
	Settings() SettingsStore
	Close() error
}

func Client() Factory { return client }

func SetClient(factory Factory) {
	client = factory
}
