package controlplane

import (
	"context"
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

func (s *Service) ApplyEntrypointV1(ctx context.Context, meta schema.Metadata, spec entrypointv1.EntrypointSpecV1) (*domain.Entrypoint, error) {
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("spec validation: %w", err)
	}

	if meta.Name == "" {
		meta.Name = ulid.Make().String()
		zap.L().Info("generated entrypoint name", zap.String("name", meta.Name))
	}

	ep := &domain.Entrypoint{
		Id:       meta.Name,
		Title:    spec.Title,
		Protocol: domain.Protocol(spec.Protocol),
		Host:     spec.Host,
		Port:     spec.Port,
		FlowId:   spec.FlowId,
	}

	if err := ep.Validate(); err != nil {
		return nil, fmt.Errorf("domain validation: %w", err)
	}

	if err := s.requireFlow(ctx, string(ports.ResourceTypeEntrypoint), ep.Id, ep.FlowId); err != nil {
		return nil, err
	}

	now := time.Now()
	existing, err := s.epRepo.Find(ctx, ep.Id)
	var existingCreated time.Time
	if err == nil {
		existingCreated = existing.CreatedAt
	}
	ep.CreatedAt, err = resolveCreatedAt(existingCreated, err, now)
	if err != nil {
		return nil, fmt.Errorf("find entrypoint: %w", err)
	}
	ep.UpdatedAt = now

	if err := s.epRepo.Save(ctx, ep); err != nil {
		return nil, fmt.Errorf("save entrypoint: %w", err)
	}

	_ = s.fireEvent(ctx, ep.Id, ports.ResourceTypeEntrypoint, ports.ChangeKindSaved)

	zap.L().Info("entrypoint applied", zap.String("name", ep.Id))
	return ep, nil
}

func (s *Service) GetEntrypoint(ctx context.Context, name string) (*domain.Entrypoint, error) {
	if name == "" {
		return nil, fmt.Errorf("entrypoint name is required")
	}
	return s.epRepo.Find(ctx, name)
}

func (s *Service) ListEntrypoints(ctx context.Context, params domain.ListParams, filter domain.EntrypointFilter) (domain.ListResult[domain.Entrypoint], error) {
	if params.PageSize > domain.DefaultMaxPageSize {
		params.PageSize = domain.DefaultMaxPageSize
	}
	return s.epRepo.List(ctx, params, filter)
}

func (s *Service) DeleteEntrypoint(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("entrypoint name is required")
	}

	if err := s.epRepo.Delete(ctx, name); err != nil {
		return err
	}

	_ = s.fireEvent(ctx, name, ports.ResourceTypeEntrypoint, ports.ChangeKindDeleted)

	zap.L().Info("entrypoint deleted", zap.String("name", name))
	return nil
}
