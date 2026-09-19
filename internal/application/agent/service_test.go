package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"

	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAgentTTL = time.Hour

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

type mockCreds struct {
	regTokens   map[string]struct{}
	agentTokens map[string]string // agentId → raw token (last issued)
	revoked     []string
	extended    []string
	consumeErr  error
	issueErr    error
}

func newMockCreds() *mockCreds {
	return &mockCreds{
		regTokens:   map[string]struct{}{},
		agentTokens: map[string]string{},
	}
}

func (m *mockCreds) CreateRegistrationToken(_ context.Context, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		return "", fmt.Errorf("TTL must be greater than 0")
	}
	token := fmt.Sprintf("pgwr_%d", len(m.regTokens)+1)
	m.regTokens[token] = struct{}{}
	return token, nil
}

func (m *mockCreds) ConsumeRegistrationToken(_ context.Context, token string) error {
	if m.consumeErr != nil {
		return m.consumeErr
	}
	if _, ok := m.regTokens[token]; !ok {
		return auth.ErrInvalidRegistrationToken
	}
	delete(m.regTokens, token)
	return nil
}

func (m *mockCreds) IssueAgentToken(_ context.Context, agentId string, ttl time.Duration) (string, error) {
	if m.issueErr != nil {
		return "", m.issueErr
	}
	if agentId == "" || ttl <= 0 {
		return "", fmt.Errorf("invalid issue args")
	}
	token := "pgw_" + agentId
	m.agentTokens[agentId] = token
	return token, nil
}

func (m *mockCreds) ExtendAgentToken(_ context.Context, rawToken string, ttl time.Duration) (time.Time, error) {
	if rawToken == "" || ttl <= 0 {
		return time.Time{}, auth.ErrInvalidToken
	}
	m.extended = append(m.extended, rawToken)
	return time.Now().Add(ttl), nil
}

func (m *mockCreds) RevokeAgentTokens(_ context.Context, agentId string) error {
	m.revoked = append(m.revoked, agentId)
	delete(m.agentTokens, agentId)
	return nil
}

func newTestService() (*Service, *mockAgentRepo, *mockCreds) {
	agents := newMockAgentRepo()
	creds := newMockCreds()
	return NewService(agents, creds, testAgentTTL), agents, creds
}

func agentCtx(agent *domain.Agent, rawToken string) context.Context {
	ctx := ports.ContextWithPrincipal(context.Background(), &domain.Principal{Agent: agent})
	return ports.ContextWithToken(ctx, rawToken)
}

