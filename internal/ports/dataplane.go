package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// ProxyResolver resolves pool members for the data plane.
// Implemented in-process by the controlplane service, or remotely by the gRPC client.
type ProxyResolver interface {
	GetProxiesByIds(ctx context.Context, ids []string) ([]*domain.Proxy, error)
	FindProxiesByLabels(ctx context.Context, labels map[string]string) ([]*domain.Proxy, error)
}
