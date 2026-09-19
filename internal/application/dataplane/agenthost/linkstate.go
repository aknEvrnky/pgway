package agenthost

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

// LinkStateConfig tunes CP reachability evaluation for distributed DP.
type LinkStateConfig struct {
	Strategy             string
	UnreachableThreshold time.Duration
	RecoverThreshold     time.Duration
	Log                  *zap.Logger
}

// LinkState tracks heartbeat and watch proofs and exposes ports.CPLinkStatus.
// Snapshot is lock-free while the published view cannot advance on wall-clock
// alone; mutations and time-crossing reads recompute under a mutex.
type LinkState struct {
	mu sync.Mutex

	strategy             string
	unreachableThreshold time.Duration
	recoverThreshold     time.Duration
	log                  *zap.Logger

	lastHB         time.Time
	watchUp        bool
	bothStaleSince time.Time
	recoverSince   time.Time
	unreachable    bool
	lastState      string // last logged observable state

	snap atomic.Pointer[ports.CPLinkSnapshot]

	now func() time.Time
}

var _ ports.CPLinkStatus = (*LinkState)(nil)

// NewLinkState builds a link-state tracker. Zero thresholds are clamped to
// sensible defaults matching process config.
func NewLinkState(cfg LinkStateConfig) *LinkState {
	if cfg.Strategy == "" {
		cfg.Strategy = ports.CPDisconnectFailOpen
	}
	if cfg.UnreachableThreshold <= 0 {
		cfg.UnreachableThreshold = 30 * time.Second
	}
	log := cfg.Log
	if log == nil {
		log = zap.NewNop()
	}
	s := &LinkState{
		strategy:             cfg.Strategy,
		unreachableThreshold: cfg.UnreachableThreshold,
		recoverThreshold:     cfg.RecoverThreshold,
		log:                  log,
		now:                  time.Now,
	}
	return s
}

// TouchHeartbeat records a successful heartbeat RPC.
func (s *LinkState) TouchHeartbeat() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastHB = s.now()
	s.recomputeLocked()
}

// MarkWatchUp records that the watch stream is open after a successful Resync.
func (s *LinkState) MarkWatchUp() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watchUp = true
	s.recomputeLocked()
}

// MarkWatchDown records that the watch stream ended.
func (s *LinkState) MarkWatchDown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watchUp = false
	s.recomputeLocked()
}

// Snapshot returns the current CP link view. The common healthy / latched
// unreachable paths avoid the mutation mutex.
func (s *LinkState) Snapshot() ports.CPLinkSnapshot {
	if published := s.snap.Load(); published != nil && !s.timeMayAdvance(*published) {
		return *published
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recomputeLocked()
	if published := s.snap.Load(); published != nil {
		return *published
	}
	return ports.CPLinkSnapshot{State: ports.CPLinkDegraded, Strategy: s.strategy}
}

// timeMayAdvance reports whether wall-clock progress alone could change state
// relative to the last published snapshot (enter/leave unreachable).
func (s *LinkState) timeMayAdvance(snap ports.CPLinkSnapshot) bool {
	now := s.now()
	hbStale := snap.LastHeartbeat.IsZero() || now.Sub(snap.LastHeartbeat) > s.unreachableThreshold
	watchStale := !snap.WatchUp
	bothStale := hbStale && watchStale

	switch snap.State {
	case ports.CPLinkUnreachable:
		// Still both-stale → latched; no lock needed.
		// Proofs look good → recover timer may elapse.
		return !bothStale
	default:
		// Both stale → may enter unreachable after threshold.
		return bothStale
	}
}

func (s *LinkState) observableStateLocked() string {
	hbStale, watchStale := s.proofsStaleLocked()
	switch {
	case s.unreachable:
		return ports.CPLinkUnreachable
	case hbStale || watchStale:
		return ports.CPLinkDegraded
	default:
		return ports.CPLinkConnected
	}
}

func (s *LinkState) proofsStaleLocked() (hbStale, watchStale bool) {
	now := s.now()
	hbStale = s.lastHB.IsZero() || now.Sub(s.lastHB) > s.unreachableThreshold
	watchStale = !s.watchUp
	return hbStale, watchStale
}

func (s *LinkState) recomputeLocked() {
	now := s.now()
	hbStale, watchStale := s.proofsStaleLocked()
	bothStale := hbStale && watchStale

	if bothStale {
		if s.bothStaleSince.IsZero() {
			s.bothStaleSince = now
		}
		s.recoverSince = time.Time{}
		if now.Sub(s.bothStaleSince) >= s.unreachableThreshold {
			s.unreachable = true
		}
	} else {
		s.bothStaleSince = time.Time{}
		if s.unreachable {
			if hbStale || watchStale {
				s.recoverSince = time.Time{}
			} else {
				if s.recoverSince.IsZero() {
					s.recoverSince = now
				}
				if now.Sub(s.recoverSince) >= s.recoverThreshold {
					s.unreachable = false
					s.recoverSince = time.Time{}
				}
			}
		}
	}

	s.logTransitionLocked()
	s.publishLocked()
}

func (s *LinkState) publishLocked() {
	state := s.observableStateLocked()
	retryAfter := int(s.unreachableThreshold / time.Second)
	if retryAfter < 1 {
		retryAfter = 1
	}
	snap := &ports.CPLinkSnapshot{
		State:             state,
		Strategy:          s.strategy,
		Rejecting:         s.unreachable && s.strategy == ports.CPDisconnectFailClosed,
		RetryAfterSeconds: retryAfter,
		LastHeartbeat:     s.lastHB,
		WatchUp:           s.watchUp,
	}
	s.snap.Store(snap)
}

func (s *LinkState) logTransitionLocked() {
	state := s.observableStateLocked()
	if state == s.lastState {
		return
	}
	from := s.lastState
	if from == "" {
		from = "init"
	}
	s.lastState = state
	rejecting := s.unreachable && s.strategy == ports.CPDisconnectFailClosed
	s.log.Info("cp link state changed",
		zap.String("from", from),
		zap.String("to", state),
		zap.String("strategy", s.strategy),
		zap.Bool("rejecting", rejecting),
		zap.Bool("watch_up", s.watchUp),
	)
}
