package server

import "github.com/aknEvrnky/pgway/internal/adapters/grpc/interceptor"

// RateLimitConfig configures unary CP RPC rate limiting.
// RPS <= 0 disables the limiter.
type RateLimitConfig struct {
	RPS   float64
	Burst int
}

func (c RateLimitConfig) toInterceptor() interceptor.RateLimitConfig {
	return interceptor.RateLimitConfig{RPS: c.RPS, Burst: c.Burst}
}
