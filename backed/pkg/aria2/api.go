package aria2

import (
	"changeme/backed/pkg/store"
	"fmt"
	"strings"

	"github.com/siku2/arigo"
)

// client 获取可用的 RPC 客户端，必要时自动重连
func (a *Aria2Service) client() (*arigo.Client, error) {
	if err := a.ensureConnected(); err != nil {
		return nil, err
	}
	return a.rpcClient, nil
}

// ==================== 添加下载 ====================

// AddURI 添加下载链接
func (a *Aria2Service) AddURI(uris []string, options *arigo.Options) (arigo.GID, error) {
	c, err := a.client()
	if err != nil {
		return arigo.GID{}, err
	}
	gid, err := c.AddURI(uris, options)
	if err != nil {
		return gid, err
	}
	a.recordNewDownload(gid.GID, uris, options)
	return gid, nil
}

// AddURIAtPosition 在指定队列位置添加下载链接
func (a *Aria2Service) AddURIAtPosition(uris []string, position uint, options *arigo.Options) (arigo.GID, error) {
	c, err := a.client()
	if err != nil {
		return arigo.GID{}, err
	}
	gid, err := c.AddURIAtPosition(uris, position, options)
	if err != nil {
		return gid, err
	}
	a.recordNewDownload(gid.GID, uris, options)
	return gid, nil
}

// AddTorrent 添加BT种子下载（传入.torrent文件内容）
func (a *Aria2Service) AddTorrent(torrent []byte, uris []string, options *arigo.Options) (arigo.GID, error) {
	c, err := a.client()
	if err != nil {
		return arigo.GID{}, err
	}
	gid, err := c.AddTorrent(torrent, uris, options)
	if err != nil {
		return gid, err
	}
	a.recordNewDownload(gid.GID, uris, options)
	return gid, nil
}

// AddTorrentAtPosition 在指定队列位置添加BT种子下载
func (a *Aria2Service) AddTorrentAtPosition(torrent []byte, uris []string, position uint, options *arigo.Options) (arigo.GID, error) {
	c, err := a.client()
	if err != nil {
		return arigo.GID{}, err
	}
	gid, err := c.AddTorrentAtPosition(torrent, uris, position, options)
	if err != nil {
		return gid, err
	}
	a.recordNewDownload(gid.GID, uris, options)
	return gid, nil
}

// AddMetalink 添加Metalink下载
func (a *Aria2Service) AddMetalink(metalink []byte, options *arigo.Options) ([]arigo.GID, error) {
	c, err := a.client()
	if err != nil {
		return nil, err
	}
	gids, err := c.AddMetalink(metalink, options)
	if err != nil {
		return gids, err
	}
	for _, gid := range gids {
		a.recordNewDownload(gid.GID, nil, options)
	}
	return gids, nil
}

// AddMetalinkAtPosition 在指定队列位置添加Metalink下载
func (a *Aria2Service) AddMetalinkAtPosition(metalink []byte, position uint, options *arigo.Options) ([]arigo.GID, error) {
	c, err := a.client()
	if err != nil {
		return nil, err
	}
	gids, err := c.AddMetalinkAtPosition(metalink, position, options)
	if err != nil {
		return gids, err
	}
	for _, gid := range gids {
		a.recordNewDownload(gid.GID, nil, options)
	}
	return gids, nil
}

// recordNewDownload 将新下载任务写入数据库，同时保存下载目录设置
func (a *Aria2Service) recordNewDownload(gid string, uris []string, options *arigo.Options) {
	if a.db == nil {
		return
	}
	url := ""
	if len(uris) > 0 {
		url = uris[0]
	}
	dir := ""
	filename := ""
	if options != nil {
		dir = options.Dir
		filename = options.Out
	}
	// 如果指定了下载目录，持久化保存为默认目录
	if dir != "" {
		a.db.SetSetting("default_download_dir", dir)
	}
	a.db.InsertDownload(&store.DownloadRecord{
		GID:      gid,
		URL:      url,
		Dir:      dir,
		Filename: filename,
		Status:   "active",
	})
	a.db.InsertEvent(gid, "added", "")
}

// ==================== 暂停/恢复 ====================

// PauseAll 暂停所有下载任务
func (a *Aria2Service) PauseAll() error {
	c, err := a.client()
	if err != nil {
		return err
	}
	return c.PauseAll()
}

// Pause 暂停指定任务，GID 不存在则自动同步状态后返回 nil
func (a *Aria2Service) Pause(gid string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	err = c.Pause(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		a.syncOneDownloadStatus(gid)
		return nil
	}
	return err
}

// ForcePause 强制暂停指定任务，GID 不存在则自动同步状态后返回 nil
func (a *Aria2Service) ForcePause(gid string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	err = c.ForcePause(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		a.syncOneDownloadStatus(gid)
		return nil
	}
	return err
}

