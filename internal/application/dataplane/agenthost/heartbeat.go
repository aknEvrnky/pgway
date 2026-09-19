package agenthost

import (
	"context"
	"errors"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

// HeartbeatOptions tunes RunHeartbeat. Zero value is fine for production.
type HeartbeatOptions struct {
	// OnSuccess is invoked after a successful Heartbeat RPC (e.g. LinkState.TouchHeartbeat).
	OnSuccess func()
}

// RunHeartbeat ticks until ctx is canceled. Transient errors are logged;
// Unauthenticated is fatal (token revoked/expired — admin must mint a new
// registration token).
func RunHeartbeat(ctx context.Context, log *zap.Logger, hb ports.AgentHeartbeater, interval time.Duration, opts HeartbeatOptions) error {
	if log == nil {
		log = zap.NewNop()
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			expiresAt, err := hb.Heartbeat(ctx)
			if err != nil {
				if isAuthRejected(err) {
					return fatalAuthError(err)
				}
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				log.Warn("agent heartbeat failed", zap.Error(err))
				continue
			}
			if opts.OnSuccess != nil {
				opts.OnSuccess()
			}
			log.Info("agent heartbeat ok",
				zap.Time("token_expires_at", expiresAt),
			)
		}
	}
}

func isAuthRejected(err error) bool {
	return errors.Is(err, ports.ErrAgentUnauthenticated)
}

func fatalAuthError(err error) error {
	return errors.Join(err, errors.New(
		"agent token rejected; delete state file and re-register with a fresh PGWAY_AGENT_REGISTRATION_TOKEN",
	))
}
