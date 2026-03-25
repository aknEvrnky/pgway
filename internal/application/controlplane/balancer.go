package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

func (s *Service) ApplyBalancerV1(ctx context.Context, meta schema.Metadata, spec balancerv1.BalancerSpecV1) (*domain.LoadBalancer, error) {
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("spec validation: %w", err)
	}

	if meta.Name == "" {
		meta.Name = ulid.Make().String()
		zap.L().Info("generated balancer name", zap.String("name", meta.Name))
	}

	lb := &domain.LoadBalancer{
		Id:     meta.Name,
		Title:  spec.Title,
		Type:   domain.BalancerType(spec.Type),
		PoolId: spec.PoolId,
	}

	if !lb.Type.IsValid() {
		return nil, fmt.Errorf("domain validation: invalid balancer type %q", lb.Type)
	}

	now := time.Now()
	if existing, err := s.lbRepo.Find(ctx, lb.Id); err == nil {
		lb.CreatedAt = existing.CreatedAt
	} else {
		lb.CreatedAt = now
	}
	lb.UpdatedAt = now

	if err := s.lbRepo.Save(ctx, lb); err != nil {
		return nil, fmt.Errorf("save balancer: %w", err)
	}

	zap.L().Info("balancer applied", zap.String("name", lb.Id))
	return lb, nil
}

func (s *Service) GetBalancer(ctx context.Context, name string) (*domain.LoadBalancer, error) {
	if name == "" {
		return nil, fmt.Errorf("balancer name is required")
	}
	return s.lbRepo.Find(ctx, name)
}

func (s *Service) ListBalancers(ctx context.Context) ([]*domain.LoadBalancer, error) {
	return s.lbRepo.GetAll(ctx)
}

func (s *Service) DeleteBalancer(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("balancer name is required")
	}

	if err := s.lbRepo.Delete(ctx, name); err != nil {
		return err
	}

	zap.L().Info("balancer deleted", zap.String("name", name))
	return nil
}
