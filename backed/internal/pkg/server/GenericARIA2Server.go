package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	dv1 "changeme/backed/api/apiserver/v1"
	"changeme/backed/internal/apiserver/controller/v1/download"
	"changeme/backed/internal/apiserver/controller/v1/event"
	"changeme/backed/internal/apiserver/controller/v1/settings"
	"changeme/backed/internal/apiserver/controller/v1/sys"
	"changeme/backed/pkg/db"
	thirdparty "changeme/backed/third_party"

	"github.com/siku2/arigo"
)

// activeServer is the currently running GenericARIA2Server instance.
// Set by SetActiveServer (called from PrepareRun), used by Config's frontend delegation methods.
var activeServer *GenericARIA2Server

// SetActiveServer sets the active server instance for Config's frontend delegation methods.
// Called by PrepareRun after controllers are wired, before the frontend can make any calls.
func SetActiveServer(s *GenericARIA2Server) {
	activeServer = s
}

// GenericARIA2Server 负责 aria2c 进程生命周期管理和 RPC 连接维护
// 同时作为 Wails 前端绑定入口，所有导出方法委托给 Controller 层处理
type GenericARIA2Server struct {
	rpcPort     string
	Endpoint    string
	rpcSecret   string
	cmd         *exec.Cmd
	rpcClient   *arigo.Client
	sessionPath string
	cancel      context.CancelFunc
	mu          sync.Mutex

	// Controller 引用（运行时注入，供前端绑定方法委托）
	downloadCtrl *download.DownloadController
	settingsCtrl *settings.SettingsController
	eventCtrl    *event.EventController
	sysCtrl      *sys.SysController
}

// SetControllers 注入 Controller 实例（由 apiserver.PrepareRun 调用）
func (g *GenericARIA2Server) SetControllers(
	dl *download.DownloadController,
	st *settings.SettingsController,
	ev *event.EventController,
	sy *sys.SysController,
) {
	g.downloadCtrl = dl
	g.settingsCtrl = st
	g.eventCtrl = ev
	g.sysCtrl = sy
}

// ServiceStartup 启动 aria2c 进程并建立 RPC 连接
func (g *GenericARIA2Server) ServiceStartup() error {
	activeServer = g

	// 确保 session 文件目录可写；若不可写（如安装到系统目录），回退到用户数据目录
	g.sessionPath = ensureSessionWritable(g.sessionPath)

	aria2cPath, err := thirdparty.ReadAndWriteForAria2c()
	if err != nil {
		return fmt.Errorf("准备 aria2c 可执行文件失败: %s\n", err.Error())
	}
	sessionDir := filepath.Dir(g.sessionPath)
	os.MkdirAll(sessionDir, 0o755)
	if _, err := os.Stat(g.sessionPath); os.IsNotExist(err) {
		os.WriteFile(g.sessionPath, []byte{}, 0o644)
	}
	g.cmd = exec.Command(aria2cPath,
		"--enable-rpc",
		"--rpc-listen-port="+g.rpcPort,
		"--rpc-secret="+g.rpcSecret,
		"--save-session="+g.sessionPath,
		"--save-session-interval=30",
		"--input-file="+g.sessionPath,
		"--auto-save-interval=30",
		"--allow-overwrite=true",
		"--continue=true",
	)
	if runtime.GOOS == "windows" {
		g.cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if err := g.cmd.Start(); err != nil {
		return fmt.Errorf("启动 aria2server 进程失败: %w", err)
	}

	log.Printf("aria2server 已启动，PID: %d", g.cmd.Process.Pid)
	time.Sleep(2 * time.Second)

	wsUrl := fmt.Sprintf("%s:%s/jsonrpc", g.Endpoint, g.rpcPort)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	g.rpcClient, err = arigo.DialContext(ctx, wsUrl, g.rpcSecret)
	if err != nil {
		return fmt.Errorf("连接 RPC 服务失败: %w", err)
	}

	go g.monitorAria2Process()
	return nil
}

// ensureSessionWritable 检测 sessionPath 所在目录是否可写。
// 若不可写（如安装到系统保护目录），回退到用户数据目录下的 api.session。
func ensureSessionWritable(sessionPath string) string {
	dir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fallbackSessionPath(sessionPath)
	}
	testFile := filepath.Join(dir, ".skdm_write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return fallbackSessionPath(sessionPath)
	}
	f.Close()
	os.Remove(testFile)
	return sessionPath
}

