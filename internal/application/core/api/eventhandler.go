package api

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/ports"

	"go.uber.org/zap"
)

func (a *Application) HandleEvent(ctx context.Context, e ports.ChangeEvent) error {
	zap.L().Info("event received",
		zap.String("id", e.ID),
		zap.String("resource_type", string(e.ResourceType)),
		zap.String("change_kind", string(e.ChangeKind)),
	)

	if e.ChangeKind == ports.ChangeKindSaved || e.ChangeKind == ports.ChangeKindDeleted {
		switch e.ResourceType {
		case ports.ResourceTypeEntrypoint, ports.ResourceTypeFlow, ports.ResourceTypeRouter:
			// warm up cache
			if err := a.warmupCache(ctx); err != nil {
				return err
			}

			// validate app
			if err := a.validateAll(ctx); err != nil {
				return err
			}
		case ports.ResourceTypePool, ports.ResourceTypeProxy, ports.ResourceTypeBalancer:
			// bootstrap load balancers
			if err := a.balancerService.Bootstrap(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}
