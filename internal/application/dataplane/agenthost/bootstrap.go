package agenthost

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// BootstrapResult is the credential set ready for CP dials.
type BootstrapResult struct {
	AgentID    string
	AgentToken string
}

// BootstrapCredentials loads persisted credentials or registers with a
// one-time token and saves the result.
func BootstrapCredentials(
	ctx context.Context,
	store ports.AgentStateStore,
	regToken string,
	agent domain.Agent,
	reg ports.AgentRegistrar,
) (*BootstrapResult, error) {
	creds, err := store.Load(ctx)
	if err == nil {
		return &BootstrapResult{AgentID: creds.AgentID, AgentToken: creds.AgentToken}, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("load agent state: %w", err)
	}

	if regToken == "" {
		return nil, fmt.Errorf("no agent state and PGWAY_AGENT_REGISTRATION_TOKEN is empty")
	}

	registered, agentToken, err := reg.Register(ctx, regToken, agent)
	if err != nil {
		return nil, fmt.Errorf("register agent: %w", err)
	}

	id := registered.Id
	if id == "" {
		id = agent.Id
	}

	creds = &ports.AgentHostCredentials{AgentID: id, AgentToken: agentToken}
	if err := store.Save(ctx, creds); err != nil {
		return nil, err
	}

	return &BootstrapResult{AgentID: id, AgentToken: agentToken}, nil
}
