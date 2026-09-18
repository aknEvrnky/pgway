package consumer

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

// Consumer drains a subscription and dispatches every event to its handlers
// in registration order. The ordering is part of the contract: later handlers
// may depend on the effects of earlier ones (e.g. the http adapter reads the
// cache that the application handler refreshes).
type Consumer struct {
	subscriber ports.EventSubscriberPort
	handlers   []ports.EventHandler
}

func NewConsumer(subscriber ports.EventSubscriberPort, handlers ...ports.EventHandler) *Consumer {
	return &Consumer{
		subscriber: subscriber,
		handlers:   handlers,
	}
}

// ConsumeEvents blocks until ctx is canceled. Handler errors are logged and
// swallowed: events are best-effort hints, a failing reload must not stop the
// consumer or the process.
func (c *Consumer) ConsumeEvents(ctx context.Context) error {
	ch := c.subscriber.Subscribe(ctx)

	for e := range ch {
		for _, handler := range c.handlers {
			if err := handler.HandleEvent(ctx, e); err != nil {
				zap.L().Warn("handle event failed",
					zap.Error(err),
					zap.String("id", e.ID),
					zap.String("resource_type", string(e.ResourceType)),
					zap.String("change_kind", string(e.ChangeKind)),
				)
			}
		}
	}

	return nil
}
