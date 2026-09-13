package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aknEvrnky/pgway/integration/testutil"
	"github.com/aknEvrnky/pgway/internal/adapters/agentruntime"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/application/consumer"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	v1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
)

type clientWatcher struct {
	watch func(ctx context.Context) error
}

func (w clientWatcher) Watch(ctx context.Context) error {
	return w.watch(ctx)
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
	app := api.NewApplication(agent, agent)
	require.NoError(t, app.Bootstrap(ctx))

	eps, err := app.EntryPoints(ctx)
	require.NoError(t, err)
	assert.Empty(t, eps)

	localBus := memory.NewPubSub(10)
	spy := &testutil.SpyHandler{}
	eventConsumer := consumer.NewConsumer(localBus, app, spy)

	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	go func() { _ = eventConsumer.ConsumeEvents(runCtx) }()
	go func() {
		_ = agentruntime.RunWatch(runCtx, clientWatcher{
			watch: func(c context.Context) error { return agent.Watch(c, localBus) },
		}, func(c context.Context) error {
			return app.Bootstrap(c)
		})
	}()

	// Allow the Watch stream to connect (onConnect Bootstrap + stream open).
	time.Sleep(50 * time.Millisecond)

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
