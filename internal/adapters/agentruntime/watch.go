package agentruntime

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Watcher opens a Watch stream until it fails or ctx is canceled.
type Watcher interface {
	Watch(ctx context.Context) error
}

// RunWatch reconnects forever until ctx is canceled. onConnect runs before
// each stream (full reload to close missed-event gaps). Transient failures
// back off; Unauthenticated is fatal.
func RunWatch(ctx context.Context, watch Watcher, onConnect func(context.Context) error) error {
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if onConnect != nil {
			if err := onConnect(ctx); err != nil {
				if isUnauthenticated(err) {
					return fatalAuthError(err)
				}
				zap.L().Warn("watch reconnect reload failed", zap.Error(err))
				if !sleepBackoff(ctx, &backoff, maxBackoff) {
					return ctx.Err()
				}
				continue
			}
		}

		err := watch.Watch(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if isUnauthenticated(err) {
			return fatalAuthError(err)
		}
		if err != nil {
			zap.L().Warn("watch stream ended", zap.Error(err))
		}

		backoff = time.Second
		if !sleepBackoff(ctx, &backoff, maxBackoff) {
			return ctx.Err()
		}
	}
}

func sleepBackoff(ctx context.Context, backoff *time.Duration, max time.Duration) bool {
	t := time.NewTimer(*backoff)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
	}
	next := *backoff * 2
	if next > max {
		next = max
	}
	*backoff = next
	return true
}