// ForcePauseAll 强制暂停所有下载任务
func (a *Aria2Service) ForcePauseAll() error {
	c, err := a.client()
	if err != nil {
		return err
	}
	return c.ForcePauseAll()
}

// Unpause 恢复暂停的下载任务，GID 不存在则自动同步状态后返回 nil
func (a *Aria2Service) Unpause(gid string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	err = c.Unpause(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		a.syncOneDownloadStatus(gid)
		return nil
	}
	return err
}

// UnpauseAll 恢复所有暂停的下载任务
func (a *Aria2Service) UnpauseAll() error {
	c, err := a.client()
	if err != nil {
		return err
	}
	return c.UnpauseAll()
}

// ==================== 删除任务 ====================

// Remove 删除指定下载任务（如果正在下载会先停止），GID 不存在则直接标记为 removed
func (a *Aria2Service) Remove(gid string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	err = c.Remove(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		// GID 在 aria2 中已不存在（可能已完成/被清除/会话丢失），直接更新 SQLite
		a.syncOneDownloadStatus(gid)
		if a.db != nil {
			a.db.UpdateDownloadStatus(gid, "removed", 0, 0, 0, 0, "")
		}
		return nil
	}
	return err
}

// ForceRemove 强制删除指定下载任务，不会执行耗时操作
func (a *Aria2Service) ForceRemove(gid string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	err = c.ForceRemove(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		a.syncOneDownloadStatus(gid)
		if a.db != nil {
			a.db.UpdateDownloadStatus(gid, "removed", 0, 0, 0, 0, "")
		}
		return nil
	}
	return err
}

// Delete 删除下载任务并同时删除已下载的文件，GID 不存在则回退到仅删 SQLite 记录
func (a *Aria2Service) Delete(gid string) error {
	c, err := a.client()
	if err != nil {
		return err
	}
	err = c.Delete(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		// GID 在 aria2 中已不存在，直接从 SQLite 删除记录
		if a.db != nil {
			a.db.DeleteDownload(gid)
		}
		return nil
	}
	return err
}

// ==================== 查询状态 ====================

// TellStatus 查询指定下载任务的详细状态，可指定返回的字段
func (a *Aria2Service) TellStatus(gid string, keys ...string) (arigo.Status, error) {
	return a.rpcClient.TellStatus(gid, keys...)
}

// TellActive 查询所有正在下载的任务
func (a *Aria2Service) TellActive(keys ...string) ([]arigo.Status, error) {
	return a.rpcClient.TellActive(keys...)
}

// TellWaiting 查询等待中的下载任务，offset 为偏移量，num 为返回数量
func (a *Aria2Service) TellWaiting(offset int, num uint, keys ...string) ([]arigo.Status, error) {
	return a.rpcClient.TellWaiting(offset, num, keys...)
}

// TellStopped 查询已停止的下载任务（已完成/出错/已删除），offset 为偏移量，num 为返回数量
func (a *Aria2Service) TellStopped(offset int, num uint, keys ...string) ([]arigo.Status, error) {
	return a.rpcClient.TellStopped(offset, num, keys...)
}

// ==================== 获取URI/文件/节点/服务器 ====================

// GetURIs 获取下载任务使用的URI列表
func (a *Aria2Service) GetURIs(gid string) ([]arigo.URI, error) {
	return a.rpcClient.GetURIs(gid)
}

// GetFiles 获取下载任务的文件列表
func (a *Aria2Service) GetFiles(gid string) ([]arigo.File, error) {
	return a.rpcClient.GetFiles(gid)
}

// GetPeers 获取BT下载的节点列表（仅BT下载有效）
func (a *Aria2Service) GetPeers(gid string) ([]arigo.Peer, error) {
	return a.rpcClient.GetPeers(gid)
}

// GetServers 获取下载任务连接的服务器列表
func (a *Aria2Service) GetServers(gid string) ([]arigo.FileServers, error) {
	return a.rpcClient.GetServers(gid)
}

// ==================== 修改队列/URI ====================

// ChangePosition 修改下载任务在队列中的位置，how 为 POS_SET/POS_CUR/POS_END
func (a *Aria2Service) ChangePosition(gid string, pos int, how arigo.PositionSetBehaviour) (int, error) {
	return a.rpcClient.ChangePosition(gid, pos, how)
}

// ChangeURI 删除并添加下载任务的URI
func (a *Aria2Service) ChangeURI(gid string, fileIndex uint, delURIs []string, addURIs []string) (uint, uint, error) {
	return a.rpcClient.ChangeURI(gid, fileIndex, delURIs, addURIs)
}

// ChangeURIAt 在指定位置删除并添加下载任务的URI
func (a *Aria2Service) ChangeURIAt(gid string, fileIndex uint, delURIs []string, addURIs []string, position uint) (uint, uint, error) {
	return a.rpcClient.ChangeURIAt(gid, fileIndex, delURIs, addURIs, position)
}

