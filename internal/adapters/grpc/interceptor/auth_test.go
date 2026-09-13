package interceptor

import (
	"context"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeAuthenticator struct {
	validToken string
	principal  *domain.Principal
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, token string) (*domain.Principal, error) {
	if token == f.validToken {
		return f.principal, nil
	}
	return nil, auth.ErrInvalidToken
}

func ctxWithAuthHeader(value string) context.Context {
	md := metadata.New(map[string]string{"authorization": value})
	return metadata.NewIncomingContext(context.Background(), md)
}

func callUnaryAs(t *testing.T, principal *domain.Principal, ctx context.Context, method string) (context.Context, error) {
	t.Helper()

	authenticator := &fakeAuthenticator{
		validToken: "pgw_valid",
		principal:  principal,
	}

	var handlerCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCtx = ctx
		return "ok", nil
	}

	_, err := UnaryAuth(authenticator)(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method}, handler)
	return handlerCtx, err
}

func callUnary(t *testing.T, ctx context.Context, method string) (context.Context, error) {
	t.Helper()
	return callUnaryAs(t, &domain.Principal{
		User: &domain.User{Id: "alice", Role: domain.RoleAdmin},
	}, ctx, method)
}

const protectedMethod = "/pgway.controlplane.v1.ProxyService/ListProxies"
const agentWriteMethod = "/pgway.controlplane.v1.ProxyService/ApplyProxyV1"
const agentHeartbeatMethod = "/pgway.controlplane.v1.AgentService/Heartbeat"

func TestUnaryAuth(t *testing.T) {
	t.Run("valid token injects principal and token into context", func(t *testing.T) {
		ctx, err := callUnary(t, ctxWithAuthHeader("Bearer pgw_valid"), protectedMethod)
		require.NoError(t, err)

		principal, ok := auth.PrincipalFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, domain.PrincipalKindUser, principal.Kind())
		assert.Equal(t, "alice", principal.User.Id)

		token, ok := auth.TokenFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "pgw_valid", token)
	})

	t.Run("agent principal allowed on heartbeat", func(t *testing.T) {
		agentPrincipal := &domain.Principal{Agent: &domain.Agent{Id: "edge-1"}}
		ctx, err := callUnaryAs(t, agentPrincipal, ctxWithAuthHeader("Bearer pgw_valid"), agentHeartbeatMethod)
		require.NoError(t, err)
		principal, ok := auth.PrincipalFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, domain.PrincipalKindAgent, principal.Kind())
	})

	t.Run("agent principal allowed on read-only list", func(t *testing.T) {
		agentPrincipal := &domain.Principal{Agent: &domain.Agent{Id: "edge-1"}}
		_, err := callUnaryAs(t, agentPrincipal, ctxWithAuthHeader("Bearer pgw_valid"), protectedMethod)
		assert.NoError(t, err)
	})

	t.Run("agent principal denied on write RPC", func(t *testing.T) {
		agentPrincipal := &domain.Principal{Agent: &domain.Agent{Id: "edge-1"}}
		_, err := callUnaryAs(t, agentPrincipal, ctxWithAuthHeader("Bearer pgw_valid"), agentWriteMethod)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
	})

	t.Run("user principal still allowed on write RPC", func(t *testing.T) {
		_, err := callUnary(t, ctxWithAuthHeader("Bearer pgw_valid"), agentWriteMethod)
		assert.NoError(t, err)
	})

	t.Run("missing metadata", func(t *testing.T) {
		_, err := callUnary(t, context.Background(), protectedMethod)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("missing authorization header", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.New(nil))
		_, err := callUnary(t, ctx, protectedMethod)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("non-bearer authorization", func(t *testing.T) {
		_, err := callUnary(t, ctxWithAuthHeader("Basic dXNlcjpwYXNz"), protectedMethod)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := callUnary(t, ctxWithAuthHeader("Bearer pgw_wrong"), protectedMethod)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	t.Run("exempt methods skip authentication", func(t *testing.T) {
		for _, method := range []string{
			"/pgway.controlplane.v1.AuthService/Login",
			"/pgway.controlplane.v1.AuthService/InitAdmin",
			"/pgway.controlplane.v1.AgentService/Register",
		} {
			_, err := callUnary(t, context.Background(), method)
			assert.NoError(t, err, "method %s should be exempt", method)
		}
	})
}

func TestExemptList(t *testing.T) {
	assert.Len(t, exemptMethods, 3)
	assert.True(t, isExempt("/pgway.controlplane.v1.AuthService/Login"))
	assert.True(t, isExempt("/pgway.controlplane.v1.AuthService/InitAdmin"))
	assert.True(t, isExempt("/pgway.controlplane.v1.AgentService/Register"))

	assert.False(t, isExempt("/pgway.controlplane.v1.AuthService/Logout"))
	assert.False(t, isExempt("/pgway.controlplane.v1.UserService/CreateUser"))
	assert.False(t, isExempt(protectedMethod))
}

func TestAgentAllowedList(t *testing.T) {
	assert.True(t, isAgentAllowed(agentHeartbeatMethod))
	assert.True(t, isAgentAllowed("/pgway.controlplane.v1.AgentService/Deregister"))
	assert.True(t, isAgentAllowed("/pgway.controlplane.v1.ChangeService/Watch"))
	assert.True(t, isAgentAllowed(protectedMethod))
	assert.False(t, isAgentAllowed(agentWriteMethod))
	assert.False(t, isAgentAllowed("/pgway.controlplane.v1.AgentService/DeleteAgent"))
}