func TestService_CreateRegistrationToken(t *testing.T) {
	svc, _, creds := newTestService()

	token, err := svc.CreateRegistrationToken(context.Background(), 24*time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	_, ok := creds.regTokens[token]
	assert.True(t, ok)
}

func TestService_Register(t *testing.T) {
	t.Run("happy path consumes token and issues agent token", func(t *testing.T) {
		svc, agents, creds := newTestService()
		reg, err := creds.CreateRegistrationToken(context.Background(), time.Hour)
		require.NoError(t, err)

		got, agentToken, err := svc.Register(context.Background(), reg, domain.Agent{
			Id:       "edge-1",
			Hostname: "edge-host",
			Version:  "v0.1.0",
		})
		require.NoError(t, err)
		assert.Equal(t, "edge-1", got.Id)
		assert.Equal(t, "pgw_edge-1", agentToken)
		assert.NotZero(t, got.CreatedAt)
		_, exists := agents.agents["edge-1"]
		assert.True(t, exists)
		_, burned := creds.regTokens[reg]
		assert.False(t, burned, "registration token must be consumed")
	})

	t.Run("invalid registration token", func(t *testing.T) {
		svc, _, _ := newTestService()
		_, _, err := svc.Register(context.Background(), "pgwr_ghost", domain.Agent{Id: "edge-1"})
		assert.ErrorIs(t, err, auth.ErrInvalidRegistrationToken)
	})

	t.Run("name conflict after consume", func(t *testing.T) {
		svc, agents, creds := newTestService()
		agents.agents["edge-1"] = &domain.Agent{Id: "edge-1"}
		reg, err := creds.CreateRegistrationToken(context.Background(), time.Hour)
		require.NoError(t, err)

		_, _, err = svc.Register(context.Background(), reg, domain.Agent{Id: "edge-1"})
		assert.ErrorIs(t, err, domain.ErrAgentExists)
		_, burned := creds.regTokens[reg]
		assert.False(t, burned, "token stays burned on conflict (fail-clean)")
	})

	t.Run("validation failure after consume", func(t *testing.T) {
		svc, _, creds := newTestService()
		reg, err := creds.CreateRegistrationToken(context.Background(), time.Hour)
		require.NoError(t, err)

		_, _, err = svc.Register(context.Background(), reg, domain.Agent{Id: ""})
		assert.Error(t, err)
		_, burned := creds.regTokens[reg]
		assert.False(t, burned)
	})
}

func TestService_Heartbeat(t *testing.T) {
	t.Run("updates last heartbeat and extends token", func(t *testing.T) {
		svc, agents, creds := newTestService()
		agents.agents["edge-1"] = &domain.Agent{Id: "edge-1", Hostname: "h"}
		raw := "pgw_edge-1"
		creds.agentTokens["edge-1"] = raw

		before := time.Now()
		expires, err := svc.Heartbeat(agentCtx(&domain.Agent{Id: "edge-1"}, raw))
		require.NoError(t, err)
		assert.True(t, expires.After(before.Add(testAgentTTL-time.Second)))
		require.NotNil(t, agents.agents["edge-1"].LastHeartbeat)
		assert.Equal(t, []string{raw}, creds.extended)
	})

	t.Run("rejects non-agent principal", func(t *testing.T) {
		svc, _, _ := newTestService()
		ctx := ports.ContextWithPrincipal(context.Background(), &domain.Principal{
			User: &domain.User{Id: "admin"},
		})
		ctx = ports.ContextWithToken(ctx, "pgw_user")
		_, err := svc.Heartbeat(ctx)
		assert.ErrorIs(t, err, ErrAgentRequired)
	})

	t.Run("rejects missing bearer token", func(t *testing.T) {
		svc, _, _ := newTestService()
		ctx := ports.ContextWithPrincipal(context.Background(), &domain.Principal{
			Agent: &domain.Agent{Id: "edge-1"},
		})
		_, err := svc.Heartbeat(ctx)
		assert.ErrorIs(t, err, ErrTokenRequired)
	})
}

func TestService_Deregister(t *testing.T) {
	svc, agents, _ := newTestService()
	now := time.Now()
	agents.agents["edge-1"] = &domain.Agent{Id: "edge-1", LastHeartbeat: &now}

	err := svc.Deregister(agentCtx(&domain.Agent{Id: "edge-1", LastHeartbeat: &now}, "pgw_edge-1"))
	require.NoError(t, err)
	assert.Nil(t, agents.agents["edge-1"].LastHeartbeat)
}

func TestService_ListAgents(t *testing.T) {
	svc, agents, _ := newTestService()
	agents.agents["a"] = &domain.Agent{Id: "a"}
	agents.agents["b"] = &domain.Agent{Id: "b"}

	result, err := svc.ListAgents(context.Background(), domain.ListParams{PageSize: 1000}, domain.AgentFilter{})
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestService_DeleteAgent(t *testing.T) {
	t.Run("revokes then deletes", func(t *testing.T) {
		svc, agents, creds := newTestService()
		agents.agents["edge-1"] = &domain.Agent{Id: "edge-1"}
		creds.agentTokens["edge-1"] = "pgw_edge-1"

		require.NoError(t, svc.DeleteAgent(context.Background(), "edge-1"))
		assert.Equal(t, []string{"edge-1"}, creds.revoked)
		_, ok := agents.agents["edge-1"]
		assert.False(t, ok)
		_, tokOk := creds.agentTokens["edge-1"]
		assert.False(t, tokOk)
	})

	t.Run("missing agent", func(t *testing.T) {
		svc, _, creds := newTestService()
		err := svc.DeleteAgent(context.Background(), "ghost")
		assert.ErrorIs(t, err, ErrAgentNotFound)
		assert.Empty(t, creds.revoked, "must not revoke before confirming agent exists")
	})
}
