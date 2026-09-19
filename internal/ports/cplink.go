package ports

import (
	"context"
	"time"
)

// CP link reachability states for distributed data-plane agents.
const (
	CPLinkConnected   = "connected"
	CPLinkDegraded    = "degraded"
	CPLinkUnreachable = "unreachable"
)

// CP disconnect strategies.
const (
	CPDisconnectFailOpen   = "fail_open"
	CPDisconnectFailClosed = "fail_closed"
)

// CPLinkSnapshot is a point-in-time view of control-plane reachability.
type CPLinkSnapshot struct {
	State             string
	Strategy          string
	Rejecting         bool // true when fail_closed and unreachable
	RetryAfterSeconds int  // hint for Retry-After when Rejecting (from unreachable threshold)
	LastHeartbeat     time.Time
	WatchUp           bool
}

// CPLinkStatus exposes CP reachability for the HTTP fail gate and future
// readiness probes (#60). Implementations must be safe for concurrent use.
type CPLinkStatus interface {
	Snapshot() CPLinkSnapshot
}

// ListenerReconciler aligns running entrypoint listeners with the DP cache
// after a full Bootstrap (Watch reconnect / periodic all-in-one resync).
type ListenerReconciler interface {
	ReconcileListeners(ctx context.Context) error
}
