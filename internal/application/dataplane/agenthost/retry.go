package agenthost

import (
	"context"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

// RetryOptions tunes RetryTransient. Zero value is fine for production.
type RetryOptions struct {
	// InitialBackoff is the first retry delay (default 1s).
	InitialBackoff time.Duration
	// MaxBackoff caps exponential growth (default 30s).
	MaxBackoff time.Duration
}

// RetryTransient calls fn until it succeeds, ctx is canceled, or isFatal
// reports true for the returned error. Transient failures are logged and
// retried with capped exponential backoff — CP unreachable at DP startup
// must not exit the process.
func RetryTransient(ctx context.Context, log *zap.Logger, what string, fn func(context.Context) error, isFatal func(error) bool, opts RetryOptions) error {
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
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if isFatal != nil && isFatal(err) {
			return err
		}
		log.Warn(what+" failed; retrying", zap.Error(err), zap.Duration("backoff", backoff))
		if !sleepBackoff(ctx, &backoff, maxBackoff) {
			return ctx.Err()
		}
	}
}

// BootstrapCredentialsWithRetry runs BootstrapCredentials until success,
// cancellation, or a permanent error (see IsPermanentBootstrapError).
func BootstrapCredentialsWithRetry(
	ctx context.Context,
	log *zap.Logger,
	store ports.AgentStateStore,
	regToken string,
	agent domain.Agent,
	reg ports.AgentRegistrar,
	opts RetryOptions,
) (*BootstrapResult, error) {
	var boot *BootstrapResult
	err := RetryTransient(ctx, log, "agent bootstrap", func(c context.Context) error {
		b, err := BootstrapCredentials(c, store, regToken, agent, reg)
		if err != nil {
			return err
		}
		boot = b
		return nil
	}, IsPermanentBootstrapError, opts)
	if err != nil {
		return nil, err
	}
	return boot, nil
}
