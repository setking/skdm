package apiserver

import (
	"fmt"

	"changeme/backed/internal/apiserver/config"
	"changeme/backed/internal/apiserver/options"
	"changeme/backed/internal/pkg/server"
)

// NewApp 创建并初始化完整的服务器实例（SQLite + GenericARIA2Server + Controllers）。
// 所有初始化在返回前同步完成，确保 Wails 前端绑定可用时一切就绪。
func NewApp() *server.Config {
	opts := options.NewOptions()
	cfg, err := config.CreateConfigFromOptions(opts)
	if err != nil {
		panic(err)
	}

	// 初始化 SQLite
	extraCfg := &ExtraConfig{SqliteOptions: cfg.SqliteOptions}
	if err := extraCfg.complete().New(); err != nil {
		panic(fmt.Sprintf("初始化 SQLite 数据库失败: %v", err))
	}

	// 创建服务器并注入 Controller
	apiSrv, err := createARIA2Server(cfg)
	if err != nil {
		panic(fmt.Sprintf("创建 ARIA2 服务器失败: %v", err))
	}
	prepared := apiSrv.PrepareRun()

	// 启动 aria2c 进程 + 状态同步 + 事件订阅 + 后台轮询
	if err := prepared.Run(); err != nil {
		panic(fmt.Sprintf("启动 aria2c 服务失败: %v", err))
	}

	return server.NewApp()
}
