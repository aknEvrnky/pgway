package badger

import (
	"context"
	"errors"
	"time"

	badgerdb "github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

// ValueLogGCDiscardRatio is the fraction of discardable data required before
// Badger rewrites a value log file. 0.5 is the commonly recommended default.
const ValueLogGCDiscardRatio = 0.5

// valueLogGCDB is the subset of *badger.DB needed for value log GC.
type valueLogGCDB interface {
	RunValueLogGC(discardRatio float64) error
}

// RunValueLogGC periodically runs Badger value log GC until ctx is cancelled.
// If interval <= 0, it returns immediately (GC disabled).
func RunValueLogGC(ctx context.Context, db *badgerdb.DB, interval time.Duration, log *zap.Logger) {
	runValueLogGC(ctx, db, interval, log)
}

func runValueLogGC(ctx context.Context, db valueLogGCDB, interval time.Duration, log *zap.Logger) {
	if interval <= 0 {
		return
	}
	if log == nil {
		log = zap.NewNop()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runValueLogGCOnce(db, log)
		}
	}
}

func runValueLogGCOnce(db valueLogGCDB, log *zap.Logger) {
	for {
		err := db.RunValueLogGC(ValueLogGCDiscardRatio)
		if err == nil {
			continue
		}
		if errors.Is(err, badgerdb.ErrNoRewrite) {
			return
		}
		log.Warn("badger value log GC", zap.Error(err))
		return
	}
}
