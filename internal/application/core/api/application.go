package api

import "github.com/aknEvrnky/pgway/internal/ports"

type Application struct {
	entryPointRepo   ports.EntryPointRepositoryPort
	flowRepo         ports.FlowRepositoryPort
	routerRepo       ports.RouterRepositoryPort
	loadBalancerRepo ports.LoadBalancerRepositoryPort
}

func NewApplication(
	epRepo ports.EntryPointRepositoryPort,
	fRepo ports.FlowRepositoryPort,
	rRepo ports.RouterRepositoryPort,
	lbRepo ports.LoadBalancerRepositoryPort,
) *Application {
	return &Application{
		entryPointRepo:   epRepo,
		flowRepo:         fRepo,
		routerRepo:       rRepo,
		loadBalancerRepo: lbRepo,
	}
}

func (a *Application) GetVersion() string {
	return "v0.0.1-dev"
}
