package api

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (a *Application) LoadEntryPoints(ctx context.Context) ([]*domain.Entrypoint, error) {
	return a.entryPointRepo.GetAll(ctx)
}

func (a *Application) GetEntryPoint(ctx context.Context, id string) (*domain.Entrypoint, error) {
	return a.entryPointRepo.Find(ctx, id)
}
