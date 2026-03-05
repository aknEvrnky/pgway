package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (a *Application) LoadRouters(ctx context.Context) ([]*domain.Router, error) {
	return a.routerRepo.GetAll(ctx)
}

func (a *Application) GetRouter(ctx context.Context, id string) (*domain.Router, error) {
	return a.routerRepo.Find(ctx, id)
}

// RouteRequest finds the router by id, evaluates rules against the request,
// and returns the target balancer id.
func (a *Application) RouteRequest(ctx context.Context, routerId string, req *http.Request) (target string, err error) {
	router, err := a.GetRouter(ctx, routerId)

	if err != nil {
		return "", fmt.Errorf("finding router %q: %w", routerId, err)
	}

	target, ok := router.Resolve(req)

	if !ok {
		return "", fmt.Errorf("router %q: %w", routerId, domain.ErrNoMatchingRule)
	}

	return target, nil

}
