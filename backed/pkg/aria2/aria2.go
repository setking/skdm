package aria2

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"strconv"
	"sync"
	"time"

	"changeme/backed/pkg/store"

	"github.com/siku2/arigo"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	Aria2RPCSecret = "my-strong-secret-token-2026"
)

// statusKeys 是查询下载状态时需要的字段列表
var statusKeys = []string{
	"gid", "status", "totalLength", "completedLength",
	"downloadSpeed", "errorCode", "errorMessage",
}

//go:embed third_party/*
var binFiles embed.FS

type Aria2Service struct {
	rpcPort     string
	rpcSecret   string
	cmd         *exec.Cmd
	rpcClient   *arigo.Client
	db          *store.Store
	sessionPath string // aria2 session 文件路径（保存/恢复下载进度）
	cancel      context.CancelFunc
	mu          sync.Mutex // 保护 rpcClient 的并发访问
}

func NewAria2Service(dbPath string) *Aria2Service {
	st, err := store.Open(dbPath)
	if err != nil {
		log.Printf("[Aria2Service] 打开数据库失败（%s）: %v，将不会记录下载历史", dbPath, err)
	}
	// session 文件与 db 同目录
	sessionPath := filepath.Join(filepath.Dir(dbPath), "aria2.session")
	return &Aria2Service{
		rpcSecret:   Aria2RPCSecret,
		rpcPort:     "6800",
		db:          st,
		sessionPath: sessionPath,
	}
}

// 在应用启动时自动调用
func (a *Aria2Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	fmt.Println("[Aria2Service] 正在启动 aria2server 进程...")
	data, err := binFiles.ReadFile("third_party/aria2c.exe")
	if err != nil {
		return fmt.Errorf("读取嵌入的 aria2server 文件失败: %w", err)
	}
	aria2Path := filepath.Join(os.TempDir(), "aria2c.exe")
	if err := os.WriteFile(aria2Path, data, 0755); err != nil {
		return fmt.Errorf("写入 aria2server 到临时目录失败: %w", err)
	}
	// 确保 session 文件存在，首次启动时创建空文件
	sessionDir := filepath.Dir(a.sessionPath)
	os.MkdirAll(sessionDir, 0755)
	if _, err := os.Stat(a.sessionPath); os.IsNotExist(err) {
		os.WriteFile(a.sessionPath, []byte{}, 0644)
	}
	a.cmd = exec.Command(aria2Path,
		"--enable-rpc",
		"--rpc-listen-port="+a.rpcPort,
		"--rpc-secret="+a.rpcSecret,
		"--save-session="+a.sessionPath,
		"--save-session-interval=30",
		"--input-file="+a.sessionPath,
		"--auto-save-interval=30",
		"--allow-overwrite=true",
		"--continue=true",
	)
	if runtime.GOOS == "windows" {
		a.cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	//启动aria2和rpc服务
	if err := a.cmd.Start(); err != nil {
		return fmt.Errorf("启动 aria2server 进程失败: %w", err)
	}

	log.Printf("aria2server 已启动，PID: %d", a.cmd.Process.Pid)
	time.Sleep(2 * time.Second)

	// 创建并连接 RPC 客户端
	wsUrl := fmt.Sprintf("ws://localhost:%s/jsonrpc", a.rpcPort)
	a.rpcClient, err = arigo.Dial(wsUrl, a.rpcSecret)
	if err != nil {
		return fmt.Errorf("连接 RPC 服务失败: %w", err)
	}

	// 监控 aria2c 进程状态（进程退出时自动重启）
	go a.monitorAria2Process()

	// 同步 aria2 当前状态到 SQLite（恢复上次会话的下载记录）
	// 注意：此时前端尚未就绪，事件推送会被忽略
	a.syncAria2State()

	// 从 SQLite 加载用户设置并应用到 aria2（必须在 subscribeToEvents 之前，
	// 因为 Subscribe 会启动内部 goroutine 消费 WebSocket 消息，干扰同步 RPC 调用）
	a.loadAndApplySettings()

	// 订阅 aria2 事件，将下载生命周期记录到 SQLite 并推送到前端
	a.subscribeToEvents()

	// 启动后台定时刷新下载进度（仅推前端不写库）
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	go a.pollActiveDownloads(ctx)

	return nil
}

// 在应用关闭时自动调用
func (a *Aria2Service) ServiceShutdown() error {
	// 停止后台轮询
	if a.cancel != nil {
		a.cancel()
	}

	if a.cmd != nil && a.cmd.Process != nil {
		log.Println("[Aria2Service] 正在关闭 aria2server 进程...")
		if runtime.GOOS == "windows" {
			if err := a.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("强制终止 aria2server 进程失败: %w", err)
			}
		} else {
			if err := a.cmd.Process.Signal(os.Interrupt); err != nil {
				return fmt.Errorf("向 aria2server 发送中断信号失败: %w", err)
			}
		}
	}

	// 关闭数据库
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			log.Printf("[Aria2Service] 关闭数据库失败: %v", err)
		}
	}
	return nil
}

