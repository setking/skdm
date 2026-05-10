package apiserver

import (
	"context"

	"changeme/backed/internal/apiserver/config"
	"changeme/backed/internal/apiserver/store"
	"changeme/backed/internal/apiserver/store/sqlite"
	genericoptions "changeme/backed/internal/pkg/options"
	genericapiserver "changeme/backed/internal/pkg/server"
)

type apiServer struct {
	genericARIA2Server *genericapiserver.GenericARIA2Server
	controllers        *Controllers
}

type preparedAPIServer struct {
	*apiServer
}

// ExtraConfig defines extra configuration for the iam-apiserver.
type ExtraConfig struct {
	SqliteOptions *genericoptions.SqliteOptions
}

func createARIA2Server(cfg *config.Config) (*apiServer, error) {
	genericConfig, err := buildGenericConfig(cfg)
	if err != nil {
		return nil, err
	}
	genericServer, err := genericConfig.Complete().New()
	if err != nil {
		return nil, err
	}
	server := &apiServer{
		genericARIA2Server: genericServer,
	}

	return server, nil
}

func (s *apiServer) PrepareRun() preparedAPIServer {
	// 创建所有 Controller（注入 RPC 提供者和数据库）
	s.controllers = initRouter(s.genericARIA2Server)

	// 将 Controller 注入到 GenericARIA2Server，使其前端绑定方法可以委托到 Controller
	s.genericARIA2Server.SetControllers(
		s.controllers.Download,
		s.controllers.Settings,
		s.controllers.Event,
		s.controllers.Sys,
	)

	// 注册为活跃服务器实例，使 Config 的前端绑定方法可以在 runFunc 返回前正常响应
	genericapiserver.SetActiveServer(s.genericARIA2Server)

	return preparedAPIServer{s}
}

func (s preparedAPIServer) Run() error {
	// 1. 启动 aria2c 进程并建立 RPC 连接
	if err := s.genericARIA2Server.ServiceStartup(); err != nil {
		return err
	}

	ctrl := s.controllers

	// 2. 同步 aria2 当前状态到 SQLite
	ctrl.Download.SyncAria2State()

	// 3. 从 SQLite 加载设置并应用到 aria2
	ctrl.Settings.LoadAndApplySettings()

	// 4. 如果用户关闭了自动开始，暂停所有任务
	ctrl.Settings.PauseAllIfAutoStartDisabled()

	// 5. 订阅 aria2 事件（写入 SQLite + 推送前端）
	ctrl.Download.SubscribeToEvents()

	// 6. 启动后台轮询（进度推送 + 全量同步）
	ctx, cancel := context.WithCancel(context.Background())
	s.genericARIA2Server.SetCancel(cancel)
	go ctrl.Download.PollActiveDownloads(ctx)

	// 7. 启动时自动检查更新（异步执行，静默将结果推送到前端）
	go ctrl.Sys.CheckForUpdateOnStartup()

	return nil
}

type completedExtraConfig struct {
	*ExtraConfig
}

// Complete fills in any fields not set that are required to have valid data and can be derived from other fields.
func (c *ExtraConfig) complete() *completedExtraConfig {
	if c.SqliteOptions.Database == "" {
		c.SqliteOptions.Database = "./app.db"
	}

	return &completedExtraConfig{c}
}

// New create a grpcAPIServer instance.
func (c *completedExtraConfig) New() error {
	storeIns, err := sqlite.GetSqliteFactoryOr(c.SqliteOptions)
	if err != nil {
		return err
	}
	store.SetClient(storeIns)

	return nil
}

func buildGenericConfig(cfg *config.Config) (genericConfig *genericapiserver.Config, lastErr error) {
	genericConfig = genericapiserver.NewConfig()
	if lastErr = cfg.GenericServerRunOptions.ApplyTo(genericConfig); lastErr != nil {
		return
	}
	return
}
