package agenthost

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWatcher struct {
	n     atomic.Int32
	fail  error
	block bool
}

func (f *fakeWatcher) Watch(ctx context.Context, afterConnect func(context.Context) error) error {
	f.n.Add(1)
	if afterConnect != nil {
		if err := afterConnect(ctx); err != nil {
			return err
		}
	}
	if f.fail != nil {
		return f.fail
	}
	if f.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

// dialFailWatcher fails before afterConnect (simulates dial / stream setup failure).
type dialFailWatcher struct {
	n atomic.Int32
}

func (f *dialFailWatcher) Watch(context.Context, func(context.Context) error) error {
	f.n.Add(1)
	return errors.New("dial failed")
}

func TestRunWatchStopsOnCancel(t *testing.T) {
	w := &fakeWatcher{block: true}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunWatch(ctx, zap.NewNop(), w, nil, WatchOptions{})
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, w.n.Load(), int32(1))
}

func TestRunWatchFatalUnauthenticated(t *testing.T) {
	w := &fakeWatcher{fail: ports.ErrAgentUnauthenticated}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := RunWatch(ctx, zap.NewNop(), w, nil, WatchOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ports.ErrAgentUnauthenticated)
}

func TestRunWatchCallsAfterConnect(t *testing.T) {
	w := &fakeWatcher{block: true}
	var connects atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunWatch(ctx, zap.NewNop(), w, func(context.Context) error {
			connects.Add(1)
			return nil
		}, WatchOptions{})
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done
	assert.GreaterOrEqual(t, connects.Load(), int32(1))
}

func TestRunWatchOnConnectedAfterAfterConnect(t *testing.T) {
	w := &fakeWatcher{block: true}
	steps := make(chan string, 2)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunWatch(ctx, zap.NewNop(), w, func(context.Context) error {
			steps <- "afterConnect"
			return nil
		}, WatchOptions{
			OnConnected: func() {
				steps <- "onConnected"
			},
		})
	}()

	require.Equal(t, "afterConnect", <-steps)
	require.Equal(t, "onConnected", <-steps)
	cancel()
	<-done
}

func TestSleepBackoffDoublesAndCaps(t *testing.T) {
	backoff := 10 * time.Millisecond
	max := 35 * time.Millisecond
	require.True(t, sleepBackoff(context.Background(), &backoff, max))
	assert.Equal(t, 20*time.Millisecond, backoff)
	require.True(t, sleepBackoff(context.Background(), &backoff, max))
	assert.Equal(t, 35*time.Millisecond, backoff)
}

func TestRunWatchBackoffGrowsOnRapidFailures(t *testing.T) {
	w := &dialFailWatcher{}
	ctx, cancel := context.WithCancel(context.Background())
	start := time.Now()

	done := make(chan error, 1)
	go func() {
		done <- RunWatch(ctx, zap.NewNop(), w, nil, WatchOptions{
			InitialBackoff: 25 * time.Millisecond,
			MaxBackoff:     200 * time.Millisecond,
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	for w.n.Load() < 3 {
		if time.Now().After(deadline) {
			cancel()
			<-done
			t.Fatalf("timed out waiting for 3 attempts, got %d", w.n.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
	elapsed := time.Since(start)
	cancel()
	<-done

	// Growing: sleep 25ms + 50ms before 2nd and 3rd attempts (≥75ms).
	// Constant-reset bug would be ~50ms for the same attempt count.
	assert.GreaterOrEqual(t, elapsed, 70*time.Millisecond)
	assert.GreaterOrEqual(t, w.n.Load(), int32(3))
}
