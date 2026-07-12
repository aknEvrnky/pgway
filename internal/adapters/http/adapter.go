package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type contextKey string

const entrypointContextKey contextKey = "entry_point_id"

// serverShutdownTimeout bounds how long HandleEvent waits for an old server
// to drain before replacing it. Without a bound a slow connection would block
// the event consumer loop indefinitely.
const serverShutdownTimeout = 10 * time.Second

type Adapter struct {
	api       ports.Application
	transport ports.ProxyTransportPort
	servers   map[string]*http.Server
	mu        sync.Mutex
}

// HandleEvent reacts to entrypoint changes: it stops the server for the old
// definition (if any) and, on save, builds and starts a server for the new
// one. It reads entrypoints through the application cache, so it must run
// after the cache-refreshing handler in the consumer chain.
func (a *Adapter) HandleEvent(ctx context.Context, e event.ChangeEvent) error {
	// skip if no modification on entrypoint
	if e.ResourceType != event.ResourceTypeEntrypoint {
		return nil
	}

	// stop and forget the old server, if one is running
	a.mu.Lock()
	oldServer, ok := a.servers[e.ID]
	delete(a.servers, e.ID)
	a.mu.Unlock()

	if ok {
		shutdownCtx, cancel := context.WithTimeout(ctx, serverShutdownTimeout)
		defer cancel()

		// best effort: a drain timeout must not prevent the new server from
		// starting; a bind conflict, if any, is logged by startServer
		if err := oldServer.Shutdown(shutdownCtx); err != nil {
			zap.L().Warn("shutting down old server", zap.Error(err), zap.String("entrypoint", e.ID))
		}
	}

	if e.ChangeKind != event.ChangeKindSaved {
		return nil
	}

	// create / update: build the server from the refreshed cache and start it
	eps, err := a.api.EntryPoints(ctx)
	if err != nil {
		return err
	}

	var ep *domain.Entrypoint
	for _, entrypoint := range eps {
		if entrypoint.Id == e.ID {
			ep = entrypoint
			break
		}
	}

	if ep == nil {
		return fmt.Errorf("entrypoint not found: %q", e.ID)
	}

	server := newServer(a.api, ep, a.transport)

	a.mu.Lock()
	a.servers[ep.Id] = server
	a.mu.Unlock()

	a.startServer(server)

	return nil
}

func NewHttpAdapter(ctx context.Context, api ports.Application, transport ports.ProxyTransportPort) (*Adapter, error) {
	entrypoints, err := api.EntryPoints(ctx)
	if err != nil {
		return nil, err
	}

	servers := make(map[string]*http.Server)

	for _, ep := range entrypoints {
		servers[ep.Id] = newServer(api, ep, transport)
	}

	return &Adapter{
		api:       api,
		servers:   servers,
		transport: transport,
	}, nil
}

func newServer(api ports.Application, ep *domain.Entrypoint, transport ports.ProxyTransportPort) *http.Server {
	handler := NewHandler(api, transport)

	mw := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), entrypointContextKey, contextKey(ep.Id))
		handler.ServeHTTP(w, r.WithContext(ctx))

	})

	return &http.Server{
		Addr:         ep.ListenAddr(),
		Handler:      mw,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// startServer runs the server's accept loop in its own goroutine. Servers
// come and go at runtime, so failures are logged rather than propagated:
// one entrypoint failing to bind must not take the whole adapter down.
func (a *Adapter) startServer(server *http.Server) {
	go func() {
		zap.L().Info("starting http server", zap.String("addr", server.Addr))
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("http server failed", zap.String("addr", server.Addr), zap.Error(err))
		}
	}()
}

// Run starts every configured server and blocks until ctx is canceled. The
// adapter's lifetime is bound to ctx, not to the number of live servers:
// entrypoints are added and removed at runtime via HandleEvent.
func (a *Adapter) Run(ctx context.Context) error {
	a.mu.Lock()
	for _, server := range a.servers {
		a.startServer(server)
	}
	a.mu.Unlock()

	<-ctx.Done()
	return nil
}

// Shutdown shutdowns the http servers
func (a *Adapter) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	servers := make([]*http.Server, 0, len(a.servers))
	for _, server := range a.servers {
		servers = append(servers, server)
	}
	a.mu.Unlock()

	g, ctx := errgroup.WithContext(ctx)

	for _, server := range servers {
		g.Go(func() error {
			return server.Shutdown(ctx)
		})
	}

	return g.Wait()
}
