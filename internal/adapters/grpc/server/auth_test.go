package server

import (
	"context"
	"testing"
	"time"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockManagers records calls; behaviour is always success.
type mockManagers struct {
	changePasswordCalls [][3]string // username, oldPassword, newPassword
	resetPasswordCalls  [][2]string // username, newPassword
}

func (m *mockManagers) CreateUser(_ context.Context, username, _ string, role domain.Role) (*domain.User, string, error) {
	return &domain.User{Id: username, Role: role}, "generated-pass", nil
}

func (m *mockManagers) ListUsers(_ context.Context, _ domain.ListParams, _ domain.UserFilter) (domain.ListResult[domain.User], error) {
	return domain.ListResult[domain.User]{}, nil
}

func (m *mockManagers) DeleteUser(_ context.Context, _ string) error { return nil }

func (m *mockManagers) ChangePassword(_ context.Context, username, oldPassword, newPassword string) error {
	m.changePasswordCalls = append(m.changePasswordCalls, [3]string{username, oldPassword, newPassword})
	return nil
}

func (m *mockManagers) ResetPassword(_ context.Context, username, newPassword string) error {
	m.resetPasswordCalls = append(m.resetPasswordCalls, [2]string{username, newPassword})
	return nil
}

func (m *mockManagers) InitAdmin(_ context.Context, _, username, _ string) (*domain.User, string, error) {
	return &domain.User{Id: username, Role: domain.RoleAdmin}, "pgw_token", nil
}

func (m *mockManagers) Login(_ context.Context, _, _ string, _ time.Duration, _ bool) (string, error) {
	return "pgw_token", nil
}

func (m *mockManagers) Logout(_ context.Context, _ string) error { return nil }

func ctxWithUser(role domain.Role, id string) context.Context {
	return auth.ContextWithUser(context.Background(), &domain.User{Id: id, Role: role})
}

func assertCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()
	st, ok := status.FromError(err)
	require.True(t, ok, "expected grpc status error, got %v", err)
	assert.Equal(t, expected, st.Code())
}

func TestAuthServer_AdminOnlyEndpoints(t *testing.T) {
	srv := NewAuthServer(&mockManagers{}, &mockManagers{})
	memberCtx := ctxWithUser(domain.RoleMember, "bob")
	adminCtx := ctxWithUser(domain.RoleAdmin, "alice")

	t.Run("member denied", func(t *testing.T) {
		_, err := srv.CreateUser(memberCtx, &controlplanev1.CreateUserRequest{Username: "x"})
		assertCode(t, err, codes.PermissionDenied)

		_, err = srv.ListUsers(memberCtx, &controlplanev1.ListUsersRequest{})
		assertCode(t, err, codes.PermissionDenied)

		_, err = srv.DeleteUser(memberCtx, &controlplanev1.DeleteUserRequest{Username: "x"})
		assertCode(t, err, codes.PermissionDenied)
	})

	t.Run("unauthenticated context denied", func(t *testing.T) {
		_, err := srv.CreateUser(context.Background(), &controlplanev1.CreateUserRequest{Username: "x"})
		assertCode(t, err, codes.Unauthenticated)
	})

	t.Run("admin allowed", func(t *testing.T) {
		resp, err := srv.CreateUser(adminCtx, &controlplanev1.CreateUserRequest{Username: "x"})
		require.NoError(t, err)
		assert.Equal(t, "x", resp.User.Id)

		_, err = srv.ListUsers(adminCtx, &controlplanev1.ListUsersRequest{})
		assert.NoError(t, err)

		_, err = srv.DeleteUser(adminCtx, &controlplanev1.DeleteUserRequest{Username: "x"})
		assert.NoError(t, err)
	})
}

func TestAuthServer_ChangePassword(t *testing.T) {
	t.Run("self change requires old password", func(t *testing.T) {
		mocks := &mockManagers{}
		srv := NewAuthServer(mocks, mocks)

		_, err := srv.ChangePassword(ctxWithUser(domain.RoleMember, "bob"), &controlplanev1.ChangePasswordRequest{
			NewPassword: "newpassword123",
		})
		assertCode(t, err, codes.InvalidArgument)
	})

	t.Run("self change passes old password through", func(t *testing.T) {
		mocks := &mockManagers{}
		srv := NewAuthServer(mocks, mocks)

		_, err := srv.ChangePassword(ctxWithUser(domain.RoleMember, "bob"), &controlplanev1.ChangePasswordRequest{
			OldPassword: "old-pass",
			NewPassword: "newpassword123",
		})
		require.NoError(t, err)
		require.Len(t, mocks.changePasswordCalls, 1)
		assert.Equal(t, [3]string{"bob", "old-pass", "newpassword123"}, mocks.changePasswordCalls[0])
	})

	t.Run("member cannot change another user's password", func(t *testing.T) {
		mocks := &mockManagers{}
		srv := NewAuthServer(mocks, mocks)

		_, err := srv.ChangePassword(ctxWithUser(domain.RoleMember, "bob"), &controlplanev1.ChangePasswordRequest{
			Username:    "alice",
			NewPassword: "newpassword123",
		})
		assertCode(t, err, codes.PermissionDenied)
	})

	t.Run("admin reset routes to ResetPassword", func(t *testing.T) {
		mocks := &mockManagers{}
		srv := NewAuthServer(mocks, mocks)

		_, err := srv.ChangePassword(ctxWithUser(domain.RoleAdmin, "alice"), &controlplanev1.ChangePasswordRequest{
			Username:    "bob",
			OldPassword: "should-be-ignored",
			NewPassword: "newpassword123",
		})
		require.NoError(t, err)
		assert.Empty(t, mocks.changePasswordCalls)
		require.Len(t, mocks.resetPasswordCalls, 1)
		assert.Equal(t, [2]string{"bob", "newpassword123"}, mocks.resetPasswordCalls[0])
	})
}

func TestAuthServer_Logout(t *testing.T) {
	mocks := &mockManagers{}
	srv := NewAuthServer(mocks, mocks)

	t.Run("without token in context", func(t *testing.T) {
		_, err := srv.Logout(context.Background(), &controlplanev1.LogoutRequest{})
		assertCode(t, err, codes.Unauthenticated)
	})

	t.Run("with token in context", func(t *testing.T) {
		ctx := auth.ContextWithToken(context.Background(), "pgw_token")
		_, err := srv.Logout(ctx, &controlplanev1.LogoutRequest{})
		assert.NoError(t, err)
	})
}
