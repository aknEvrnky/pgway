package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"

	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
)

type ControlPlane interface {
	ProxyControlPlane
	// PoolControlPlane    — upcoming...
}

type ProxyControlPlane interface {
	ApplyProxyV1(ctx context.Context, meta schema.Metadata, spec proxyv1.ProxySpecV1) (*domain.Proxy, error)
	GetProxy(ctx context.Context, name string) (*domain.Proxy, error)
	ListProxies(ctx context.Context) ([]*domain.Proxy, error)
	DeleteProxy(ctx context.Context, name string) error
}
