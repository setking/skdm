package event

import (
	dv1 "changeme/backed/api/apiserver/v1"
	ctrlv1 "changeme/backed/internal/apiserver/controller/v1"
	"changeme/backed/internal/apiserver/store"
	srvv1 "changeme/backed/internal/apiserver/service/v1"
)

// EventController 事件管理控制器
type EventController struct {
	srv srvv1.Service
	rpc ctrlv1.Aria2ClientProvider
}

// NewEventController 创建事件控制器
func NewEventController(store store.Factory, rpc ctrlv1.Aria2ClientProvider) *EventController {
	return &EventController{
		srv: srvv1.NewService(store),
		rpc: rpc,
	}
}

// ListEventsByGID 获取指定下载任务的事件记录
func (e *EventController) ListEventsByGID(gid string) ([]dv1.EventRecord, error) {
	return e.srv.Events().ListEventsByGID(gid)
}
