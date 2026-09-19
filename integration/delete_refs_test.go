package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aknEvrnky/pgway/integration/testutil"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteReferentialIntegrity(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	proxySpec := proxyv1.ProxySpecV1{Protocol: "http", Host: "127.0.0.1", Port: 8080}

	t.Run("unused resources delete successfully", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)

		_, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "orphan-proxy"}, proxySpec)
		require.NoError(t, err)
		require.NoError(t, svc.DeleteProxy(ctx, "orphan-proxy"))

		_, err = svc.ApplyPoolV1(ctx, schema.Metadata{Name: "orphan-pool"}, poolv1.PoolSpecV1{
			Title: "o", Type: "dynamic",
			Selector: &poolv1.SelectorSpec{Allow: map[string]string{"env": "x"}},
		})
		require.NoError(t, err)
		require.NoError(t, svc.DeletePool(ctx, "orphan-pool"))
	})

	t.Run("delete proxy in static pool rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "p1"}, proxySpec)
		require.NoError(t, err)
		_, err = svc.ApplyPoolV1(ctx, schema.Metadata{Name: "static-pool"}, poolv1.PoolSpecV1{
			Title: "s", Type: "static", Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
		})
		require.NoError(t, err)

		err = svc.DeleteProxy(ctx, "p1")
		require.Error(t, err)
		var inUse *controlplane.ResourceInUseError
		require.True(t, errors.As(err, &inUse))
		assert.Equal(t, "proxy", inUse.ResourceType)
		assert.Equal(t, "static-pool", inUse.Dependents[0].Name)
	})

	t.Run("delete proxy only matched by dynamic selector succeeds", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "dyn-p", Labels: map[string]string{"env": "prod"}}, proxySpec)
		require.NoError(t, err)
		_, err = svc.ApplyPoolV1(ctx, schema.Metadata{Name: "dyn-pool"}, poolv1.PoolSpecV1{
			Title: "d", Type: "dynamic",
			Selector: &poolv1.SelectorSpec{Allow: map[string]string{"env": "prod"}},
		})
		require.NoError(t, err)

		require.NoError(t, svc.DeleteProxy(ctx, "dyn-p"))
	})

	t.Run("delete pool referenced by balancer rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "p1"}, proxySpec)
		require.NoError(t, err)
		_, err = svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-1"}, poolv1.PoolSpecV1{
			Title: "s", Type: "static", Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
		})
		require.NoError(t, err)
		_, err = svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-1"}, balancerv1.BalancerSpecV1{
			Title: "lb", Type: "round-robin", PoolId: "pool-1",
		})
		require.NoError(t, err)

		err = svc.DeletePool(ctx, "pool-1")
		require.Error(t, err)
		var inUse *controlplane.ResourceInUseError
		require.True(t, errors.As(err, &inUse))
		assert.Equal(t, "balancer", inUse.Dependents[0].Type)
		assert.Equal(t, "lb-1", inUse.Dependents[0].Name)
	})

	t.Run("delete balancer referenced by flow rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		seedBalancerGraph(t, svc)
		_, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-1"}, flowv1.FlowSpecV1{BalancerId: "lb-1"})
		require.NoError(t, err)

		err = svc.DeleteBalancer(ctx, "lb-1")
		require.Error(t, err)
		var inUse *controlplane.ResourceInUseError
		require.True(t, errors.As(err, &inUse))
		assert.Equal(t, "flow", inUse.Dependents[0].Type)
	})

	t.Run("delete balancer referenced by router target rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		seedBalancerGraph(t, svc)
		_, err := svc.ApplyRouterV1(ctx, schema.Metadata{Name: "router-1"}, routerv1.RouterSpecV1{
			Title: "r",
			Rules: []routerv1.RuleSpec{{
				Id: "r1", Match: routerv1.MatchSpec{Type: "catch_all"}, Target: "lb-1",
			}},
		})
		require.NoError(t, err)

		err = svc.DeleteBalancer(ctx, "lb-1")
		require.Error(t, err)
		var inUse *controlplane.ResourceInUseError
		require.True(t, errors.As(err, &inUse))
		assert.Equal(t, "router", inUse.Dependents[0].Type)
	})

	t.Run("delete router referenced by flow rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		seedBalancerGraph(t, svc)
		_, err := svc.ApplyRouterV1(ctx, schema.Metadata{Name: "router-1"}, routerv1.RouterSpecV1{
			Title: "r",
			Rules: []routerv1.RuleSpec{{
				Id: "r1", Match: routerv1.MatchSpec{Type: "catch_all"}, Target: "lb-1",
			}},
		})
		require.NoError(t, err)
		_, err = svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-1"}, flowv1.FlowSpecV1{
			RouterId: "router-1", BalancerId: "lb-1",
		})
		require.NoError(t, err)

		err = svc.DeleteRouter(ctx, "router-1")
		require.Error(t, err)
		var inUse *controlplane.ResourceInUseError
		require.True(t, errors.As(err, &inUse))
		assert.Equal(t, "flow", inUse.Dependents[0].Type)
	})

	t.Run("delete flow referenced by entrypoint rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		seedBalancerGraph(t, svc)
		_, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-1"}, flowv1.FlowSpecV1{BalancerId: "lb-1"})
		require.NoError(t, err)
		_, err = svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "ep-1"}, entrypointv1.EntrypointSpecV1{
			Title: "e", Protocol: "http", Host: "0.0.0.0", Port: 9090, FlowId: "flow-1",
		})
		require.NoError(t, err)

		err = svc.DeleteFlow(ctx, "flow-1")
		require.Error(t, err)
		var inUse *controlplane.ResourceInUseError
		require.True(t, errors.As(err, &inUse))
		assert.Equal(t, "entrypoint", inUse.Dependents[0].Type)
	})

	t.Run("delete entrypoint succeeds while flow remains", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		seedBalancerGraph(t, svc)
		_, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-1"}, flowv1.FlowSpecV1{BalancerId: "lb-1"})
		require.NoError(t, err)
		_, err = svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "ep-1"}, entrypointv1.EntrypointSpecV1{
			Title: "e", Protocol: "http", Host: "0.0.0.0", Port: 9090, FlowId: "flow-1",
		})
		require.NoError(t, err)

		require.NoError(t, svc.DeleteEntrypoint(ctx, "ep-1"))
		_, err = svc.GetFlow(ctx, "flow-1")
		require.NoError(t, err)
	})

	t.Run("ordered teardown succeeds", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "p1"}, proxySpec)
		require.NoError(t, err)
		_, err = svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-1"}, poolv1.PoolSpecV1{
			Title: "s", Type: "static", Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
		})
		require.NoError(t, err)
		_, err = svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-1"}, balancerv1.BalancerSpecV1{
			Title: "lb", Type: "round-robin", PoolId: "pool-1",
		})
		require.NoError(t, err)
		_, err = svc.ApplyRouterV1(ctx, schema.Metadata{Name: "router-1"}, routerv1.RouterSpecV1{
			Title: "r",
			Rules: []routerv1.RuleSpec{{
				Id: "r1", Match: routerv1.MatchSpec{Type: "catch_all"}, Target: "lb-1",
			}},
		})
		require.NoError(t, err)
		_, err = svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-1"}, flowv1.FlowSpecV1{
			RouterId: "router-1", BalancerId: "lb-1",
		})
		require.NoError(t, err)
		_, err = svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "ep-1"}, entrypointv1.EntrypointSpecV1{
			Title: "e", Protocol: "http", Host: "0.0.0.0", Port: 9090, FlowId: "flow-1",
		})
		require.NoError(t, err)

		require.NoError(t, svc.DeleteEntrypoint(ctx, "ep-1"))
		require.NoError(t, svc.DeleteFlow(ctx, "flow-1"))
		require.NoError(t, svc.DeleteRouter(ctx, "router-1"))
		require.NoError(t, svc.DeleteBalancer(ctx, "lb-1"))
		require.NoError(t, svc.DeletePool(ctx, "pool-1"))
		require.NoError(t, svc.DeleteProxy(ctx, "p1"))
	})
}

func seedBalancerGraph(t *testing.T, svc *controlplane.Service) {
	t.Helper()
	ctx := context.Background()
	_, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "p1"}, proxyv1.ProxySpecV1{
		Protocol: "http", Host: "127.0.0.1", Port: 8080,
	})
	require.NoError(t, err)
	_, err = svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-1"}, poolv1.PoolSpecV1{
		Title: "s", Type: "static", Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
	})
	require.NoError(t, err)
	_, err = svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-1"}, balancerv1.BalancerSpecV1{
		Title: "lb", Type: "round-robin", PoolId: "pool-1",
	})
	require.NoError(t, err)
}
