package testutil

import (
	"context"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
	"github.com/stretchr/testify/require"
)

var defaultProxySpec = proxyv1.ProxySpecV1{
	Protocol: "http",
	Host:     "127.0.0.1",
	Port:     8080,
}

// MustApplyProxy persists a minimal HTTP proxy.
func MustApplyProxy(t *testing.T, svc *controlplane.Service, name string) {
	t.Helper()
	_, err := svc.ApplyProxyV1(context.Background(), schema.Metadata{Name: name}, defaultProxySpec)
	require.NoError(t, err)
}

// MustApplyStaticPool persists a static pool referencing the given proxy IDs.
func MustApplyStaticPool(t *testing.T, svc *controlplane.Service, name string, proxyIDs ...string) {
	t.Helper()
	if len(proxyIDs) == 0 {
		proxyIDs = []string{"p1"}
	}
	members := make([]poolv1.PoolMemberSpec, 0, len(proxyIDs))
	for _, id := range proxyIDs {
		members = append(members, poolv1.PoolMemberSpec{ProxyId: id})
	}
	_, err := svc.ApplyPoolV1(context.Background(), schema.Metadata{Name: name}, poolv1.PoolSpecV1{
		Title: name, Type: "static", Members: members,
	})
	require.NoError(t, err)
}

// MustApplyBalancer persists a round-robin balancer for the given pool.
func MustApplyBalancer(t *testing.T, svc *controlplane.Service, name, poolID string) {
	t.Helper()
	_, err := svc.ApplyBalancerV1(context.Background(), schema.Metadata{Name: name}, balancerv1.BalancerSpecV1{
		Title: name, Type: "round-robin", PoolId: poolID,
	})
	require.NoError(t, err)
}

// MustApplyRouter persists a catch-all router targeting the balancer.
func MustApplyRouter(t *testing.T, svc *controlplane.Service, name, balancerID string) {
	t.Helper()
	_, err := svc.ApplyRouterV1(context.Background(), schema.Metadata{Name: name}, routerv1.RouterSpecV1{
		Title: name,
		Rules: []routerv1.RuleSpec{{
			Id: "r1", Match: routerv1.MatchSpec{Type: "catch_all"}, Target: balancerID,
		}},
	})
	require.NoError(t, err)
}

// MustApplyFlow persists a flow pointing at the balancer (optional router).
func MustApplyFlow(t *testing.T, svc *controlplane.Service, name, balancerID, routerID string) {
	t.Helper()
	_, err := svc.ApplyFlowV1(context.Background(), schema.Metadata{Name: name}, flowv1.FlowSpecV1{
		BalancerId: balancerID,
		RouterId:   routerID,
	})
	require.NoError(t, err)
}

// MustApplyEntrypoint persists an HTTP entrypoint for the flow.
func MustApplyEntrypoint(t *testing.T, svc *controlplane.Service, name, flowID string, port uint16) {
	t.Helper()
	_, err := svc.ApplyEntrypointV1(context.Background(), schema.Metadata{Name: name}, entrypointv1.EntrypointSpecV1{
		Title: name, Protocol: "http", Host: "0.0.0.0", Port: port, FlowId: flowID,
	})
	require.NoError(t, err)
}

// MustSeedThroughPool creates proxy "p1" and pool "pool-1".
func MustSeedThroughPool(t *testing.T, svc *controlplane.Service) {
	t.Helper()
	MustApplyProxy(t, svc, "p1")
	MustApplyStaticPool(t, svc, "pool-1", "p1")
}

// MustSeedThroughBalancer creates p1 → pool-1 → lb-1.
func MustSeedThroughBalancer(t *testing.T, svc *controlplane.Service) {
	t.Helper()
	MustSeedThroughPool(t, svc)
	MustApplyBalancer(t, svc, "lb-1", "pool-1")
}

// MustSeedThroughFlow creates p1 → pool-1 → lb-1 → flow-1.
func MustSeedThroughFlow(t *testing.T, svc *controlplane.Service) {
	t.Helper()
	MustSeedThroughBalancer(t, svc)
	MustApplyFlow(t, svc, "flow-1", "lb-1", "")
}
