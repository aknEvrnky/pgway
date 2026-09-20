package probes

import (
	"context"
	"sync"

	"github.com/aknEvrnky/pgway/internal/ports"
)

const (
	ReasonOK                 = "ok"
	ReasonShuttingDown       = "shutting_down"
	ReasonStorageUnavailable = "storage_unavailable"
	ReasonGRPCNotServing     = "grpc_not_serving"
	ReasonCPUnreachable      = "cp_unreachable"
)

// ReadyGateConfig configures which readiness checks apply for a binary.
type ReadyGateConfig struct {
	Storage     ports.StoragePinger // nil = skip
	Link        ports.CPLinkStatus  // nil = skip
	RequireGRPC bool                // true for pgway / pgway-cp
}

// ReadyGate evaluates process readiness for /readyz.
type ReadyGate struct {
	storage     ports.StoragePinger
	link        ports.CPLinkStatus
	requireGRPC bool

	mu           sync.Mutex
	shuttingDown bool
	grpcServing  bool
}

// NewReadyGate builds a gate; starts not-ready when RequireGRPC is set.
func NewReadyGate(cfg ReadyGateConfig) *ReadyGate {
	return &ReadyGate{
		storage:     cfg.Storage,
		link:        cfg.Link,
		requireGRPC: cfg.RequireGRPC,
	}
}

// MarkShuttingDown flips readiness to not-ready for orchestrator drain.
func (g *ReadyGate) MarkShuttingDown() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.shuttingDown = true
}

// MarkGRPCServing records whether the control-plane gRPC server is accepting.
func (g *ReadyGate) MarkGRPCServing(serving bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.grpcServing = serving
}

// Check returns ready=true with reason "ok", or ready=false with a short reason.
func (g *ReadyGate) Check(ctx context.Context) (ready bool, reason string) {
	g.mu.Lock()
	shuttingDown := g.shuttingDown
	grpcServing := g.grpcServing
	requireGRPC := g.requireGRPC
	storage := g.storage
	link := g.link
	g.mu.Unlock()

	if shuttingDown {
		return false, ReasonShuttingDown
	}
	if storage != nil {
		if err := storage.Ping(ctx); err != nil {
			return false, ReasonStorageUnavailable
		}
	}
	if requireGRPC && !grpcServing {
		return false, ReasonGRPCNotServing
	}
	if link != nil {
		snap := link.Snapshot()
		// Align with fail_closed only: fail_open keeps serving stale traffic,
		// so readiness must stay ready or K8s would drain working pods.
		if snap.Rejecting {
			return false, ReasonCPUnreachable
		}
	}
	return true, ReasonOK
}
