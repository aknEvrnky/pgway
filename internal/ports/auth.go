package ports

import (
	"context"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// UserManager covers user lifecycle operations. Authorization (who may call
// what) is enforced by the transport adapters, not here.
type UserManager interface {
	// CreateUser creates a user. When password is empty a temporary password
	// is generated and returned; it is never persisted in plain text.
	CreateUser(ctx context.Context, username, password string, role domain.Role) (*domain.User, string, error)
	ListUsers(ctx context.Context, params domain.ListParams, filter domain.UserFilter) (domain.ListResult[domain.User], error)
	DeleteUser(ctx context.Context, username string) error
	// ChangePassword sets a new password after verifying the current one.
	ChangePassword(ctx context.Context, username, oldPassword, newPassword string) error
	// ResetPassword sets a new password without verification. Adapters must
	// gate this behind an admin check.
	ResetPassword(ctx context.Context, username, newPassword string) error
}

type AuthManager interface {
	// InitAdmin consumes the one-time bootstrap token, creates the first
	// admin user and returns it together with a login token.
	InitAdmin(ctx context.Context, bootstrapToken, username, password string) (*domain.User, string, error)
	// Login exchanges credentials for an opaque bearer token. Zero ttl means
	// the server default; noExpiry issues a token that never expires.
	Login(ctx context.Context, username, password string, ttl time.Duration, noExpiry bool) (string, error)
	Logout(ctx context.Context, token string) error
}

// TokenAuthenticator resolves a bearer token to its user. Used by transport
// interceptors on every call.
type TokenAuthenticator interface {
	Authenticate(ctx context.Context, token string) (*domain.User, error)
}
