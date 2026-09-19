package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"

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

	if flow.RouterId != "" {
		if err := s.requireRouter(ctx, string(ports.ResourceTypeFlow), flow.Id, flow.RouterId); err != nil {
			return nil, err
		}
	}
	if flow.BalancerId != "" {
		if err := s.requireBalancer(ctx, string(ports.ResourceTypeFlow), flow.Id, flow.BalancerId); err != nil {
			return nil, err
		}
	}

	now := time.Now()
	existing, err := s.flowRepo.Find(ctx, flow.Id)
	var existingCreated time.Time
	if err == nil {
		existingCreated = existing.CreatedAt
	}
	flow.CreatedAt, err = resolveCreatedAt(existingCreated, err, now)
	if err != nil {
		return nil, fmt.Errorf("find flow: %w", err)
	}
	flow.UpdatedAt = now

	if err := s.flowRepo.Save(ctx, flow); err != nil {
		return nil, fmt.Errorf("save flow: %w", err)
	}

	_ = s.fireEvent(ctx, flow.Id, ports.ResourceTypeFlow, ports.ChangeKindSaved)

	zap.L().Info("flow applied", zap.String("name", flow.Id))
	return flow, nil
}

func (s *Service) GetFlow(ctx context.Context, name string) (*domain.Flow, error) {
	if name == "" {
		return nil, fmt.Errorf("flow name is required")
	}
	return s.flowRepo.Find(ctx, name)
}

func (s *Service) ListFlows(ctx context.Context, params domain.ListParams, filter domain.FlowFilter) (domain.ListResult[domain.Flow], error) {
	if params.PageSize > domain.DefaultMaxPageSize {
		params.PageSize = domain.DefaultMaxPageSize
	}
	return s.flowRepo.List(ctx, params, filter)
}

func (s *Service) DeleteFlow(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("flow name is required")
	}

	if err := s.rejectIfEntrypointsReferenceFlow(ctx, name); err != nil {
		return err
	}

	if err := s.flowRepo.Delete(ctx, name); err != nil {
		return err
	}

	_ = s.fireEvent(ctx, name, ports.ResourceTypeFlow, ports.ChangeKindDeleted)

	zap.L().Info("flow deleted", zap.String("name", name))
	return nil
}
