package dashboard

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"time"

	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/aknEvrnky/pgway/internal/ports"
	"go.uber.org/zap"
)

type Adapter struct {
	cp            ports.ControlPlane
	authenticator ports.TokenAuthenticator
	authManager   ports.AuthManager
	users         ports.UserManager
	cfg           config.DashboardConfig
	server        *http.Server
	// staticFS overrides the build-tagged UI embed (tests only).
	staticFS fs.FS
}

func NewAdapter(cp ports.ControlPlane, authenticator ports.TokenAuthenticator, authManager ports.AuthManager, users ports.UserManager, cfg config.DashboardConfig) *Adapter {
	a := &Adapter{
		cp:            cp,
		authenticator: authenticator,
		authManager:   authManager,
		users:         users,
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

// SetStaticFS installs a UI filesystem (tests) and remounts routes so the SPA
// catch-all is registered. Production uses the build-tagged ui embed instead.
func (a *Adapter) SetStaticFS(fsys fs.FS) {
	a.staticFS = fsys
	a.server.Handler = a.routes()
}

func (a *Adapter) Run(ctx context.Context) error {
	zap.L().Info("starting dashboard server", zap.String("addr", a.server.Addr))
	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *Adapter) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
