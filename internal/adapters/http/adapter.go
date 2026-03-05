package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	EntryPointIDKey = "entry_point_id"
)

type Adapter struct {
	api     ports.Application
	servers map[string]*http.Server
}

func NewHttpAdapter(ctx context.Context, api ports.Application) (*Adapter, error) {
	entrypoints, err := api.LoadEntryPoints(ctx)
	if err != nil {
		return nil, err
	}

	servers := make(map[string]*http.Server)

	for _, ep := range entrypoints {
		servers[ep.Id] = createServer(ep)
	}

	return &Adapter{
		api:     api,
		servers: servers,
	}, nil
}

func createServer(ep *domain.Entrypoint) *http.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), EntryPointIDKey, ep.Id)
		handleRequest(w, r.WithContext(ctx))
	})

	return &http.Server{
		Addr:         ep.ListenAddr(),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func (a *Adapter) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, server := range a.servers {
		g.Go(func() error {
			zap.L().Info("starting http server", zap.String("addr", server.Addr))
			err := server.ListenAndServe()
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		})
	}

	return g.Wait()
}

// Shutdown shutdowns the http servers
func (a *Adapter) Shutdown(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, server := range a.servers {
		g.Go(func() error {
			return server.Shutdown(ctx)
		})
	}

	return g.Wait()
}
