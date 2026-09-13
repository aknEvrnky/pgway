package badger_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aknEvrnky/pgway/integration/testutil"
	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

func TestFlowRepository(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Save and Find", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			flow := testutil.NewTestFlow()
			err := store.Flows.Save(context.Background(), flow)
			require.NoError(t, err)
			got, err := store.Flows.Find(context.Background(), flow.Id)
			require.NoError(t, err)
			assert.Equal(t, flow, got)
		}},
		{"List returns all", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			f1 := testutil.NewTestFlow()
			f2 := testutil.NewTestFlow()
			f2.Id = "flow-2"
			ctx := context.Background()
			require.NoError(t, store.Flows.Save(ctx, f1))
			require.NoError(t, store.Flows.Save(ctx, f2))
			result, err := store.Flows.List(ctx, domain.ListParams{}, domain.FlowFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, f := range result.Items {
				ids[f.Id] = true
			}
			assert.True(t, ids["flow-1"], "flow-1 should be in results")
			assert.True(t, ids["flow-2"], "flow-2 should be in results")
		}},
		{"Save overwrite", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			flow := testutil.NewTestFlow()
			require.NoError(t, store.Flows.Save(context.Background(), flow))
			flow.BalancerId = "lb-updated"
			require.NoError(t, store.Flows.Save(context.Background(), flow))
			got, err := store.Flows.Find(context.Background(), flow.Id)
			require.NoError(t, err)
			assert.Equal(t, "lb-updated", got.BalancerId)
		}},
		{"Delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			flow := testutil.NewTestFlow()
			ctx := context.Background()
			require.NoError(t, store.Flows.Save(ctx, flow))
			require.NoError(t, store.Flows.Delete(ctx, flow.Id))
			_, err := store.Flows.Find(ctx, flow.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Find after delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			flow := testutil.NewTestFlow()
			require.NoError(t, store.Flows.Save(context.Background(), flow))
			require.NoError(t, store.Flows.Delete(context.Background(), flow.Id))
			_, err := store.Flows.Find(context.Background(), flow.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Delete non-existent", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			err := store.Flows.Delete(context.Background(), "unknown-flow-id")
			assert.ErrorContains(t, err, "not found")
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestFlowFilter(t *testing.T) {
	t.Parallel()

	t.Run("filter by router_id", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		f1 := testutil.NewTestFlow()
		f1.Id = "flow-a"
		f1.RouterId = "router-1"

		f2 := testutil.NewTestFlow()
		f2.Id = "flow-b"
		f2.RouterId = "router-2"

		require.NoError(t, store.Flows.Save(ctx, f1))
		require.NoError(t, store.Flows.Save(ctx, f2))

		result, err := store.Flows.List(ctx, domain.ListParams{}, domain.FlowFilter{RouterId: "router-1"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "flow-a", result.Items[0].Id)
	})

	t.Run("filter by balancer_id", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		f1 := testutil.NewTestFlow()
		f1.Id = "flow-a"
		f1.BalancerId = "lb-1"

		f2 := testutil.NewTestFlow()
		f2.Id = "flow-b"
		f2.BalancerId = "lb-2"

		require.NoError(t, store.Flows.Save(ctx, f1))
		require.NoError(t, store.Flows.Save(ctx, f2))

		result, err := store.Flows.List(ctx, domain.ListParams{}, domain.FlowFilter{BalancerId: "lb-1"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "flow-a", result.Items[0].Id)
	})

	t.Run("search by id", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		f1 := testutil.NewTestFlow()
		f1.Id = "api-flow"

		f2 := testutil.NewTestFlow()
		f2.Id = "web-flow"

		require.NoError(t, store.Flows.Save(ctx, f1))
		require.NoError(t, store.Flows.Save(ctx, f2))

		result, err := store.Flows.List(ctx, domain.ListParams{}, domain.FlowFilter{Search: "api"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "api-flow", result.Items[0].Id)
	})

	t.Run("combined filter and pagination", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		for i := 1; i <= 3; i++ {
			f := testutil.NewTestFlow()
			f.Id = fmt.Sprintf("flow-r1-%d", i)
			f.RouterId = "router-1"
			require.NoError(t, store.Flows.Save(ctx, f))
		}
		for i := 1; i <= 2; i++ {
			f := testutil.NewTestFlow()
			f.Id = fmt.Sprintf("flow-r2-%d", i)
			f.RouterId = "router-2"
			require.NoError(t, store.Flows.Save(ctx, f))
		}

		result, err := store.Flows.List(ctx, domain.ListParams{PageSize: 2}, domain.FlowFilter{RouterId: "router-1"})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, 3, result.TotalCount)
		assert.NotEmpty(t, result.NextCursor)

		result2, err := store.Flows.List(ctx, domain.ListParams{PageSize: 2, Cursor: result.NextCursor}, domain.FlowFilter{RouterId: "router-1"})
		require.NoError(t, err)
		assert.Len(t, result2.Items, 1)
		assert.Equal(t, 3, result2.TotalCount)
		assert.Empty(t, result2.NextCursor)
	})
}
