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

func TestProxyRepository(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Save and Find", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			proxy := testutil.NewTestProxy()
			err := store.Proxies.Save(context.Background(), proxy)
			require.NoError(t, err)
			got, err := store.Proxies.Find(context.Background(), proxy.Id)
			require.NoError(t, err)
			assert.Equal(t, proxy, got)
		}},
		{"List returns all", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			p1 := testutil.NewTestProxy()
			p2 := testutil.NewTestProxy()
			p2.Id = "p2"
			ctx := context.Background()
			require.NoError(t, store.Proxies.Save(ctx, p1))
			require.NoError(t, store.Proxies.Save(ctx, p2))
			result, err := store.Proxies.List(ctx, domain.ListParams{}, domain.ProxyFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, p := range result.Items {
				ids[p.Id] = true
			}
			assert.True(t, ids["p1"], "p1 should be in results")
			assert.True(t, ids["p2"], "p2 should be in results")
		}},
		{"Save overwrite", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			proxy := testutil.NewTestProxy()
			require.NoError(t, store.Proxies.Save(context.Background(), proxy))
			proxy.Host = "10.0.0.1"
			require.NoError(t, store.Proxies.Save(context.Background(), proxy))
			got, err := store.Proxies.Find(context.Background(), proxy.Id)
			require.NoError(t, err)
			assert.Equal(t, "10.0.0.1", got.Host)
		}},
		{"Delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			proxy := testutil.NewTestProxy()
			ctx := context.Background()
			require.NoError(t, store.Proxies.Save(ctx, proxy))
			require.NoError(t, store.Proxies.Delete(ctx, proxy.Id))
			_, err := store.Proxies.Find(ctx, proxy.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Find after delete", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			proxy := testutil.NewTestProxy()
			require.NoError(t, store.Proxies.Save(context.Background(), proxy))
			require.NoError(t, store.Proxies.Delete(context.Background(), proxy.Id))
			_, err := store.Proxies.Find(context.Background(), proxy.Id)
			assert.ErrorContains(t, err, "not found")
		}},
		{"Delete non-existent", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			err := store.Proxies.Delete(context.Background(), "unknown-proxy-id")
			assert.ErrorContains(t, err, "not found")
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestProxyListPagination(t *testing.T) {
	t.Parallel()

	store := badgerutil.NewBadgerStore(t)
	ctx := context.Background()

	// Insert 5 proxies
	for i := 1; i <= 5; i++ {
		p := testutil.NewTestProxy()
		p.Id = fmt.Sprintf("proxy-%d", i)
		require.NoError(t, store.Proxies.Save(ctx, p))
	}

	t.Run("no params returns all", func(t *testing.T) {
		result, err := store.Proxies.List(ctx, domain.ListParams{}, domain.ProxyFilter{})
		require.NoError(t, err)
		assert.Len(t, result.Items, 5)
		assert.Equal(t, 5, result.TotalCount)
		assert.Empty(t, result.NextCursor)
	})

	t.Run("page_size=2 returns first page", func(t *testing.T) {
		result, err := store.Proxies.List(ctx, domain.ListParams{PageSize: 2}, domain.ProxyFilter{})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, 5, result.TotalCount)
		assert.NotEmpty(t, result.NextCursor)
	})

	t.Run("paginate through all items", func(t *testing.T) {
		var allItems []*domain.Proxy
		cursor := ""

		for {
			result, err := store.Proxies.List(ctx, domain.ListParams{PageSize: 2, Cursor: cursor}, domain.ProxyFilter{})
			require.NoError(t, err)
			assert.Equal(t, 5, result.TotalCount)
			allItems = append(allItems, result.Items...)

			if result.NextCursor == "" {
				break
			}
			cursor = result.NextCursor
		}

		assert.Len(t, allItems, 5)

		// Verify no duplicates
		seen := map[string]bool{}
		for _, p := range allItems {
			assert.False(t, seen[p.Id], "duplicate item: %s", p.Id)
			seen[p.Id] = true
		}
	})

	t.Run("empty database", func(t *testing.T) {
		emptyStore := badgerutil.NewBadgerStore(t)
		result, err := emptyStore.Proxies.List(ctx, domain.ListParams{PageSize: 10}, domain.ProxyFilter{})
		require.NoError(t, err)
		assert.Len(t, result.Items, 0)
		assert.Equal(t, 0, result.TotalCount)
		assert.Empty(t, result.NextCursor)
	})

	t.Run("page_size larger than total", func(t *testing.T) {
		result, err := store.Proxies.List(ctx, domain.ListParams{PageSize: 100}, domain.ProxyFilter{})
		require.NoError(t, err)
		assert.Len(t, result.Items, 5)
		assert.Equal(t, 5, result.TotalCount)
		assert.Empty(t, result.NextCursor)
	})
}

