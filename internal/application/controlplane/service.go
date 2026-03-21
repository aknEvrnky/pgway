package controlplane

import "github.com/aknEvrnky/pgway/internal/ports"

type Service struct {
	proxyRepo ports.ProxyRepositoryPort
	poolRepo  ports.PoolRepositoryPort
}

func NewService(proxyRepo ports.ProxyRepositoryPort, poolRepo ports.PoolRepositoryPort) *Service {
	return &Service{proxyRepo: proxyRepo, poolRepo: poolRepo}
}
