package balancer

import (
	"context"
	"fmt"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

type Service struct {
	cp       ports.ControlPlaneReader
	resolver ports.ProxyResolver
	mu       sync.RWMutex
	registry map[string]LoadBalancer
}

func NewService(
	cp ports.ControlPlaneReader,
	resolver ports.ProxyResolver,
) *Service {
	return &Service{
		cp:       cp,
		resolver: resolver,
		registry: make(map[string]LoadBalancer),
	}
}

// Bootstrap reloads all load balancers from the control plane into a fresh
// registry (deleted balancers are dropped).
func (s *Service) Bootstrap(ctx context.Context) error {
	result, err := s.cp.ListBalancers(ctx, domain.ListParams{}, domain.BalancerFilter{})

	if err != nil {
		return fmt.Errorf("loading balancers: %w", err)
	}

	next := make(map[string]LoadBalancer, len(result.Items))

	for _, lb := range result.Items {
		pool, err := s.cp.GetPool(ctx, lb.PoolId)

		if err != nil {
			return fmt.Errorf("loading pool: %w", err)
		}

		// load proxies for pool
		proxies, err := s.resolveProxies(ctx, pool)

		if err != nil {
			return fmt.Errorf("resolving proxies: %w", err)
		}

		pool.LoadResolvedProxies(proxies)

		instance, err := Build(lb, pool)

		if err != nil {
			return fmt.Errorf("balancer %q: %w", lb.Id, err)
		}

		next[lb.Id] = instance
	}

	s.mu.Lock()
	s.registry = next
	s.mu.Unlock()

	return nil
}

func (s *Service) Get(id string) (LoadBalancer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, ok := s.registry[id]

	if !ok {
		return nil, ErrBalancerNotFound
	}

	return instance, nil
}

// Next returns the next proxy for given load balancer ID
func (s *Service) Next(id string) (*domain.Proxy, error) {
	lb, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	return lb.Next()
}

func (s *Service) resolveProxies(ctx context.Context, pool *domain.Pool) ([]*domain.Proxy, error) {
	switch pool.Type {
	case domain.PoolTypeStatic:
		ids := pool.MemberProxyIds()
		proxies, err := s.resolver.GetProxiesByIds(ctx, ids)
		if err != nil {
			return nil, err
		}
		if len(proxies) == 0 {
			return proxies, nil
		}
		return orderProxiesByIDs(proxies, ids)

	case domain.PoolTypeDynamic:
		return s.resolver.FindProxiesByLabels(ctx, pool.Selector.Allow)

	default:
		return nil, fmt.Errorf("unknown pool type: %q", pool.Type)
	}
}

// orderProxiesByIDs returns proxies in the same order as ids.
func orderProxiesByIDs(proxies []*domain.Proxy, ids []string) ([]*domain.Proxy, error) {
	byID := make(map[string]*domain.Proxy, len(proxies))
	for _, p := range proxies {
		byID[p.Id] = p
	}
	ordered := make([]*domain.Proxy, 0, len(ids))
	for _, id := range ids {
		p, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("proxy %q not found", id)
		}
		ordered = append(ordered, p)
	}
	return ordered, nil
}
