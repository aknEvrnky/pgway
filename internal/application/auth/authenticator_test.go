package auth

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAgentRepo struct {
	agents map[string]*domain.Agent
}

func newMockAgentRepo() *mockAgentRepo {
	return &mockAgentRepo{agents: map[string]*domain.Agent{}}
}

func (m *mockAgentRepo) List(_ context.Context, _ domain.ListParams, filter domain.AgentFilter) (domain.ListResult[domain.Agent], error) {
	var items []*domain.Agent
outer:
	for _, a := range m.agents {
		if filter.Search != "" &&
			!strings.Contains(a.Id, filter.Search) &&
			!strings.Contains(a.Hostname, filter.Search) {
			continue
		}
		for k, v := range filter.Labels {
			if a.Labels[k] != v {
				continue outer
			}
		}
		items = append(items, a)
	}
	return domain.ListResult[domain.Agent]{Items: items, TotalCount: len(items)}, nil
}

func (m *mockAgentRepo) Find(_ context.Context, id string) (*domain.Agent, error) {
	a, ok := m.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent %q not found", id)
	}
	copied := *a
	return &copied, nil
}

func (m *mockAgentRepo) Create(_ context.Context, agent *domain.Agent) error {
	if _, ok := m.agents[agent.Id]; ok {
		return domain.ErrAgentExists
	}
	copied := *agent
	m.agents[agent.Id] = &copied
	return nil
}

func (m *mockAgentRepo) Save(_ context.Context, agent *domain.Agent) error {
	copied := *agent
	m.agents[agent.Id] = &copied
	return nil
}

func (m *mockAgentRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.agents[id]; !ok {
		return fmt.Errorf("agent %q not found", id)
	}
	delete(m.agents, id)
	return nil
}

// --- helpers ---

func newTestAuthenticator() (*Authenticator, *mockUserRepo, *mockAgentRepo, *mockTokenRepo) {
	users := newMockUserRepo()
	agents := newMockAgentRepo()
	tokens := newMockTokenRepo()
	return NewAuthenticator(users, agents, tokens), users, agents, tokens
}

// seedToken stores a token record for the given raw token and returns its hash.
func seedToken(tokens *mockTokenRepo, raw string, record domain.Token) string {
	record.Hash = hashToken(raw)
	tokens.tokens[record.Hash] = &record
	return record.Hash
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
