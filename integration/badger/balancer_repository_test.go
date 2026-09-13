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

func TestBalancerRepository(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Save and Find", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			lb := testutil.NewTestLB()
			err := store.LBs.Save(context.Background(), lb)
			require.NoError(t, err)
			got, err := store.LBs.Find(context.Background(), lb.Id)
			require.NoError(t, err)
			assert.Equal(t, lb, got)
		}},
		{"List returns all", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			lb1 := testutil.NewTestLB()
			lb2 := testutil.NewTestLB()
			lb2.Id = "lb-2"
			ctx := context.Background()
			require.NoError(t, store.LBs.Save(ctx, lb1))
			require.NoError(t, store.LBs.Save(ctx, lb2))
			result, err := store.LBs.List(ctx, domain.ListParams{}, domain.BalancerFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, lb := range result.Items {
				ids[lb.Id] = true
			}
			assert.True(t, ids["lb-1"], "lb-1 should be in results")
			assert.True(t, ids["lb-2"], "lb-2 should be in results")
		}},
		{"Save overwrite", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			lb := testutil.NewTestLB()
			require.NoError(t, store.LBs.Save(context.Background(), lb))
			lb.Title = "updated-lb"
			require.NoError(t, store.LBs.Save(context.Background(), lb))
			got, err := store.LBs.Find(context.Background(), lb.Id)
			require.NoError(t, err)
			assert.Equal(t, "updated-lb", got.Title)
		}},
		{"Delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			lb := testutil.NewTestLB()
			ctx := context.Background()
			require.NoError(t, store.LBs.Save(ctx, lb))
			require.NoError(t, store.LBs.Delete(ctx, lb.Id))
			_, err := store.LBs.Find(ctx, lb.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Find after delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			lb := testutil.NewTestLB()
			require.NoError(t, store.LBs.Save(context.Background(), lb))
			require.NoError(t, store.LBs.Delete(context.Background(), lb.Id))
			_, err := store.LBs.Find(context.Background(), lb.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Delete non-existent", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			err := store.LBs.Delete(context.Background(), "unknown-lb-id")
			assert.ErrorContains(t, err, "not found")
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestBalancerFilter(t *testing.T) {
	t.Parallel()

	t.Run("filter by pool_id", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		lb1 := testutil.NewTestLB()
		lb1.Id = "lb-a"
		lb1.PoolId = "pool-1"

		lb2 := testutil.NewTestLB()
		lb2.Id = "lb-b"
		lb2.PoolId = "pool-2"

		require.NoError(t, store.LBs.Save(ctx, lb1))
		require.NoError(t, store.LBs.Save(ctx, lb2))

		result, err := store.LBs.List(ctx, domain.ListParams{}, domain.BalancerFilter{PoolId: "pool-1"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "lb-a", result.Items[0].Id)
	})

	t.Run("filter by type", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		lb1 := testutil.NewTestLB()
		lb1.Id = "lb-rr"
		lb1.Type = domain.BalancerTypeRoundRobin

		lb2 := testutil.NewTestLB()
		lb2.Id = "lb-w"
		lb2.Type = domain.BalancerTypeWeighted

		require.NoError(t, store.LBs.Save(ctx, lb1))
		require.NoError(t, store.LBs.Save(ctx, lb2))

		result, err := store.LBs.List(ctx, domain.ListParams{}, domain.BalancerFilter{Type: "round-robin"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "lb-rr", result.Items[0].Id)
	})
}
