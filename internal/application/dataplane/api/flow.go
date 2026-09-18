package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (a *Application) getFlow(_ context.Context, id string) (*domain.Flow, error) {
	return a.cache.GetFlow(id)
}

// ExecuteFlow executes the flow for given entry point
// and returns the next proxy
func (a *Application) ExecuteFlow(ctx context.Context, entrypointId string, req *http.Request) (proxy *domain.Proxy, balancerId string, err error) {
	ep, err := a.getEntryPoint(ctx, entrypointId)

	if err != nil {
		return nil, "", fmt.Errorf("entrypoint: %w", err)
	}

	flow, err := a.getFlow(ctx, ep.FlowId)

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

	proxy, err = a.balancerService.Next(balancerId)

	return proxy, balancerId, err
}
