package auth

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type contextKey int

const (
	principalContextKey contextKey = iota
	tokenContextKey
)

// ContextWithPrincipal carries the authenticated principal (user or agent)
// into the handler context. Set only by transport auth middleware.
func ContextWithPrincipal(ctx context.Context, principal *domain.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func PrincipalFromContext(ctx context.Context) (*domain.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(*domain.Principal)
	return principal, ok
}

// ContextWithToken carries the raw bearer token so handlers like Logout can
// revoke the exact credential the call was made with.
func ContextWithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, tokenContextKey, token)
}

func TokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(tokenContextKey).(string)
	return token, ok
}
