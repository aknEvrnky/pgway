package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

type Adapter struct {
	cp            ports.ControlPlane
	authenticator ports.TokenAuthenticator
	cfg           config.RestConfig
	server        *http.Server
}

func NewRestAdapter(cp ports.ControlPlane, authenticator ports.TokenAuthenticator, cfg config.RestConfig) *Adapter {
	a := &Adapter{
		cp:            cp,
		authenticator: authenticator,
		cfg:           cfg,
	}

	a.server = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      a.routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return a
}

// Handler exposes the HTTP handler for tests.
func (a *Adapter) Handler() http.Handler {
	return a.server.Handler
}

func (a *Adapter) Run(ctx context.Context) error {
	zap.L().Info("starting rest server", zap.String("addr", a.server.Addr))
	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *Adapter) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
