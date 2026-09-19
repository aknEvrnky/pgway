package interceptor

import (
	"context"
	"net"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func callRateLimited(t *testing.T, cfg RateLimitConfig, ctx context.Context) error {
	t.Helper()
	handler := func(context.Context, any) (any, error) { return "ok", nil }
	_, err := UnaryRateLimit(cfg)(ctx, nil, &grpc.UnaryServerInfo{FullMethod: protectedMethod}, handler)
	return err
}

func ctxWithPeerIP(ip string) context.Context {
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.ParseIP(ip), Port: 12345},
	})
}

func ctxWithUser(id string) context.Context {
	return ports.ContextWithPrincipal(context.Background(), &domain.Principal{
		User: &domain.User{Id: id, Role: domain.RoleAdmin},
	})
}

func ctxWithAgent(id string) context.Context {
	return ports.ContextWithPrincipal(context.Background(), &domain.Principal{
		Agent: &domain.Agent{Id: id},
	})
}

func TestUnaryRateLimit_Disabled(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RPS: 0, Burst: 1}
	for range 20 {
		require.NoError(t, callRateLimited(t, cfg, ctxWithUser("alice")))
	}
}

func TestUnaryRateLimit_AllowsUnderBurst(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RPS: 100, Burst: 5}
	ctx := ctxWithUser("alice")
	for range 5 {
		require.NoError(t, callRateLimited(t, cfg, ctx))
	}
}

func TestUnaryRateLimit_RejectsAfterBurst(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RPS: 1, Burst: 2}
	// Very low RPS so refill does not race the burst drain.
	interceptor := UnaryRateLimit(cfg)
	handler := func(context.Context, any) (any, error) { return "ok", nil }
	info := &grpc.UnaryServerInfo{FullMethod: protectedMethod}
	ctx := ctxWithUser("bob")

	for range 2 {
		_, err := interceptor(ctx, nil, info, handler)
		require.NoError(t, err)
	}
	_, err := interceptor(ctx, nil, info, handler)
	require.Error(t, err)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestUnaryRateLimit_SeparateBucketsPerPrincipal(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RPS: 1, Burst: 1}
	interceptor := UnaryRateLimit(cfg)
	handler := func(context.Context, any) (any, error) { return "ok", nil }
	info := &grpc.UnaryServerInfo{FullMethod: protectedMethod}

	_, err := interceptor(ctxWithUser("alice"), nil, info, handler)
	require.NoError(t, err)
	_, err = interceptor(ctxWithUser("alice"), nil, info, handler)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))

	_, err = interceptor(ctxWithUser("carol"), nil, info, handler)
	require.NoError(t, err)
}

func TestUnaryRateLimit_UserAndAgentIndependent(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RPS: 1, Burst: 1}
	interceptor := UnaryRateLimit(cfg)
	handler := func(context.Context, any) (any, error) { return "ok", nil }
	info := &grpc.UnaryServerInfo{FullMethod: protectedMethod}

	_, err := interceptor(ctxWithUser("same-id"), nil, info, handler)
	require.NoError(t, err)
	_, err = interceptor(ctxWithAgent("same-id"), nil, info, handler)
	require.NoError(t, err)
}

func TestUnaryRateLimit_PeerIPKey(t *testing.T) {
	t.Parallel()
	cfg := RateLimitConfig{RPS: 1, Burst: 1}
	interceptor := UnaryRateLimit(cfg)
	handler := func(context.Context, any) (any, error) { return "ok", nil }
	info := &grpc.UnaryServerInfo{FullMethod: protectedMethod}

	ctxA := ctxWithPeerIP("10.0.0.1")
	ctxB := ctxWithPeerIP("10.0.0.2")

	_, err := interceptor(ctxA, nil, info, handler)
	require.NoError(t, err)
	_, err = interceptor(ctxA, nil, info, handler)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))

	_, err = interceptor(ctxB, nil, info, handler)
	require.NoError(t, err)
}

func TestRateLimitKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "user:alice", rateLimitKey(ctxWithUser("alice")))
	assert.Equal(t, "agent:edge-1", rateLimitKey(ctxWithAgent("edge-1")))
	assert.Equal(t, "ip:127.0.0.1", rateLimitKey(ctxWithPeerIP("127.0.0.1")))
	assert.Equal(t, "anon", rateLimitKey(context.Background()))
}
