package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"

	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
)

type ControlPlane interface {
	ProxyControlPlane
	PoolControlPlane
	BalancerControlPlane
	RouterControlPlane
	FlowControlPlane
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

type BalancerControlPlane interface {
	ApplyBalancerV1(ctx context.Context, meta schema.Metadata, spec balancerv1.BalancerSpecV1) (*domain.LoadBalancer, error)
	GetBalancer(ctx context.Context, name string) (*domain.LoadBalancer, error)
	ListBalancers(ctx context.Context) ([]*domain.LoadBalancer, error)
	DeleteBalancer(ctx context.Context, name string) error
}

type RouterControlPlane interface {
	ApplyRouterV1(ctx context.Context, meta schema.Metadata, spec routerv1.RouterSpecV1) (*domain.Router, error)
	GetRouter(ctx context.Context, name string) (*domain.Router, error)
	ListRouters(ctx context.Context) ([]*domain.Router, error)
	DeleteRouter(ctx context.Context, name string) error
}

type FlowControlPlane interface {
	ApplyFlowV1(ctx context.Context, meta schema.Metadata, spec flowv1.FlowSpecV1) (*domain.Flow, error)
	GetFlow(ctx context.Context, name string) (*domain.Flow, error)
	ListFlows(ctx context.Context) ([]*domain.Flow, error)
	DeleteFlow(ctx context.Context, name string) error
}
