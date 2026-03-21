package controlplane

import "github.com/aknEvrnky/pgway/internal/ports"

type Service struct {
	proxyRepo ports.ProxyRepositoryPort
}

func NewService(proxyRepo ports.ProxyRepositoryPort) *Service {
	return &Service{proxyRepo: proxyRepo}
}
