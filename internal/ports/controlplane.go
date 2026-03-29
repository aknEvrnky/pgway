package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
)

// ControlPlaneReader is used for read only cp access
type ControlPlaneReader interface {
	GetProxy(ctx context.Context, name string) (*domain.Proxy, error)
	ListProxies(ctx context.Context, params domain.ListParams, filter domain.ProxyFilter) (domain.ListResult[domain.Proxy], error)
	GetProxiesByIds(ctx context.Context, ids []string) ([]*domain.Proxy, error)
	FindProxiesByLabels(ctx context.Context, labels map[string]string) ([]*domain.Proxy, error)

	GetPool(ctx context.Context, name string) (*domain.Pool, error)
	ListPools(ctx context.Context, params domain.ListParams, filter domain.PoolFilter) (domain.ListResult[domain.Pool], error)

	GetBalancer(ctx context.Context, name string) (*domain.LoadBalancer, error)
	ListBalancers(ctx context.Context, params domain.ListParams, filter domain.BalancerFilter) (domain.ListResult[domain.LoadBalancer], error)

	GetRouter(ctx context.Context, name string) (*domain.Router, error)
	ListRouters(ctx context.Context, params domain.ListParams, filter domain.RouterFilter) (domain.ListResult[domain.Router], error)

	GetFlow(ctx context.Context, name string) (*domain.Flow, error)
	ListFlows(ctx context.Context, params domain.ListParams, filter domain.FlowFilter) (domain.ListResult[domain.Flow], error)

	GetEntrypoint(ctx context.Context, name string) (*domain.Entrypoint, error)
	ListEntrypoints(ctx context.Context, params domain.ListParams, filter domain.EntrypointFilter) (domain.ListResult[domain.Entrypoint], error)
}

// ControlPlaneWriter is used for write only cp access
type ControlPlaneWriter interface {
	ApplyProxyV1(ctx context.Context, meta schema.Metadata, spec proxyv1.ProxySpecV1) (*domain.Proxy, error)
	DeleteProxy(ctx context.Context, name string) error

	ApplyPoolV1(ctx context.Context, meta schema.Metadata, spec poolv1.PoolSpecV1) (*domain.Pool, error)
	DeletePool(ctx context.Context, name string) error

	ApplyBalancerV1(ctx context.Context, meta schema.Metadata, spec balancerv1.BalancerSpecV1) (*domain.LoadBalancer, error)
	DeleteBalancer(ctx context.Context, name string) error

	ApplyRouterV1(ctx context.Context, meta schema.Metadata, spec routerv1.RouterSpecV1) (*domain.Router, error)
	DeleteRouter(ctx context.Context, name string) error

	ApplyFlowV1(ctx context.Context, meta schema.Metadata, spec flowv1.FlowSpecV1) (*domain.Flow, error)
	DeleteFlow(ctx context.Context, name string) error

	ApplyEntrypointV1(ctx context.Context, meta schema.Metadata, spec entrypointv1.EntrypointSpecV1) (*domain.Entrypoint, error)
	DeleteEntrypoint(ctx context.Context, name string) error
}

type ControlPlane interface {
	ControlPlaneReader
	ControlPlaneWriter
}
