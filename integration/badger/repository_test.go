package badger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/integration/testutil"
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
		{"GetAll", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			p1 := testutil.NewTestProxy()
			p2 := testutil.NewTestProxy()
			p2.Id = "p2"
			require.NoError(t, store.Proxies.Save(context.Background(), p1))
			require.NoError(t, store.Proxies.Save(context.Background(), p2))
			all, err := store.Proxies.GetAll(context.Background())
			require.NoError(t, err)
			assert.Len(t, all, 2)
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
			require.NoError(t, store.Proxies.Save(context.Background(), proxy))
			err := store.Proxies.Delete(context.Background(), proxy.Id)
			assert.NoError(t, err)
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
		tt := tt
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
		{"GetAll", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			p1 := testutil.NewTestPool()
			p2 := testutil.NewTestPool()
			p2.Id = "pool-2"
			require.NoError(t, store.Pools.Save(context.Background(), p1))
			require.NoError(t, store.Pools.Save(context.Background(), p2))
			all, err := store.Pools.GetAll(context.Background())
			require.NoError(t, err)
			assert.Len(t, all, 2)
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
			require.NoError(t, store.Pools.Save(context.Background(), pool))
			err := store.Pools.Delete(context.Background(), pool.Id)
			assert.NoError(t, err)
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
		tt := tt
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
		{"GetAll", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			lb1 := testutil.NewTestLB()
			lb2 := testutil.NewTestLB()
			lb2.Id = "lb-2"
			require.NoError(t, store.LBs.Save(context.Background(), lb1))
			require.NoError(t, store.LBs.Save(context.Background(), lb2))
			all, err := store.LBs.GetAll(context.Background())
			require.NoError(t, err)
			assert.Len(t, all, 2)
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
			require.NoError(t, store.LBs.Save(context.Background(), lb))
			err := store.LBs.Delete(context.Background(), lb.Id)
			assert.NoError(t, err)
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
		tt := tt
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
		{"GetAll", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			r1 := testutil.NewTestRouter()
			r2 := testutil.NewTestRouter()
			r2.Id = "router-2"
			require.NoError(t, store.Routers.Save(context.Background(), r1))
			require.NoError(t, store.Routers.Save(context.Background(), r2))
			all, err := store.Routers.GetAll(context.Background())
			require.NoError(t, err)
			assert.Len(t, all, 2)
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
			require.NoError(t, store.Routers.Save(context.Background(), router))
			err := store.Routers.Delete(context.Background(), router.Id)
			assert.NoError(t, err)
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
		tt := tt
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
		{"GetAll", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			f1 := testutil.NewTestFlow()
			f2 := testutil.NewTestFlow()
			f2.Id = "flow-2"
			require.NoError(t, store.Flows.Save(context.Background(), f1))
			require.NoError(t, store.Flows.Save(context.Background(), f2))
			all, err := store.Flows.GetAll(context.Background())
			require.NoError(t, err)
			assert.Len(t, all, 2)
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
			require.NoError(t, store.Flows.Save(context.Background(), flow))
			err := store.Flows.Delete(context.Background(), flow.Id)
			assert.NoError(t, err)
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
		tt := tt
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
		{"GetAll", func(t *testing.T) {
			store := badgerutil.NewBadgerStore(t)
			ep1 := testutil.NewTestEntrypoint()
			ep2 := testutil.NewTestEntrypoint()
			ep2.Id = "ep-2"
			require.NoError(t, store.EPs.Save(context.Background(), ep1))
			require.NoError(t, store.EPs.Save(context.Background(), ep2))
			all, err := store.EPs.GetAll(context.Background())
			require.NoError(t, err)
			assert.Len(t, all, 2)
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
			require.NoError(t, store.EPs.Save(context.Background(), ep))
			err := store.EPs.Delete(context.Background(), ep.Id)
			assert.NoError(t, err)
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
