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
	default:
		return nil, fmt.Errorf("unknown balancer type %q", lb.Type)
	}
}
