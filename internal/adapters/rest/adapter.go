package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

type Adapter struct {
	cp     ports.ControlPlane
	server *http.Server
}

func NewRestAdapter(cp ports.ControlPlane, addr string) *Adapter {
	a := &Adapter{cp: cp}

	a.server = &http.Server{
		Addr:         addr,
		Handler:      a.routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return a
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
