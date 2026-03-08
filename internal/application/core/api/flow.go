package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (a *Application) LoadFlows(ctx context.Context) ([]*domain.Flow, error) {
	return a.flowRepo.GetAll(ctx)
}

func (a *Application) GetFlow(ctx context.Context, id string) (*domain.Flow, error) {
	return a.flowRepo.Find(ctx, id)
}

// ExecuteFlow executes the flow for given entry point
// and returns the next proxy
func (a *Application) ExecuteFlow(ctx context.Context, entrypointId string, req *http.Request) (proxy *domain.Proxy, balancerId string, err error) {
	ep, err := a.GetEntryPoint(ctx, entrypointId)

	if err != nil {
		return nil, "", fmt.Errorf("entrypoint: %w", err)
	}

	flow, err := a.GetFlow(ctx, ep.Flow)

	if err != nil {
		return nil, "", fmt.Errorf("flow: %w", err)
	}

	if flow.RouterId != "" {
		routedBalancer, err := a.RouteRequest(ctx, flow.RouterId, req)
		if err != nil {
			return nil, "", fmt.Errorf("router: %w", err)
		}

		balancerId = routedBalancer
	} else if flow.BalancerId != "" {
		balancerId = flow.BalancerId
	} else {
		return nil, "", fmt.Errorf("no router or balancer for flow: %q", flow.Id)
	}

	proxy, err = a.BalancerService.Next(balancerId)

	return proxy, balancerId, err
}
