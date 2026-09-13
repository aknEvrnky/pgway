package algorithm

import (
	"fmt"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// Weighted implements nginx-style smooth weighted round-robin.
type Weighted struct {
	proxies []*domain.Proxy
	weights []int
	current []int
	total   int
	mu      sync.Mutex
}

func (w *Weighted) Release(result domain.BalancerResult) {
	// nothing to do for weighted
}

func NewWeighted(pool *domain.Pool) (*Weighted, error) {
	if pool == nil {
		return nil, domain.ErrNoPool
	}
	if pool.Type != domain.PoolTypeStatic {
		return nil, fmt.Errorf("weighted balancer requires static pool %q", pool.Id)
	}
	if !pool.HasProxiesResolved() {
		return nil, fmt.Errorf("pool %q proxies not resolved", pool.Id)
	}

	proxies := pool.ResolvedProxies()
	if len(proxies) == 0 {
		return nil, domain.ErrNoProxy
	}
	if len(proxies) != len(pool.Members) {
		return nil, fmt.Errorf("pool %q resolved proxy count %d != members %d", pool.Id, len(proxies), len(pool.Members))
	}

	weights := make([]int, len(pool.Members))
	total := 0
	for i, m := range pool.Members {
		if m.Weight < 1 {
			return nil, fmt.Errorf("pool %q member %q: weight must be >= 1", pool.Id, m.ProxyId)
		}
		if proxies[i].Id != m.ProxyId {
			return nil, fmt.Errorf("pool %q member order mismatch at %d: want %q got %q", pool.Id, i, m.ProxyId, proxies[i].Id)
		}
		weights[i] = m.Weight
		total += m.Weight
	}

	return &Weighted{
		proxies: proxies,
		weights: weights,
		current: make([]int, len(weights)),
		total:   total,
	}, nil
}

func (w *Weighted) Next() (*domain.Proxy, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	best := -1
	for i := range w.proxies {
		w.current[i] += w.weights[i]
		if best < 0 || w.current[i] > w.current[best] {
			best = i
		}
	}
	w.current[best] -= w.total
	return w.proxies[best], nil
}
