package api

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/event"
	"go.uber.org/zap"
)

func (a *Application) HandleEvent(ctx context.Context, e event.ChangeEvent) error {
	zap.L().Info("event received",
		zap.String("id", e.ID),
		zap.String("resource_type", string(e.ResourceType)),
		zap.String("change_kind", string(e.ChangeKind)),
	)

	if e.ChangeKind == event.ChangeKindSaved || e.ChangeKind == event.ChangeKindDeleted {
		switch e.ResourceType {
		case event.ResourceTypeEntrypoint, event.ResourceTypeFlow, event.ResourceTypeRouter:
			// warm up cache
			if err := a.warmupCache(ctx); err != nil {
				return err
			}

			// validate app
			if err := a.validateAll(ctx); err != nil {
				return err
			}
		case event.ResourceTypePool, event.ResourceTypeProxy, event.ResourceTypeBalancer:
			// bootstrap load balancers
			if err := a.balancerService.Bootstrap(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}
