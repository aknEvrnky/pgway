package integration

import (
	"context"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/integration/testutil"
	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/application/consumer"
	"github.com/aknEvrnky/pgway/internal/application/core/api"
	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/aknEvrnky/pgway/internal/schema"
	v1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplication_Event_Handler(t *testing.T) {
	t.Parallel()

	pubSub := memory.NewPubSub(10)
	spy := &testutil.SpyHandler{}
	svc := testutil.NewSvc(t, pubSub)
	app := api.NewApplication(svc, svc)

	// init the consumer
	eventConsumer := consumer.NewConsumer(pubSub, app, spy)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = eventConsumer.ConsumeEvents(ctx) }()

	// gate: wait until subscriber starts to consume
	require.Eventually(t, func() bool {
		_ = pubSub.Publish(ctx, event.ChangeEvent{ID: "sentinel"})
		return spy.Count() > 0
	}, 2*time.Second, 10*time.Millisecond)

	// assert no entrypoint in dataplane cache
	eps, err := app.EntryPoints(ctx)
	require.NoError(t, err)
	assert.Len(t, eps, 0)

	_, err = svc.ApplyEntrypointV1(ctx, schema.Metadata{
		Name: "test-ep",
	}, v1.EntrypointSpecV1{
		Title:    "test-ep",
		Protocol: "http",
		Host:     "0.0.0.0",
		Port:     18080,
		FlowId:   "test-flow",
	})

	require.NoError(t, err)

	// ASSERT — wait until dp applies the cache
	require.Eventually(t, func() bool {
		eps, err := app.EntryPoints(ctx)
		if err != nil {
			return false
		}
		for _, ep := range eps {
			if ep.Id == "test-ep" {
				return true
			}
		}
		return false
	}, 2*time.Second, 20*time.Millisecond, "apply did not reach the DP cache")

	// ACT — delete - ASSERT — wait until dp deletes from the cache
	require.NoError(t, svc.DeleteEntrypoint(ctx, "test-ep"))
	require.Eventually(t, func() bool {
		eps, err := app.EntryPoints(ctx)
		return err == nil && len(eps) == 0
	}, 2*time.Second, 20*time.Millisecond, "delete did not reach the DP cache")
}
