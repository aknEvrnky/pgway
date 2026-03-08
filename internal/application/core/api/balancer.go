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

func (a *Application) Release(ctx context.Context, balancerId string, result domain.BalancerResult) error {
	lb, err := a.BalancerService.Get(balancerId)
	if err != nil {
		return err
	}

	lb.Release(result)

	return nil
}
