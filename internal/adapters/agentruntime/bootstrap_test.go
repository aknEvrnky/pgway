package agentruntime

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRegistrar struct {
	calls int
	token string
	err   error
}

func (f *fakeRegistrar) Register(_ context.Context, regToken string, agent domain.Agent) (*domain.Agent, string, error) {
	f.calls++
	if f.err != nil {
		return nil, "", f.err
	}
	return &domain.Agent{Id: agent.Id}, f.token, nil
}

func TestBootstrapCredentialsUsesStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	require.NoError(t, SaveState(path, &State{AgentID: "a1", AgentToken: "tok"}))

	reg := &fakeRegistrar{token: "unused"}
	got, err := BootstrapCredentials(context.Background(), path, "reg", domain.Agent{Id: "a1"}, reg)
	require.NoError(t, err)
	assert.Equal(t, "a1", got.AgentID)
	assert.Equal(t, "tok", got.AgentToken)
	assert.Equal(t, 0, reg.calls)
}

func TestBootstrapCredentialsRegistersWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "agent.json")
	reg := &fakeRegistrar{token: "issued"}

	got, err := BootstrapCredentials(context.Background(), path, "reg-secret", domain.Agent{Id: "edge-1"}, reg)
	require.NoError(t, err)
	assert.Equal(t, "edge-1", got.AgentID)
	assert.Equal(t, "issued", got.AgentToken)
	assert.Equal(t, 1, reg.calls)

	loaded, err := LoadState(path)
	require.NoError(t, err)
	assert.Equal(t, "edge-1", loaded.AgentID)
	assert.Equal(t, "issued", loaded.AgentToken)
}

func TestBootstrapCredentialsRequiresRegToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	_, err := BootstrapCredentials(context.Background(), path, "", domain.Agent{Id: "x"}, &fakeRegistrar{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PGWAY_REGISTRATION_TOKEN")
}
