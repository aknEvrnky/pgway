package agenthost

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHB struct {
	calls int
	err   error
}

func (f *fakeHB) Heartbeat(context.Context) (time.Time, error) {
	f.calls++
	if f.err != nil {
		return time.Time{}, f.err
	}
	return time.Now().Add(time.Hour), nil
}

func TestRunHeartbeatStopsOnCancel(t *testing.T) {
	hb := &fakeHB{}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunHeartbeat(ctx, zap.NewNop(), hb, 20*time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, hb.calls, 1)
}

func TestRunHeartbeatFatalOnUnauthenticated(t *testing.T) {
	hb := &fakeHB{err: ports.ErrAgentUnauthenticated}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := RunHeartbeat(ctx, zap.NewNop(), hb, 10*time.Millisecond)
	require.Error(t, err)
	assert.ErrorIs(t, err, ports.ErrAgentUnauthenticated)
	assert.Contains(t, err.Error(), "PGWAY_REGISTRATION_TOKEN")
}

func TestRunHeartbeatContinuesOnTransient(t *testing.T) {
	hb := &fakeHB{err: errors.New("unavailable")}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- RunHeartbeat(ctx, zap.NewNop(), hb, 15*time.Millisecond)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, hb.calls, 2)
}
