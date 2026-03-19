package router

import (
	"context"
	"errors"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
)

var (
	ErrConfigNotLoaded = errors.New("config is not loaded")
	ErrRouterNotFound  = errors.New("router not found")
)

type ConfigRepository struct {
	routers map[string]*domain.Router
}

func NewConfigRepository(c *config.Config) (*ConfigRepository, error) {
	if c == nil {
		return nil, ErrConfigNotLoaded
	}

	m := make(map[string]*domain.Router, len(c.Routers))
	for _, r := range c.Routers {
		mapped := mapToDomain(r)
		m[mapped.Id] = mapped
	}

	return &ConfigRepository{routers: m}, nil
}

func (r *ConfigRepository) GetAll(ctx context.Context) ([]*domain.Router, error) {
	results := make([]*domain.Router, 0, len(r.routers))

	for _, r := range r.routers {
		results = append(results, r)
	}

	return results, nil
}

func (r *ConfigRepository) Find(ctx context.Context, id string) (*domain.Router, error) {
	ep, ok := r.routers[id]
	if !ok {
		return nil, ErrRouterNotFound
	}
	return ep, nil
}

func mapToCondition(rc config.RouterConditionConfig) *domain.RouterCondition {
	return &domain.RouterCondition{
		Type:  domain.MatchType(rc.Type),
		Value: rc.Value,
	}
}

func mapToMatch(rm config.RouterMatchConfig) domain.RouterMatch {
	all := make([]domain.RouterCondition, 0, len(rm.All))

	for _, rc := range rm.All {
		condition := *mapToCondition(rc)
		all = append(all, condition)
	}

	anyConds := make([]domain.RouterCondition, 0, len(rm.Any))

	for _, rc := range rm.Any {
		condition := *mapToCondition(rc)
		anyConds = append(anyConds, condition)
	}

	var notCond *domain.RouterCondition

	if rm.Not != nil {
		notCond = mapToCondition(*rm.Not)
	}

	return domain.RouterMatch{
		Type:  domain.MatchType(rm.Type),
		Value: rm.Value,
		All:   all,
		Any:   anyConds,
		Not:   notCond,
	}
}

func mapToRule(rr config.RouterRuleConfig) *domain.RouterRule {
	return &domain.RouterRule{
		Id:     rr.Id,
		Match:  mapToMatch(rr.Match),
		Target: rr.Target,
	}
}

func mapToDomain(rc config.RouterConfig) *domain.Router {
	rules := make([]*domain.RouterRule, 0, len(rc.Rules))

	for _, rule := range rc.Rules {
		rules = append(rules, mapToRule(rule))
	}

	return &domain.Router{
		Id:          rc.Id,
		Title:       rc.Title,
		Description: rc.Description,
		Rules:       rules,
	}
}
