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

func TestEntrypointRepository(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Save and Find", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			ep := testutil.NewTestEntrypoint()
			err := store.EPs.Save(context.Background(), ep)
			require.NoError(t, err)
			got, err := store.EPs.Find(context.Background(), ep.Id)
			require.NoError(t, err)
			assert.Equal(t, ep, got)
		}},
		{"List returns all", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			ep1 := testutil.NewTestEntrypoint()
			ep2 := testutil.NewTestEntrypoint()
			ep2.Id = "ep-2"
			ctx := context.Background()
			require.NoError(t, store.EPs.Save(ctx, ep1))
			require.NoError(t, store.EPs.Save(ctx, ep2))
			result, err := store.EPs.List(ctx, domain.ListParams{}, domain.EntrypointFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, ep := range result.Items {
				ids[ep.Id] = true
			}
			assert.True(t, ids["ep-1"], "ep-1 should be in results")
			assert.True(t, ids["ep-2"], "ep-2 should be in results")
		}},
		{"Save overwrite", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			ep := testutil.NewTestEntrypoint()
			require.NoError(t, store.EPs.Save(context.Background(), ep))
			ep.Title = "updated-entrypoint"
			require.NoError(t, store.EPs.Save(context.Background(), ep))
			got, err := store.EPs.Find(context.Background(), ep.Id)
			require.NoError(t, err)
			assert.Equal(t, "updated-entrypoint", got.Title)
		}},
		{"Delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			ep := testutil.NewTestEntrypoint()
			ctx := context.Background()
			require.NoError(t, store.EPs.Save(ctx, ep))
			require.NoError(t, store.EPs.Delete(ctx, ep.Id))
			_, err := store.EPs.Find(ctx, ep.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Find after delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			ep := testutil.NewTestEntrypoint()
			require.NoError(t, store.EPs.Save(context.Background(), ep))
			require.NoError(t, store.EPs.Delete(context.Background(), ep.Id))
			_, err := store.EPs.Find(context.Background(), ep.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Delete non-existent", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			err := store.EPs.Delete(context.Background(), "unknown-ep-id")
			assert.ErrorContains(t, err, "not found")
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestEntrypointFilter(t *testing.T) {
	t.Parallel()

	t.Run("filter by protocol", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		ep1 := testutil.NewTestEntrypoint()
		ep1.Id = "ep-http"
		ep1.Protocol = domain.ProtocolHTTP

		ep2 := testutil.NewTestEntrypoint()
		ep2.Id = "ep-socks"
		ep2.Protocol = "socks5"

		require.NoError(t, store.EPs.Save(ctx, ep1))
		require.NoError(t, store.EPs.Save(ctx, ep2))

		result, err := store.EPs.List(ctx, domain.ListParams{}, domain.EntrypointFilter{Protocol: "http"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "ep-http", result.Items[0].Id)
	})

	t.Run("filter by host substring", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		ep1 := testutil.NewTestEntrypoint()
		ep1.Id = "ep-local"
		ep1.Host = "localhost"

		ep2 := testutil.NewTestEntrypoint()
		ep2.Id = "ep-remote"
		ep2.Host = "10.0.0.1"

		require.NoError(t, store.EPs.Save(ctx, ep1))
		require.NoError(t, store.EPs.Save(ctx, ep2))

		result, err := store.EPs.List(ctx, domain.ListParams{}, domain.EntrypointFilter{Host: "local"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "ep-local", result.Items[0].Id)
	})

	t.Run("search by title", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		ep1 := testutil.NewTestEntrypoint()
		ep1.Id = "ep1"
		ep1.Title = "Production Gateway"

		ep2 := testutil.NewTestEntrypoint()
		ep2.Id = "ep2"
		ep2.Title = "Staging Gateway"

		require.NoError(t, store.EPs.Save(ctx, ep1))
		require.NoError(t, store.EPs.Save(ctx, ep2))

		result, err := store.EPs.List(ctx, domain.ListParams{}, domain.EntrypointFilter{Search: "production"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "ep1", result.Items[0].Id)
	})

	t.Run("search is case insensitive", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		ep1 := testutil.NewTestEntrypoint()
		ep1.Id = "ep1"
		ep1.Title = "MyGateway"

		require.NoError(t, store.EPs.Save(ctx, ep1))

		result, err := store.EPs.List(ctx, domain.ListParams{}, domain.EntrypointFilter{Search: "MYGATEWAY"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
	})

	t.Run("combined protocol and host filter", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		ep1 := testutil.NewTestEntrypoint()
		ep1.Id = "ep1"
		ep1.Protocol = domain.ProtocolHTTP
		ep1.Host = "localhost"

		ep2 := testutil.NewTestEntrypoint()
		ep2.Id = "ep2"
		ep2.Protocol = domain.ProtocolHTTP
		ep2.Host = "10.0.0.1"

		ep3 := testutil.NewTestEntrypoint()
		ep3.Id = "ep3"
		ep3.Protocol = "socks5"
		ep3.Host = "localhost"

		require.NoError(t, store.EPs.Save(ctx, ep1))
		require.NoError(t, store.EPs.Save(ctx, ep2))
		require.NoError(t, store.EPs.Save(ctx, ep3))

		result, err := store.EPs.List(ctx, domain.ListParams{}, domain.EntrypointFilter{Protocol: "http", Host: "local"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "ep1", result.Items[0].Id)
	})
}
