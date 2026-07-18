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

	// Pins the T2 interim gate: agents authenticate fine but stay locked out
	// until the per-method policy map exists (T6, #44). PermissionDenied, not
	// Unauthenticated — the token itself is valid.
	t.Run("agent principal denied until policy map lands", func(t *testing.T) {
		agentPrincipal := &domain.Principal{Agent: &domain.Agent{Id: "edge-1"}}
		_, err := callUnaryAs(t, agentPrincipal, ctxWithAuthHeader("Bearer pgw_valid"), protectedMethod)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
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
		} {
			_, err := callUnary(t, context.Background(), method)
			assert.NoError(t, err, "method %s should be exempt", method)
		}
	})
}

// TestExemptList pins the exemption list: only the two credential-issuing
// RPCs may ever bypass authentication.
func TestExemptList(t *testing.T) {
	assert.Len(t, exemptMethods, 2)
	assert.True(t, isExempt("/pgway.controlplane.v1.AuthService/Login"))
	assert.True(t, isExempt("/pgway.controlplane.v1.AuthService/InitAdmin"))

	assert.False(t, isExempt("/pgway.controlplane.v1.AuthService/Logout"))
	assert.False(t, isExempt("/pgway.controlplane.v1.UserService/CreateUser"))
	assert.False(t, isExempt(protectedMethod))
}
