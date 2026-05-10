package apiserver

import (
	"context"

	"changeme/backed/pkg/aria2"
	"changeme/backed/pkg/store"

	"github.com/siku2/arigo"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Aria2Service struct {
	svc *aria2.Aria2Service
}

func NewAria2Service(dbPath string) *Aria2Service {
	return &Aria2Service{
		svc: aria2.NewAria2Service(dbPath),
	}
}

func (a *Aria2Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return a.svc.ServiceStartup(ctx, options)
}

func (a *Aria2Service) ServiceShutdown() error {
	return a.svc.ServiceShutdown()
}

// ==================== 添加下载 ====================

func (a *Aria2Service) AddURI(uris []string, options *arigo.Options) (arigo.GID, error) {
	return a.svc.AddURI(uris, options)
}

func (a *Aria2Service) AddURIAtPosition(uris []string, position uint, options *arigo.Options) (arigo.GID, error) {
	return a.svc.AddURIAtPosition(uris, position, options)
}

func (a *Aria2Service) AddTorrent(torrent []byte, uris []string, options *arigo.Options) (arigo.GID, error) {
	return a.svc.AddTorrent(torrent, uris, options)
}

func (a *Aria2Service) AddTorrentAtPosition(torrent []byte, uris []string, position uint, options *arigo.Options) (arigo.GID, error) {
	return a.svc.AddTorrentAtPosition(torrent, uris, position, options)
}

func (a *Aria2Service) AddMetalink(metalink []byte, options *arigo.Options) ([]arigo.GID, error) {
	return a.svc.AddMetalink(metalink, options)
}

func (a *Aria2Service) AddMetalinkAtPosition(metalink []byte, position uint, options *arigo.Options) ([]arigo.GID, error) {
	return a.svc.AddMetalinkAtPosition(metalink, position, options)
}

// ==================== 暂停/恢复 ====================

func (a *Aria2Service) PauseAll() error {
	return a.svc.PauseAll()
}

func (a *Aria2Service) Pause(gid string) error {
	return a.svc.Pause(gid)
}

func (a *Aria2Service) ForcePause(gid string) error {
	return a.svc.ForcePause(gid)
}

func (a *Aria2Service) ForcePauseAll() error {
	return a.svc.ForcePauseAll()
}

func (a *Aria2Service) Unpause(gid string) error {
	return a.svc.Unpause(gid)
}

func (a *Aria2Service) UnpauseAll() error {
	return a.svc.UnpauseAll()
}

// ==================== 删除任务 ====================

func (a *Aria2Service) Remove(gid string) error {
	return a.svc.Remove(gid)
}

func (a *Aria2Service) ForceRemove(gid string) error {
	return a.svc.ForceRemove(gid)
}

func (a *Aria2Service) Delete(gid string) error {
	return a.svc.Delete(gid)
}

// ==================== 查询状态 ====================

func (a *Aria2Service) TellStatus(gid string, keys ...string) (arigo.Status, error) {
	return a.svc.TellStatus(gid, keys...)
}

func (a *Aria2Service) TellActive(keys ...string) ([]arigo.Status, error) {
	return a.svc.TellActive(keys...)
}

func (a *Aria2Service) TellWaiting(offset int, num uint, keys ...string) ([]arigo.Status, error) {
	return a.svc.TellWaiting(offset, num, keys...)
}

func (a *Aria2Service) TellStopped(offset int, num uint, keys ...string) ([]arigo.Status, error) {
	return a.svc.TellStopped(offset, num, keys...)
}

// ==================== 获取URI/文件/节点/服务器 ====================

func (a *Aria2Service) GetURIs(gid string) ([]arigo.URI, error) {
	return a.svc.GetURIs(gid)
}

func (a *Aria2Service) GetFiles(gid string) ([]arigo.File, error) {
	return a.svc.GetFiles(gid)
}

func (a *Aria2Service) GetPeers(gid string) ([]arigo.Peer, error) {
	return a.svc.GetPeers(gid)
}

func (a *Aria2Service) GetServers(gid string) ([]arigo.FileServers, error) {
	return a.svc.GetServers(gid)
}

// ==================== 修改队列/URI ====================

func (a *Aria2Service) ChangePosition(gid string, pos int, how arigo.PositionSetBehaviour) (int, error) {
	return a.svc.ChangePosition(gid, pos, how)
}

func (a *Aria2Service) ChangeURI(gid string, fileIndex uint, delURIs []string, addURIs []string) (uint, uint, error) {
	return a.svc.ChangeURI(gid, fileIndex, delURIs, addURIs)
}

func (a *Aria2Service) ChangeURIAt(gid string, fileIndex uint, delURIs []string, addURIs []string, position uint) (uint, uint, error) {
	return a.svc.ChangeURIAt(gid, fileIndex, delURIs, addURIs, position)
}

// ==================== 选项管理 ====================

func (a *Aria2Service) GetOptions(gid string) (arigo.Options, error) {
	return a.svc.GetOptions(gid)
}

func (a *Aria2Service) ChangeOptions(gid string, options arigo.Options) error {
	return a.svc.ChangeOptions(gid, options)
}

func (a *Aria2Service) GetGlobalOptions() (arigo.Options, error) {
	return a.svc.GetGlobalOptions()
}

func (a *Aria2Service) ChangeGlobalOptions(options arigo.Options) error {
	return a.svc.ChangeGlobalOptions(options)
}

// ==================== 全局状态/版本/会话 ====================

func (a *Aria2Service) GetGlobalStats() (arigo.Stats, error) {
	return a.svc.GetGlobalStats()
}

func (a *Aria2Service) GetVersion() (arigo.VersionInfo, error) {
	return a.svc.GetVersion()
}

func (a *Aria2Service) GetSessionInfo() (arigo.SessionInfo, error) {
	return a.svc.GetSessionInfo()
}

// ==================== 下载结果/会话管理 ====================

func (a *Aria2Service) PurgeDownloadResults() error {
	return a.svc.PurgeDownloadResults()
}

func (a *Aria2Service) RemoveDownloadResult(gid string) error {
	return a.svc.RemoveDownloadResult(gid)
}

// ==================== 关闭/保存会话 ====================

func (a *Aria2Service) Shutdown() error {
	return a.svc.Shutdown()
}

func (a *Aria2Service) ForceShutdown() error {
	return a.svc.ForceShutdown()
}

func (a *Aria2Service) SaveSession() error {
	return a.svc.SaveSession()
}

// ==================== 事件系统 ====================

func (a *Aria2Service) Subscribe(evtType arigo.EventType, listener arigo.EventListener) arigo.UnsubscribeFunc {
	return a.svc.Subscribe(evtType, listener)
}

func (a *Aria2Service) WaitForDownload(gid string) error {
	return a.svc.WaitForDownload(gid)
}

func (a *Aria2Service) Download(uris []string, options *arigo.Options) (arigo.Status, error) {
	return a.svc.Download(uris, options)
}

// ==================== 批量调用/GID工厂 ====================

func (a *Aria2Service) MultiCall(methods ...*arigo.MethodCall) ([]arigo.MethodResult, error) {
	return a.svc.MultiCall(methods...)
}

func (a *Aria2Service) GetGID(gid string) arigo.GID {
	return a.svc.GetGID(gid)
}

// ==================== 继续下载 ====================

// ContinueDownload 从历史记录继续未完成的下载
func (a *Aria2Service) ContinueDownload(gid string) (arigo.GID, error) {
	return a.svc.ContinueDownload(gid)
}

// ==================== 数据库查询 ====================

// ListDownloads 分页查询下载记录，status为空时返回全部
func (a *Aria2Service) ListDownloads(status string, offset, limit int) ([]store.DownloadRecord, int, error) {
	return a.svc.ListDownloads(status, offset, limit)
}

// GetDownload 获取单条下载记录
func (a *Aria2Service) GetDownload(gid string) (*store.DownloadRecord, error) {
	return a.svc.GetDownload(gid)
}

// ListEventsByGID 获取指定下载任务的事件记录
func (a *Aria2Service) ListEventsByGID(gid string) ([]store.EventRecord, error) {
	return a.svc.ListEventsByGID(gid)
}

// DeleteDownloadRecord 删除下载记录
func (a *Aria2Service) DeleteDownloadRecord(gid string) error {
	return a.svc.DeleteDownloadRecord(gid)
}

// FindDownloadByURL 根据 URL 查找已存在的下载记录（用于重复检测，返回 nil 表示无重复）
func (a *Aria2Service) FindDownloadByURL(url string) (*store.DownloadRecord, error) {
	return a.svc.FindDownloadByURL(url)
}

// GetDefaultDownloadDir 获取上次使用的下载目录（从 SQLite 读取）
func (a *Aria2Service) GetDefaultDownloadDir() (string, error) {
	return a.svc.GetDefaultDownloadDir()
}

// ==================== 设置管理 ====================

// GetSettings 获取用户设置（从 SQLite 读取，未设置的返回默认值）
func (a *Aria2Service) GetSettings() (*store.Settings, error) {
	return a.svc.GetSettings()
}

// SaveSettings 保存用户设置到 SQLite 并同步到 aria2
func (a *Aria2Service) SaveSettings(s *store.Settings) error {
	return a.svc.SaveSettings(s)
}

// ==================== 版本与更新 ====================

// GetAppVersion 返回应用版本号
func (a *Aria2Service) GetAppVersion() string {
	return a.svc.GetAppVersion()
}

// CheckForUpdate 检查 GitHub Release 是否有新版本
func (a *Aria2Service) CheckForUpdate() *aria2.UpdateCheckResult {
	return a.svc.CheckForUpdate()
}
