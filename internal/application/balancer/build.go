package balancer

import (
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/balancer/algorithm"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func Build(lb *domain.LoadBalancer, p *domain.Pool) (LoadBalancer, error) {
	switch lb.Type {
	case domain.BalancerTypeRoundRobin:
		return algorithm.NewRoundRobin(p)
	case domain.BalancerTypeWeighted:
		return algorithm.NewWeighted(p)
	case domain.BalancerTypeLeastBytes:
		return algorithm.NewLeastBytes(p, lb.ResetInterval)
	default:
		return nil, fmt.Errorf("unknown balancer type %q", lb.Type)
	}
}
