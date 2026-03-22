package controlplane

import "github.com/aknEvrnky/pgway/internal/ports"

type Service struct {
	proxyRepo  ports.ProxyRepositoryPort
	poolRepo   ports.PoolRepositoryPort
	lbRepo     ports.LoadBalancerRepositoryPort
	routerRepo ports.RouterRepositoryPort
	flowRepo   ports.FlowRepositoryPort
}

func NewService(
	proxyRepo ports.ProxyRepositoryPort,
	poolRepo ports.PoolRepositoryPort,
	lbRepo ports.LoadBalancerRepositoryPort,
	routerRepo ports.RouterRepositoryPort,
	flowRepo ports.FlowRepositoryPort,
) *Service {
	return &Service{
		proxyRepo:  proxyRepo,
		poolRepo:   poolRepo,
		lbRepo:     lbRepo,
		routerRepo: routerRepo,
		flowRepo:   flowRepo,
	}
}
