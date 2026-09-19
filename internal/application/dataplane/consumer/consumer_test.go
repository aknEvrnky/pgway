package consumer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/aknEvrnky/pgway/internal/ports"

	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingHandler records the order it was invoked in via a shared log.
type recordingHandler struct {
	name string
	err  error
	log  *callLog
}

type callLog struct {
	mu    sync.Mutex
	calls []string
}

func (l *callLog) append(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, name)
}

func (l *callLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.calls...)
}

func (h *recordingHandler) HandleEvent(_ context.Context, _ ports.ChangeEvent) error {
	h.log.append(h.name)
	return h.err
}

type eventLog struct {
	mu     sync.Mutex
	events []ports.ChangeEvent
}

func (l *eventLog) append(e ports.ChangeEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, e)
}

func (l *eventLog) snapshot() []ports.ChangeEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]ports.ChangeEvent(nil), l.events...)
}

type eventRecordingHandler struct {
	log *eventLog
}

func (h *eventRecordingHandler) HandleEvent(_ context.Context, e ports.ChangeEvent) error {
	h.log.append(e)
	return nil
}

func testEvent() ports.ChangeEvent {
	return ports.ChangeEvent{ID: "x", ResourceType: ports.ResourceTypeEntrypoint, ChangeKind: ports.ChangeKindSaved}
}

func startConsumer(t *testing.T, cfg CoalesceConfig, handlers ...ports.EventHandler) (*memory.PubSub, context.CancelFunc, <-chan struct{}) {
	t.Helper()
	ps := memory.NewPubSub(256)
	c := NewConsumer(zap.NewNop(), ps, cfg, handlers...)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = c.ConsumeEvents(ctx)
		close(done)
	}()
	// Subscribe runs before the receive loop; give the goroutine a moment to register.
	time.Sleep(20 * time.Millisecond)
	return ps, cancel, done
}

func stopConsumer(t *testing.T, cancel context.CancelFunc, done <-chan struct{}) {
	t.Helper()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ConsumeEvents did not return after ctx cancel")
	}
}

func TestConsumer_ConsumeEvents_CallsHandlersInOrder(t *testing.T) {
	log := &callLog{}
	first := &recordingHandler{name: "first", log: log}
	second := &recordingHandler{name: "second", log: log}

	ps, cancel, done := startConsumer(t, CoalesceConfig{}, first, second)

	require.Eventually(t, func() bool {
		require.NoError(t, ps.Publish(context.Background(), testEvent()))
		return len(log.snapshot()) >= 2
	}, 2*time.Second, 20*time.Millisecond)

	stopConsumer(t, cancel, done)

	calls := log.snapshot()
	require.GreaterOrEqual(t, len(calls), 2)
	for i := 0; i+1 < len(calls); i += 2 {
		assert.Equal(t, "first", calls[i])
		assert.Equal(t, "second", calls[i+1])
	}
}

func TestConsumer_ConsumeEvents_ContinuesAfterHandlerError(t *testing.T) {
	log := &callLog{}
	failing := &recordingHandler{name: "failing", err: errors.New("boom"), log: log}
	second := &recordingHandler{name: "second", log: log}

	ps, cancel, done := startConsumer(t, CoalesceConfig{}, failing, second)

	require.Eventually(t, func() bool {
		require.NoError(t, ps.Publish(context.Background(), testEvent()))
		calls := log.snapshot()
		for _, c := range calls {
			if c == "second" {
				return true
			}
		}
		return false
	}, 2*time.Second, 20*time.Millisecond, "second handler must run even when the first fails")

	stopConsumer(t, cancel, done)
}

func TestConsumer_ConsumeEvents_ReturnsNilOnCtxCancel(t *testing.T) {
	ps := memory.NewPubSub(10)
	consumer := NewConsumer(zap.NewNop(), ps, CoalesceConfig{}, &recordingHandler{name: "h", log: &callLog{}})

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() { errCh <- consumer.ConsumeEvents(ctx) }()

	cancel()
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("ConsumeEvents did not return after ctx cancel")
	}
}

func TestConsumer_Coalesce_ProxyBurstOneBalancerEvent(t *testing.T) {
	elog := &eventLog{}
	ps, cancel, done := startConsumer(t, CoalesceConfig{
		Window:    40 * time.Millisecond,
		MaxBuffer: 256,
	}, &eventRecordingHandler{log: elog})

	for i := 0; i < 30; i++ {
		require.NoError(t, ps.Publish(context.Background(), ports.ChangeEvent{
			ID:           "p",
			ResourceType: ports.ResourceTypeProxy,
			ChangeKind:   ports.ChangeKindSaved,
		}))
	}

	require.Eventually(t, func() bool {
		return len(elog.snapshot()) >= 1
	}, 2*time.Second, 5*time.Millisecond)

	time.Sleep(60 * time.Millisecond) // ensure no second flush
	events := elog.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, ports.ResourceTypeProxy, events[0].ResourceType)
	assert.Equal(t, coalescedEventID, events[0].ID)

	stopConsumer(t, cancel, done)
}

