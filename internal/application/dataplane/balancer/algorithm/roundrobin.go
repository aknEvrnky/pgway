package algorithm

import (
	"fmt"
	"sync/atomic"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type RoundRobin struct {
	pool    *domain.Pool
	window  uint32
	counter atomic.Uint32
}

func (r *RoundRobin) Release(result domain.BalancerResult) {
	// nothing to do for round-robin
}

func NewRoundRobin(pool *domain.Pool) (*RoundRobin, error) {
	r := &RoundRobin{
		pool: pool,
	}

	if pool == nil {
		return nil, domain.ErrNoPool
	}

	if !pool.HasProxiesResolved() {
		return nil, fmt.Errorf("pool %q proxies not resolved", pool.Id)
	}

	// Empty is allowed at build/bootstrap (e.g. dynamic pool with no label
	// matches yet). Next() returns ErrNoProxy until proxies appear and the
	// balancer is rebuilt.
	r.window = uint32(len(pool.ResolvedProxies()))

	return r, nil
}

func (r *RoundRobin) Next() (*domain.Proxy, error) {
	if r.window == 0 {
		return nil, domain.ErrNoProxy
	}
	val := (r.counter.Add(1) - 1) % r.window
	proxy := r.pool.ResolvedProxies()[val]
	return proxy, nil
}
