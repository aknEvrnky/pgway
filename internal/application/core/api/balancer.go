package api

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (a *Application) Release(ctx context.Context, balancerId string, result domain.BalancerResult) error {
	lb, err := a.balancerService.Get(balancerId)
	if err != nil {
		return err
	}

	lb.Release(result)

	return nil
}
