package options

import (
	"changeme/backed/internal/pkg/server"
)

// ServerRunOptions contains the options while running a generic api server.
type ServerRunOptions struct {
	RpcPort     string `json:"rpc_port"        mapstructure:"rpc_port"`
	RpcSecret   string `json:"rpc_secret"     mapstructure:"rpc_secret"`
	SessionPath string `json:"session_path" mapstructure:"session_path"`
	Endpoint    string `json:"endpoint" mapstructure:"endpoint"`
}

// NewServerRunOptions creates a new ServerRunOptions object with default parameters.
func NewServerRunOptions() *ServerRunOptions {
	defaults := server.NewConfig()

	return &ServerRunOptions{
		RpcPort:     defaults.RpcPort,
		RpcSecret:   defaults.RpcSecret,
		SessionPath: defaults.SessionPath,
		Endpoint:    defaults.Endpoint,
	}
}

// ApplyTo applies the run options to the method receiver and returns self.
func (s *ServerRunOptions) ApplyTo(c *server.Config) error {
	c.RpcPort = s.RpcPort
	c.RpcSecret = s.RpcSecret
	c.SessionPath = s.SessionPath
	c.Endpoint = s.Endpoint

	return nil
}

// Validate checks validation of ServerRunOptions.
func (s *ServerRunOptions) Validate() []error {
	errors := []error{}

	return errors
}
