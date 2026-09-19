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

func TestApplyForwardRefs(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	proxySpec := proxyv1.ProxySpecV1{Protocol: "http", Host: "127.0.0.1", Port: 8080}

	t.Run("pool with existing proxies succeeds", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		testutil.MustApplyProxy(t, svc, "p1")
		_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-1"}, poolv1.PoolSpecV1{
			Title: "s", Type: "static", Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
		})
		require.NoError(t, err)
	})

	t.Run("pool with unknown proxy rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-1"}, poolv1.PoolSpecV1{
			Title: "s", Type: "static", Members: []poolv1.PoolMemberSpec{{ProxyId: "missing"}},
		})
		require.Error(t, err)
		var missing *controlplane.ResourceMissingRefError
		require.True(t, errors.As(err, &missing))
		assert.Equal(t, "proxy", missing.MissingType)
		assert.Equal(t, "missing", missing.MissingName)
	})

	t.Run("balancer missing pool rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-1"}, balancerv1.BalancerSpecV1{
			Title: "lb", Type: "round-robin", PoolId: "nope",
		})
		require.Error(t, err)
		var missing *controlplane.ResourceMissingRefError
		require.True(t, errors.As(err, &missing))
		assert.Equal(t, "pool", missing.MissingType)
	})

	t.Run("router missing balancer rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyRouterV1(ctx, schema.Metadata{Name: "r1"}, routerv1.RouterSpecV1{
			Title: "r",
			Rules: []routerv1.RuleSpec{{Id: "x", Match: routerv1.MatchSpec{Type: "catch_all"}, Target: "ghost-lb"}},
		})
		require.Error(t, err)
		var missing *controlplane.ResourceMissingRefError
		require.True(t, errors.As(err, &missing))
		assert.Equal(t, "balancer", missing.MissingType)
	})

	t.Run("flow missing balancer rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "f1"}, flowv1.FlowSpecV1{BalancerId: "ghost"})
		require.Error(t, err)
		var missing *controlplane.ResourceMissingRefError
		require.True(t, errors.As(err, &missing))
		assert.Equal(t, "balancer", missing.MissingType)
	})

	t.Run("entrypoint missing flow rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "ep"}, entrypointv1.EntrypointSpecV1{
			Title: "e", Protocol: "http", Host: "0.0.0.0", Port: 1, FlowId: "ghost",
		})
		require.Error(t, err)
		var missing *controlplane.ResourceMissingRefError
		require.True(t, errors.As(err, &missing))
		assert.Equal(t, "flow", missing.MissingType)
	})

	t.Run("dynamic pool skips proxy existence", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "dyn"}, poolv1.PoolSpecV1{
			Title: "d", Type: "dynamic",
			Selector: &poolv1.SelectorSpec{Allow: map[string]string{"env": "prod"}},
		})
		require.NoError(t, err)
	})

	t.Run("update retarget to missing id rejects", func(t *testing.T) {
		svc, _ := testutil.NewSvcWithPublisher(t)
		testutil.MustSeedThroughBalancer(t, svc)
		_, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "f1"}, flowv1.FlowSpecV1{BalancerId: "lb-1"})
		require.NoError(t, err)
		_, err = svc.ApplyFlowV1(ctx, schema.Metadata{Name: "f1"}, flowv1.FlowSpecV1{BalancerId: "missing-lb"})
		require.Error(t, err)
		var missing *controlplane.ResourceMissingRefError
		require.True(t, errors.As(err, &missing))
		assert.Equal(t, "missing-lb", missing.MissingName)
	})

	t.Run("ordered multi-resource apply succeeds", func(t *testing.T) {
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
		_, err = svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-1"}, flowv1.FlowSpecV1{BalancerId: "lb-1"})
		require.NoError(t, err)
		_, err = svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "ep-1"}, entrypointv1.EntrypointSpecV1{
			Title: "e", Protocol: "http", Host: "0.0.0.0", Port: 9090, FlowId: "flow-1",
		})
		require.NoError(t, err)
	})
}
