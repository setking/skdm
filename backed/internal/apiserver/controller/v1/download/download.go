package download

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	dv1 "changeme/backed/api/apiserver/v1"
	ctrlv1 "changeme/backed/internal/apiserver/controller/v1"
	srvv1 "changeme/backed/internal/apiserver/service/v1"
	"changeme/backed/internal/apiserver/store"

	"github.com/siku2/arigo"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// statusKeys 是查询下载状态时需要的字段列表
var statusKeys = []string{
	"gid", "status", "totalLength", "completedLength",
	"downloadSpeed", "errorCode", "errorMessage",
}

// DownloadController 下载管理控制器
type DownloadController struct {
	srv srvv1.Service
	rpc ctrlv1.Aria2ClientProvider
}

// NewDownloadController 创建下载控制器
func NewDownloadController(store store.Factory, rpc ctrlv1.Aria2ClientProvider) *DownloadController {
	return &DownloadController{
		srv: srvv1.NewService(store),
		rpc: rpc,
	}
}

// ==================== 启动生命周期（服务器进程就绪后由外部编排调用） ====================

// SyncAria2State 将 aria2 当前的所有下载任务同步到 SQLite（应用启动时调用）
func (d *DownloadController) SyncAria2State() {
	if _, err := d.rpc.Client(); err != nil {
		return
	}

	type downloadInfo struct {
		GID             string
		Status          string
		TotalLength     int64
		CompletedLength int64
		DownloadSpeed   int64
		ErrorCode       int
		ErrorMessage    string
	}
	seen := map[string]bool{}
	var downloads []downloadInfo

	collect := func(statuses []arigo.Status) {
		for _, st := range statuses {
			if seen[st.GID] {
				continue
			}
			seen[st.GID] = true
			downloads = append(downloads, downloadInfo{
				GID:             st.GID,
				Status:          string(st.Status),
				TotalLength:     int64(st.TotalLength),
				CompletedLength: int64(st.CompletedLength),
				DownloadSpeed:   int64(st.DownloadSpeed),
				ErrorCode:       int(st.ErrorCode),
				ErrorMessage:    st.ErrorMessage,
			})
		}
	}

	if active, err := d.TellActive(statusKeys...); err == nil {
		collect(active)
	}
	if waiting, err := d.TellWaiting(0, 1000, statusKeys...); err == nil {
		collect(waiting)
	}
	if stopped, err := d.TellStopped(0, 1000, statusKeys...); err == nil {
		collect(stopped)
	}

	for _, dl := range downloads {
		existing, err := d.srv.Downloads().GetDownload(dl.GID)
		if err != nil || existing == nil {
			d.srv.Downloads().InsertDownload(context.TODO(), &dv1.DownloadRecord{
				GID:             dl.GID,
				Status:          dl.Status,
				TotalLength:     dl.TotalLength,
				CompletedLength: dl.CompletedLength,
				DownloadSpeed:   dl.DownloadSpeed,
				ErrorCode:       dl.ErrorCode,
				ErrorMessage:    dl.ErrorMessage,
			})
		} else {
			d.srv.Downloads().UpdateDownloadStatus(dl.GID, dl.Status,
				dl.CompletedLength, dl.TotalLength, dl.DownloadSpeed,
				dl.ErrorCode, dl.ErrorMessage)
		}
	}
}

