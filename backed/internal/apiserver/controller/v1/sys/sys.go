package sys

import (
	"log"

	ctrlv1 "changeme/backed/internal/apiserver/controller/v1"
	"changeme/backed/internal/apiserver/store"
	srvv1 "changeme/backed/internal/apiserver/service/v1"

	"github.com/siku2/arigo"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SysController 系统管理控制器
type SysController struct {
	srv srvv1.Service
	rpc ctrlv1.Aria2ClientProvider
}

// NewSysController 创建系统控制器
func NewSysController(store store.Factory, rpc ctrlv1.Aria2ClientProvider) *SysController {
	return &SysController{
		srv: srvv1.NewService(store),
		rpc: rpc,
	}
}

// GetGlobalStats 获取全局下载统计信息
func (s *SysController) GetGlobalStats() (arigo.Stats, error) {
	c, err := s.rpc.Client()
	if err != nil {
		return arigo.Stats{}, err
	}
	return c.GetGlobalStats()
}

// GetVersion 获取aria2版本信息
func (s *SysController) GetVersion() (arigo.VersionInfo, error) {
	c, err := s.rpc.Client()
	if err != nil {
		return arigo.VersionInfo{}, err
	}
	return c.GetVersion()
}

// GetSessionInfo 获取当前会话ID
func (s *SysController) GetSessionInfo() (arigo.SessionInfo, error) {
	c, err := s.rpc.Client()
	if err != nil {
		return arigo.SessionInfo{}, err
	}
	return c.GetSessionInfo()
}

// Shutdown 优雅关闭aria2服务
func (s *SysController) Shutdown() error {
	c, err := s.rpc.Client()
	if err != nil {
		return err
	}
	return c.Shutdown()
}

// ForceShutdown 强制关闭aria2服务
func (s *SysController) ForceShutdown() error {
	c, err := s.rpc.Client()
	if err != nil {
		return err
	}
	return c.ForceShutdown()
}

// SaveSession 保存当前会话到文件
func (s *SysController) SaveSession() error {
	c, err := s.rpc.Client()
	if err != nil {
		return err
	}
	return c.SaveSession()
}

// Subscribe 订阅下载事件
func (s *SysController) Subscribe(evtType arigo.EventType, listener arigo.EventListener) arigo.UnsubscribeFunc {
	c, _ := s.rpc.Client()
	if c == nil {
		return func() bool { return false }
	}
	return c.Subscribe(evtType, listener)
}

// WaitForDownload 等待指定下载任务完成
func (s *SysController) WaitForDownload(gid string) error {
	c, err := s.rpc.Client()
	if err != nil {
		return err
	}
	return c.WaitForDownload(gid)
}

// Download 添加下载链接并等待下载完成
func (s *SysController) Download(uris []string, options *arigo.Options) (arigo.Status, error) {
	c, err := s.rpc.Client()
	if err != nil {
		return arigo.Status{}, err
	}
	return c.Download(uris, options)
}

// MultiCall 批量执行多个RPC调用
func (s *SysController) MultiCall(methods ...*arigo.MethodCall) ([]arigo.MethodResult, error) {
	c, err := s.rpc.Client()
	if err != nil {
		return nil, err
	}
	return c.MultiCall(methods...)
}

// GetAppVersion 返回当前应用版本号
func (s *SysController) GetAppVersion() string {
	return GetAppVersion()
}

// CheckForUpdate 检查 GitHub Release 是否有新版本（供前端手动调用）
func (s *SysController) CheckForUpdate() *UpdateCheckResult {
	return CheckForUpdate()
}

// CheckForUpdateOnStartup 启动时自动检查更新，结果通过 "update-check" 事件推送到前端。
// 仅在启动时调用一次，静默处理错误（不弹窗打扰用户）。
func (s *SysController) CheckForUpdateOnStartup() {
	result := CheckForUpdate()
	if result.Error != "" {
		log.Printf("[Update] 启动时检查更新失败: %s", result.Error)
	} else if result.HasUpdate {
		log.Printf("[Update] 发现新版本 v%s → v%s", result.CurrentVersion, result.LatestVersion)
	}
	application.Get().Event.Emit("update-check", *result)
}

// GetGID 创建GID包装对象
func (s *SysController) GetGID(gid string) arigo.GID {
	c, err := s.rpc.Client()
	if err != nil {
		return arigo.GID{}
	}
	return c.GetGID(gid)
}
