package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

func (s *Service) ApplyPoolV1(ctx context.Context, meta schema.Metadata, spec poolv1.PoolSpecV1) (*domain.Pool, error) {
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("spec validation: %w", err)
	}

	if meta.Name == "" {
		meta.Name = ulid.Make().String()
		zap.L().Info("generated pool name", zap.String("name", meta.Name))
	}

	pool := poolFromSpecV1(meta, spec)

	if err := pool.Validate(); err != nil {
		return nil, fmt.Errorf("domain validation: %w", err)
	}

	if pool.Type == domain.PoolTypeStatic {
		for _, m := range pool.Members {
			if err := s.requireProxy(ctx, string(ports.ResourceTypePool), pool.Id, m.ProxyId); err != nil {
				return nil, err
			}
		}
	}

	if pool.Type != domain.PoolTypeStatic {
		if err := s.rejectIfWeightedBalancersReference(ctx, pool.Id); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	existing, err := s.poolRepo.Find(ctx, pool.Id)
	var existingCreated time.Time
	if err == nil {
		existingCreated = existing.CreatedAt
	}
	pool.CreatedAt, err = resolveCreatedAt(existingCreated, err, now)
	if err != nil {
		return nil, fmt.Errorf("find pool: %w", err)
	}
	pool.UpdatedAt = now

	if err := s.poolRepo.Save(ctx, pool); err != nil {
		return nil, fmt.Errorf("save pool: %w", err)
	}

	_ = s.fireEvent(ctx, pool.Id, ports.ResourceTypePool, ports.ChangeKindSaved)

	zap.L().Info("pool applied", zap.String("name", pool.Id))
	return pool, nil
}

func (s *Service) GetPool(ctx context.Context, name string) (*domain.Pool, error) {
	if name == "" {
		return nil, fmt.Errorf("pool name is required")
	}
	return s.poolRepo.Find(ctx, name)
}

func (s *Service) ListPools(ctx context.Context, params domain.ListParams, filter domain.PoolFilter) (domain.ListResult[domain.Pool], error) {
	if params.PageSize > domain.DefaultMaxPageSize {
		params.PageSize = domain.DefaultMaxPageSize
	}
	return s.poolRepo.List(ctx, params, filter)
}

func (s *Service) DeletePool(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("pool name is required")
	}

	if err := s.rejectIfBalancersReferencePool(ctx, name); err != nil {
		return err
	}

	if err := s.poolRepo.Delete(ctx, name); err != nil {
		return err
	}

	_ = s.fireEvent(ctx, name, ports.ResourceTypePool, ports.ChangeKindDeleted)

	zap.L().Info("pool deleted", zap.String("name", name))
	return nil
}

func poolFromSpecV1(meta schema.Metadata, spec poolv1.PoolSpecV1) *domain.Pool {
	pool := &domain.Pool{
		Id:     meta.Name,
		Title:  spec.Title,
		Type:   domain.PoolType(spec.Type),
		Labels: meta.Labels,
	}

	switch pool.Type {
	case domain.PoolTypeStatic:
		pool.Members = make([]domain.PoolMember, 0, len(spec.Members))
		for _, m := range spec.Members {
			pool.Members = append(pool.Members, domain.PoolMember{
				ProxyId: m.ProxyId,
				Weight:  m.ResolvedWeight(),
			})
		}
	case domain.PoolTypeDynamic:
		pool.Selector = &domain.LabelSelector{
			Allow: spec.Selector.Allow,
		}
	}

	return pool
}

func (s *Service) rejectIfWeightedBalancersReference(ctx context.Context, poolID string) error {
	result, err := s.lbRepo.List(ctx, domain.ListParams{PageSize: 0}, domain.BalancerFilter{
		PoolId: poolID,
		Type:   string(domain.BalancerTypeWeighted),
	})
	if err != nil {
		return fmt.Errorf("checking weighted balancers for pool %q: %w", poolID, err)
	}
	if len(result.Items) > 0 {
		return fmt.Errorf("pool %q is referenced by weighted balancer %q; change or delete the balancer before making the pool non-static", poolID, result.Items[0].Id)
	}
	return nil
}
