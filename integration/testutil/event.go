package testutil

import (
	"context"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/event"
)

type SpyPublisher struct {
	mu     sync.Mutex
	Events []event.ChangeEvent
}

func (s *SpyPublisher) Publish(ctx context.Context, e event.ChangeEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, e)

	return nil
}
