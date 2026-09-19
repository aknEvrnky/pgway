package agenthost

import (
	"context"
	"io/fs"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	creds *ports.AgentHostCredentials
	err   error
}

func (m *memStore) Load(context.Context) (*ports.AgentHostCredentials, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.creds == nil {
		return nil, fs.ErrNotExist
	}
	return m.creds, nil
}

func (m *memStore) Save(_ context.Context, creds *ports.AgentHostCredentials) error {
	m.creds = creds
	return nil
}

type fakeRegistrar struct {
	calls int
	token string
	err   error
}

func (f *fakeRegistrar) Register(_ context.Context, _ string, agent domain.Agent) (*domain.Agent, string, error) {
	f.calls++
	if f.err != nil {
		return nil, "", f.err
	}
	return &domain.Agent{Id: agent.Id}, f.token, nil
}

func TestBootstrapCredentialsUsesStore(t *testing.T) {
	store := &memStore{creds: &ports.AgentHostCredentials{AgentID: "a1", AgentToken: "tok"}}
	reg := &fakeRegistrar{token: "unused"}

	got, err := BootstrapCredentials(context.Background(), store, "reg", domain.Agent{Id: "a1"}, reg)
	require.NoError(t, err)
	assert.Equal(t, "a1", got.AgentID)
	assert.Equal(t, "tok", got.AgentToken)
	assert.Equal(t, 0, reg.calls)
}

func TestBootstrapCredentialsRegistersWhenMissing(t *testing.T) {
	store := &memStore{}
	reg := &fakeRegistrar{token: "issued"}

	got, err := BootstrapCredentials(context.Background(), store, "reg-secret", domain.Agent{Id: "edge-1"}, reg)
	require.NoError(t, err)
	assert.Equal(t, "edge-1", got.AgentID)
	assert.Equal(t, "issued", got.AgentToken)
	assert.Equal(t, 1, reg.calls)
	require.NotNil(t, store.creds)
	assert.Equal(t, "issued", store.creds.AgentToken)
}

func TestBootstrapCredentialsRequiresRegToken(t *testing.T) {
	_, err := BootstrapCredentials(context.Background(), &memStore{}, "", domain.Agent{Id: "x"}, &fakeRegistrar{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PGWAY_AGENT_REGISTRATION_TOKEN")
}
