package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- in-memory mocks ---

type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: map[string]*domain.User{}}
}

func (m *mockUserRepo) List(_ context.Context, _ domain.ListParams, filter domain.UserFilter) (domain.ListResult[domain.User], error) {
	var items []*domain.User
	for _, u := range m.users {
		if filter.Role != "" && string(u.Role) != filter.Role {
			continue
		}
		items = append(items, u)
	}
	return domain.ListResult[domain.User]{Items: items, TotalCount: len(items)}, nil
}

func (m *mockUserRepo) Find(_ context.Context, id string) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("user %q not found", id)
	}
	copied := *u
	return &copied, nil
}

func (m *mockUserRepo) Count(_ context.Context) (int, error) {
	return len(m.users), nil
}

func (m *mockUserRepo) Save(_ context.Context, user *domain.User) error {
	copied := *user
	m.users[user.Id] = &copied
	return nil
}

func (m *mockUserRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.users[id]; !ok {
		return fmt.Errorf("user %q not found", id)
	}
	delete(m.users, id)
	return nil
}

type mockTokenRepo struct {
	tokens map[string]*domain.Token
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{tokens: map[string]*domain.Token{}}
}

func (m *mockTokenRepo) Find(_ context.Context, hash string) (*domain.Token, error) {
	t, ok := m.tokens[hash]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}
	copied := *t
	return &copied, nil
}

func (m *mockTokenRepo) Save(_ context.Context, token *domain.Token) error {
	copied := *token
	m.tokens[token.Hash] = &copied
	return nil
}

func (m *mockTokenRepo) Delete(_ context.Context, hash string) error {
	if _, ok := m.tokens[hash]; !ok {
		return fmt.Errorf("token not found")
	}
	delete(m.tokens, hash)
	return nil
}

func (m *mockTokenRepo) DeleteByUserId(_ context.Context, userId string) error {
	for hash, t := range m.tokens {
		if t.UserId == userId {
			delete(m.tokens, hash)
		}
	}
	return nil
}

func (m *mockTokenRepo) DeleteByAgentId(_ context.Context, agentId string) error {
	for hash, t := range m.tokens {
		if t.AgentId == agentId {
			delete(m.tokens, hash)
		}
	}
	return nil
}

// --- helpers ---

const testTTL = time.Hour

func newTestService() (*Service, *mockUserRepo, *mockTokenRepo) {
	users := newMockUserRepo()
	tokens := newMockTokenRepo()
	return NewService(users, tokens, testTTL), users, tokens
}

// initializedService returns a service with an admin user and a valid token.
func initializedService(t *testing.T) (*Service, *mockUserRepo, *mockTokenRepo, string) {
	t.Helper()
	svc, users, tokens := newTestService()
	ctx := context.Background()

	require.NoError(t, svc.Bootstrap(ctx))
	_, token, err := svc.InitAdmin(ctx, svc.BootstrapToken(), "admin", "password123")
	require.NoError(t, err)

	return svc, users, tokens, token
}

// --- tests ---

func TestService_Bootstrap(t *testing.T) {
	ctx := context.Background()

	t.Run("generates token when no users", func(t *testing.T) {
		svc, _, _ := newTestService()
		require.NoError(t, svc.Bootstrap(ctx))
		assert.NotEmpty(t, svc.BootstrapToken())
	})

	t.Run("no token when users exist", func(t *testing.T) {
		svc, users, _ := newTestService()
		users.users["admin"] = &domain.User{Id: "admin", Role: domain.RoleAdmin, PasswordHash: "h"}
		require.NoError(t, svc.Bootstrap(ctx))
		assert.Empty(t, svc.BootstrapToken())
	})
}

func TestService_InitAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("success creates admin and returns token", func(t *testing.T) {
		svc, users, _ := newTestService()
		require.NoError(t, svc.Bootstrap(ctx))

		user, token, err := svc.InitAdmin(ctx, svc.BootstrapToken(), "admin", "password123")
		require.NoError(t, err)

		assert.Equal(t, "admin", user.Id)
		assert.Equal(t, domain.RoleAdmin, user.Role)
		assert.NotEmpty(t, token)
		assert.NotEqual(t, "password123", users.users["admin"].PasswordHash)

		// bootstrap token is single-use
		assert.Empty(t, svc.BootstrapToken())
	})

	t.Run("wrong bootstrap token", func(t *testing.T) {
		svc, _, _ := newTestService()
		require.NoError(t, svc.Bootstrap(ctx))

		_, _, err := svc.InitAdmin(ctx, "wrong", "admin", "password123")
		assert.ErrorIs(t, err, ErrInvalidBootstrapToken)
	})

	t.Run("already initialized", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		_, _, err := svc.InitAdmin(ctx, "anything", "admin2", "password123")
		assert.ErrorIs(t, err, ErrAlreadyInitialized)
	})

	t.Run("weak password rejected", func(t *testing.T) {
		svc, _, _ := newTestService()
		require.NoError(t, svc.Bootstrap(ctx))

		_, _, err := svc.InitAdmin(ctx, svc.BootstrapToken(), "admin", "short")
		assert.ErrorIs(t, err, ErrWeakPassword)
	})
}