func fallbackSessionPath(originalPath string) string {
	fallback := filepath.Join(db.UserDataDir(), "api.session")
	log.Printf("[Aria2Service] session 目录不可写 (%s)，回退到 %s", filepath.Dir(originalPath), fallback)
	return fallback
}

// ServiceShutdown 在应用关闭时自动调用，终止 aria2c 进程
func (g *GenericARIA2Server) ServiceShutdown() error {
	if g.cancel != nil {
		g.cancel()
	}

	if g.cmd != nil && g.cmd.Process != nil {
		log.Println("[Aria2Service] 正在关闭 aria2server 进程...")
		if runtime.GOOS == "windows" {
			if err := g.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("强制终止 aria2server 进程失败: %w", err)
			}
		} else {
			if err := g.cmd.Process.Signal(os.Interrupt); err != nil {
				return fmt.Errorf("向 aria2server 发送中断信号失败: %w", err)
			}
		}
	}
	return nil
}

// ==================== RPC 客户端访问 ====================

// Client 返回可用的 RPC 客户端，必要时自动重连
func (g *GenericARIA2Server) Client() (*arigo.Client, error) {
	if err := g.ensureConnected(); err != nil {
		return nil, err
	}
	return g.rpcClient, nil
}

// RPCClient 返回原始的 RPC 客户端指针（不做重连检查）
func (g *GenericARIA2Server) RPCClient() *arigo.Client {
	return g.rpcClient
}

func (g *GenericARIA2Server) ensureConnected() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Liveness check: 用 goroutine + timeout 防止 GetVersion 在断开的 WebSocket 上无限阻塞
	if g.rpcClient != nil {
		alive := make(chan error, 1)
		go func() {
			_, err := g.rpcClient.GetVersion()
			alive <- err
		}()
		select {
		case err := <-alive:
			if err == nil {
				return nil
			}
			log.Printf("[Aria2Service] RPC 连接已断开: %v，尝试重连...", err)
		case <-time.After(5 * time.Second):
			log.Printf("[Aria2Service] RPC 连接探测超时，强制重连...")
		}
	}

	wsUrl := fmt.Sprintf("%s:%s/jsonrpc", g.Endpoint, g.rpcPort)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := arigo.DialContext(ctx, wsUrl, g.rpcSecret)
	if err != nil {
		return fmt.Errorf("重连 RPC 服务失败: %w", err)
	}
	g.rpcClient = client
	log.Println("[Aria2Service] RPC 重连成功")
	return nil
}

// ==================== 进程监控 ====================

