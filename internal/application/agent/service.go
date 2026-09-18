package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

// Service owns the agent lifecycle. Credential mechanics stay in auth via
// ports.AgentCredentials — this package orchestrates registry + tokens.
type Service struct {
	agents        ports.AgentRepositoryPort
	creds         ports.AgentCredentials
	agentTokenTTL time.Duration
}

func NewService(
	agents ports.AgentRepositoryPort,
	creds ports.AgentCredentials,
	agentTokenTTL time.Duration,
) *Service {
	return &Service{
		agents:        agents,
		creds:         creds,
		agentTokenTTL: agentTokenTTL,
	}
}

func (s *Service) CreateRegistrationToken(ctx context.Context, ttl time.Duration) (string, error) {
	return s.creds.CreateRegistrationToken(ctx, ttl)
}

// Register consumes a single-use registration token, creates the agent record,
// then issues a per-agent bearer token. Ordering is fail-clean: consume →
// create → issue. A failure after consume leaves "token burned, registry
// clean" — admin mints a fresh token and retries.
func (s *Service) Register(ctx context.Context, regToken string, agent domain.Agent) (*domain.Agent, string, error) {
	if err := s.creds.ConsumeRegistrationToken(ctx, regToken); err != nil {
		return nil, "", err
	}

	if err := agent.Validate(); err != nil {
		return nil, "", fmt.Errorf("validate agent: %w", err)
	}

	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now

	if err := s.agents.Create(ctx, &agent); err != nil {
		return nil, "", err
	}

	agentToken, err := s.creds.IssueAgentToken(ctx, agent.Id, s.agentTokenTTL)
	if err != nil {
		return nil, "", fmt.Errorf("issue agent token: %w", err)
	}

	return &agent, agentToken, nil
}

// Heartbeat refreshes LastHeartbeat and slides the agent token TTL forward.
// Identity comes from the authenticated principal; the raw bearer is needed
// so ExtendAgentToken can push that exact credential's expiry.
func (s *Service) Heartbeat(ctx context.Context) (time.Time, error) {
	agent, err := agentFromContext(ctx)
	if err != nil {
		return time.Time{}, err
	}

	rawToken, ok := ports.TokenFromContext(ctx)
	if !ok || rawToken == "" {
		return time.Time{}, ErrTokenRequired
	}

	now := time.Now()
	agent.LastHeartbeat = &now
	agent.UpdatedAt = now

	if err := s.agents.Save(ctx, agent); err != nil {
		return time.Time{}, fmt.Errorf("save agent: %w", err)
	}

	if err := s.creds.ExtendAgentToken(ctx, rawToken, s.agentTokenTTL); err != nil {
		return time.Time{}, fmt.Errorf("extend agent token: %w", err)
	}

	return now.Add(s.agentTokenTTL), nil
}

// Deregister clears LastHeartbeat so status derives to passive. The agent
// record and token stay — a restart reuses them without burning a new
// registration token.
func (s *Service) Deregister(ctx context.Context) error {
	agent, err := agentFromContext(ctx)
	if err != nil {
		return err
	}

	agent.LastHeartbeat = nil
	agent.UpdatedAt = time.Now()

	if err := s.agents.Save(ctx, agent); err != nil {
		return fmt.Errorf("save agent: %w", err)
	}

	return nil
}

func (s *Service) ListAgents(ctx context.Context, params domain.ListParams, filter domain.AgentFilter) (domain.ListResult[domain.Agent], error) {
	if params.PageSize > domain.DefaultMaxPageSize {
		params.PageSize = domain.DefaultMaxPageSize
	}
	return s.agents.List(ctx, params, filter)
}

// DeleteAgent revokes tokens first, then removes the registry record.
// Revoke-first is fail-safe: a crash leaves "tokens dead, record listed"
// (harmless, delete again) instead of "record gone, token alive".
func (s *Service) DeleteAgent(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("agent name is required")
	}

	if _, err := s.agents.Find(ctx, name); err != nil {
		return ErrAgentNotFound
	}

	if err := s.creds.RevokeAgentTokens(ctx, name); err != nil {
		return fmt.Errorf("revoke agent tokens: %w", err)
	}

	if err := s.agents.Delete(ctx, name); err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}

	return nil
}

func agentFromContext(ctx context.Context) (*domain.Agent, error) {
	principal, ok := ports.PrincipalFromContext(ctx)
	if !ok || principal == nil || principal.Agent == nil {
		return nil, ErrAgentRequired
	}
	// Copy so callers can mutate without touching the principal's snapshot.
	copied := *principal.Agent
	return &copied, nil
}

var _ ports.AgentManager = (*Service)(nil)
