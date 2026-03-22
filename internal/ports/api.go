package ports

import (
	"context"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type Application interface {
	Bootstrap(ctx context.Context) error
	EntryPoints(ctx context.Context) ([]*domain.Entrypoint, error)
	ExecuteFlow(ctx context.Context, entrypointId string, req *http.Request) (proxy *domain.Proxy, balancerId string, err error)
	Release(ctx context.Context, balancerId string, result domain.BalancerResult) error
}
