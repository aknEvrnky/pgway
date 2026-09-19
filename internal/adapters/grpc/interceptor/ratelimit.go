package interceptor

import (
	"context"
	"net"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimitConfig configures the unary token-bucket rate limiter.
// RPS <= 0 disables limiting (no-op interceptor).
type RateLimitConfig struct {
	RPS   float64
	Burst int
}

// UnaryRateLimit rejects unary RPCs that exceed the per-client token bucket.
// Authenticated callers are keyed by principal; others by peer IP.
func UnaryRateLimit(cfg RateLimitConfig) grpc.UnaryServerInterceptor {
	if cfg.RPS <= 0 {
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}

	limiters := &sync.Map{}
	limit := rate.Limit(cfg.RPS)
	burst := cfg.Burst

	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		key := rateLimitKey(ctx)
		limiter := loadOrStoreLimiter(limiters, key, limit, burst)
		if !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}

func loadOrStoreLimiter(m *sync.Map, key string, limit rate.Limit, burst int) *rate.Limiter {
	if v, ok := m.Load(key); ok {
		return v.(*rate.Limiter)
	}
	limiter := rate.NewLimiter(limit, burst)
	actual, _ := m.LoadOrStore(key, limiter)
	return actual.(*rate.Limiter)
}

func rateLimitKey(ctx context.Context) string {
	if p, ok := ports.PrincipalFromContext(ctx); ok {
		switch p.Kind() {
		case domain.PrincipalKindUser:
			if p.User != nil && p.User.Id != "" {
				return "user:" + p.User.Id
			}
		case domain.PrincipalKindAgent:
			if p.Agent != nil && p.Agent.Id != "" {
				return "agent:" + p.Agent.Id
			}
		}
	}

	if pr, ok := peer.FromContext(ctx); ok && pr.Addr != nil {
		host := pr.Addr.String()
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "" {
			return "ip:" + host
		}
	}

	return "anon"
}
