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
