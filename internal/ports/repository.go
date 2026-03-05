package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type EntryPointRepositoryPort interface {
	GetAll(ctx context.Context) ([]*domain.Entrypoint, error)
	Find(ctx context.Context, id string) (*domain.Entrypoint, error)
}

type FlowRepositoryPort interface {
	GetAll(ctx context.Context) ([]*domain.Flow, error)
	Find(ctx context.Context, id string) (*domain.Flow, error)
}

type RouterRepositoryPort interface {
	GetAll(ctx context.Context) ([]*domain.Router, error)
	Find(ctx context.Context, id string) (*domain.Router, error)
}

type LoadBalancerRepositoryPort interface {
	GetAll(ctx context.Context) ([]*domain.LoadBalancer, error)
	Find(ctx context.Context, id string) (*domain.LoadBalancer, error)
}
