package config

import (
	"context"
	"errors"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
)

var (
	ErrConfigNotLoaded    = errors.New("config is not loaded")
	ErrEntryPointNotFound = errors.New("entrypoint not found")
)

type ConfigRepository struct {
	entrypoints map[string]*domain.Entrypoint
}

func NewConfigRepository(c *config.Config) (*ConfigRepository, error) {
	if c == nil {
		return nil, ErrConfigNotLoaded
	}

	m := make(map[string]*domain.Entrypoint, len(c.EntryPoints))
	for _, ep := range c.EntryPoints {
		mapped := mapToDomain(ep)
		m[mapped.Id] = mapped
	}

	return &ConfigRepository{entrypoints: m}, nil
}

func (r *ConfigRepository) GetAll(ctx context.Context) ([]*domain.Entrypoint, error) {
	results := make([]*domain.Entrypoint, 0, len(r.entrypoints))

	for _, ep := range r.entrypoints {
		results = append(results, ep)
	}

	return results, nil
}

func (r *ConfigRepository) Find(ctx context.Context, id string) (*domain.Entrypoint, error) {
	ep, ok := r.entrypoints[id]
	if !ok {
		return nil, ErrEntryPointNotFound
	}
	return ep, nil
}

func mapToDomain(ep config.EntrypointConfig) *domain.Entrypoint {
	return &domain.Entrypoint{
		Id:       ep.Id,
		Title:    ep.Title,
		Protocol: ep.Protocol,
		Host:     ep.Host,
		Port:     ep.Port,
		Flow:     ep.Flow,
	}
}
