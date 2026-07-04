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
	user       *domain.User
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, token string) (*domain.User, error) {
	if token == f.validToken {
		return f.user, nil
	}
	return nil, auth.ErrInvalidToken
}

func ctxWithAuthHeader(value string) context.Context {
	md := metadata.New(map[string]string{"authorization": value})
	return metadata.NewIncomingContext(context.Background(), md)
}

func callUnary(t *testing.T, ctx context.Context, method string) (context.Context, error) {
	t.Helper()

	authenticator := &fakeAuthenticator{
		validToken: "pgw_valid",
		user:       &domain.User{Id: "alice", Role: domain.RoleAdmin},
	}

	var handlerCtx context.Context
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCtx = ctx
		return "ok", nil
	}

	_, err := UnaryAuth(authenticator)(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method}, handler)
	return handlerCtx, err
}

const protectedMethod = "/pgway.controlplane.v1.ProxyService/ListProxies"

func TestUnaryAuth(t *testing.T) {
	t.Run("valid token injects user and token into context", func(t *testing.T) {
		ctx, err := callUnary(t, ctxWithAuthHeader("Bearer pgw_valid"), protectedMethod)
		require.NoError(t, err)

		user, ok := auth.UserFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "alice", user.Id)

		token, ok := auth.TokenFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, "pgw_valid", token)
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