// ==================== 前端事件推送 ====================

// pushDownloadUpdate 将下载记录推送到前端 UI（通过 Wails Events 实时推送）
func (a *Aria2Service) pushDownloadUpdate(dr *store.DownloadRecord) {
	application.Get().Event.Emit("download-update", *dr)
}

// pushDownloadRemoved 通知前端某个 GID 已被移除
func (a *Aria2Service) pushDownloadRemoved(gid string) {
	application.Get().Event.Emit("download-removed", gid)
}

// fetchAndPush 从 SQLite 读取完整记录并推送到前端
func (a *Aria2Service) fetchAndPush(gid string) {
	if a.db == nil {
		return
	}
	dr, err := a.db.GetDownload(gid)
	if err != nil || dr == nil {
		return
	}
	a.pushDownloadUpdate(dr)
}

// ==================== 事件订阅（状态变更 → SQLite + 前端） ====================

// subscribeToEvents 订阅 aria2 下载生命周期事件
// 策略：状态变更 → 写入 SQLite（持久化） + 推送到前端（实时 UI）
func (a *Aria2Service) subscribeToEvents() {
	if a.db == nil || a.rpcClient == nil {
		return
	}

	a.rpcClient.Subscribe(arigo.StartEvent, func(ev *arigo.DownloadEvent) {
		status, err := a.rpcClient.TellStatus(ev.GID, statusKeys...)
		if err != nil {
			log.Printf("[Event] TellStatus(GID=%s) 失败: %v", ev.GID, err)
			return
		}
		a.db.UpdateDownloadStatus(ev.GID, string(status.Status),
			int64(status.CompletedLength), int64(status.TotalLength),
			int64(status.DownloadSpeed), int(status.ErrorCode), status.ErrorMessage)
		a.db.InsertEvent(ev.GID, "start", "")
		// 实时推送到前端
		a.fetchAndPush(ev.GID)
	})

	a.rpcClient.Subscribe(arigo.PauseEvent, func(ev *arigo.DownloadEvent) {
		a.db.UpdateDownloadStatus(ev.GID, "paused", 0, 0, 0, 0, "")
		a.db.InsertEvent(ev.GID, "pause", "")
		a.fetchAndPush(ev.GID)
	})

	a.rpcClient.Subscribe(arigo.CompleteEvent, func(ev *arigo.DownloadEvent) {
		status, err := a.rpcClient.TellStatus(ev.GID, statusKeys...)
		if err == nil {
			a.db.UpdateDownloadStatus(ev.GID, string(status.Status),
				int64(status.CompletedLength), int64(status.TotalLength),
				0, 0, "")
		} else {
			a.db.UpdateDownloadStatus(ev.GID, "complete", 0, 0, 0, 0, "")
		}
		a.db.InsertEvent(ev.GID, "complete", "")
		a.fetchAndPush(ev.GID)
	})

	a.rpcClient.Subscribe(arigo.ErrorEvent, func(ev *arigo.DownloadEvent) {
		status, err := a.rpcClient.TellStatus(ev.GID, statusKeys...)
		errMsg := ""
		errCode := 0
		if err == nil {
			errMsg = status.ErrorMessage
			errCode = int(status.ErrorCode)
		}
		a.db.UpdateDownloadStatus(ev.GID, "error", 0, 0, 0, errCode, errMsg)
		a.db.InsertEvent(ev.GID, "error", fmt.Sprintf(`{"code":%d,"message":"%s"}`, errCode, errMsg))
		a.fetchAndPush(ev.GID)
	})

	a.rpcClient.Subscribe(arigo.StopEvent, func(ev *arigo.DownloadEvent) {
		status, err := a.rpcClient.TellStatus(ev.GID, statusKeys...)
		if err == nil {
			a.db.UpdateDownloadStatus(ev.GID, string(status.Status),
				int64(status.CompletedLength), int64(status.TotalLength), 0,
				int(status.ErrorCode), status.ErrorMessage)
		} else {
			a.db.UpdateDownloadStatus(ev.GID, "removed", 0, 0, 0, 0, "")
		}
		a.db.InsertEvent(ev.GID, "stop", "")
		a.fetchAndPush(ev.GID)
	})
}

