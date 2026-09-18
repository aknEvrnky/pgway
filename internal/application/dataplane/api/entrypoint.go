package api

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (a *Application) EntryPoints(_ context.Context) ([]*domain.Entrypoint, error) {
	return a.cache.AllEntrypoints(), nil
}

func (a *Application) getEntryPoint(_ context.Context, id string) (*domain.Entrypoint, error) {
	return a.cache.GetEntrypoint(id)
}
