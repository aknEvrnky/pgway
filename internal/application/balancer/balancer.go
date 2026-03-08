package balancer

import (
	"errors"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

var (
	ErrBalancerNotFound = errors.New("balancer not found")
)

type LoadBalancer interface {
	Next() (*domain.Proxy, error)
	Release(result domain.BalancerResult)
}