// ==================== 数据库查询（供 apiserver 调用） ====================

// ListDownloads 分页查询下载记录
func (a *Aria2Service) ListDownloads(status string, offset, limit int) ([]store.DownloadRecord, int, error) {
	if a.db == nil {
		return nil, 0, fmt.Errorf("数据库不可用")
	}
	return a.db.ListDownloads(status, offset, limit)
}

// GetDownload 获取单条下载记录
func (a *Aria2Service) GetDownload(gid string) (*store.DownloadRecord, error) {
	if a.db == nil {
		return nil, fmt.Errorf("数据库不可用")
	}
	return a.db.GetDownload(gid)
}

// ListEventsByGID 获取指定下载任务的事件记录
func (a *Aria2Service) ListEventsByGID(gid string) ([]store.EventRecord, error) {
	if a.db == nil {
		return nil, fmt.Errorf("数据库不可用")
	}
	return a.db.ListEventsByGID(gid)
}

// DeleteDownloadRecord 删除下载记录，并通知前端
func (a *Aria2Service) DeleteDownloadRecord(gid string) error {
	if a.db == nil {
		return fmt.Errorf("数据库不可用")
	}
	if err := a.db.DeleteDownload(gid); err != nil {
		return err
	}
	a.pushDownloadRemoved(gid)
	return nil
}

// FindDownloadByURL 根据 URL 查找已存在的下载记录（用于重复检测）
func (a *Aria2Service) FindDownloadByURL(url string) (*store.DownloadRecord, error) {
	if a.db == nil {
		return nil, fmt.Errorf("数据库不可用")
	}
	dr, err := a.db.FindDownloadByURL(url)
	if err != nil {
		return nil, nil // 查询失败或无匹配均返回 nil,nil
	}
	return dr, nil
}

// ==================== 设置管理 ====================

// GetSettings 获取当前设置（从 SQLite 读取，未设置的返回默认值）
func (a *Aria2Service) GetSettings() (*store.Settings, error) {
	if a.db == nil {
		return store.DefaultSettings(), nil
	}
	return a.db.GetSettings()
}

// SaveSettings 保存设置到 SQLite 并同步到 aria2
func (a *Aria2Service) SaveSettings(s *store.Settings) error {
	if a.db == nil {
		return fmt.Errorf("数据库不可用")
	}
	if err := a.db.SaveSettings(s); err != nil {
		return fmt.Errorf("保存设置失败: %w", err)
	}
	// 立即将可同步的选项应用到 aria2
	a.applySettingsToAria2(s)
	return nil
}

// applySettingsToAria2 将当前设置应用到 aria2 全局选项
func (a *Aria2Service) applySettingsToAria2(s *store.Settings) {
	// 使用 ensureConnected 获取可用连接并加锁，避免并发 RPC 调用冲突
	if err := a.ensureConnected(); err != nil {
		log.Printf("[Settings] 连接 aria2 失败: %v", err)
		return
	}
	limit := strconv.FormatInt(s.MaxDownloadLimit, 10)
	if s.MaxDownloadLimit <= 0 {
		limit = "0"
	}
	opts := arigo.Options{
		MaxConcurrentDownloads: uint(s.MaxConcurrentDownloads),
		MaxConnectionPerServer: uint(s.MaxConnectionPerServer),
		Split:                  uint(s.Split),
		MaxDownloadLimit:       limit,
		Continue:               s.Continue,
		AllowOverwrite:         s.AllowOverwrite,
		AutoFileRenaming:       s.AutoFileRenaming,
	}
	if err := a.rpcClient.ChangeGlobalOptions(opts); err != nil {
		log.Printf("[Settings] 应用设置到 aria2 失败: %v", err)
	} else {
		log.Printf("[Settings] 已同步设置到 aria2: maxConcurrent=%d maxConn=%d split=%d limit=%s",
			s.MaxConcurrentDownloads, s.MaxConnectionPerServer, s.Split, limit)
	}
}

