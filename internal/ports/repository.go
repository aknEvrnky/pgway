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
	Save(ctx context.Context, lb *domain.LoadBalancer) error
	Delete(ctx context.Context, id string) error
}

type PoolRepositoryPort interface {
	GetAll(ctx context.Context) ([]*domain.Pool, error)
	Find(ctx context.Context, id string) (*domain.Pool, error)
	Save(ctx context.Context, pool *domain.Pool) error
	Delete(ctx context.Context, id string) error
}

type ProxyRepositoryPort interface {
	GetAll(ctx context.Context) ([]*domain.Proxy, error)
	Find(ctx context.Context, id string) (*domain.Proxy, error)
	GetByIds(ctx context.Context, ids []string) ([]*domain.Proxy, error)
	FindByLabels(ctx context.Context, labels map[string]string) ([]*domain.Proxy, error)
	Save(ctx context.Context, proxy *domain.Proxy) error
	Delete(ctx context.Context, id string) error
}
