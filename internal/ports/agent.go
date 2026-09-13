package ports

import (
	"context"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type AgentManager interface {
	CreateRegistrationToken(ctx context.Context, ttl time.Duration) (string, error)                   // admin
	Register(ctx context.Context, regToken string, agent domain.Agent) (*domain.Agent, string, error) // → agent, agentToken
	Heartbeat(ctx context.Context) (time.Time, error)                                                 // identity from principal; → new token expiry
	Deregister(ctx context.Context) error                                                             // identity from principal
	ListAgents(ctx context.Context, params domain.ListParams, filter domain.AgentFilter) (domain.ListResult[domain.Agent], error)
	DeleteAgent(ctx context.Context, name string) error
}
