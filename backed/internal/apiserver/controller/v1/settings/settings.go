package settings

import (
	"log"
	"strconv"

	dv1 "changeme/backed/api/apiserver/v1"
	ctrlv1 "changeme/backed/internal/apiserver/controller/v1"
	"changeme/backed/internal/apiserver/store"
	srvv1 "changeme/backed/internal/apiserver/service/v1"

	"github.com/siku2/arigo"
)

// SettingsController 设置管理控制器
type SettingsController struct {
	srv srvv1.Service
	rpc ctrlv1.Aria2ClientProvider
}

// NewSettingsController 创建设置控制器
func NewSettingsController(store store.Factory, rpc ctrlv1.Aria2ClientProvider) *SettingsController {
	return &SettingsController{
		srv: srvv1.NewService(store),
		rpc: rpc,
	}
}

// ==================== 启动生命周期 ====================

// LoadAndApplySettings 启动时从 SQLite 加载设置并应用到 aria2
func (s *SettingsController) LoadAndApplySettings() {
	settings, err := s.srv.Settings().GetSettings()
	if err != nil {
		log.Printf("[Settings] 读取设置失败: %v", err)
		return
	}
	log.Printf("[Settings] 已加载设置: dir=%s maxConcurrent=%d maxConn=%d split=%d",
		settings.DefaultDownloadDir, settings.MaxConcurrentDownloads,
		settings.MaxConnectionPerServer, settings.Split)
	s.applySettingsToAria2(settings)
}

// PauseAllIfAutoStartDisabled 如果用户关闭了"启动时自动开始未完成任务"，暂停所有任务
func (s *SettingsController) PauseAllIfAutoStartDisabled() {
	settings, err := s.srv.Settings().GetSettings()
	if err != nil {
		log.Printf("[AutoStart] 读取设置失败: %v", err)
		return
	}
	if settings.AutoStartUnfinished {
		return
	}
	log.Println("[AutoStart] 已关闭自动开始未完成任务，暂停所有任务...")
	c, err := s.rpc.Client()
	if err != nil {
		log.Printf("[AutoStart] 连接 aria2 失败: %v", err)
		return
	}
	if err := c.PauseAll(); err != nil {
		log.Printf("[AutoStart] 暂停所有任务失败: %v", err)
	}
}

// ==================== 选项管理 ====================

// GetGlobalOptions 获取全局选项
func (s *SettingsController) GetGlobalOptions() (arigo.Options, error) {
	c, err := s.rpc.Client()
	if err != nil {
		return arigo.Options{}, err
	}
	return c.GetGlobalOptions()
}

// ChangeGlobalOptions 动态修改全局选项
func (s *SettingsController) ChangeGlobalOptions(options arigo.Options) error {
	c, err := s.rpc.Client()
	if err != nil {
		return err
	}
	return c.ChangeGlobalOptions(options)
}

// ==================== 设置管理 ====================

// GetSettings 获取当前设置
func (s *SettingsController) GetSettings() (*dv1.Settings, error) {
	return s.srv.Settings().GetSettings()
}

// SaveSettings 保存设置到 SQLite 并同步到 aria2
func (s *SettingsController) SaveSettings(st *dv1.Settings) error {
	if err := s.srv.Settings().SaveSettings(st); err != nil {
		return err
	}
	s.applySettingsToAria2(st)
	return nil
}

// GetDefaultDownloadDir 获取默认下载目录
func (s *SettingsController) GetDefaultDownloadDir() string {
	return s.srv.Settings().GetDefaultDownloadDir()
}

// ==================== 内部辅助 ====================

// applySettingsToAria2 将设置应用到 aria2 全局选项
func (s *SettingsController) applySettingsToAria2(st *dv1.Settings) {
	c, err := s.rpc.Client()
	if err != nil {
		log.Printf("[Settings] 连接 api 失败: %v", err)
		return
	}
	limit := strconv.FormatInt(st.MaxDownloadLimit, 10)
	if st.MaxDownloadLimit <= 0 {
		limit = "0"
	}
	opts := arigo.Options{
		MaxConcurrentDownloads: uint(st.MaxConcurrentDownloads),
		MaxConnectionPerServer: uint(st.MaxConnectionPerServer),
		Split:                  uint(st.Split),
		MaxDownloadLimit:       limit,
		Continue:               st.Continue,
		AllowOverwrite:         st.AllowOverwrite,
		AutoFileRenaming:       st.AutoFileRenaming,
	}
	if err := c.ChangeGlobalOptions(opts); err != nil {
		log.Printf("[Settings] 应用设置到 api 失败: %v", err)
	} else {
		log.Printf("[Settings] 已同步设置到 api: maxConcurrent=%d maxConn=%d split=%d limit=%s",
			st.MaxConcurrentDownloads, st.MaxConnectionPerServer, st.Split, limit)
	}
}
