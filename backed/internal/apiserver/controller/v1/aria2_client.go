package v1

import "github.com/siku2/arigo"

// Aria2ClientProvider 提供访问 aria2 RPC 客户端的接口
type Aria2ClientProvider interface {
	Client() (*arigo.Client, error)
}
