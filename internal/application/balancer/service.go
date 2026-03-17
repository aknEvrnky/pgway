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
	lbRepo   ports.LoadBalancerRepositoryPort
	poolRepo ports.PoolRepositoryPort
	mu       sync.RWMutex
	registry map[string]LoadBalancer
}

func NewService(lb ports.LoadBalancerRepositoryPort, p ports.PoolRepositoryPort) *Service {
	return &Service{
		lbRepo:   lb,
		poolRepo: p,
		registry: make(map[string]LoadBalancer),
	}
}

// Bootstrap is a function that gets all load balancers
// and necessary pools and registers
func (s *Service) Bootstrap(ctx context.Context) error {
	lbs, err := s.lbRepo.GetAll(ctx)

	if err != nil {
		return fmt.Errorf("loading balancers: %w", err)
	}

	for _, lb := range lbs {
		pool, err := s.poolRepo.Find(ctx, lb.PoolId)

		if err != nil {
			return fmt.Errorf("loading pool: %w", err)
		}

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
