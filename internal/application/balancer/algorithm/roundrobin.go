package algorithm

import (
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

	r.window = uint32(len(pool.Proxies))

	if r.window == 0 {
		return nil, domain.ErrNoProxy
	}

	return r, nil
}

func (r *RoundRobin) Next() (*domain.Proxy, error) {
	val := (r.counter.Add(1) - 1) % r.window
	proxy := r.pool.Proxies[val]
	return proxy, nil
}