// SubscribeToEvents 订阅 aria2 下载生命周期事件，写入 SQLite 并推送到前端
func (d *DownloadController) SubscribeToEvents() {
	rpcClient := d.getRPCClient()
	if rpcClient == nil {
		return
	}

	rpcClient.Subscribe(arigo.StartEvent, func(ev *arigo.DownloadEvent) {
		status, err := rpcClient.TellStatus(ev.GID, statusKeys...)
		if err != nil {
			log.Printf("[Event] TellStatus(GID=%s) 失败: %v", ev.GID, err)
			return
		}
		d.srv.Downloads().UpdateDownloadStatus(ev.GID, string(status.Status),
			int64(status.CompletedLength), int64(status.TotalLength),
			int64(status.DownloadSpeed), int(status.ErrorCode), status.ErrorMessage)
		d.srv.Events().InsertEvent(ev.GID, "start", "")
		d.fetchAndPush(ev.GID)
	})

	rpcClient.Subscribe(arigo.PauseEvent, func(ev *arigo.DownloadEvent) {
		d.srv.Downloads().UpdateDownloadStatus(ev.GID, "paused", 0, 0, 0, 0, "")
		d.srv.Events().InsertEvent(ev.GID, "pause", "")
		d.fetchAndPush(ev.GID)
	})

	rpcClient.Subscribe(arigo.CompleteEvent, func(ev *arigo.DownloadEvent) {
		status, err := rpcClient.TellStatus(ev.GID, statusKeys...)
		if err == nil {
			d.srv.Downloads().UpdateDownloadStatus(ev.GID, string(status.Status),
				int64(status.CompletedLength), int64(status.TotalLength),
				0, 0, "")
		} else {
			d.srv.Downloads().UpdateDownloadStatus(ev.GID, "complete", 0, 0, 0, 0, "")
		}
		d.srv.Events().InsertEvent(ev.GID, "complete", "")
		d.fetchAndPush(ev.GID)
	})

	rpcClient.Subscribe(arigo.ErrorEvent, func(ev *arigo.DownloadEvent) {
		status, err := rpcClient.TellStatus(ev.GID, statusKeys...)
		errMsg := ""
		errCode := 0
		if err == nil {
			errMsg = status.ErrorMessage
			errCode = int(status.ErrorCode)
		}
		d.srv.Downloads().UpdateDownloadStatus(ev.GID, "error", 0, 0, 0, errCode, errMsg)
		d.srv.Events().InsertEvent(ev.GID, "error", fmt.Sprintf(`{"code":%d,"message":"%s"}`, errCode, errMsg))
		d.fetchAndPush(ev.GID)
	})

	rpcClient.Subscribe(arigo.StopEvent, func(ev *arigo.DownloadEvent) {
		status, err := rpcClient.TellStatus(ev.GID, statusKeys...)
		if err == nil {
			d.srv.Downloads().UpdateDownloadStatus(ev.GID, string(status.Status),
				int64(status.CompletedLength), int64(status.TotalLength), 0,
				int(status.ErrorCode), status.ErrorMessage)
		} else {
			d.srv.Downloads().UpdateDownloadStatus(ev.GID, "removed", 0, 0, 0, 0, "")
		}
		d.srv.Events().InsertEvent(ev.GID, "stop", "")
		d.fetchAndPush(ev.GID)
	})
}

// PollActiveDownloads 定时轮询 aria2 进度并推送到前端（后台 goroutine）
func (d *DownloadController) PollActiveDownloads(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	fullSyncTicker := time.NewTicker(300 * time.Second)
	defer ticker.Stop()
	defer fullSyncTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fullSyncTicker.C:
			d.SyncAria2State()
			d.pushAllDownloadsToFrontend()
			log.Printf("[Poll] 全量同步完成")
		case <-ticker.C:
			active, err := d.TellActive(statusKeys...)
			if err != nil {
				log.Printf("[Poll] TellActive 失败: %v", err)
				continue
			}
			for _, st := range active {
				existing, err := d.srv.Downloads().GetDownload(st.GID)
				if err != nil || existing == nil {
					continue
				}
				existing.CompletedLength = int64(st.CompletedLength)
				existing.TotalLength = int64(st.TotalLength)
				existing.DownloadSpeed = int64(st.DownloadSpeed)
				existing.Status = string(st.Status)
				d.pushDownloadUpdate(existing)
			}
		}
	}
}

// ==================== 前端事件推送 ====================

func (d *DownloadController) pushDownloadUpdate(dr *dv1.DownloadRecord) {
	application.Get().Event.Emit("download-update", *dr)
}

func (d *DownloadController) pushDownloadRemoved(gid string) {
	application.Get().Event.Emit("download-removed", gid)
}

func (d *DownloadController) fetchAndPush(gid string) {
	dr, err := d.srv.Downloads().GetDownload(gid)
	if err != nil || dr == nil {
		return
	}
	d.pushDownloadUpdate(dr)
}

func (d *DownloadController) pushAllDownloadsToFrontend() {
	records, _, err := d.srv.Downloads().ListDownloads("", 0, 1000)
	if err != nil {
		log.Printf("[Poll] 读取全量下载记录失败: %v", err)
		return
	}
	for i := range records {
		d.pushDownloadUpdate(&records[i])
	}
}

// ==================== 添加下载 ====================

// AddURI 添加下载链接
func (d *DownloadController) AddURI(uris []string, options *arigo.Options) (arigo.GID, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return arigo.GID{}, err
	}
	gid, err := c.AddURI(uris, options)
	if err != nil {
		return gid, err
	}
	d.recordNewDownload(context.TODO(), gid.GID, uris, options)
	return gid, nil
}

