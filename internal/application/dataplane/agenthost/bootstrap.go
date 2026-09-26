package agenthost

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

var (
	// ErrBootstrapConfig marks unretryable bootstrap misconfiguration.
	ErrBootstrapConfig = errors.New("bootstrap misconfigured")
	// ErrBootstrapState marks an unusable local agent state store.
	ErrBootstrapState = errors.New("agent state unusable")
)

// IsPermanentBootstrapError reports whether err must not be retried at DP
// startup: misconfiguration, an unusable state store, a taken agent name,
// or auth rejection.
func IsPermanentBootstrapError(err error) bool {
	return errors.Is(err, ErrBootstrapConfig) ||
		errors.Is(err, ErrBootstrapState) ||
		errors.Is(err, domain.ErrAgentExists) ||
		isAuthRejected(err)
}

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
		return nil, fmt.Errorf("%w: %w", ErrBootstrapState, err)
	}

	if regToken == "" {
		return nil, fmt.Errorf("%w: no agent state and PGWAY_AGENT_REGISTRATION_TOKEN is empty", ErrBootstrapConfig)
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
		// Registration succeeded but the credentials were not persisted;
		// the single-use token is consumed, so retrying Register cannot
		// recover and must not burn another attempt.
		return nil, fmt.Errorf("%w: save after register: %w", ErrBootstrapState, err)
	}

	return &BootstrapResult{AgentID: id, AgentToken: agentToken}, nil
}
