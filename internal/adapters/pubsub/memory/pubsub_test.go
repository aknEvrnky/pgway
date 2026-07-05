package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPubSub_Publish_Subscribes(t *testing.T) {
	ps := NewPubSub(10)
	ctx := context.Background()

	// create a listener
	ch := ps.Subscribe(ctx)

	chEvent := event.ChangeEvent{
		ID:           "1",
		ResourceType: event.ResourceTypeBalancer,
		ChangeKind:   event.ChangeKindSaved,
	}

	err := ps.Publish(ctx, chEvent)
	require.NoError(t, err)

	select {
	case got := <-ch:
		assert.Equal(t, chEvent, got)
	case <-time.After(time.Second):

	}
}

func TestPubSub_Publish_Publishes_To_Multiple_Subscribers(t *testing.T) {
	ps := NewPubSub(10)
	ctx := context.Background()

	// create a listener
	ch1 := ps.Subscribe(ctx)
	ch2 := ps.Subscribe(ctx)

	chEvent := event.ChangeEvent{
		ID:           "1",
		ResourceType: event.ResourceTypeBalancer,
		ChangeKind:   event.ChangeKindSaved,
	}

	err := ps.Publish(ctx, chEvent)
	require.NoError(t, err)

	for i, ch := range []<-chan event.ChangeEvent{ch1, ch2} {
		select {
		case got := <-ch:
			assert.Equal(t, chEvent, got)
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d received no event", i)
		}
	}
}

func TestPubSub_Concurrent(t *testing.T) {
	ps := NewPubSub(10)
	const workers = 50
	var wg sync.WaitGroup

	// 1 - add concurrent publishers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ps.Publish(context.Background(), event.ChangeEvent{
				ID:           "router-1",
				ResourceType: event.ResourceTypeRouter,
				ChangeKind:   event.ChangeKindSaved,
			})
		}()
	}

	// 2- increasing / decreasing subscribers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(context.Background())
			ch := ps.Subscribe(ctx)

			done := make(chan struct{})
			go func() {
				// channel range drains close call
				for range ch {
				}

				close(done)
			}()

			cancel()
			<-done
		}()
	}

	wg.Wait()

	// validate subscriber count
	ps.mu.Lock()
	count := len(ps.subs)
	ps.mu.Unlock()

	assert.Equal(t, 0, count, "all subscribers are expected to be cleaned")
}