// AddURIAtPosition 在指定队列位置添加下载链接
func (d *DownloadController) AddURIAtPosition(uris []string, position uint, options *arigo.Options) (arigo.GID, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return arigo.GID{}, err
	}
	gid, err := c.AddURIAtPosition(uris, position, options)
	if err != nil {
		return gid, err
	}
	d.recordNewDownload(context.TODO(), gid.GID, uris, options)
	return gid, nil
}

// AddTorrent 添加BT种子下载（传入.torrent文件内容）
func (d *DownloadController) AddTorrent(torrent []byte, uris []string, options *arigo.Options) (arigo.GID, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return arigo.GID{}, err
	}
	gid, err := c.AddTorrent(torrent, uris, options)
	if err != nil {
		return gid, err
	}
	d.recordNewDownload(context.TODO(), gid.GID, uris, options)
	return gid, nil
}

// AddTorrentAtPosition 在指定队列位置添加BT种子下载
func (d *DownloadController) AddTorrentAtPosition(torrent []byte, uris []string, position uint, options *arigo.Options) (arigo.GID, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return arigo.GID{}, err
	}
	gid, err := c.AddTorrentAtPosition(torrent, uris, position, options)
	if err != nil {
		return gid, err
	}
	d.recordNewDownload(context.TODO(), gid.GID, uris, options)
	return gid, nil
}

// AddMetalink 添加Metalink下载
func (d *DownloadController) AddMetalink(metalink []byte, options *arigo.Options) ([]arigo.GID, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	gids, err := c.AddMetalink(metalink, options)
	if err != nil {
		return gids, err
	}
	for _, gid := range gids {
		d.recordNewDownload(context.TODO(), gid.GID, nil, options)
	}
	return gids, nil
}

// AddMetalinkAtPosition 在指定队列位置添加Metalink下载
func (d *DownloadController) AddMetalinkAtPosition(metalink []byte, position uint, options *arigo.Options) ([]arigo.GID, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	gids, err := c.AddMetalinkAtPosition(metalink, position, options)
	if err != nil {
		return gids, err
	}
	for _, gid := range gids {
		d.recordNewDownload(context.TODO(), gid.GID, nil, options)
	}
	return gids, nil
}

// ==================== 暂停/恢复 ====================

// PauseAll 暂停所有下载任务
func (d *DownloadController) PauseAll() error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	return c.PauseAll()
}

// Pause 暂停指定任务，GID 不存在则自动同步状态后返回 nil
func (d *DownloadController) Pause(gid string) error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	err = c.Pause(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		d.syncOneDownloadStatus(gid)
		return nil
	}
	return err
}

// ForcePause 强制暂停指定任务
func (d *DownloadController) ForcePause(gid string) error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	err = c.ForcePause(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		d.syncOneDownloadStatus(gid)
		return nil
	}
	return err
}

// ForcePauseAll 强制暂停所有下载任务
func (d *DownloadController) ForcePauseAll() error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	return c.ForcePauseAll()
}

// Unpause 恢复暂停的下载任务
func (d *DownloadController) Unpause(gid string) error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	err = c.Unpause(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		d.syncOneDownloadStatus(gid)
		return nil
	}
	return err
}

// UnpauseAll 恢复所有暂停的下载任务
func (d *DownloadController) UnpauseAll() error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	return c.UnpauseAll()
}

// ==================== 删除任务 ====================

// Remove 删除指定下载任务
func (d *DownloadController) Remove(gid string) error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	err = c.Remove(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		d.syncOneDownloadStatus(gid)
		d.srv.Downloads().UpdateDownloadStatus(gid, "removed", 0, 0, 0, 0, "")
		d.fetchAndPush(gid)
		return nil
	}
	return err
}

// ForceRemove 强制删除指定下载任务
func (d *DownloadController) ForceRemove(gid string) error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	err = c.ForceRemove(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		d.syncOneDownloadStatus(gid)
		d.srv.Downloads().UpdateDownloadStatus(gid, "removed", 0, 0, 0, 0, "")
		d.fetchAndPush(gid)
		return nil
	}
	return err
}

// Delete 删除下载任务并同时删除已下载的文件
func (d *DownloadController) Delete(gid string) error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	err = c.Delete(gid)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return d.deleteDownloadRecord(gid)
	}
	return err
}

