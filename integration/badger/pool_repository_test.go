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

func TestPoolRepository(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Save and Find", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			pool := testutil.NewTestPool()
			err := store.Pools.Save(context.Background(), pool)
			require.NoError(t, err)
			got, err := store.Pools.Find(context.Background(), pool.Id)
			require.NoError(t, err)
			assert.Equal(t, pool, got)
		}},
		{"List returns all", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			p1 := testutil.NewTestPool()
			p2 := testutil.NewTestPool()
			p2.Id = "pool-2"
			ctx := context.Background()
			require.NoError(t, store.Pools.Save(ctx, p1))
			require.NoError(t, store.Pools.Save(ctx, p2))
			result, err := store.Pools.List(ctx, domain.ListParams{}, domain.PoolFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, p := range result.Items {
				ids[p.Id] = true
			}
			assert.True(t, ids["pool-1"], "pool-1 should be in results")
			assert.True(t, ids["pool-2"], "pool-2 should be in results")
		}},
		{"Save overwrite", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			pool := testutil.NewTestPool()
			require.NoError(t, store.Pools.Save(context.Background(), pool))
			pool.Title = "updated-pool"
			require.NoError(t, store.Pools.Save(context.Background(), pool))
			got, err := store.Pools.Find(context.Background(), pool.Id)
			require.NoError(t, err)
			assert.Equal(t, "updated-pool", got.Title)
		}},
		{"Delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			pool := testutil.NewTestPool()
			ctx := context.Background()
			require.NoError(t, store.Pools.Save(ctx, pool))
			require.NoError(t, store.Pools.Delete(ctx, pool.Id))
			_, err := store.Pools.Find(ctx, pool.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Find after delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			pool := testutil.NewTestPool()
			require.NoError(t, store.Pools.Save(context.Background(), pool))
			require.NoError(t, store.Pools.Delete(context.Background(), pool.Id))
			_, err := store.Pools.Find(context.Background(), pool.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Delete non-existent", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			err := store.Pools.Delete(context.Background(), "unknown-pool-id")
			assert.ErrorContains(t, err, "not found")
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestPoolFilter(t *testing.T) {
	t.Parallel()

	t.Run("filter by type", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		pool1 := testutil.NewTestPool()
		pool1.Id = "pool-static"
		pool1.Type = domain.PoolTypeStatic

		pool2 := testutil.NewTestPool()
		pool2.Id = "pool-dynamic"
		pool2.Type = domain.PoolTypeDynamic

		require.NoError(t, store.Pools.Save(ctx, pool1))
		require.NoError(t, store.Pools.Save(ctx, pool2))

		result, err := store.Pools.List(ctx, domain.ListParams{}, domain.PoolFilter{Type: "static"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "pool-static", result.Items[0].Id)
	})

	t.Run("search by title", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		pool1 := testutil.NewTestPool()
		pool1.Id = "pool-alpha"
		pool1.Title = "alpha-pool"

		pool2 := testutil.NewTestPool()
		pool2.Id = "pool-beta"
		pool2.Title = "beta-pool"

		require.NoError(t, store.Pools.Save(ctx, pool1))
		require.NoError(t, store.Pools.Save(ctx, pool2))

		result, err := store.Pools.List(ctx, domain.ListParams{}, domain.PoolFilter{Search: "alpha"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "pool-alpha", result.Items[0].Id)
	})
}
