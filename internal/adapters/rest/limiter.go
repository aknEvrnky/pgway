package rest

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultLimiterIdleTTL       = 15 * time.Minute
	defaultLimiterSweepInterval = time.Minute
)

// limiterRegistry tracks one token bucket per client key and evicts buckets
// idle longer than idleTTL so key spaces bounded by client IPs cannot grow
// without bound over process lifetime.
type limiterRegistry struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*limiterEntry

	idleTTL       time.Duration
	sweepInterval time.Duration
	lastSweep     time.Time
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newLimiterRegistry(limit rate.Limit, burst int) *limiterRegistry {
	return &limiterRegistry{
		limit:         limit,
		burst:         burst,
		entries:       make(map[string]*limiterEntry),
		idleTTL:       defaultLimiterIdleTTL,
		sweepInterval: defaultLimiterSweepInterval,
		lastSweep:     time.Now(),
	}
}

// allow reports whether one request for key fits its bucket, creating the
// bucket on first use.
func (r *limiterRegistry) allow(key string) bool {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.entries[key]
	if !ok {
		e = &limiterEntry{limiter: rate.NewLimiter(r.limit, r.burst)}
		r.entries[key] = e
	}
	e.lastSeen = now

	if now.Sub(r.lastSweep) >= r.sweepInterval {
		r.sweep(now)
	}
	return e.limiter.Allow()
}

// sweep drops buckets idle beyond idleTTL. Caller must hold r.mu.
func (r *limiterRegistry) sweep(now time.Time) {
	r.lastSweep = now
	for key, e := range r.entries {
		if now.Sub(e.lastSeen) > r.idleTTL {
			delete(r.entries, key)
		}
	}
}
