package aria2server

import (
	"context"

	"changeme/backed/pkg/aria2"

	"github.com/siku2/arigo"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type Aria2Service struct {
	svc *aria2.Aria2Service
}

func NewAria2Service() *Aria2Service {
	return &Aria2Service{
		svc: aria2.NewAria2Service(),
	}
}

func (a *Aria2Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return a.svc.ServiceStartup(ctx, options)
}

func (a *Aria2Service) ServiceShutdown() error {
	return a.svc.ServiceShutdown()
}

func (a *Aria2Service) AddURI(uris []string, options *arigo.Options) (arigo.GID, error) {
	return a.svc.AddURI(uris, options)
}

func (a *Aria2Service) PauseAll() error {
	return a.svc.PauseAll()
}
