package auth

import (
	"context"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func newTestAuthenticator() (*Authenticator, *mockUserRepo, *mockAgentRepo, *mockTokenRepo) {
	users := newMockUserRepo()
	agents := newMockAgentRepo()
	tokens := newMockTokenRepo()
	return NewAuthenticator(users, agents, tokens), users, agents, tokens
}

// --- tests ---

func TestAuthenticator_Authenticate(t *testing.T) {
	ctx := context.Background()

	t.Run("empty token", func(t *testing.T) {
		authr, _, _, _ := newTestAuthenticator()
		_, err := authr.Authenticate(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("unknown token", func(t *testing.T) {
		authr, _, _, _ := newTestAuthenticator()
		_, err := authr.Authenticate(ctx, "pgw_unknown")
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("expired token rejected and cleaned up", func(t *testing.T) {
		authr, users, _, tokens := newTestAuthenticator()
		users.users["admin"] = &domain.User{Id: "admin", Role: domain.RoleAdmin, PasswordHash: "h"}

		past := time.Now().Add(-time.Minute)
		hash := seedToken(tokens, "pgw_expired", domain.Token{UserId: "admin", ExpiresAt: &past})

		_, err := authr.Authenticate(ctx, "pgw_expired")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.NotContains(t, tokens.tokens, hash)
	})

	t.Run("user token resolves to user principal", func(t *testing.T) {
		authr, users, _, tokens := newTestAuthenticator()
		users.users["admin"] = &domain.User{Id: "admin", Role: domain.RoleAdmin, PasswordHash: "h"}
		seedToken(tokens, "pgw_user", domain.Token{UserId: "admin"})

		principal, err := authr.Authenticate(ctx, "pgw_user")
		require.NoError(t, err)
		assert.Equal(t, domain.PrincipalKindUser, principal.Kind())
		require.NotNil(t, principal.User)
		assert.Equal(t, "admin", principal.User.Id)
		assert.Nil(t, principal.Agent)
	})

	t.Run("token of deleted user rejected and cleaned up", func(t *testing.T) {
		authr, _, _, tokens := newTestAuthenticator()
		hash := seedToken(tokens, "pgw_orphan_user", domain.Token{UserId: "ghost"})

		_, err := authr.Authenticate(ctx, "pgw_orphan_user")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.NotContains(t, tokens.tokens, hash)
	})

	t.Run("agent token resolves to agent principal", func(t *testing.T) {
		authr, _, agents, tokens := newTestAuthenticator()
		agents.agents["edge-1"] = &domain.Agent{Id: "edge-1", Hostname: "edge-host"}
		seedToken(tokens, "pgw_agent", domain.Token{AgentId: "edge-1"})

		principal, err := authr.Authenticate(ctx, "pgw_agent")
		require.NoError(t, err)
		assert.Equal(t, domain.PrincipalKindAgent, principal.Kind())
		require.NotNil(t, principal.Agent)
		assert.Equal(t, "edge-1", principal.Agent.Id)
		assert.Nil(t, principal.User)
	})

	t.Run("token of deleted agent rejected and cleaned up", func(t *testing.T) {
		authr, _, _, tokens := newTestAuthenticator()
		hash := seedToken(tokens, "pgw_orphan_agent", domain.Token{AgentId: "ghost-agent"})

		_, err := authr.Authenticate(ctx, "pgw_orphan_agent")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.NotContains(t, tokens.tokens, hash)
	})

	t.Run("subjectless token rejected and cleaned up", func(t *testing.T) {
		authr, _, _, tokens := newTestAuthenticator()
		hash := seedToken(tokens, "pgw_subjectless", domain.Token{})

		_, err := authr.Authenticate(ctx, "pgw_subjectless")
		assert.ErrorIs(t, err, ErrInvalidToken)
		assert.NotContains(t, tokens.tokens, hash)
	})

	// bridge test: a token minted by the user Service authenticates through
	// the Authenticator when both share the same stores.
	t.Run("login token authenticates", func(t *testing.T) {
		authr, users, _, tokens := newTestAuthenticator()
		svc := NewService(users, tokens, testTTL)

		require.NoError(t, svc.Bootstrap(ctx))
		_, token, err := svc.InitAdmin(ctx, svc.BootstrapToken(), "admin", "password123")
		require.NoError(t, err)

		principal, err := authr.Authenticate(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, "admin", principal.User.Id)
	})
}
