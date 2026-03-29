package controlplane

import (
	"context"
	"fmt"
	"time"

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

	now := time.Now()
	if existing, err := s.poolRepo.Find(ctx, pool.Id); err == nil {
		pool.CreatedAt = existing.CreatedAt
	} else {
		pool.CreatedAt = now
	}
	pool.UpdatedAt = now

	if err := s.poolRepo.Save(ctx, pool); err != nil {
		return nil, fmt.Errorf("save pool: %w", err)
	}

	zap.L().Info("pool applied", zap.String("name", pool.Id))
	return pool, nil
}

func (s *Service) GetPool(ctx context.Context, name string) (*domain.Pool, error) {
	if name == "" {
		return nil, fmt.Errorf("pool name is required")
	}
	return s.poolRepo.Find(ctx, name)
}

func (s *Service) ListPools(ctx context.Context, params domain.ListParams) (domain.ListResult[domain.Pool], error) {
	if params.PageSize > domain.DefaultMaxPageSize {
		params.PageSize = domain.DefaultMaxPageSize
	}
	return s.poolRepo.List(ctx, params)
}

func (s *Service) DeletePool(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("pool name is required")
	}

	if err := s.poolRepo.Delete(ctx, name); err != nil {
		return err
	}

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
		pool.ProxyIds = spec.ProxyIds
	case domain.PoolTypeDynamic:
		pool.Selector = &domain.LabelSelector{
			Allow: spec.Selector.Allow,
		}
	}

	return pool
}