func TestService_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("valid credentials issue a token bound to the user", func(t *testing.T) {
		svc, _, tokens, _ := initializedService(t)

		token, err := svc.Login(ctx, "admin", "password123", 0, false)
		require.NoError(t, err)

		record, ok := tokens.tokens[hashToken(token)]
		require.True(t, ok, "token must be persisted by hash")
		assert.Equal(t, "admin", record.UserId)
		assert.Empty(t, record.AgentId)
	})

	t.Run("wrong password", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		_, err := svc.Login(ctx, "admin", "wrong-password", 0, false)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("unknown user", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		_, err := svc.Login(ctx, "ghost", "password123", 0, false)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("default ttl applied", func(t *testing.T) {
		svc, _, tokens, _ := initializedService(t)

		token, err := svc.Login(ctx, "admin", "password123", 0, false)
		require.NoError(t, err)

		record := tokens.tokens[hashToken(token)]
		require.NotNil(t, record.ExpiresAt)
		assert.WithinDuration(t, time.Now().Add(testTTL), *record.ExpiresAt, time.Minute)
	})

	t.Run("noExpiry issues token without expiry", func(t *testing.T) {
		svc, _, tokens, _ := initializedService(t)

		token, err := svc.Login(ctx, "admin", "password123", 0, true)
		require.NoError(t, err)

		record := tokens.tokens[hashToken(token)]
		assert.Nil(t, record.ExpiresAt)
	})
}

func TestService_Logout(t *testing.T) {
	ctx := context.Background()

	t.Run("revokes token", func(t *testing.T) {
		svc, _, tokens, token := initializedService(t)

		require.NoError(t, svc.Logout(ctx, token))

		assert.NotContains(t, tokens.tokens, hashToken(token))
	})

	t.Run("unknown token", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		assert.ErrorIs(t, svc.Logout(ctx, "pgw_unknown"), ErrInvalidToken)
	})
}

func TestService_CreateUser(t *testing.T) {
	ctx := context.Background()

	t.Run("with explicit password", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)

		user, generated, err := svc.CreateUser(ctx, "bob", "bobpassword", domain.RoleMember)
		require.NoError(t, err)
		assert.Equal(t, "bob", user.Id)
		assert.Equal(t, domain.RoleMember, user.Role)
		assert.Empty(t, generated)

		_, err = svc.Login(ctx, "bob", "bobpassword", 0, false)
		assert.NoError(t, err)
	})

	t.Run("without password generates one", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)

		_, generated, err := svc.CreateUser(ctx, "bob", "", domain.RoleMember)
		require.NoError(t, err)
		require.NotEmpty(t, generated)

		_, err = svc.Login(ctx, "bob", generated, 0, false)
		assert.NoError(t, err)
	})

	t.Run("empty role defaults to member", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)

		user, _, err := svc.CreateUser(ctx, "bob", "bobpassword", "")
		require.NoError(t, err)
		assert.Equal(t, domain.RoleMember, user.Role)
	})

	t.Run("duplicate username", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)

		_, _, err := svc.CreateUser(ctx, "admin", "password123", domain.RoleMember)
		assert.ErrorIs(t, err, ErrUserExists)
	})
}

func TestService_DeleteUser(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes user and revokes tokens", func(t *testing.T) {
		svc, users, tokens, _ := initializedService(t)

		_, _, err := svc.CreateUser(ctx, "bob", "bobpassword", domain.RoleMember)
		require.NoError(t, err)
		bobToken, err := svc.Login(ctx, "bob", "bobpassword", 0, false)
		require.NoError(t, err)

		require.NoError(t, svc.DeleteUser(ctx, "bob"))

		assert.NotContains(t, users.users, "bob")
		assert.NotContains(t, tokens.tokens, hashToken(bobToken))
	})

	t.Run("last admin cannot be deleted", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		assert.ErrorIs(t, svc.DeleteUser(ctx, "admin"), ErrLastAdmin)
	})

	t.Run("admin deletable when another admin exists", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)

		_, _, err := svc.CreateUser(ctx, "admin2", "password123", domain.RoleAdmin)
		require.NoError(t, err)

		assert.NoError(t, svc.DeleteUser(ctx, "admin"))
	})

	t.Run("unknown user", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		assert.ErrorContains(t, svc.DeleteUser(ctx, "ghost"), "not found")
	})
}

func TestService_ChangePassword(t *testing.T) {
	ctx := context.Background()

	t.Run("with old password verification", func(t *testing.T) {
		svc, _, tokens, token := initializedService(t)

		require.NoError(t, svc.ChangePassword(ctx, "admin", "password123", "newpassword123"))

		// old token revoked
		assert.NotContains(t, tokens.tokens, hashToken(token))

		// new password works, old one does not
		_, err := svc.Login(ctx, "admin", "newpassword123", 0, false)
		assert.NoError(t, err)
		_, err = svc.Login(ctx, "admin", "password123", 0, false)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("wrong old password", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		err := svc.ChangePassword(ctx, "admin", "wrong", "newpassword123")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("empty old password never skips verification", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		err := svc.ChangePassword(ctx, "admin", "", "newpassword123")
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("weak new password", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		err := svc.ChangePassword(ctx, "admin", "password123", "short")
		assert.ErrorIs(t, err, ErrWeakPassword)
	})
}

func TestService_ResetPassword(t *testing.T) {
	ctx := context.Background()

	t.Run("sets password without verification and revokes tokens", func(t *testing.T) {
		svc, _, tokens, token := initializedService(t)

		require.NoError(t, svc.ResetPassword(ctx, "admin", "newpassword123"))

		assert.NotContains(t, tokens.tokens, hashToken(token))

		_, err := svc.Login(ctx, "admin", "newpassword123", 0, false)
		assert.NoError(t, err)
	})

	t.Run("unknown user", func(t *testing.T) {
		svc, _, _, _ := initializedService(t)
		assert.ErrorContains(t, svc.ResetPassword(ctx, "ghost", "newpassword123"), "not found")
	})
}
