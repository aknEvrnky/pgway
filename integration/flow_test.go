package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
)

func TestFlow_BootstrapAndExecute(t *testing.T) {
	t.Parallel()
	store := badgerutil.NewBadgerStore(t)
	svc := controlplane.NewService(store.Proxies, store.Pools, store.LBs, store.Routers, store.Flows, store.EPs)
	ctx := context.Background()

	// 1. Proxy
	proxy, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "proxy-1"}, proxyv1.ProxySpecV1{
		Protocol: "http",
		Host:     "10.0.0.1",
		Port:     3128,
	})
	require.NoError(t, err)

	// 2. Pool (references proxy)
	pool, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-1"}, poolv1.PoolSpecV1{
		Title:    "test-pool",
		Type:     "static",
		ProxyIds: []string{proxy.Id},
	})
	require.NoError(t, err)

	// 3. Load Balancer (references pool)
	lb, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-1"}, balancerv1.BalancerSpecV1{
		Title:  "test-lb",
		Type:   "round-robin",
		PoolId: pool.Id,
	})
	require.NoError(t, err)

	// 4. Flow (references LB — no back-reference to Entrypoint, so no cycle)
	flow, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-1"}, flowv1.FlowSpecV1{
		BalancerId: lb.Id,
	})
	require.NoError(t, err)

	// 5. Entrypoint (references flow)
	ep, err := svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "ep-1"}, entrypointv1.EntrypointSpecV1{
		Title:    "test-ep",
		Protocol: "http",
		Host:     "0.0.0.0",
		Port:     18080,
		FlowId:   flow.Id,
	})
	require.NoError(t, err)

	// Bootstrap Application from real DB
	app := api.NewApplication(svc)
	err = app.Bootstrap(ctx)
	require.NoError(t, err)

	// Execute: route a request through the entrypoint's flow
	// req is passed to ExecuteFlow; routing is driven by entrypointId directly (no router in this flow)
	req := httptest.NewRequest(http.MethodConnect, "http://example.com:80", nil)
	req.Host = ep.ListenAddr()

	result, _, err := app.ExecuteFlow(ctx, ep.Id, req)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, proxy.Host, result.Host)
	assert.Equal(t, proxy.Port, result.Port)
}
