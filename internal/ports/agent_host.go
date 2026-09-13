package ports

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// ErrAgentUnauthenticated signals a rejected/expired agent bearer.
// gRPC adapters should wrap Unauthenticated with this sentinel so core
// can classify without importing gRPC.
var ErrAgentUnauthenticated = errors.New("agent unauthenticated")

// AgentHostCredentials are CP-issued credentials persisted on the DP host.
type AgentHostCredentials struct {
	AgentID    string
	AgentToken string
}

// AgentStateStore loads and saves DP agent credentials.
type AgentStateStore interface {
	// Load returns the persisted credentials, or an error matching
	// fs.ErrNotExist when none are stored yet.
	Load(ctx context.Context) (*AgentHostCredentials, error)
	Save(ctx context.Context, creds *AgentHostCredentials) error
}

// AgentProcessLock ensures at most one DP process per state directory.
type AgentProcessLock interface {
	// Acquire takes an exclusive non-blocking lock. The returned Closer
	// releases it (also released by the kernel on process death).
	Acquire() (io.Closer, error)
}

// AgentRegistrar exchanges a registration token for agent credentials.
type AgentRegistrar interface {
	Register(ctx context.Context, regToken string, agent domain.Agent) (*domain.Agent, string, error)
}

// AgentHeartbeater sends liveness signals; identity comes from the bearer.
type AgentHeartbeater interface {
	Heartbeat(ctx context.Context) (time.Time, error)
}

// AgentChangeWatcher opens a change stream until it fails or ctx is canceled.
type AgentChangeWatcher interface {
	Watch(ctx context.Context) error
}
