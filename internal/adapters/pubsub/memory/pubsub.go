package memory

import (
	"context"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/google/uuid"
)

type PubSub struct {
	bufferSize int
	mu         sync.Mutex
	subs       map[string]chan event.ChangeEvent
}

func NewPubSub(bufferSize int) *PubSub {
	subs := make(map[string]chan event.ChangeEvent, bufferSize)

	return &PubSub{
		bufferSize: bufferSize,
		subs:       subs,
		mu:         sync.Mutex{},
	}
}

func (p *PubSub) Publish(ctx context.Context, event event.ChangeEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ch := range p.subs {
		select {
		case ch <- event:
		default:
		}
	}

	return nil
}

func (p *PubSub) Subscribe(ctx context.Context) <-chan event.ChangeEvent {
	ch := make(chan event.ChangeEvent, p.bufferSize)
	chId := uuid.New().String()
	p.mu.Lock()
	p.subs[chId] = ch
	p.mu.Unlock()

	go func(id string) {
		<-ctx.Done()

		p.mu.Lock()
		delete(p.subs, id)
		p.mu.Unlock()

		close(ch)
	}(chId)

	return ch
}
