package integration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aknEvrnky/pgway/integration/testutil"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/agenthost"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/api"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/consumer"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	v1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
)

type clientWatcher struct {
	watch func(ctx context.Context, afterConnect func(context.Context) error) error
}

func (w clientWatcher) Watch(ctx context.Context, afterConnect func(context.Context) error) error {
	return w.watch(ctx, afterConnect)
}

// TestWatchHotReload verifies distributed CP→DP reload over ChangeService.Watch
// with agent authentication.
func TestWatchHotReload(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	addr, authService := testutil.NewAuthTestServer(t)

	anon := newAuthedClient(t, addr, "")
	_, adminToken, err := anon.InitAdmin(ctx, authService.BootstrapToken(), "admin", "password123")
	require.NoError(t, err)
	admin := newAuthedClient(t, addr, adminToken)

	regToken, err := admin.CreateRegistrationToken(ctx, time.Hour)
	require.NoError(t, err)

	_, agentToken, err := anon.Register(ctx, regToken, domain.Agent{
		Id:       "watch-edge",
		Hostname: "watch-host",
		Version:  "v0.1.0",
	})
	require.NoError(t, err)

	agent := newAuthedClient(t, addr, agentToken)
	app := api.NewApplication(agent, agent, zap.NewNop())
	require.NoError(t, app.Bootstrap(ctx))

	eps, err := app.EntryPoints(ctx)
	require.NoError(t, err)
	assert.Empty(t, eps)

	localBus := memory.NewPubSub(10)
	spy := &testutil.SpyHandler{}
	eventConsumer := consumer.NewConsumer(zap.NewNop(), localBus, consumer.CoalesceConfig{}, app, spy)

	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	watchReady := make(chan struct{})
	var readyOnce sync.Once

	go func() { _ = eventConsumer.ConsumeEvents(runCtx) }()
	go func() {
		_ = agenthost.RunWatch(runCtx, zap.NewNop(), clientWatcher{
			watch: func(c context.Context, after func(context.Context) error) error {
				return agent.Watch(c, localBus, after)
			},
		}, func(c context.Context) error {
			return app.Bootstrap(c)
		}, agenthost.WatchOptions{
			OnConnected: func() {
				readyOnce.Do(func() { close(watchReady) })
			},
		})
	}()

	select {
	case <-watchReady:
	case <-time.After(5 * time.Second):
		t.Fatal("watch stream did not become ready")
	}

	_, err = admin.ApplyProxyV1(ctx, schema.Metadata{Name: "p1"}, proxyv1.ProxySpecV1{
		Protocol: "http", Host: "127.0.0.1", Port: 8080,
	})
	require.NoError(t, err)
	_, err = admin.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-1"}, poolv1.PoolSpecV1{
		Title: "s", Type: "static", Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
	})
	require.NoError(t, err)
	_, err = admin.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-1"}, balancerv1.BalancerSpecV1{
		Title: "lb", Type: "round-robin", PoolId: "pool-1",
	})
	require.NoError(t, err)
	_, err = admin.ApplyFlowV1(ctx, schema.Metadata{Name: "any-flow"}, flowv1.FlowSpecV1{BalancerId: "lb-1"})
	require.NoError(t, err)

	_, err = admin.ApplyEntrypointV1(ctx, schema.Metadata{Name: "watched-ep"}, v1.EntrypointSpecV1{
		Title:    "watched-ep",
		Protocol: "http",
		Host:     "0.0.0.0",
		Port:     19090,
		FlowId:   "any-flow",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		eps, err := app.EntryPoints(ctx)
		if err != nil {
			return false
		}
		for _, ep := range eps {
			if ep.Id == "watched-ep" {
				return true
			}
		}
		return false
	}, 3*time.Second, 20*time.Millisecond, "watch hint did not reload DP cache")

	require.NoError(t, admin.DeleteEntrypoint(ctx, "watched-ep"))
	require.Eventually(t, func() bool {
		eps, err := app.EntryPoints(ctx)
		return err == nil && len(eps) == 0
	}, 3*time.Second, 20*time.Millisecond, "delete hint did not clear DP cache")
}
