package algorithm

import (
	"fmt"
	"sync"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// LeastBytes selects the proxy with the lowest accumulated downstream byte count.
// Counters update only on Release; reset is lazy on Next when resetInterval elapses.
type LeastBytes struct {
	pool          *domain.Pool
	counters      map[string]int64
	resetInterval time.Duration
	lastReset     time.Time
	mu            sync.Mutex
}

func NewLeastBytes(pool *domain.Pool, resetInterval time.Duration) (*LeastBytes, error) {
	if pool == nil {
		return nil, domain.ErrNoPool
	}
	if !pool.HasProxiesResolved() {
		return nil, fmt.Errorf("pool %q proxies not resolved", pool.Id)
	}
	if len(pool.ResolvedProxies()) == 0 {
		return nil, domain.ErrNoProxy
	}
	if resetInterval <= 0 {
		resetInterval = time.Minute
	}

	return &LeastBytes{
		pool:          pool,
		counters:      make(map[string]int64),
		resetInterval: resetInterval,
		lastReset:     time.Now(),
	}, nil
}

func (l *LeastBytes) Next() (*domain.Proxy, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	proxies := l.pool.ResolvedProxies()
	if len(proxies) == 0 {
		return nil, domain.ErrNoProxy
	}

	now := time.Now()
	if now.Sub(l.lastReset) >= l.resetInterval {
		l.counters = make(map[string]int64)
		l.lastReset = now
	}

	best := 0
	bestBytes := l.counters[proxies[0].Id]
	for i := 1; i < len(proxies); i++ {
		n := l.counters[proxies[i].Id]
		if n < bestBytes {
			best = i
			bestBytes = n
		}
	}
	return proxies[best], nil
}

func (l *LeastBytes) Release(result domain.BalancerResult) {
	if result.Bytes <= 0 || result.ProxyId == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.counters[result.ProxyId] += result.Bytes
}
