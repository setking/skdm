package apiserver

import "changeme/backed/internal/apiserver/config"

func Run(cfg *config.Config) error {
	server, err := createARIA2Server(cfg)
	if err != nil {
		return err
	}
	return server.PrepareRun().Run()
}
