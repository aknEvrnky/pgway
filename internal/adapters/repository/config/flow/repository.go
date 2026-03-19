package flow

import (
	"context"
	"errors"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
)

var (
	ErrConfigNotLoaded = errors.New("config is not loaded")
	ErrFlowNotFound    = errors.New("flow not found")
)

type ConfigRepository struct {
	flows map[string]*domain.Flow
}

func NewConfigRepository(c *config.Config) (*ConfigRepository, error) {
	if c == nil {
		return nil, ErrConfigNotLoaded
	}

	m := make(map[string]*domain.Flow, len(c.Flows))
	for _, f := range c.Flows {
		mapped := mapToDomain(f)
		m[mapped.Id] = mapped
	}

	return &ConfigRepository{flows: m}, nil
}

func (r *ConfigRepository) GetAll(ctx context.Context) ([]*domain.Flow, error) {
	results := make([]*domain.Flow, 0, len(r.flows))

	for _, ep := range r.flows {
		results = append(results, ep)
	}

	return results, nil
}

func (r *ConfigRepository) Find(ctx context.Context, id string) (*domain.Flow, error) {
	f, ok := r.flows[id]
	if !ok {
		return nil, ErrFlowNotFound
	}
	return f, nil
}

func mapToDomain(f config.FlowConfig) *domain.Flow {
	return &domain.Flow{
		Id:         f.Id,
		RouterId:   f.RouterId,
		BalancerId: f.BalancerId,
	}
}
