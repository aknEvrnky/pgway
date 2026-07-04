package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/aknEvrnky/pgway/integration/testutil"
	grpcclient "github.com/aknEvrnky/pgway/internal/adapters/grpc/client"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
)

func newAuthedClient(t *testing.T, addr, token string) *grpcclient.Client {
	t.Helper()
	client, err := grpcclient.NewClient(addr, token)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })
	return client
}

func assertGrpcCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, expected, status.Code(err))
}

// TestAuthFlow exercises the full lifecycle over a real TCP gRPC connection:
// bootstrap → init → login → authorized calls → role checks → revocation.
func TestAuthFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	addr, authService := testutil.NewAuthTestServer(t)

	bootstrapToken := authService.BootstrapToken()
	require.NotEmpty(t, bootstrapToken, "fresh server must expose a bootstrap token")

	anon := newAuthedClient(t, addr, "")

	t.Run("without token everything except init/login is rejected", func(t *testing.T) {
		_, err := anon.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
		assertGrpcCode(t, err, codes.Unauthenticated)

		_, _, err = anon.CreateUser(ctx, "x", "password123", domain.RoleMember)
		assertGrpcCode(t, err, codes.Unauthenticated)
	})

	t.Run("init with wrong bootstrap token", func(t *testing.T) {
		_, _, err := anon.InitAdmin(ctx, "pgw_wrong", "admin", "password123")
		assertGrpcCode(t, err, codes.Unauthenticated)
	})

	var adminToken string

	t.Run("init with correct bootstrap token creates admin", func(t *testing.T) {
		user, token, err := anon.InitAdmin(ctx, bootstrapToken, "admin", "password123")
		require.NoError(t, err)
		assert.Equal(t, "admin", user.Id)
		assert.Equal(t, domain.RoleAdmin, user.Role)
		require.NotEmpty(t, token)
		adminToken = token
	})

	t.Run("second init is rejected", func(t *testing.T) {
		_, _, err := anon.InitAdmin(ctx, bootstrapToken, "admin2", "password123")
		assertGrpcCode(t, err, codes.FailedPrecondition)
	})

	admin := newAuthedClient(t, addr, adminToken)

	t.Run("admin token grants resource access", func(t *testing.T) {
		_, err := admin.ApplyProxyV1(ctx, schema.Metadata{Name: "p1"}, proxyv1.ProxySpecV1{
			URL: "http://user:pass@127.0.0.1:8080",
		})
		require.NoError(t, err)

		result, err := admin.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
		require.NoError(t, err)
		assert.Equal(t, 1, result.TotalCount)
	})

	t.Run("garbage token is rejected", func(t *testing.T) {
		bad := newAuthedClient(t, addr, "pgw_garbage")
		_, err := bad.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
		assertGrpcCode(t, err, codes.Unauthenticated)
	})

	var memberToken string

	t.Run("admin creates member who can log in", func(t *testing.T) {
		user, generated, err := admin.CreateUser(ctx, "bob", "", domain.RoleMember)
		require.NoError(t, err)
		assert.Equal(t, domain.RoleMember, user.Role)
		require.NotEmpty(t, generated, "server must generate a temporary password")

		memberToken, err = anon.Login(ctx, "bob", generated, 0, false)
		require.NoError(t, err)

		// change the temporary password
		member := newAuthedClient(t, addr, memberToken)
		require.NoError(t, member.ChangePassword(ctx, "", generated, "bob-password-1"))

		// password change revokes the session
		_, err = member.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
		assertGrpcCode(t, err, codes.Unauthenticated)

		memberToken, err = anon.Login(ctx, "bob", "bob-password-1", 0, false)
		require.NoError(t, err)
	})

	member := newAuthedClient(t, addr, memberToken)

	t.Run("member can access resources but not user management", func(t *testing.T) {
		_, err := member.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
		require.NoError(t, err)

		_, _, err = member.CreateUser(ctx, "eve", "password123", domain.RoleMember)
		assertGrpcCode(t, err, codes.PermissionDenied)

		_, err = member.ListUsers(ctx, domain.ListParams{}, domain.UserFilter{})
		assertGrpcCode(t, err, codes.PermissionDenied)

		err = member.DeleteUser(ctx, "admin")
		assertGrpcCode(t, err, codes.PermissionDenied)

		err = member.ResetPassword(ctx, "admin", "hacked-password")
		assertGrpcCode(t, err, codes.PermissionDenied)
	})

	t.Run("admin user management", func(t *testing.T) {
		result, err := admin.ListUsers(ctx, domain.ListParams{}, domain.UserFilter{})
		require.NoError(t, err)
		assert.Equal(t, 2, result.TotalCount) // admin + bob

		// admin resets bob's password without knowing the old one
		require.NoError(t, admin.ResetPassword(ctx, "bob", "bob-password-2"))

		_, err = anon.Login(ctx, "bob", "bob-password-2", 0, false)
		require.NoError(t, err)

		// last admin is protected
		err = admin.DeleteUser(ctx, "admin")
		assertGrpcCode(t, err, codes.FailedPrecondition)

		require.NoError(t, admin.DeleteUser(ctx, "bob"))
	})

	t.Run("logout revokes the token", func(t *testing.T) {
		token, err := anon.Login(ctx, "admin", "password123", 0, false)
		require.NoError(t, err)

		session := newAuthedClient(t, addr, token)
		_, err = session.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
		require.NoError(t, err)

		require.NoError(t, session.Logout(ctx, token))

		_, err = session.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
		assertGrpcCode(t, err, codes.Unauthenticated)
	})
}
