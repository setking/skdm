package config

import "changeme/backed/internal/apiserver/options"

type Config struct {
	*options.Options
}

func CreateConfigFromOptions(opts *options.Options) (*Config, error) {
	return &Config{
		Options: opts,
	}, nil
}
