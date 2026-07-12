package consumer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/pubsub/memory"
	"github.com/aknEvrnky/pgway/internal/application/event"
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

func (h *recordingHandler) HandleEvent(_ context.Context, _ event.ChangeEvent) error {
	h.log.append(h.name)
	return h.err
}

func testEvent() event.ChangeEvent {
	return event.ChangeEvent{ID: "x", ResourceType: event.ResourceTypeEntrypoint, ChangeKind: event.ChangeKindSaved}
}

func TestConsumer_ConsumeEvents_CallsHandlersInOrder(t *testing.T) {
	ps := memory.NewPubSub(10)
	log := &callLog{}
	first := &recordingHandler{name: "first", log: log}
	second := &recordingHandler{name: "second", log: log}

	consumer := NewConsumer(ps, first, second)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = consumer.ConsumeEvents(ctx)
		close(done)
	}()

	// wait until the consumer is subscribed, then publish
	require.Eventually(t, func() bool {
		require.NoError(t, ps.Publish(context.Background(), testEvent()))
		return len(log.snapshot()) >= 2
	}, 2*time.Second, 20*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ConsumeEvents did not return after ctx cancel")
	}

	calls := log.snapshot()
	require.GreaterOrEqual(t, len(calls), 2)
	// order must hold for every consumed event: first, second, first, second...
	for i := 0; i+1 < len(calls); i += 2 {
		assert.Equal(t, "first", calls[i])
		assert.Equal(t, "second", calls[i+1])
	}
}

func TestConsumer_ConsumeEvents_ContinuesAfterHandlerError(t *testing.T) {
	ps := memory.NewPubSub(10)
	log := &callLog{}
	failing := &recordingHandler{name: "failing", err: errors.New("boom"), log: log}
	second := &recordingHandler{name: "second", log: log}

	consumer := NewConsumer(ps, failing, second)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = consumer.ConsumeEvents(ctx)
		close(done)
	}()

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

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ConsumeEvents did not return after ctx cancel")
	}
}

func TestConsumer_ConsumeEvents_ReturnsNilOnCtxCancel(t *testing.T) {
	ps := memory.NewPubSub(10)
	consumer := NewConsumer(ps, &recordingHandler{name: "h", log: &callLog{}})

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