// ==================== 查询状态 ====================

// TellStatus 查询指定下载任务的详细状态
func (d *DownloadController) TellStatus(gid string, keys ...string) (arigo.Status, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return arigo.Status{}, err
	}
	return c.TellStatus(gid, keys...)
}

// TellActive 查询所有正在下载的任务
func (d *DownloadController) TellActive(keys ...string) ([]arigo.Status, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	return c.TellActive(keys...)
}

// TellWaiting 查询等待中的下载任务
func (d *DownloadController) TellWaiting(offset int, num uint, keys ...string) ([]arigo.Status, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	return c.TellWaiting(offset, num, keys...)
}

// TellStopped 查询已停止的下载任务
func (d *DownloadController) TellStopped(offset int, num uint, keys ...string) ([]arigo.Status, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	return c.TellStopped(offset, num, keys...)
}

// ==================== 获取URI/文件/节点/服务器 ====================

// GetURIs 获取下载任务使用的URI列表
func (d *DownloadController) GetURIs(gid string) ([]arigo.URI, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	return c.GetURIs(gid)
}

// GetFiles 获取下载任务的文件列表
func (d *DownloadController) GetFiles(gid string) ([]arigo.File, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	return c.GetFiles(gid)
}

// GetPeers 获取BT下载的节点列表
func (d *DownloadController) GetPeers(gid string) ([]arigo.Peer, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	return c.GetPeers(gid)
}

// GetServers 获取下载任务连接的服务器列表
func (d *DownloadController) GetServers(gid string) ([]arigo.FileServers, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return nil, err
	}
	return c.GetServers(gid)
}

// ==================== 修改队列/URI ====================

// ChangePosition 修改下载任务在队列中的位置
func (d *DownloadController) ChangePosition(gid string, pos int, how arigo.PositionSetBehaviour) (int, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return 0, err
	}
	return c.ChangePosition(gid, pos, how)
}

// ChangeURI 删除并添加下载任务的URI
func (d *DownloadController) ChangeURI(gid string, fileIndex uint, delURIs []string, addURIs []string) (uint, uint, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return 0, 0, err
	}
	return c.ChangeURI(gid, fileIndex, delURIs, addURIs)
}

// ChangeURIAt 在指定位置删除并添加下载任务的URI
func (d *DownloadController) ChangeURIAt(gid string, fileIndex uint, delURIs []string, addURIs []string, position uint) (uint, uint, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return 0, 0, err
	}
	return c.ChangeURIAt(gid, fileIndex, delURIs, addURIs, position)
}

// ==================== 选项管理 ====================

// GetOptions 获取指定下载任务的选项
func (d *DownloadController) GetOptions(gid string) (arigo.Options, error) {
	c, err := d.rpc.Client()
	if err != nil {
		return arigo.Options{}, err
	}
	return c.GetOptions(gid)
}

// ChangeOptions 动态修改指定下载任务的选项
func (d *DownloadController) ChangeOptions(gid string, options arigo.Options) error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	return c.ChangeOptions(gid, options)
}

// ==================== 下载结果管理 ====================

// PurgeDownloadResults 清除所有已完成/出错/已删除的下载记录
func (d *DownloadController) PurgeDownloadResults() error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	return c.PurgeDownloadResults()
}

// RemoveDownloadResult 清除指定下载任务的完成/出错/删除记录
func (d *DownloadController) RemoveDownloadResult(gid string) error {
	c, err := d.rpc.Client()
	if err != nil {
		return err
	}
	return c.RemoveDownloadResult(gid)
}

// ==================== 继续下载（断点续传） ====================

// ContinueDownload 从 SQLite 历史记录中取出下载参数，重新提交
func (d *DownloadController) ContinueDownload(gid string) (arigo.GID, error) {
	record, err := d.srv.Downloads().GetDownload(gid)
	if err != nil {
		return arigo.GID{}, fmt.Errorf("未找到下载记录: %w", err)
	}
	opts := &arigo.Options{
		Dir: record.Dir,
		Out: record.Filename,
	}
	gid2, err := d.AddURI([]string{record.URL}, opts)
	if err != nil {
		return arigo.GID{}, err
	}
	d.srv.Downloads().DeleteDownload(gid)
	return gid2, nil
}

// ==================== 数据库查询 ====================

