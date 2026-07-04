package auth

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type contextKey int

const (
	userContextKey contextKey = iota
	tokenContextKey
)

func ContextWithUser(ctx context.Context, user *domain.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(*domain.User)
	return user, ok
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
