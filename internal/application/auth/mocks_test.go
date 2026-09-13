package auth

// In-memory test doubles shared by every test file in this package.
// Rule of thumb: fakes stay package-private; they graduate to an exported
// `authtest` package only when 2-3+ other packages need the same fake.

import (
	"context"
	"fmt"
	"strings"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

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

type mockRegTokenRepo struct {
	regTokens map[string]*domain.RegistrationToken
}

func newMockRegTokenRepo() *mockRegTokenRepo {
	return &mockRegTokenRepo{regTokens: map[string]*domain.RegistrationToken{}}
}

func (m *mockRegTokenRepo) Save(_ context.Context, token *domain.RegistrationToken) error {
	copied := *token
	m.regTokens[token.Hash] = &copied
	return nil
}

func (m *mockRegTokenRepo) Consume(_ context.Context, hash string) (*domain.RegistrationToken, error) {
	t, ok := m.regTokens[hash]
	if !ok {
		return nil, fmt.Errorf("registration token not found")
	}
	delete(m.regTokens, hash)
	copied := *t
	return &copied, nil
}

// seedToken stores a token record for the given raw token and returns its hash.
func seedToken(tokens *mockTokenRepo, raw string, record domain.Token) string {
	record.Hash = hashToken(raw)
	tokens.tokens[record.Hash] = &record
	return record.Hash
}
