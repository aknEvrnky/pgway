package api

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/balancer"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"golang.org/x/sync/errgroup"
)

type Application struct {
	entryPointRepo ports.EntryPointRepositoryPort
	flowRepo       ports.FlowRepositoryPort
	routerRepo     ports.RouterRepositoryPort

	cache           *ResourceCache
	balancerService *balancer.Service
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
		entryPointRepo:  epRepo,
		flowRepo:        fRepo,
		routerRepo:      rRepo,
		cache:           NewResourceCache(),
		balancerService: balancer.NewService(lbRepo, pRepo, prRepo),
	}
}

func (a *Application) Bootstrap(ctx context.Context) error {
	// warm up cahce
	if err := a.warmupCache(ctx); err != nil {
		return err
	}

	// validate app
	if err := a.validateAll(ctx); err != nil {
		return err
	}

	// bootstrap load balancers
	if err := a.balancerService.Bootstrap(ctx); err != nil {
		return err
	}

	return nil
}

func (a *Application) warmupCache(ctx context.Context) error {
	// get all necessary resources into cache
	g, gctx := errgroup.WithContext(ctx)

	var entrypoints []*domain.Entrypoint
	var flows []*domain.Flow
	var routers []*domain.Router

	g.Go(func() error {
		res, err := a.entryPointRepo.GetAll(gctx)
		if err != nil {
			return err
		}

		entrypoints = res
		return nil
	})

	g.Go(func() error {
		res, err := a.flowRepo.GetAll(gctx)
		if err != nil {
			return err
		}

		flows = res
		return nil
	})

	g.Go(func() error {
		res, err := a.routerRepo.GetAll(gctx)
		if err != nil {
			return err
		}

		routers = res
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("cache bootstrap failed: %w", err)
	}

	a.cache.Reload(entrypoints, flows, routers)

	return nil
}

func (a *Application) validateAll(ctx context.Context) error {
	// validate entrypoints
	eps, err := a.EntryPoints(ctx)

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
