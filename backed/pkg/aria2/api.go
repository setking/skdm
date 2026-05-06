package aria2

import (
	"fmt"

	"github.com/siku2/arigo"
)

func (a *Aria2Service) AddURI(uris []string, options *arigo.Options) (arigo.GID, error) {
	fmt.Printf("添加下载链接%s", uris)
	return a.rpcClient.AddURI(uris, options)
}

func (a *Aria2Service) PauseAll() error {
	return a.rpcClient.PauseAll()
}
