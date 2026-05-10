package apiserver

import (
	"changeme/backed/internal/apiserver"
	"changeme/backed/internal/pkg/server"
)

func Start() *server.Config {
	return apiserver.NewApp()
}
