package agentruntime

import (
	"context"
	"fmt"
	"os"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// Registrar performs the one-shot Register exchange.
type Registrar interface {
	Register(ctx context.Context, regToken string, agent domain.Agent) (*domain.Agent, string, error)
}

// BootstrapResult is the credential set ready for CP dials.
type BootstrapResult struct {
	AgentID    string
	AgentToken string
}

// BootstrapCredentials loads the state file or registers with a one-time token.
func BootstrapCredentials(
	ctx context.Context,
	statePath string,
	regToken string,
	agent domain.Agent,
	reg Registrar,
) (*BootstrapResult, error) {
	state, err := LoadState(statePath)
	if err == nil {
		return &BootstrapResult{AgentID: state.AgentID, AgentToken: state.AgentToken}, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("load agent state: %w", err)
	}

	if regToken == "" {
		return nil, fmt.Errorf("no agent state at %s and PGWAY_REGISTRATION_TOKEN is empty", statePath)
	}

	registered, agentToken, err := reg.Register(ctx, regToken, agent)
	if err != nil {
		return nil, fmt.Errorf("register agent: %w", err)
	}

	id := registered.Id
	if id == "" {
		id = agent.Id
	}

	if err := SaveState(statePath, &State{AgentID: id, AgentToken: agentToken}); err != nil {
		return nil, err
	}

	return &BootstrapResult{AgentID: id, AgentToken: agentToken}, nil
}

// EnsureHostname fills Hostname when empty.
func EnsureHostname(agent *domain.Agent) error {
	if agent.Hostname != "" {
		return nil
	}
	h, err := os.Hostname()
	if err != nil {
		return err
	}
	agent.Hostname = h
	return nil
}
