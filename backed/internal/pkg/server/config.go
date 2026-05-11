package server

import (
	"fmt"
	"path/filepath"

	dv1 "changeme/backed/api/apiserver/v1"
	"changeme/backed/internal/apiserver/controller/v1/sys"

	"github.com/siku2/arigo"
	"github.com/spf13/viper"
)

const (
	Aria2RPCSecret = "my-strong-secret-token-2026"
)

type Config struct {
	RpcPort     string
	Endpoint    string
	RpcSecret   string
	SessionPath string // api session 文件路径（保存/恢复下载进度）
	runFunc     RunFunc
}
type Option func(*Config)

// RunFunc defines the application's startup callback function.
type RunFunc func() error

// WithRunFunc is used to set the application startup callback function option.
func WithRunFunc(run RunFunc) Option {
	return func(a *Config) {
		a.runFunc = run
	}
}

// NewApp creates a new application instance based on the given application name,
// binary name, and other options.
func NewApp(opts ...Option) *Config {
	a := &Config{}

	for _, o := range opts {
		o(a)
	}

	return a
}

// NewConfig returns a Config struct with the default values.
func NewConfig() *Config {
	sessionPath := filepath.Join(filepath.Dir("./app.db"), "api.session")
	return &Config{
		RpcSecret:   Aria2RPCSecret,
		RpcPort:     "6800",
		SessionPath: sessionPath,
		Endpoint:    "ws://localhost",
	}
}

type CompletedConfig struct {
	*Config
}

func (c *Config) Complete() CompletedConfig {
	return CompletedConfig{c}
}

// New returns a new instance of GenericAPIServer from the given config.
func (c CompletedConfig) New() (*GenericARIA2Server, error) {

	s := &GenericARIA2Server{
		rpcPort:     c.RpcPort,
		rpcSecret:   c.RpcSecret,
		sessionPath: c.SessionPath,
		Endpoint:    c.Endpoint,
	}
	return s, nil
}

// LoadConfig 使用 viper 读取 configs/apiserver.yaml 配置，绑定到 Config 结构体。
func LoadConfig() (*Config, error) {
	v := viper.New()
	v.SetConfigName("apiserver")
	v.SetConfigType("yaml")
	v.AddConfigPath("configs")
	v.AddConfigPath("./configs")

	v.SetDefault("aria2.rpcPort", "6800")
	v.SetDefault("aria2.rpcSecret", Aria2RPCSecret)
	v.SetDefault("aria2.sessionPath", "./app.db")
	v.SetDefault("aria2.endpoint", "ws://localhost")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	sessionPath := v.GetString("aria2.sessionPath")
	if sessionPath == "" {
		sessionPath = filepath.Join(filepath.Dir("./app.db"), "api.session")
	}

	return &Config{
		RpcPort:     v.GetString("aria2.rpcPort"),
		RpcSecret:   v.GetString("aria2.rpcSecret"),
		SessionPath: sessionPath,
	}, nil
}

// ==================== 前端绑定方法（委托给 activeServer） ====================
// 所有方法在 activeServer 为 nil 时返回零值 + error，防止初始化竞态导致 panic。

var errServerNotReady = fmt.Errorf("服务器尚未就绪，请稍后重试")

func (c *Config) AddURI(uris []string, options *arigo.Options) (arigo.GID, error) {
	if activeServer == nil {
		return arigo.GID{}, errServerNotReady
	}
	return activeServer.AddURI(uris, options)
}
func (c *Config) Pause(gid string) error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.Pause(gid)
}
func (c *Config) Unpause(gid string) error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.Unpause(gid)
}
func (c *Config) Remove(gid string) error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.Remove(gid)
}
func (c *Config) ListDownloads(status string, offset, limit int) ([]dv1.DownloadRecord, int, error) {
	if activeServer == nil {
		return nil, 0, errServerNotReady
	}
	return activeServer.ListDownloads(status, offset, limit)
}
func (c *Config) GetDefaultDownloadDir() (string, error) {
	if activeServer == nil {
		return "", errServerNotReady
	}
	return activeServer.GetDefaultDownloadDir()
}
func (c *Config) FindDownloadByURL(url string) (*dv1.DownloadRecord, error) {
	if activeServer == nil {
		return nil, errServerNotReady
	}
	return activeServer.FindDownloadByURL(url)
}
func (c *Config) DeleteDownloadRecord(gid string) error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.DeleteDownloadRecord(gid)
}
func (c *Config) OpenFileLocation(gid string) error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.OpenFileLocation(gid)
}
func (c *Config) DeleteWithLocalFile(gid string) error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.DeleteWithLocalFile(gid)
}
func (c *Config) ContinueDownload(gid string) (arigo.GID, error) {
	if activeServer == nil {
		return arigo.GID{}, errServerNotReady
	}
	return activeServer.ContinueDownload(gid)
}
func (c *Config) RemoveDownloadResult(gid string) error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.RemoveDownloadResult(gid)
}
func (c *Config) PurgeDownloadResults() error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.PurgeDownloadResults()
}
func (c *Config) GetSettings() (*dv1.Settings, error) {
	if activeServer == nil {
		return nil, errServerNotReady
	}
	return activeServer.GetSettings()
}
func (c *Config) SaveSettings(st *dv1.Settings) error {
	if activeServer == nil {
		return errServerNotReady
	}
	return activeServer.SaveSettings(st)
}
func (c *Config) GetAppVersion() string {
	if activeServer == nil {
		return ""
	}
	return activeServer.GetAppVersion()
}
func (c *Config) CheckForUpdate() *sys.UpdateCheckResult {
	if activeServer == nil {
		return &sys.UpdateCheckResult{Error: "服务器尚未就绪"}
	}
	return activeServer.CheckForUpdate()
}

// ServiceShutdown 实现 Wails v3 服务生命周期，应用退出时自动调用
func (c *Config) ServiceShutdown() error {
	if activeServer != nil {
		return activeServer.ServiceShutdown()
	}
	return nil
}
