package api

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/balancer"
	"github.com/aknEvrnky/pgway/internal/ports"
)

type Application struct {
	entryPointRepo   ports.EntryPointRepositoryPort
	flowRepo         ports.FlowRepositoryPort
	routerRepo       ports.RouterRepositoryPort
	loadBalancerRepo ports.LoadBalancerRepositoryPort
	poolRepo         ports.PoolRepositoryPort
	proxyRepo        ports.ProxyRepositoryPort

	BalancerService *balancer.Service
}

func NewApplication(
	epRepo ports.EntryPointRepositoryPort,
	fRepo ports.FlowRepositoryPort,
	rRepo ports.RouterRepositoryPort,
	lbRepo ports.LoadBalancerRepositoryPort,
	pRepo ports.PoolRepositoryPort,
	prRepo ports.ProxyRepositoryPort,
) *Application {
	return &Application{
		entryPointRepo:   epRepo,
		flowRepo:         fRepo,
		routerRepo:       rRepo,
		loadBalancerRepo: lbRepo,
		poolRepo:         pRepo,
		proxyRepo:        prRepo,
		BalancerService:  balancer.NewService(lbRepo, pRepo, prRepo),
	}
}

func (a *Application) Bootstrap(ctx context.Context) error {
	if err := a.ValidateAll(ctx); err != nil {
		return err
	}

	if err := a.BalancerService.Bootstrap(ctx); err != nil {
		return err
	}

	return nil
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
