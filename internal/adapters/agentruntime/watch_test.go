package agentruntime

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeWatcher struct {
	n     atomic.Int32
	fail  error
	block bool
}

func (f *fakeWatcher) Watch(ctx context.Context) error {
	f.n.Add(1)
	if f.fail != nil {
		return f.fail
	}
	if f.block {
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func TestRunWatchStopsOnCancel(t *testing.T) {
	w := &fakeWatcher{block: true}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunWatch(ctx, w, nil)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, w.n.Load(), int32(1))
}

func TestRunWatchFatalUnauthenticated(t *testing.T) {
	w := &fakeWatcher{fail: status.Error(codes.Unauthenticated, "revoked")}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := RunWatch(ctx, w, nil)
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRunWatchCallsOnConnect(t *testing.T) {
	w := &fakeWatcher{block: true}
	var connects atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunWatch(ctx, w, func(context.Context) error {
			connects.Add(1)
			return nil
		})
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done
	assert.GreaterOrEqual(t, connects.Load(), int32(1))
}
