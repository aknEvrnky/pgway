package agenthost

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/ports"
)

// Resync reloads DP cache/LB state from the control plane and reconciles
// HTTP entrypoint listeners so missed change events are healed.
func Resync(ctx context.Context, app ports.Application, listeners ports.ListenerReconciler) error {
	if err := app.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	if listeners == nil {
		return nil
	}
	if err := listeners.ReconcileListeners(ctx); err != nil {
		return fmt.Errorf("reconcile listeners: %w", err)
	}
	return nil
}