// ==================== 选项管理 ====================

// GetOptions 获取指定下载任务的选项
func (a *Aria2Service) GetOptions(gid string) (arigo.Options, error) {
	return a.rpcClient.GetOptions(gid)
}

// ChangeOptions 动态修改指定下载任务的选项
func (a *Aria2Service) ChangeOptions(gid string, options arigo.Options) error {
	return a.rpcClient.ChangeOptions(gid, options)
}

// GetGlobalOptions 获取全局选项
func (a *Aria2Service) GetGlobalOptions() (arigo.Options, error) {
	return a.rpcClient.GetGlobalOptions()
}

// ChangeGlobalOptions 动态修改全局选项
func (a *Aria2Service) ChangeGlobalOptions(options arigo.Options) error {
	return a.rpcClient.ChangeGlobalOptions(options)
}

// ==================== 全局状态/版本/会话 ====================

// GetGlobalStats 获取全局下载统计信息（下载/上传速度、各状态任务数量）
func (a *Aria2Service) GetGlobalStats() (arigo.Stats, error) {
	return a.rpcClient.GetGlobalStats()
}

// GetVersion 获取aria2版本信息和启用的功能列表
func (a *Aria2Service) GetVersion() (arigo.VersionInfo, error) {
	return a.rpcClient.GetVersion()
}

// GetSessionInfo 获取当前会话ID
func (a *Aria2Service) GetSessionInfo() (arigo.SessionInfo, error) {
	return a.rpcClient.GetSessionInfo()
}

// ==================== 下载结果/会话管理 ====================

// PurgeDownloadResults 清除所有已完成/出错/已删除的下载记录
func (a *Aria2Service) PurgeDownloadResults() error {
	return a.rpcClient.PurgeDownloadResults()
}

// RemoveDownloadResult 清除指定下载任务的完成/出错/删除记录
func (a *Aria2Service) RemoveDownloadResult(gid string) error {
	return a.rpcClient.RemoveDownloadResult(gid)
}

// ==================== 关闭/保存会话 ====================

// Shutdown 优雅关闭aria2服务
func (a *Aria2Service) Shutdown() error {
	return a.rpcClient.Shutdown()
}

// ForceShutdown 强制关闭aria2服务
func (a *Aria2Service) ForceShutdown() error {
	return a.rpcClient.ForceShutdown()
}

// SaveSession 保存当前会话到文件（需要启用--save-session选项）
func (a *Aria2Service) SaveSession() error {
	return a.rpcClient.SaveSession()
}

// ==================== 事件系统 ====================

// Subscribe 订阅下载事件（start/pause/stop/complete/btcomplete/error），返回取消订阅函数
func (a *Aria2Service) Subscribe(evtType arigo.EventType, listener arigo.EventListener) arigo.UnsubscribeFunc {
	return a.rpcClient.Subscribe(evtType, listener)
}

// WaitForDownload 等待指定下载任务完成，返回nil表示成功，否则返回错误
func (a *Aria2Service) WaitForDownload(gid string) error {
	return a.rpcClient.WaitForDownload(gid)
}

// Download 添加下载链接并等待下载完成，返回最终状态
func (a *Aria2Service) Download(uris []string, options *arigo.Options) (arigo.Status, error) {
	return a.rpcClient.Download(uris, options)
}

// ==================== 批量调用/GID工厂 ====================

// MultiCall 批量执行多个RPC调用
func (a *Aria2Service) MultiCall(methods ...*arigo.MethodCall) ([]arigo.MethodResult, error) {
	return a.rpcClient.MultiCall(methods...)
}

// GetGID 创建GID包装对象，提供面向对象的操作方式
func (a *Aria2Service) GetGID(gid string) arigo.GID {
	return a.rpcClient.GetGID(gid)
}

// ==================== 继续下载（断点续传） ====================

// ContinueDownload 从 SQLite 历史记录中取出下载参数，重新提交到 aria2 继续下载
// 返回新的 GID；原 SQLite 记录在成功提交后删除
func (a *Aria2Service) ContinueDownload(gid string) (arigo.GID, error) {
	if a.db == nil {
		return arigo.GID{}, fmt.Errorf("数据库不可用")
	}
	record, err := a.db.GetDownload(gid)
	if err != nil {
		return arigo.GID{}, fmt.Errorf("未找到下载记录: %w", err)
	}
	// 调用 AddURI 重新提交（会自动写入新的 SQLite 记录）
	opts := &arigo.Options{
		Dir: record.Dir,
		Out: record.Filename,
	}
	gid2, err := a.AddURI([]string{record.URL}, opts)
	if err != nil {
		return arigo.GID{}, err
	}
	// 删除旧的 SQLite 记录（aria2 session 中的旧记录由 aria2 自行管理）
	a.db.DeleteDownload(gid)
	return gid2, nil
}
