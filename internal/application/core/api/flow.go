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

func (a *Application) ExecuteFlow(ctx context.Context, entrypointId string, req *http.Request) (target string, err error) {
	ep, err := a.GetEntryPoint(ctx, entrypointId)

	if err != nil {
		return "", fmt.Errorf("entrypoint: %w", err)
	}

	flow, err := a.GetFlow(ctx, ep.Flow)

	if err != nil {
		return "", fmt.Errorf("flow: %w", err)
	}

	if flow.RouterId != "" {
		balancerId, err := a.RouteRequest(ctx, flow.RouterId, req)
		if err != nil {
			return "", fmt.Errorf("router: %w", err)
		}

		return balancerId, nil
	}

	if flow.BalancerId == "" {
		return "", fmt.Errorf("flow %q: no router or balancer defined", flow.Id)
	}

	return flow.BalancerId, nil
}
