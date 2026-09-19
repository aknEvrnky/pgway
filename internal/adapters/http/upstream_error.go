package http

import (
	"context"
	"errors"
	"net"
	"net/http"
	"syscall"
)

// classifyUpstreamError maps dial/round-trip failures to an HTTP status and a
// short client-facing message. Full error detail belongs in logs, not the body.
func classifyUpstreamError(err error) (status int, clientMsg string) {
	if err == nil {
		return http.StatusBadGateway, "bad gateway"
	}

	if errors.Is(err, context.Canceled) {
		return http.StatusBadGateway, "bad gateway"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, "gateway timeout"
	}

	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return http.StatusGatewayTimeout, "gateway timeout"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return http.StatusBadGateway, "dns lookup failed"
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		return http.StatusBadGateway, "connection refused"
	}

	return http.StatusBadGateway, "bad gateway"
}
