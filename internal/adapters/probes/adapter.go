package probes

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Adapter is a dedicated HTTP listener for process probes.
type Adapter struct {
	gate   *ReadyGate
	server *http.Server
}

// New builds a probe adapter bound to addr (e.g. ":8082").
func New(addr string, gate *ReadyGate) *Adapter {
	a := &Adapter{gate: gate}
	a.server = &http.Server{
		Addr:         addr,
		Handler:      a.handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return a
}

func (a *Adapter) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", a.liveness)
	mux.HandleFunc("/readyz", a.readiness)
	return mux
}

func (a *Adapter) liveness(w http.ResponseWriter, _ *http.Request) {
	writePlain(w, http.StatusOK, ReasonOK)
}

func (a *Adapter) readiness(w http.ResponseWriter, r *http.Request) {
	ok, reason := a.gate.Check(r.Context())
	if ok {
		writePlain(w, http.StatusOK, reason)
		return
	}
	writePlain(w, http.StatusServiceUnavailable, reason)
}

func writePlain(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// Run blocks until the probe server stops.
func (a *Adapter) Run() error {
	zap.L().Info("starting probes server", zap.String("addr", a.server.Addr))
	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully stops the probe server.
func (a *Adapter) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
