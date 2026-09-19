package agenthost

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resyncApp struct {
	bootstraps int
	bootErr    error
}

func (a *resyncApp) Bootstrap(context.Context) error {
	a.bootstraps++
	return a.bootErr
}
func (a *resyncApp) EntryPoints(context.Context) ([]*domain.Entrypoint, error) { return nil, nil }
func (a *resyncApp) ExecuteFlow(context.Context, string, *http.Request) (*domain.Proxy, string, error) {
	return nil, "", errors.New("unused")
}
func (a *resyncApp) Release(context.Context, string, domain.BalancerResult) error { return nil }

type resyncListeners struct {
	calls int
	err   error
}

func (r *resyncListeners) ReconcileListeners(context.Context) error {
	r.calls++
	return r.err
}

func TestResync_BootstrapsThenReconciles(t *testing.T) {
	app := &resyncApp{}
	lis := &resyncListeners{}
	require.NoError(t, Resync(context.Background(), app, lis))
	assert.Equal(t, 1, app.bootstraps)
	assert.Equal(t, 1, lis.calls)
}

func TestResync_BootstrapErrorSkipsReconcile(t *testing.T) {
	app := &resyncApp{bootErr: errors.New("cp down")}
	lis := &resyncListeners{}
	err := Resync(context.Background(), app, lis)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bootstrap")
	assert.Equal(t, 0, lis.calls)
}
