package apiserver

import (
	ctrlv1 "changeme/backed/internal/apiserver/controller/v1"
	"changeme/backed/internal/apiserver/controller/v1/download"
	"changeme/backed/internal/apiserver/controller/v1/event"
	"changeme/backed/internal/apiserver/controller/v1/settings"
	"changeme/backed/internal/apiserver/controller/v1/sys"
	"changeme/backed/internal/apiserver/store/sqlite"
)

// Controllers 持有所有 Controller 实例，供服务器编排启动流程
type Controllers struct {
	Download *download.DownloadController
	Event    *event.EventController
	Settings *settings.SettingsController
	Sys      *sys.SysController
}

func initRouter(rpc ctrlv1.Aria2ClientProvider) *Controllers {
	storeIns, _ := sqlite.GetSqliteFactoryOr(nil)
	return &Controllers{
		Download: download.NewDownloadController(storeIns, rpc),
		Event:    event.NewEventController(storeIns, rpc),
		Settings: settings.NewSettingsController(storeIns, rpc),
		Sys:      sys.NewSysController(storeIns, rpc),
	}
}
