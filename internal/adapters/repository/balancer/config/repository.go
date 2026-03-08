package config

import (
	"context"
	"errors"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
)

var (
	ErrConfigNotLoaded      = errors.New("config is not loaded")
	ErrLoadBalancerNotFound = errors.New("load balancer not found")
)

type ConfigRepository struct {
	balancers map[string]*domain.LoadBalancer
}

func NewConfigRepository(c *config.Config) (*ConfigRepository, error) {
	if c == nil {
		return nil, ErrConfigNotLoaded
	}

	m := make(map[string]*domain.LoadBalancer, len(c.LoadBalancers))
	for _, lb := range c.LoadBalancers {
		mapped := mapToDomain(lb)
		m[mapped.Id] = mapped
	}

	return &ConfigRepository{balancers: m}, nil
}

func (r *ConfigRepository) GetAll(ctx context.Context) ([]*domain.LoadBalancer, error) {
	results := make([]*domain.LoadBalancer, 0, len(r.balancers))

	for _, lb := range r.balancers {
		results = append(results, lb)
	}

	return results, nil
}

func (r *ConfigRepository) Find(ctx context.Context, id string) (*domain.LoadBalancer, error) {
	lb, ok := r.balancers[id]
	if !ok {
		return nil, ErrLoadBalancerNotFound
	}
	return lb, nil
}

func mapToDomain(lb config.LoadBalancerConfig) *domain.LoadBalancer {
	return &domain.LoadBalancer{
		Id:     lb.Id,
		Title:  lb.Title,
		Type:   domain.BalancerType(lb.Type),
		PoolId: lb.Pool,
	}
}
