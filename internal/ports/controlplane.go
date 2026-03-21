package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"

	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
)

type ControlPlane interface {
	ProxyControlPlane
	PoolControlPlane
}

type ProxyControlPlane interface {
	ApplyProxyV1(ctx context.Context, meta schema.Metadata, spec proxyv1.ProxySpecV1) (*domain.Proxy, error)
	GetProxy(ctx context.Context, name string) (*domain.Proxy, error)
	ListProxies(ctx context.Context) ([]*domain.Proxy, error)
	DeleteProxy(ctx context.Context, name string) error
}

type PoolControlPlane interface {
	ApplyPoolV1(ctx context.Context, meta schema.Metadata, spec poolv1.PoolSpecV1) (*domain.Pool, error)
	GetPool(ctx context.Context, name string) (*domain.Pool, error)
	ListPools(ctx context.Context) ([]*domain.Pool, error)
	DeletePool(ctx context.Context, name string) error
}
