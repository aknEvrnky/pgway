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
	cache           *ResourceCache
	controlPlane    ports.ControlPlaneReader
	balancerService *balancer.Service
}

func NewApplication(
	cp ports.ControlPlaneReader,
) *Application {
	return &Application{
		cache:           NewResourceCache(),
		balancerService: balancer.NewService(cp),
		controlPlane:    cp,
	}
}

func (a *Application) Bootstrap(ctx context.Context) error {
	// warm up cache
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
		res, err := a.controlPlane.ListEntrypoints(gctx, domain.ListParams{})
		if err != nil {
			return err
		}

		entrypoints = res.Items
		return nil
	})

	g.Go(func() error {
		res, err := a.controlPlane.ListFlows(gctx, domain.ListParams{})
		if err != nil {
			return err
		}

		flows = res.Items
		return nil
	})

	g.Go(func() error {
		res, err := a.controlPlane.ListRouters(gctx, domain.ListParams{})
		if err != nil {
			return err
		}

		routers = res.Items
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
