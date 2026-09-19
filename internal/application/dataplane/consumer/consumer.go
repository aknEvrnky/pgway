package consumer

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

const coalescedEventID = "_coalesced"

type coalesceKey string

const (
	keyBalancer   coalesceKey = "balancer"
	keyTopology   coalesceKey = "topology"
	keyEntrypoint coalesceKey = "entrypoint"
)

// CoalesceConfig controls setTimeout-style event batching on the consumer path.
// Window <= 0 disables coalescing (immediate dispatch).
type CoalesceConfig struct {
	Window    time.Duration
	MaxBuffer int
}

// Consumer drains a subscription and dispatches every event to its handlers
// in registration order. The ordering is part of the contract: later handlers
// may depend on the effects of earlier ones (e.g. the http adapter reads the
// cache that the application handler refreshes).
//
// When CoalesceConfig.Window > 0, events are buffered per key and flushed
// after the window (or earlier when MaxBuffer is reached).
type Consumer struct {
	subscriber ports.EventSubscriberPort
	handlers   []ports.EventHandler
	log        *zap.Logger
	cfg        CoalesceConfig

	mu     sync.Mutex
	keys   map[coalesceKey]*keyState
	closed bool
}

type keyState struct {
	timer       *time.Timer
	count       int
	dirty       bool                        // balancer / topology
	entrypoints map[string]ports.ChangeKind // last-wins
}

func NewConsumer(log *zap.Logger, subscriber ports.EventSubscriberPort, cfg CoalesceConfig, handlers ...ports.EventHandler) *Consumer {
	if log == nil {
		log = zap.NewNop()
	}
	return &Consumer{
		subscriber: subscriber,
		handlers:   handlers,
		log:        log,
		cfg:        cfg,
		keys:       make(map[coalesceKey]*keyState),
	}
}

// ConsumeEvents blocks until ctx is canceled. Handler errors are logged and
// swallowed: events are best-effort hints, a failing reload must not stop the
// consumer or the process.
func (c *Consumer) ConsumeEvents(ctx context.Context) error {
	ch := c.subscriber.Subscribe(ctx)

	for e := range ch {
		if c.cfg.Window <= 0 {
			c.dispatch(ctx, e)
			continue
		}
		c.offer(ctx, e)
	}

	if c.cfg.Window > 0 {
		// Subscription closed because ctx was canceled; flush with a live context.
		c.flushAll(context.WithoutCancel(ctx))
	}
	return nil
}

func (c *Consumer) offer(ctx context.Context, e ports.ChangeEvent) {
	key, ok := coalesceKeyFor(e.ResourceType)
	if !ok {
		c.dispatch(ctx, e)
		return
	}

	var flushNow bool
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		c.dispatch(ctx, e)
		return
	}

	st := c.keys[key]
	if st == nil {
		st = &keyState{entrypoints: make(map[string]ports.ChangeKind)}
		c.keys[key] = st
	}

	st.count++
	switch key {
	case keyBalancer, keyTopology:
		st.dirty = true
	case keyEntrypoint:
		st.entrypoints[e.ID] = e.ChangeKind
	}

	if st.timer == nil {
		k := key
		st.timer = time.AfterFunc(c.cfg.Window, func() {
			c.flushKey(ctx, k)
		})
	}

	if c.cfg.MaxBuffer > 0 && st.count >= c.cfg.MaxBuffer {
		flushNow = true
	}
	c.mu.Unlock()

	if flushNow {
		c.flushKey(ctx, key)
	}
}

func (c *Consumer) flushAll(ctx context.Context) {
	c.mu.Lock()
	c.closed = true
	keys := make([]coalesceKey, 0, len(c.keys))
	for k := range c.keys {
		keys = append(keys, k)
	}
	c.mu.Unlock()

	for _, k := range keys {
		c.flushKey(ctx, k)
	}
}

func (c *Consumer) flushKey(ctx context.Context, key coalesceKey) {
	events := c.takeKeyEvents(key)
	for _, e := range events {
		c.dispatch(ctx, e)
	}
}

func (c *Consumer) takeKeyEvents(key coalesceKey) []ports.ChangeEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	st := c.keys[key]
	if st == nil {
		return nil
	}
	if st.timer != nil {
		st.timer.Stop()
		st.timer = nil
	}

	var out []ports.ChangeEvent
	switch key {
	case keyBalancer:
		if st.dirty {
			out = append(out, ports.ChangeEvent{
				ID:           coalescedEventID,
				ResourceType: ports.ResourceTypeProxy,
				ChangeKind:   ports.ChangeKindSaved,
			})
		}
	case keyTopology:
		if st.dirty {
			out = append(out, ports.ChangeEvent{
				ID:           coalescedEventID,
				ResourceType: ports.ResourceTypeFlow,
				ChangeKind:   ports.ChangeKindSaved,
			})
		}
	case keyEntrypoint:
		if len(st.entrypoints) > 0 {
			ids := make([]string, 0, len(st.entrypoints))
			for id := range st.entrypoints {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			for _, id := range ids {
				out = append(out, ports.ChangeEvent{
					ID:           id,
					ResourceType: ports.ResourceTypeEntrypoint,
					ChangeKind:   st.entrypoints[id],
				})
			}
		}
	}

	delete(c.keys, key)
	return out
}

func (c *Consumer) dispatch(ctx context.Context, e ports.ChangeEvent) {
	for _, handler := range c.handlers {
		if err := handler.HandleEvent(ctx, e); err != nil {
			c.log.Warn("handle event failed",
				zap.Error(err),
				zap.String("id", e.ID),
				zap.String("resource_type", string(e.ResourceType)),
				zap.String("change_kind", string(e.ChangeKind)),
			)
		}
	}
}

func coalesceKeyFor(t ports.ResourceType) (coalesceKey, bool) {
	switch t {
	case ports.ResourceTypeProxy, ports.ResourceTypePool, ports.ResourceTypeBalancer:
		return keyBalancer, true
	case ports.ResourceTypeFlow, ports.ResourceTypeRouter:
		return keyTopology, true
	case ports.ResourceTypeEntrypoint:
		return keyEntrypoint, true
	default:
		return "", false
	}
}
