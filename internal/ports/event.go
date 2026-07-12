package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/event"
)

type EventPublisherPort interface {
	Publish(ctx context.Context, event event.ChangeEvent) error
}

type EventSubscriberPort interface {
	// Subscribe returns a channel that is closed when ctx is canceled.
	Subscribe(ctx context.Context) <-chan event.ChangeEvent
}

type EventHandler interface {
	HandleEvent(ctx context.Context, e event.ChangeEvent) error
}
