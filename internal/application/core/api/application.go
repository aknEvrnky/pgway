package api

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/ports"
)

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

func (a *Application) ValidateAll(ctx context.Context) error {
	// validate entrypoints
	eps, err := a.LoadEntryPoints(ctx)

	if err != nil {
		return fmt.Errorf("loading entrypoints: %w", err)
	}

	for _, ep := range eps {
		if err = ep.Validate(); err != nil {
			return fmt.Errorf("entrypoint %q: %w", ep.Id, err)
		}
	}

	return nil
}

func (a *Application) GetVersion() string {
	return "v0.0.1-dev"
}
