package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

func (s *Service) ApplyRouterV1(ctx context.Context, meta schema.Metadata, spec routerv1.RouterSpecV1) (*domain.Router, error) {
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("spec validation: %w", err)
	}

	if meta.Name == "" {
		meta.Name = ulid.Make().String()
		zap.L().Info("generated router name", zap.String("name", meta.Name))
	}

	router := routerFromSpecV1(meta, spec)

	if err := router.Validate(); err != nil {
		return nil, fmt.Errorf("domain validation: %w", err)
	}

	now := time.Now()
	if existing, err := s.routerRepo.Find(ctx, router.Id); err == nil {
		router.CreatedAt = existing.CreatedAt
	} else {
		router.CreatedAt = now
	}
	router.UpdatedAt = now

	if err := s.routerRepo.Save(ctx, router); err != nil {
		return nil, fmt.Errorf("save router: %w", err)
	}

	zap.L().Info("router applied", zap.String("name", router.Id))
	return router, nil
}

func (s *Service) GetRouter(ctx context.Context, name string) (*domain.Router, error) {
	if name == "" {
		return nil, fmt.Errorf("router name is required")
	}
	return s.routerRepo.Find(ctx, name)
}

func (s *Service) ListRouters(ctx context.Context, params domain.ListParams, filter domain.RouterFilter) (domain.ListResult[domain.Router], error) {
	if params.PageSize > domain.DefaultMaxPageSize {
		params.PageSize = domain.DefaultMaxPageSize
	}
	return s.routerRepo.List(ctx, params, filter)
}

func (s *Service) DeleteRouter(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("router name is required")
	}

	if err := s.routerRepo.Delete(ctx, name); err != nil {
		return err
	}

	zap.L().Info("router deleted", zap.String("name", name))
	return nil
}

func routerFromSpecV1(meta schema.Metadata, spec routerv1.RouterSpecV1) *domain.Router {
	rules := make([]*domain.RouterRule, 0, len(spec.Rules))

	for _, r := range spec.Rules {
		rule := &domain.RouterRule{
			Id:     r.Id,
			Match:  matchFromSpec(r.Match),
			Target: r.Target,
		}
		rules = append(rules, rule)
	}

	return &domain.Router{
		Id:          meta.Name,
		Title:       spec.Title,
		Description: spec.Description,
		Rules:       rules,
	}
}

func matchFromSpec(m routerv1.MatchSpec) domain.RouterMatch {
	match := domain.RouterMatch{
		Type:  domain.MatchType(m.Type),
		Value: m.Value,
	}

	for _, c := range m.All {
		match.All = append(match.All, domain.RouterCondition{
			Type:  domain.MatchType(c.Type),
			Value: c.Value,
		})
	}

	for _, c := range m.Any {
		match.Any = append(match.Any, domain.RouterCondition{
			Type:  domain.MatchType(c.Type),
			Value: c.Value,
		})
	}

	if m.Not != nil {
		match.Not = &domain.RouterCondition{
			Type:  domain.MatchType(m.Not.Type),
			Value: m.Not.Value,
		}
	}

	return match
}
