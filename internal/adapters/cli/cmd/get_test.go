package cmd

import (
	"context"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubControlPlane embeds Client so only the methods under test need to be
// implemented; calling any other method panics.
type stubControlPlane struct {
	Client

	getFlowCalls   []string
	getRouterCalls []string
}

func (s *stubControlPlane) GetFlow(_ context.Context, name string) (*domain.Flow, error) {
	s.getFlowCalls = append(s.getFlowCalls, name)
	return &domain.Flow{Id: name}, nil
}

func (s *stubControlPlane) GetRouter(_ context.Context, name string) (*domain.Router, error) {
	s.getRouterCalls = append(s.getRouterCalls, name)
	return &domain.Router{Id: name}, nil
}

func TestGetFlowCmd_SingleFlowQueriesFlows(t *testing.T) {
	cp := &stubControlPlane{}

	cmd := newGetFlowCmd(&Deps{Client: cp})
	cmd.SetArgs([]string{"main-flow"})

	require.NoError(t, cmd.Execute())

	assert.Equal(t, []string{"main-flow"}, cp.getFlowCalls)
	assert.Empty(t, cp.getRouterCalls, "get flow must not query routers")
}
