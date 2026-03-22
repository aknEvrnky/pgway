package controlplane

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

func (s *Service) ApplyFlowV1(ctx context.Context, meta schema.Metadata, spec flowv1.FlowSpecV1) (*domain.Flow, error) {
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("spec validation: %w", err)
	}

	if meta.Name == "" {
		meta.Name = ulid.Make().String()
		zap.L().Info("generated flow name", zap.String("name", meta.Name))
	}

	flow := &domain.Flow{
		Id:         meta.Name,
		RouterId:   spec.RouterId,
		BalancerId: spec.BalancerId,
	}

	if err := s.flowRepo.Save(ctx, flow); err != nil {
		return nil, fmt.Errorf("save flow: %w", err)
	}

	zap.L().Info("flow applied", zap.String("name", flow.Id))
	return flow, nil
}

func (s *Service) GetFlow(ctx context.Context, name string) (*domain.Flow, error) {
	if name == "" {
		return nil, fmt.Errorf("flow name is required")
	}
	return s.flowRepo.Find(ctx, name)
}

func (s *Service) ListFlows(ctx context.Context) ([]*domain.Flow, error) {
	return s.flowRepo.GetAll(ctx)
}

func (s *Service) DeleteFlow(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("flow name is required")
	}

	if err := s.flowRepo.Delete(ctx, name); err != nil {
		return err
	}

	zap.L().Info("flow deleted", zap.String("name", name))
	return nil
}
