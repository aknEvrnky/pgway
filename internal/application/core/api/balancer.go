package api

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (a *Application) LoadLoadBalancers(ctx context.Context) ([]*domain.LoadBalancer, error) {
	return a.loadBalancerRepo.GetAll(ctx)
}

func (a *Application) GetLoadBalancer(ctx context.Context, id string) (*domain.LoadBalancer, error) {
	return a.loadBalancerRepo.Find(ctx, id)
}