func (g *GenericARIA2Server) monitorAria2Process() {
	if g.cmd == nil {
		return
	}
	err := g.cmd.Wait()
	if err != nil {
		log.Printf("[Aria2Service] aria2c 进程异常退出: %v", err)
	} else {
		log.Println("[Aria2Service] aria2c 进程正常退出")
	}

	time.Sleep(1 * time.Second)
	log.Println("[Aria2Service] 正在重新启动 aria2c...")

	aria2cPath, pathErr := thirdparty.ReadAndWriteForAria2c()
	if pathErr != nil {
		log.Printf("[Aria2Service] 准备 aria2c 可执行文件失败: %v", pathErr)
		return
	}
	if _, err := os.Stat(g.sessionPath); os.IsNotExist(err) {
		os.WriteFile(g.sessionPath, []byte{}, 0o644)
	}
	g.cmd = exec.Command(aria2cPath,
		"--enable-rpc",
		"--rpc-listen-port="+g.rpcPort,
		"--rpc-secret="+g.rpcSecret,
		"--save-session="+g.sessionPath,
		"--save-session-interval=30",
		"--input-file="+g.sessionPath,
		"--auto-save-interval=30",
		"--allow-overwrite=true",
		"--continue=true",
	)
	if runtime.GOOS == "windows" {
		g.cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
	if err := g.cmd.Start(); err != nil {
		log.Printf("[Aria2Service] 重新启动 aria2c 失败: %v", err)
		return
	}
	log.Printf("[Aria2Service] aria2c 已重新启动，PID: %d", g.cmd.Process.Pid)
	time.Sleep(2 * time.Second)

	if err := g.ensureConnected(); err != nil {
		log.Printf("[Aria2Service] 重连失败: %v", err)
	}

	go g.monitorAria2Process()
}

// SetCancel 设置 cancel 函数（供外部 Controller 层控制后台 goroutine）
func (g *GenericARIA2Server) SetCancel(cancel context.CancelFunc) {
	g.cancel = cancel
}

// ==================== 前端绑定方法（委托给 Controller） ====================

// AddURI 添加下载链接
func (g *GenericARIA2Server) AddURI(uris []string, options *arigo.Options) (arigo.GID, error) {
	return g.downloadCtrl.AddURI(uris, options)
}

// Pause 暂停指定任务
func (g *GenericARIA2Server) Pause(gid string) error {
	return g.downloadCtrl.Pause(gid)
}

// Unpause 恢复暂停的下载任务
func (g *GenericARIA2Server) Unpause(gid string) error {
	return g.downloadCtrl.Unpause(gid)
}

// Remove 删除指定下载任务
func (g *GenericARIA2Server) Remove(gid string) error {
	return g.downloadCtrl.Remove(gid)
}

// ListDownloads 分页查询下载记录
func (g *GenericARIA2Server) ListDownloads(status string, offset, limit int) ([]dv1.DownloadRecord, int, error) {
	return g.downloadCtrl.ListDownloads(status, offset, limit)
}

// GetDefaultDownloadDir 获取默认下载目录
func (g *GenericARIA2Server) GetDefaultDownloadDir() (string, error) {
	dir := g.settingsCtrl.GetDefaultDownloadDir()
	return dir, nil
}

// FindDownloadByURL 根据 URL 查找已存在的下载记录
func (g *GenericARIA2Server) FindDownloadByURL(url string) (*dv1.DownloadRecord, error) {
	return g.downloadCtrl.FindDownloadByURL(url)
}

// CleanDuplicateByURL 清理同 URL 的旧下载记录和 aria2 缓存（如有暂停先解锁）
func (g *GenericARIA2Server) CleanDuplicateByURL(url string) error {
	return g.downloadCtrl.CleanDuplicateByURL(url)
}

// DeleteDownloadRecord 删除下载记录
func (g *GenericARIA2Server) DeleteDownloadRecord(gid string) error {
	return g.downloadCtrl.Delete(gid)
}

// OpenFileLocation 在文件管理器中打开下载文件所在目录并选中该文件
func (g *GenericARIA2Server) OpenFileLocation(gid string) error {
	return g.downloadCtrl.OpenFileLocation(gid)
}

// DeleteWithLocalFile 删除数据库记录并同时删除本地下载文件
func (g *GenericARIA2Server) DeleteWithLocalFile(gid string) error {
	return g.downloadCtrl.DeleteWithLocalFile(gid)
}

// ContinueDownload 从历史记录恢复下载
func (g *GenericARIA2Server) ContinueDownload(gid string) (arigo.GID, error) {
	return g.downloadCtrl.ContinueDownload(gid)
}

// RemoveDownloadResult 清除指定下载任务的结果记录
func (g *GenericARIA2Server) RemoveDownloadResult(gid string) error {
	return g.downloadCtrl.RemoveDownloadResult(gid)
}

// PurgeDownloadResults 清除所有已完成/出错/已删除的下载记录
func (g *GenericARIA2Server) PurgeDownloadResults() error {
	return g.downloadCtrl.PurgeDownloadResults()
}

// GetSettings 获取当前设置
func (g *GenericARIA2Server) GetSettings() (*dv1.Settings, error) {
	return g.settingsCtrl.GetSettings()
}

// SaveSettings 保存设置到 SQLite 并同步到 aria2
func (g *GenericARIA2Server) SaveSettings(st *dv1.Settings) error {
	return g.settingsCtrl.SaveSettings(st)
}

// GetAppVersion 返回当前应用版本号
func (g *GenericARIA2Server) GetAppVersion() string {
	return g.sysCtrl.GetAppVersion()
}

// CheckForUpdate 检查 GitHub Release 是否有新版本
func (g *GenericARIA2Server) CheckForUpdate() *sys.UpdateCheckResult {
	return g.sysCtrl.CheckForUpdate()
}
