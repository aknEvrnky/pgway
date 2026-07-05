package ports

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/event"
)

type EventPublisher interface {
	Publish(ctx context.Context, event event.ChangeEvent) error
}

type EventSubscriber interface {
	// Subscribe returns a channel that is closed when ctx is canceled.
	Subscribe(ctx context.Context) <-chan event.ChangeEvent
}
