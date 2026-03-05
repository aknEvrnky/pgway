package api

import "github.com/aknEvrnky/pgway/internal/ports"

type Application struct {
	entryPointRepo ports.EntryPointRepositoryPort
	flowRepo       ports.FlowRepositoryPort
	routerRepo     ports.RouterRepositoryPort
}

func NewApplication(
	epRepo ports.EntryPointRepositoryPort,
	fRepo ports.FlowRepositoryPort,
	rRepo ports.RouterRepositoryPort) *Application {
	return &Application{
		entryPointRepo: epRepo,
		flowRepo:       fRepo,
		routerRepo:     rRepo,
	}
}

func (a *Application) GetVersion() string {
	return "v0.0.1-dev"
}
