package api

import (
	"context"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func (a *Application) LoadPools(ctx context.Context) ([]*domain.Pool, error) {
	return a.poolRepo.GetAll(ctx)
}

func (a *Application) GetPool(ctx context.Context, id string) (*domain.Pool, error) {
	return a.poolRepo.Find(ctx, id)
}
