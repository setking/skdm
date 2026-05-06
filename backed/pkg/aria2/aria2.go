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
	"time"

	"github.com/siku2/arigo"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	Aria2RPCSecret = "my-strong-secret-token-2026"
)

//go:embed third_party/*
var binFiles embed.FS

type Aria2Service struct {
	rpcPort   string
	rpcSecret string
	cmd       *exec.Cmd
	rpcClient *arigo.Client
}

func NewAria2Service() *Aria2Service {
	return &Aria2Service{
		rpcSecret: Aria2RPCSecret,
		rpcPort:   "6800",
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
	a.cmd = exec.Command(aria2Path,
		"--enable-rpc",
		"--rpc-listen-port="+a.rpcPort,
		"--rpc-secret="+a.rpcSecret,
	)
	a.cmd.Stdout = os.Stdout
	a.cmd.Stderr = os.Stderr
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
	return nil
}

// 在应用关闭时自动调用
func (a *Aria2Service) ServiceShutdown() error {
	if a.cmd != nil && a.cmd.Process != nil {
		log.Println("[Aria2Service] 正在关闭 aria2server 进程...")
		if runtime.GOOS == "windows" {
			// Windows 下直接结束进程
			if err := a.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("强制终止 aria2server 进程失败: %w", err)
			}
		} else {
			// Unix-like 系统发送中断信号
			if err := a.cmd.Process.Signal(os.Interrupt); err != nil {
				return fmt.Errorf("向 aria2server 发送中断信号失败: %w", err)
			}
		}
	}
	return nil
}
