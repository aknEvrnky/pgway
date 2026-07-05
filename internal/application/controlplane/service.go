package controlplane

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

type Service struct {
	proxyRepo  ports.ProxyRepositoryPort
	poolRepo   ports.PoolRepositoryPort
	lbRepo     ports.LoadBalancerRepositoryPort
	routerRepo ports.RouterRepositoryPort
	flowRepo   ports.FlowRepositoryPort
	epRepo     ports.EntryPointRepositoryPort

	eventPublisher ports.EventPublisher
}

func NewService(
	proxyRepo ports.ProxyRepositoryPort,
	poolRepo ports.PoolRepositoryPort,
	lbRepo ports.LoadBalancerRepositoryPort,
	routerRepo ports.RouterRepositoryPort,
	flowRepo ports.FlowRepositoryPort,
	epRepo ports.EntryPointRepositoryPort,
	eventPublisher ports.EventPublisher,
) *Service {
	return &Service{
		proxyRepo:      proxyRepo,
		poolRepo:       poolRepo,
		lbRepo:         lbRepo,
		routerRepo:     routerRepo,
		flowRepo:       flowRepo,
		epRepo:         epRepo,
		eventPublisher: eventPublisher,
	}
}

func (s *Service) fireEvent(ctx context.Context, id string, resType event.ResourceType, chKind event.ChangeKind) error {
	changeEvent := event.ChangeEvent{
		ID:           id,
		ResourceType: resType,
		ChangeKind:   chKind,
	}

	err := s.eventPublisher.Publish(ctx, changeEvent)
	if err != nil {
		zap.L().Warn(fmt.Sprintf("event could not published: %s", err.Error()),
			zap.String("resource_id", changeEvent.ID),
			zap.String("resource_type", string(changeEvent.ResourceType)),
			zap.String("change_kind", string(changeEvent.ChangeKind)),
		)

		return err
	}

	return nil
}
