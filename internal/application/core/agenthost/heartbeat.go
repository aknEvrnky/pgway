package agenthost

import (
	"context"
	"errors"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

// RunHeartbeat ticks until ctx is canceled. Transient errors are logged;
// Unauthenticated is fatal (token revoked/expired — admin must mint a new
// registration token).
func RunHeartbeat(ctx context.Context, hb ports.AgentHeartbeater, interval time.Duration) error {
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
				zap.L().Warn("agent heartbeat failed", zap.Error(err))
				continue
			}
			zap.L().Debug("agent heartbeat ok", zap.Time("token_expires_at", expiresAt))
		}
	}
}

func isAuthRejected(err error) bool {
	return errors.Is(err, ports.ErrAgentUnauthenticated)
}

func fatalAuthError(err error) error {
	return errors.Join(err, errors.New(
		"agent token rejected; delete state file and re-register with a fresh PGWAY_REGISTRATION_TOKEN",
	))
}