// loadAndApplySettings 启动时从 SQLite 加载设置并应用到 aria2
func (a *Aria2Service) loadAndApplySettings() {
	if a.db == nil {
		return
	}
	settings, err := a.db.GetSettings()
	if err != nil {
		log.Printf("[Settings] 读取设置失败: %v", err)
		return
	}
	log.Printf("[Settings] 已加载设置: dir=%s maxConcurrent=%d maxConn=%d split=%d",
		settings.DefaultDownloadDir, settings.MaxConcurrentDownloads,
		settings.MaxConnectionPerServer, settings.Split)
	a.applySettingsToAria2(settings)
}

// syncAria2State 将 aria2 当前的所有下载任务同步到 SQLite（应用启动时调用）
// 此时前端尚未就绪，不推送事件（前端挂载时会通过 ListDownloads 获取初始状态）
func (a *Aria2Service) syncAria2State() {
	if a.db == nil || a.rpcClient == nil {
		return
	}

	// 合并来自 Active、Waiting、Stopped 的下载记录
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

	if active, err := a.rpcClient.TellActive(statusKeys...); err == nil {
		collect(active)
	}
	if waiting, err := a.rpcClient.TellWaiting(0, 1000, statusKeys...); err == nil {
		collect(waiting)
	}
	if stopped, err := a.rpcClient.TellStopped(0, 1000, statusKeys...); err == nil {
		collect(stopped)
	}

	for _, d := range downloads {
		existing, err := a.db.GetDownload(d.GID)
		if err != nil || existing == nil {
			// 新记录（可能来自 aria2 历史 session 恢复）
			a.db.InsertDownload(&store.DownloadRecord{
				GID:             d.GID,
				Status:          d.Status,
				TotalLength:     d.TotalLength,
				CompletedLength: d.CompletedLength,
				DownloadSpeed:   d.DownloadSpeed,
				ErrorCode:       d.ErrorCode,
				ErrorMessage:    d.ErrorMessage,
			})
		} else {
			// 更新已有记录的状态
			a.db.UpdateDownloadStatus(d.GID, d.Status,
				d.CompletedLength, d.TotalLength, d.DownloadSpeed,
				d.ErrorCode, d.ErrorMessage)
		}
	}
}

// ==================== 设置项 ====================

// GetDefaultDownloadDir 获取默认下载目录
func (a *Aria2Service) GetDefaultDownloadDir() (string, error) {
	if a.db == nil {
		return "./download", nil
	}
	dir, err := a.db.GetSetting("default_download_dir")
	if err != nil {
		return "./download", nil
	}
	return dir, nil
}

// ensureConnected 检查并恢复 RPC 连接
func (a *Aria2Service) ensureConnected() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.rpcClient != nil {
		// 检测连接是否存活（轻量级调用）
		_, err := a.rpcClient.GetVersion()
		if err == nil {
			return nil
		}
		log.Printf("[Aria2Service] RPC 连接已断开: %v，尝试重连...", err)
	}

	wsUrl := fmt.Sprintf("ws://localhost:%s/jsonrpc", a.rpcPort)
	client, err := arigo.Dial(wsUrl, a.rpcSecret)
	if err != nil {
		return fmt.Errorf("重连 RPC 服务失败: %w", err)
	}
	a.rpcClient = client
	// 重连后重新订阅事件
	a.subscribeToEvents()
	log.Println("[Aria2Service] RPC 重连成功")
	return nil
}

