package http

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type contextKey string

const entrypointContextKey contextKey = "entry_point_id"

// serverShutdownTimeout bounds how long ReconcileListeners waits for an old
// server to drain before replacing it. Without a bound a slow connection
// would block the event consumer / resync path indefinitely.
const serverShutdownTimeout = 10 * time.Second

type Adapter struct {
	api          ports.Application
	transport    ports.ProxyTransportPort
	link         ports.CPLinkStatus // optional; nil disables fail_closed gate
	maxBodyBytes int64
	servers      map[string]*http.Server
	mu           sync.Mutex
}

var _ ports.ListenerReconciler = (*Adapter)(nil)

// HandleEvent reacts to entrypoint change hints by reconciling all listeners
// against the application cache (same path as Watch/periodic Resync). It must
// run after the cache-refreshing handler in the consumer chain.
func (a *Adapter) HandleEvent(ctx context.Context, e ports.ChangeEvent) error {
	if e.ResourceType != ports.ResourceTypeEntrypoint {
		return nil
	}
	return a.ReconcileListeners(ctx)
}

// ReconcileListeners brings running entrypoint servers in line with the
// application cache. Bind-address changes start the new listener before
// draining the old one (ports differ, so there is no bind conflict). Removals
// and replacements shut down in parallel.
func (a *Adapter) ReconcileListeners(ctx context.Context) error {
	eps, err := a.api.EntryPoints(ctx)
	if err != nil {
		return err
	}

	desired := make(map[string]*domain.Entrypoint, len(eps))
	for _, ep := range eps {
		desired[ep.Id] = ep
	}

	a.mu.Lock()
	var toStop []*http.Server
	var toStart []*http.Server

	// Removals and address changes → schedule stop of the old server.
	for id, srv := range a.servers {
		ep, ok := desired[id]
		if !ok || srv.Addr != ep.ListenAddr() {
			toStop = append(toStop, srv)
			delete(a.servers, id)
		}
	}

	// Additions and replacements → start new servers while old ones still drain.
	for id, ep := range desired {
		if _, ok := a.servers[id]; ok {
			continue
		}
		srv := newServer(a.api, ep, a.transport, a.link, a.maxBodyBytes)
		a.servers[id] = srv
		toStart = append(toStart, srv)
	}
	a.mu.Unlock()

	for _, srv := range toStart {
		a.startServer(srv)
	}

	if len(toStop) == 0 {
		return nil
	}

	g, gctx := errgroup.WithContext(ctx)
	for _, srv := range toStop {
		g.Go(func() error {
			shutdownCtx, cancel := context.WithTimeout(gctx, serverShutdownTimeout)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				zap.L().Warn("reconcile shutdown", zap.Error(err), zap.String("addr", srv.Addr))
			}
			return nil // best-effort; never fail the whole reconcile
		})
	}
	_ = g.Wait()
	return nil
}

func NewHttpAdapter(ctx context.Context, api ports.Application, transport ports.ProxyTransportPort, maxBodyBytes int64, link ports.CPLinkStatus) (*Adapter, error) {
	entrypoints, err := api.EntryPoints(ctx)
	if err != nil {
		return nil, err
	}

	servers := make(map[string]*http.Server)

	for _, ep := range entrypoints {
		servers[ep.Id] = newServer(api, ep, transport, link, maxBodyBytes)
	}

	return &Adapter{
		api:          api,
		servers:      servers,
		transport:    transport,
		link:         link,
		maxBodyBytes: maxBodyBytes,
	}, nil
}

func newServer(api ports.Application, ep *domain.Entrypoint, transport ports.ProxyTransportPort, link ports.CPLinkStatus, maxBodyBytes int64) *http.Server {
	handler := NewHandler(api, transport, maxBodyBytes, link)

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
