package ports

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrChangeEventInvalid = errors.New("change event is invalid")
)

type ResourceType string

const (
	ResourceTypeEntrypoint ResourceType = "entrypoint"
	ResourceTypeFlow       ResourceType = "flow"
	ResourceTypeRouter     ResourceType = "router"
	ResourceTypeProxy      ResourceType = "proxy"
	ResourceTypePool       ResourceType = "pool"
	// ResourceTypeBalancer is the event/wire name for schema kind LoadBalancer.
	ResourceTypeBalancer ResourceType = "balancer"
)

func (t ResourceType) IsValid() bool {
	switch t {
	case ResourceTypeEntrypoint, ResourceTypeFlow, ResourceTypeRouter,
		ResourceTypeProxy, ResourceTypePool, ResourceTypeBalancer:
		return true
	}
	return false
}

type ChangeKind string

const (
	ChangeKindSaved   ChangeKind = "saved"
	ChangeKindDeleted ChangeKind = "deleted"
)

func (k ChangeKind) IsValid() bool {
	switch k {
	case ChangeKindSaved, ChangeKindDeleted:
		return true
	}
	return false
}

type ChangeEvent struct {
	ID           string
	ResourceType ResourceType
	ChangeKind   ChangeKind
}

func (e *ChangeEvent) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("id is empty: %w", ErrChangeEventInvalid)
	}

	if !e.ResourceType.IsValid() {
		return fmt.Errorf("resource type %q is invalid: %w", e.ResourceType, ErrChangeEventInvalid)
	}

	if !e.ChangeKind.IsValid() {
		return fmt.Errorf("change kind %q is invalid: %w", e.ChangeKind, ErrChangeEventInvalid)
	}

	return nil
}

type EventPublisherPort interface {
	Publish(ctx context.Context, e ChangeEvent) error
}

type EventSubscriberPort interface {
	// Subscribe returns a channel that is closed when ctx is canceled.
	Subscribe(ctx context.Context) <-chan ChangeEvent
}

type EventHandler interface {
	HandleEvent(ctx context.Context, e ChangeEvent) error
}
