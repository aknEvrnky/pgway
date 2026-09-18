package testutil

import (
	"context"
	"errors"
	"sync"

	"github.com/aknEvrnky/pgway/internal/ports"
)

type SpyPublisher struct {
	mu     sync.Mutex
	Events []ports.ChangeEvent
}

func (s *SpyPublisher) Publish(ctx context.Context, e ports.ChangeEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, e)

	return nil
}

type SpyHandler struct {
	mu     sync.Mutex
	events []ports.ChangeEvent
}

func (s *SpyHandler) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}

func (s *SpyHandler) HandleEvent(ctx context.Context, e ports.ChangeEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)

	return nil
}

// FailingPublisher always fails to publish. It exists to prove that event
// publishing is best-effort: a broken publisher must never fail a write.
type FailingPublisher struct{}

func (FailingPublisher) Publish(_ context.Context, _ ports.ChangeEvent) error {
	return errors.New("publish failed")
}