func TestProxyFilterAndSearch(t *testing.T) {
	t.Parallel()

	t.Run("filter by protocol", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		p1 := testutil.NewTestProxy()
		p1.Id = "p-http"
		p1.Protocol = "http"

		p2 := testutil.NewTestProxy()
		p2.Id = "p-socks5"
		p2.Protocol = "socks5"

		require.NoError(t, store.Proxies.Save(ctx, p1))
		require.NoError(t, store.Proxies.Save(ctx, p2))

		result, err := store.Proxies.List(ctx, domain.ListParams{}, domain.ProxyFilter{Protocol: "http"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "p-http", result.Items[0].Id)
	})

	t.Run("filter by labels", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		p1 := testutil.NewTestProxy()
		p1.Id = "p-labeled"
		p1.Labels = map[string]string{"env": "prod"}

		p2 := testutil.NewTestProxy()
		p2.Id = "p-other"
		p2.Labels = map[string]string{"env": "staging"}

		require.NoError(t, store.Proxies.Save(ctx, p1))
		require.NoError(t, store.Proxies.Save(ctx, p2))

		result, err := store.Proxies.List(ctx, domain.ListParams{}, domain.ProxyFilter{Labels: map[string]string{"env": "prod"}})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "p-labeled", result.Items[0].Id)
	})

	t.Run("search by host", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		p1 := testutil.NewTestProxy()
		p1.Id = "p-example"
		p1.Host = "example.com"

		p2 := testutil.NewTestProxy()
		p2.Id = "p-other"
		p2.Host = "other.com"

		require.NoError(t, store.Proxies.Save(ctx, p1))
		require.NoError(t, store.Proxies.Save(ctx, p2))

		result, err := store.Proxies.List(ctx, domain.ListParams{}, domain.ProxyFilter{Search: "example"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "p-example", result.Items[0].Id)
	})

	t.Run("search is case insensitive", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		p1 := testutil.NewTestProxy()
		p1.Id = "p-example"
		p1.Host = "example.com"

		p2 := testutil.NewTestProxy()
		p2.Id = "p-other"
		p2.Host = "other.com"

		require.NoError(t, store.Proxies.Save(ctx, p1))
		require.NoError(t, store.Proxies.Save(ctx, p2))

		result, err := store.Proxies.List(ctx, domain.ListParams{}, domain.ProxyFilter{Search: "EXAMPLE"})
		require.NoError(t, err)
		require.Len(t, result.Items, 1)
		assert.Equal(t, "p-example", result.Items[0].Id)
	})

	t.Run("combined filter and pagination", func(t *testing.T) {
		store := badgerutil.NewBadgerStore(t)
		ctx := context.Background()

		// Save 3 http proxies and 2 socks5 proxies
		for i := 1; i <= 3; i++ {
			p := testutil.NewTestProxy()
			p.Id = fmt.Sprintf("http-%d", i)
			p.Protocol = "http"
			require.NoError(t, store.Proxies.Save(ctx, p))
		}
		for i := 1; i <= 2; i++ {
			p := testutil.NewTestProxy()
			p.Id = fmt.Sprintf("socks5-%d", i)
			p.Protocol = "socks5"
			require.NoError(t, store.Proxies.Save(ctx, p))
		}

		// First page: filter http with page_size=2
		result, err := store.Proxies.List(ctx, domain.ListParams{PageSize: 2}, domain.ProxyFilter{Protocol: "http"})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, 3, result.TotalCount)
		assert.NotEmpty(t, result.NextCursor)

		// All items should be http
		for _, p := range result.Items {
			assert.Equal(t, domain.Protocol("http"), p.Protocol)
		}

		// Second page
		result2, err := store.Proxies.List(ctx, domain.ListParams{PageSize: 2, Cursor: result.NextCursor}, domain.ProxyFilter{Protocol: "http"})
		require.NoError(t, err)
		assert.Len(t, result2.Items, 1)
		assert.Equal(t, 3, result2.TotalCount)
		assert.Empty(t, result2.NextCursor)
		assert.Equal(t, domain.Protocol("http"), result2.Items[0].Protocol)
	})
}
