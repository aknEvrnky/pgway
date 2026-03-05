package ports

import (
	"context"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type Application interface {
	LoadEntryPoints(ctx context.Context) ([]*domain.Entrypoint, error)
	GetEntryPoint(ctx context.Context, id string) (*domain.Entrypoint, error)

	LoadRouters(ctx context.Context) ([]*domain.Router, error)
	GetRouter(ctx context.Context, id string) (*domain.Router, error)

	LoadFlows(ctx context.Context) ([]*domain.Flow, error)
	GetFlow(ctx context.Context, id string) (*domain.Flow, error)
	ExecuteFlow(ctx context.Context, entrypointId string, req *http.Request) (target string, err error)

	GetVersion() string
}
