package controlplane

import "github.com/aknEvrnky/pgway/internal/ports"

type Service struct {
	proxyRepo  ports.ProxyRepositoryPort
	poolRepo   ports.PoolRepositoryPort
	lbRepo     ports.LoadBalancerRepositoryPort
	routerRepo ports.RouterRepositoryPort
}

func NewService(
	proxyRepo ports.ProxyRepositoryPort,
	poolRepo ports.PoolRepositoryPort,
	lbRepo ports.LoadBalancerRepositoryPort,
	routerRepo ports.RouterRepositoryPort,
) *Service {
	return &Service{
		proxyRepo:  proxyRepo,
		poolRepo:   poolRepo,
		lbRepo:     lbRepo,
		routerRepo: routerRepo,
	}
}
