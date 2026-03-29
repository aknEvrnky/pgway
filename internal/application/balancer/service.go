package balancer

import (
	"context"
	"fmt"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// todo could we use sync.Map?
type Service struct {
	cp       ports.ControlPlaneReader
	mu       sync.RWMutex
	registry map[string]LoadBalancer
}

func NewService(
	cp ports.ControlPlaneReader,
) *Service {
	return &Service{
		cp:       cp,
		registry: make(map[string]LoadBalancer),
	}
}

// Bootstrap is a function that gets all load balancers
// and necessary pools and registers
func (s *Service) Bootstrap(ctx context.Context) error {
	result, err := s.cp.ListBalancers(ctx, domain.ListParams{})

	if err != nil {
		return fmt.Errorf("loading balancers: %w", err)
	}

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

		s.mu.Lock()
		s.registry[lb.Id] = instance
		s.mu.Unlock()
	}

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
		proxies, err := s.cp.GetProxiesByIds(ctx, pool.ProxyIds)
		if err != nil {
			return nil, err
		}

		return proxies, nil

	case domain.PoolTypeDynamic:
		return s.cp.FindProxiesByLabels(ctx, pool.Selector.Allow)

	default:
		return nil, fmt.Errorf("unknown pool type: %q", pool.Type)
	}
}
