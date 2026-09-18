package badger

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"testing"
	"time"

	badgerdb "github.com/dgraph-io/badger/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeGCDB struct {
	calls     atomic.Int32
	lastRatio atomic.Uint64
	errs      []error
}

func (f *fakeGCDB) RunValueLogGC(discardRatio float64) error {
	n := int(f.calls.Add(1) - 1)
	f.lastRatio.Store(math.Float64bits(discardRatio))
	if n < len(f.errs) {
		return f.errs[n]
	}
	return badgerdb.ErrNoRewrite
}

func TestRunValueLogGC_DisabledWhenIntervalNonPositive(t *testing.T) {
	t.Parallel()

	for _, interval := range []time.Duration{0, -time.Second} {
		db := &fakeGCDB{}
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func(interval time.Duration) {
			defer close(done)
			runValueLogGC(ctx, db, interval, zap.NewNop())
		}(interval)

		select {
		case <-done:
		case <-time.After(time.Second):
			cancel()
			t.Fatalf("expected immediate return when interval is %v", interval)
		}
		cancel()
		assert.Equal(t, int32(0), db.calls.Load())
	}
}

func TestRunValueLogGC_StopsOnCancel(t *testing.T) {
	t.Parallel()

	db := &fakeGCDB{errs: []error{badgerdb.ErrNoRewrite}}
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		runValueLogGC(ctx, db, 20*time.Millisecond, zap.NewNop())
	}()

	require.Eventually(t, func() bool {
		return db.calls.Load() >= 1
	}, time.Second, 5*time.Millisecond)

	assert.Equal(t, ValueLogGCDiscardRatio, math.Float64frombits(db.lastRatio.Load()))

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GC loop did not stop after cancel")
	}
}

func TestRunValueLogGCOnce_RewritesUntilNoRewrite(t *testing.T) {
	t.Parallel()

	db := &fakeGCDB{errs: []error{nil, nil, badgerdb.ErrNoRewrite}}
	runValueLogGCOnce(db, zap.NewNop())
	assert.Equal(t, int32(3), db.calls.Load())
}

func TestRunValueLogGCOnce_WarnsAndStopsOnOtherError(t *testing.T) {
	t.Parallel()

	db := &fakeGCDB{errs: []error{errors.New("boom")}}
	runValueLogGCOnce(db, zap.NewNop())
	assert.Equal(t, int32(1), db.calls.Load())
}
