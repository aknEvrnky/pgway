package ports

import (
	"context"
	"net"
	"net/http"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type ProxyTransportPort interface {
	RoundTrip(ctx context.Context, proxy *domain.Proxy, r *http.Request) (*http.Response, error)
	Dial(ctx context.Context, proxy *domain.Proxy, target string) (net.Conn, error)
}
