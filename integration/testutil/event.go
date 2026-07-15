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

type SpyHandler struct {
	mu     sync.Mutex
	events []event.ChangeEvent
}

func (s *SpyHandler) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

func (s *SpyHandler) HandleEvent(ctx context.Context, e event.ChangeEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)

	return nil
}