// monitorAria2Process 监控 aria2c 进程，意外退出时尝试重新启动
func (a *Aria2Service) monitorAria2Process() {
	if a.cmd == nil {
		return
	}
	err := a.cmd.Wait()
	if err != nil {
		log.Printf("[Aria2Service] aria2c 进程异常退出: %v", err)
	} else {
		log.Println("[Aria2Service] aria2c 进程正常退出")
	}

	// 尝试重新启动 aria2c
	time.Sleep(1 * time.Second)
	log.Println("[Aria2Service] 正在重新启动 aria2c...")
	aria2Path := filepath.Join(os.TempDir(), "aria2c.exe")
	// 确保 session 文件存在
	if _, err := os.Stat(a.sessionPath); os.IsNotExist(err) {
		os.WriteFile(a.sessionPath, []byte{}, 0644)
	}
	a.cmd = exec.Command(aria2Path,
		"--enable-rpc",
		"--rpc-listen-port="+a.rpcPort,
		"--rpc-secret="+a.rpcSecret,
		"--save-session="+a.sessionPath,
		"--save-session-interval=30",
		"--input-file="+a.sessionPath,
		"--auto-save-interval=30",
		"--allow-overwrite=true",
		"--continue=true",
	)
	if runtime.GOOS == "windows" {
		a.cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if err := a.cmd.Start(); err != nil {
		log.Printf("[Aria2Service] 重新启动 aria2c 失败: %v", err)
		return
	}
	log.Printf("[Aria2Service] aria2c 已重新启动，PID: %d", a.cmd.Process.Pid)
	time.Sleep(2 * time.Second)

	// 等待重连完成
	if err := a.ensureConnected(); err != nil {
		log.Printf("[Aria2Service] 重连失败: %v", err)
	}

	// 重新开始监控
	go a.monitorAria2Process()
}

// syncOneDownloadStatus 从 aria2 查询单个 GID 的真实状态并同步到 SQLite
// 用于处理前端操作（Pause/Remove）时发现 aria2 中 GID 已不在活跃列表的情况
func (a *Aria2Service) syncOneDownloadStatus(gid string) {
	if a.db == nil || a.rpcClient == nil {
		return
	}
	// 先尝试 TellStatus（覆盖 active/waiting/paused）
	status, err := a.rpcClient.TellStatus(gid, statusKeys...)
	if err == nil {
		a.db.UpdateDownloadStatus(gid, string(status.Status),
			int64(status.CompletedLength), int64(status.TotalLength),
			int64(status.DownloadSpeed), int(status.ErrorCode), status.ErrorMessage)
		a.fetchAndPush(gid)
		return
	}
	// TellStatus 失败，在 stopped 列表中查找（completed/error/removed）
	stopped, err2 := a.rpcClient.TellStopped(0, 1000, statusKeys...)
	if err2 != nil {
		return
	}
	for _, st := range stopped {
		if st.GID == gid {
			a.db.UpdateDownloadStatus(gid, string(st.Status),
				int64(st.CompletedLength), int64(st.TotalLength), 0,
				int(st.ErrorCode), st.ErrorMessage)
			a.fetchAndPush(gid)
			return
		}
	}
}

// pollActiveDownloads 定时轮询
// 策略：
//   - 每 3 秒：从 aria2 拉活跃下载的进度，直接推送到前端（不写 SQLite）
//   - 每 30 秒：全量同步到 SQLite（兜底纠错）并推送全部下载到前端
func (a *Aria2Service) pollActiveDownloads(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	fullSyncTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer fullSyncTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fullSyncTicker.C:
			// 全量同步：确保 SQLite 状态与 aria2 保持一致
			if a.db != nil && a.rpcClient != nil {
				a.syncAria2State()
				// 全量同步后推送所有记录到前端（从 SQLite 读取完整记录）
				a.pushAllDownloadsToFrontend()
				log.Printf("[Poll] 全量同步完成")
			}
		case <-ticker.C:
			if a.db == nil || a.rpcClient == nil {
				continue
			}
			active, err := a.rpcClient.TellActive(statusKeys...)
			if err != nil {
				log.Printf("[Poll] TellActive 失败: %v", err)
				continue
			}
			for _, st := range active {
				// 方案：从 SQLite 读取完整记录，用 aria2 的实时进度覆盖
				existing, err := a.db.GetDownload(st.GID)
				if err != nil || existing == nil {
					continue
				}
				existing.CompletedLength = int64(st.CompletedLength)
				existing.TotalLength = int64(st.TotalLength)
				existing.DownloadSpeed = int64(st.DownloadSpeed)
				existing.Status = string(st.Status)
				// 仅推前端，不写 SQLite
				a.pushDownloadUpdate(existing)
			}
		}
	}
}

// pushAllDownloadsToFrontend 从 SQLite 读取所有下载记录并推送到前端
// 用于全量同步兜底，确保前端状态最终一致
func (a *Aria2Service) pushAllDownloadsToFrontend() {
	if a.db == nil {
		return
	}
	records, _, err := a.db.ListDownloads("", 0, 1000)
	if err != nil {
		log.Printf("[Poll] 读取全量下载记录失败: %v", err)
		return
	}
	for i := range records {
		a.pushDownloadUpdate(&records[i])
	}
}
