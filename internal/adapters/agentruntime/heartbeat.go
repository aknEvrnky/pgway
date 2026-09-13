package agentruntime

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Heartbeater sends periodic Heartbeat RPCs. Identity comes from the bearer token.
type Heartbeater interface {
	Heartbeat(ctx context.Context) (time.Time, error)
}

// RunHeartbeat ticks until ctx is canceled. Transient errors are logged;
// Unauthenticated is fatal (token revoked/expired — admin must mint a new
// registration token).
func RunHeartbeat(ctx context.Context, hb Heartbeater, interval time.Duration) error {
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
				if isUnauthenticated(err) {
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

func isUnauthenticated(err error) bool {
	return status.Code(err) == codes.Unauthenticated
}

func fatalAuthError(err error) error {
	return errors.Join(err, errors.New(
		"agent token rejected; delete state file and re-register with a fresh PGWAY_REGISTRATION_TOKEN",
	))
}
