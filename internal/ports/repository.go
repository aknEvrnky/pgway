package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type EntryPointRepositoryPort interface {
	List(ctx context.Context, params domain.ListParams, filter domain.EntrypointFilter) (domain.ListResult[domain.Entrypoint], error)
	Find(ctx context.Context, id string) (*domain.Entrypoint, error)
	Save(ctx context.Context, ep *domain.Entrypoint) error
	Delete(ctx context.Context, id string) error
}

type FlowRepositoryPort interface {
	List(ctx context.Context, params domain.ListParams, filter domain.FlowFilter) (domain.ListResult[domain.Flow], error)
	Find(ctx context.Context, id string) (*domain.Flow, error)
	Save(ctx context.Context, flow *domain.Flow) error
	Delete(ctx context.Context, id string) error
}

type RouterRepositoryPort interface {
	List(ctx context.Context, params domain.ListParams, filter domain.RouterFilter) (domain.ListResult[domain.Router], error)
	Find(ctx context.Context, id string) (*domain.Router, error)
	Save(ctx context.Context, router *domain.Router) error
	Delete(ctx context.Context, id string) error
}

type LoadBalancerRepositoryPort interface {
	List(ctx context.Context, params domain.ListParams, filter domain.BalancerFilter) (domain.ListResult[domain.LoadBalancer], error)
	Find(ctx context.Context, id string) (*domain.LoadBalancer, error)
	Save(ctx context.Context, lb *domain.LoadBalancer) error
	Delete(ctx context.Context, id string) error
}

type PoolRepositoryPort interface {
	List(ctx context.Context, params domain.ListParams, filter domain.PoolFilter) (domain.ListResult[domain.Pool], error)
	Find(ctx context.Context, id string) (*domain.Pool, error)
	Save(ctx context.Context, pool *domain.Pool) error
	Delete(ctx context.Context, id string) error
}

type UserRepositoryPort interface {
	List(ctx context.Context, params domain.ListParams, filter domain.UserFilter) (domain.ListResult[domain.User], error)
	Find(ctx context.Context, id string) (*domain.User, error)
	Count(ctx context.Context) (int, error)
	Save(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id string) error
}

type TokenRepositoryPort interface {
	Find(ctx context.Context, hash string) (*domain.Token, error)
	Save(ctx context.Context, token *domain.Token) error
	Delete(ctx context.Context, hash string) error
	DeleteByUserId(ctx context.Context, userId string) error
	DeleteByAgentId(ctx context.Context, agentId string) error
}

type ProxyRepositoryPort interface {
	List(ctx context.Context, params domain.ListParams, filter domain.ProxyFilter) (domain.ListResult[domain.Proxy], error)
	Find(ctx context.Context, id string) (*domain.Proxy, error)
	GetByIds(ctx context.Context, ids []string) ([]*domain.Proxy, error)
	FindByLabels(ctx context.Context, labels map[string]string) ([]*domain.Proxy, error)
	Save(ctx context.Context, proxy *domain.Proxy) error
	Delete(ctx context.Context, id string) error
}

type AgentRepositoryPort interface {
	List(ctx context.Context, params domain.ListParams, filter domain.AgentFilter) (domain.ListResult[domain.Agent], error)
	Find(ctx context.Context, id string) (*domain.Agent, error)
	// Create inserts a new agent. It fails with domain.ErrAgentExists when
	// the id is taken; the uniqueness check is atomic at the storage layer.
	Create(ctx context.Context, agent *domain.Agent) error
	Save(ctx context.Context, agent *domain.Agent) error
	Delete(ctx context.Context, id string) error
}

type RegistrationTokenRepositoryPort interface {
	Save(ctx context.Context, token *domain.RegistrationToken) error
	// Consume atomically finds and deletes the token by hash: exactly one
	// concurrent caller succeeds. The deleted record is returned so the
	// caller can run an expiry check.
	Consume(ctx context.Context, hash string) (*domain.RegistrationToken, error)
}
