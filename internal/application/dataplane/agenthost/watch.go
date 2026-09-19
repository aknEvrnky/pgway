package agenthost

import (
	"context"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

// WatchOptions tunes RunWatch. Zero value is fine for production.
type WatchOptions struct {
	// OnConnected is invoked after the watch stream is established and
	// afterConnect (bootstrap) has succeeded — useful for test handshakes
	// and LinkState.MarkWatchUp.
	OnConnected func()
	// OnDisconnected is invoked when a watch session ends (before backoff),
	// including after a successful session — useful for LinkState.MarkWatchDown.
	OnDisconnected func()

	// InitialBackoff is the first reconnect delay (default 1s).
	InitialBackoff time.Duration
	// MaxBackoff caps exponential growth (default 30s).
	MaxBackoff time.Duration
}

// RunWatch reconnects forever until ctx is canceled.
//
// Per connection the sequence is:
//  1. Open the authenticated watch stream (server registers the subscriber)
//  2. afterConnect — typically a full Bootstrap to close missed-event gaps
//  3. OnConnected (optional)
//  4. Consume change hints until the stream ends
//
// Transient failures back off; Unauthenticated is fatal.
func RunWatch(ctx context.Context, log *zap.Logger, watch ports.AgentChangeWatcher, afterConnect func(context.Context) error, opts WatchOptions) error {
	if log == nil {
		log = zap.NewNop()
	}
	initial := opts.InitialBackoff
	if initial <= 0 {
		initial = time.Second
	}
	maxBackoff := opts.MaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}
	backoff := initial

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		sessionOK := false
		err := watch.Watch(ctx, func(c context.Context) error {
			if afterConnect != nil {
				if err := afterConnect(c); err != nil {
					return err
				}
			}
			sessionOK = true
			if opts.OnConnected != nil {
				opts.OnConnected()
			}
			return nil
		})
		if opts.OnDisconnected != nil {
			opts.OnDisconnected()
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if isAuthRejected(err) {
			return fatalAuthError(err)
		}
		if err != nil {
			log.Warn("watch stream ended", zap.Error(err))
		}

		// Reset after a successful session; keep growing across rapid failures.
		if sessionOK {
			backoff = initial
		}
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