func TestConsumer_Coalesce_EntrypointLastWins(t *testing.T) {
	elog := &eventLog{}
	ps, cancel, done := startConsumer(t, CoalesceConfig{
		Window:    40 * time.Millisecond,
		MaxBuffer: 256,
	}, &eventRecordingHandler{log: elog})

	require.NoError(t, ps.Publish(context.Background(), ports.ChangeEvent{
		ID: "ep-1", ResourceType: ports.ResourceTypeEntrypoint, ChangeKind: ports.ChangeKindSaved,
	}))
	require.NoError(t, ps.Publish(context.Background(), ports.ChangeEvent{
		ID: "ep-1", ResourceType: ports.ResourceTypeEntrypoint, ChangeKind: ports.ChangeKindDeleted,
	}))

	require.Eventually(t, func() bool {
		return len(elog.snapshot()) >= 1
	}, 2*time.Second, 5*time.Millisecond)

	time.Sleep(60 * time.Millisecond)
	events := elog.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "ep-1", events[0].ID)
	assert.Equal(t, ports.ChangeKindDeleted, events[0].ChangeKind)

	stopConsumer(t, cancel, done)
}

func TestConsumer_Coalesce_MaxBufferEarlyFlush(t *testing.T) {
	elog := &eventLog{}
	ps, cancel, done := startConsumer(t, CoalesceConfig{
		Window:    5 * time.Second, // would be slow without early flush
		MaxBuffer: 5,
	}, &eventRecordingHandler{log: elog})

	for i := 0; i < 5; i++ {
		require.NoError(t, ps.Publish(context.Background(), ports.ChangeEvent{
			ID: "p", ResourceType: ports.ResourceTypeProxy, ChangeKind: ports.ChangeKindSaved,
		}))
	}

	require.Eventually(t, func() bool {
		return len(elog.snapshot()) >= 1
	}, 500*time.Millisecond, 5*time.Millisecond, "expected early flush before window")

	stopConsumer(t, cancel, done)
}

func TestConsumer_Coalesce_KeyedIndependence(t *testing.T) {
	elog := &eventLog{}
	ps, cancel, done := startConsumer(t, CoalesceConfig{
		Window:    50 * time.Millisecond,
		MaxBuffer: 256,
	}, &eventRecordingHandler{log: elog})

	require.NoError(t, ps.Publish(context.Background(), ports.ChangeEvent{
		ID: "p1", ResourceType: ports.ResourceTypeProxy, ChangeKind: ports.ChangeKindSaved,
	}))
	require.NoError(t, ps.Publish(context.Background(), ports.ChangeEvent{
		ID: "ep-1", ResourceType: ports.ResourceTypeEntrypoint, ChangeKind: ports.ChangeKindSaved,
	}))

	require.Eventually(t, func() bool {
		return len(elog.snapshot()) >= 2
	}, 2*time.Second, 5*time.Millisecond)

	events := elog.snapshot()
	var sawProxy, sawEP bool
	for _, e := range events {
		if e.ResourceType == ports.ResourceTypeProxy {
			sawProxy = true
		}
		if e.ResourceType == ports.ResourceTypeEntrypoint {
			sawEP = true
		}
	}
	assert.True(t, sawProxy)
	assert.True(t, sawEP)

	stopConsumer(t, cancel, done)
}

func TestConsumer_Coalesce_DisabledPassthrough(t *testing.T) {
	elog := &eventLog{}
	ps, cancel, done := startConsumer(t, CoalesceConfig{Window: 0}, &eventRecordingHandler{log: elog})

	for i := 0; i < 3; i++ {
		require.NoError(t, ps.Publish(context.Background(), ports.ChangeEvent{
			ID: "p", ResourceType: ports.ResourceTypeProxy, ChangeKind: ports.ChangeKindSaved,
		}))
	}

	require.Eventually(t, func() bool {
		return len(elog.snapshot()) >= 3
	}, 2*time.Second, 5*time.Millisecond)

	assert.Len(t, elog.snapshot(), 3)
	stopConsumer(t, cancel, done)
}
