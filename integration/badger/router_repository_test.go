package badger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aknEvrnky/pgway/integration/testutil"
	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func TestRouterRepository(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Save and Find", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			router := testutil.NewTestRouter()
			err := store.Routers.Save(context.Background(), router)
			require.NoError(t, err)
			got, err := store.Routers.Find(context.Background(), router.Id)
			require.NoError(t, err)
			assert.Equal(t, router, got)
		}},
		{"List returns all", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			r1 := testutil.NewTestRouter()
			r2 := testutil.NewTestRouter()
			r2.Id = "router-2"
			ctx := context.Background()
			require.NoError(t, store.Routers.Save(ctx, r1))
			require.NoError(t, store.Routers.Save(ctx, r2))
			result, err := store.Routers.List(ctx, domain.ListParams{}, domain.RouterFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, r := range result.Items {
				ids[r.Id] = true
			}
			assert.True(t, ids["router-1"], "router-1 should be in results")
			assert.True(t, ids["router-2"], "router-2 should be in results")
		}},
		{"Save overwrite", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			router := testutil.NewTestRouter()
			require.NoError(t, store.Routers.Save(context.Background(), router))
			router.Title = "updated-router"
			require.NoError(t, store.Routers.Save(context.Background(), router))
			got, err := store.Routers.Find(context.Background(), router.Id)
			require.NoError(t, err)
			assert.Equal(t, "updated-router", got.Title)
		}},
		{"Delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			router := testutil.NewTestRouter()
			ctx := context.Background()
			require.NoError(t, store.Routers.Save(ctx, router))
			require.NoError(t, store.Routers.Delete(ctx, router.Id))
			_, err := store.Routers.Find(ctx, router.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Find after delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			router := testutil.NewTestRouter()
			require.NoError(t, store.Routers.Save(context.Background(), router))
			require.NoError(t, store.Routers.Delete(context.Background(), router.Id))
			_, err := store.Routers.Find(context.Background(), router.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Delete non-existent", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			err := store.Routers.Delete(context.Background(), "unknown-router-id")
			assert.ErrorContains(t, err, "not found")
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestRouterFilter(t *testing.T) {
	t.Parallel()

	t.Run("search by id", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		r1 := testutil.NewTestRouter()
		r1.Id = "api-router"
		r1.Title = "API Router"

		r2 := testutil.NewTestRouter()
		r2.Id = "web-router"
		r2.Title = "Web Router"

		require.NoError(t, store.Routers.Save(ctx, r1))
		require.NoError(t, store.Routers.Save(ctx, r2))

		result, err := store.Routers.List(ctx, domain.ListParams{}, domain.RouterFilter{Search: "api"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "api-router", result.Items[0].Id)
	})

	t.Run("search by title", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		r1 := testutil.NewTestRouter()
		r1.Id = "r1"
		r1.Title = "Production Router"

		r2 := testutil.NewTestRouter()
		r2.Id = "r2"
		r2.Title = "Staging Router"

		require.NoError(t, store.Routers.Save(ctx, r1))
		require.NoError(t, store.Routers.Save(ctx, r2))

		result, err := store.Routers.List(ctx, domain.ListParams{}, domain.RouterFilter{Search: "production"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "r1", result.Items[0].Id)
	})

	t.Run("search is case insensitive", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		r1 := testutil.NewTestRouter()
		r1.Id = "r1"
		r1.Title = "MyRouter"

		require.NoError(t, store.Routers.Save(ctx, r1))

		result, err := store.Routers.List(ctx, domain.ListParams{}, domain.RouterFilter{Search: "MYROUTER"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
	})

	t.Run("empty filter returns all", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		r1 := testutil.NewTestRouter()
		r1.Id = "r1"
		r2 := testutil.NewTestRouter()
		r2.Id = "r2"

		require.NoError(t, store.Routers.Save(ctx, r1))
		require.NoError(t, store.Routers.Save(ctx, r2))

		result, err := store.Routers.List(ctx, domain.ListParams{}, domain.RouterFilter{})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, 2, result.TotalCount)
	})
}
