package agenthost

import (
	"context"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

// RunWatch reconnects forever until ctx is canceled. onConnect runs before
// each stream (full reload to close missed-event gaps). Transient failures
// back off; Unauthenticated is fatal.
func RunWatch(ctx context.Context, log *zap.Logger, watch ports.AgentChangeWatcher, onConnect func(context.Context) error) error {
	if log == nil {
		log = zap.NewNop()
	}
	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if onConnect != nil {
			if err := onConnect(ctx); err != nil {
				if isAuthRejected(err) {
					return fatalAuthError(err)
				}
				log.Warn("watch reconnect reload failed", zap.Error(err))
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
		if isAuthRejected(err) {
			return fatalAuthError(err)
		}
		if err != nil {
			log.Warn("watch stream ended", zap.Error(err))
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