// ListDownloads 分页查询下载记录
func (d *DownloadController) ListDownloads(status string, offset, limit int) ([]dv1.DownloadRecord, int, error) {
	return d.srv.Downloads().ListDownloads(status, offset, limit)
}

// GetDownload 获取单条下载记录
func (d *DownloadController) GetDownload(gid string) (*dv1.DownloadRecord, error) {
	return d.srv.Downloads().GetDownload(gid)
}

// FindDownloadByURL 根据 URL 查找已存在的下载记录
func (d *DownloadController) FindDownloadByURL(url string) (*dv1.DownloadRecord, error) {
	dr, err := d.srv.Downloads().FindDownloadByURL(url)
	if err != nil {
		return nil, nil
	}
	return dr, nil
}

// ==================== 内部辅助方法 ====================

// getRPCClient 获取原始的 RPC 客户端指针（不做重连检查）
func (d *DownloadController) getRPCClient() *arigo.Client {
	if p, ok := d.rpc.(interface{ RPCClient() *arigo.Client }); ok {
		return p.RPCClient()
	}
	c, _ := d.rpc.Client()
	return c
}

// recordNewDownload 将新下载任务写入数据库，同时保存下载目录设置
func (d *DownloadController) recordNewDownload(ctx context.Context, gid string, uris []string, options *arigo.Options) {
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

	if dir != "" {
		if err := d.srv.Settings().SetSetting("default_download_dir", dir); err != nil {
			log.Printf("[Download] 保存默认下载目录失败: %v", err)
		}
	}
	if err := d.srv.Downloads().InsertDownload(ctx, &dv1.DownloadRecord{
		GID:      gid,
		URL:      url,
		Dir:      dir,
		Filename: filename,
		Status:   "active",
	}); err != nil {
		log.Printf("[Download] 记录下载任务失败(GID=%s): %v", gid, err)
	}
	if err := d.srv.Events().InsertEvent(gid, "added", ""); err != nil {
		log.Printf("[Download] 记录事件失败(GID=%s): %v", gid, err)
	}
}

// deleteDownloadRecord 删除下载记录
func (d *DownloadController) deleteDownloadRecord(gid string) error {
	if err := d.srv.Downloads().DeleteDownload(gid); err != nil {
		return err
	}
	d.pushDownloadRemoved(gid)
	return nil
}

// OpenFileLocation 在文件管理器中打开下载文件所在目录并选中该文件
func (d *DownloadController) OpenFileLocation(gid string) error {
	record, err := d.srv.Downloads().GetDownload(gid)
	if err != nil || record == nil {
		return fmt.Errorf("未找到下载记录: %s", gid)
	}
	filePath := filepath.Join(record.Dir, record.Filename)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", "/select,", filePath)
	case "darwin":
		cmd = exec.Command("open", "-R", filePath)
	default:
		cmd = exec.Command("xdg-open", record.Dir)
	}
	return cmd.Start()
}

// DeleteWithLocalFile 删除数据库记录并同时删除本地下载文件
func (d *DownloadController) DeleteWithLocalFile(gid string) error {
	record, err := d.srv.Downloads().GetDownload(gid)
	if err != nil || record == nil {
		// 即使没找到记录，也尝试清理数据库
		d.deleteDownloadRecord(gid)
		return fmt.Errorf("未找到下载记录: %s", gid)
	}
	filePath := filepath.Join(record.Dir, record.Filename)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除本地文件失败: %w", err)
	}
	return d.deleteDownloadRecord(gid)
}

// syncOneDownloadStatus 从 aria2 查询单个 GID 的真实状态并同步到 SQLite
func (d *DownloadController) syncOneDownloadStatus(gid string) {
	status, err := d.TellStatus(gid, statusKeys...)
	if err == nil {
		d.srv.Downloads().UpdateDownloadStatus(gid, string(status.Status),
			int64(status.CompletedLength), int64(status.TotalLength),
			int64(status.DownloadSpeed), int(status.ErrorCode), status.ErrorMessage)
		d.fetchAndPush(gid)
		return
	}
	stopped, err2 := d.TellStopped(0, 1000, statusKeys...)
	if err2 != nil {
		return
	}
	for _, st := range stopped {
		if st.GID == gid {
			d.srv.Downloads().UpdateDownloadStatus(gid, string(st.Status),
				int64(st.CompletedLength), int64(st.TotalLength), 0,
				int(st.ErrorCode), st.ErrorMessage)
			d.fetchAndPush(gid)
			return
		}
	}
}
