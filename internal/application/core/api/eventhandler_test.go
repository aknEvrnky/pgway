package api

import (
	"context"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplication_HandleEvent_WarmupCache(t *testing.T) {
	cp := &mockControlPlane{entrypoints: []*domain.Entrypoint{}}

	app := newApp(cp)
	require.NoError(t, app.warmupCache(context.Background()))
	eps, err := app.EntryPoints(context.Background())
	require.NoError(t, err)
	assert.Len(t, eps, 0)

	cp.entrypoints = append(cp.entrypoints, testEP)

	// create a mock event
	mockEvent := event.ChangeEvent{
		ID:           testEP.Id,
		ResourceType: event.ResourceTypeEntrypoint,
		ChangeKind:   event.ChangeKindSaved,
	}

	err = app.HandleEvent(context.Background(), mockEvent)
	require.NoError(t, err)

	eps, err = app.EntryPoints(context.Background())
	require.NoError(t, err)
	assert.Len(t, eps, 1)
	assert.Equal(t, testEP.Id, eps[0].Id)
}

func TestApplication_HandleEvent_BootstrapApplication(t *testing.T) {
	cp := &mockControlPlane{
		entrypoints: []*domain.Entrypoint{testEP},
		lbs:         []*domain.LoadBalancer{testLB},
		pools:       map[string]*domain.Pool{"pool-1": testPool},
		proxies:     []*domain.Proxy{testProxy},
	}

	app := newApp(cp)
	err := app.Bootstrap(context.Background())
	require.NoError(t, err)

	proxy, err := app.balancerService.Next(testLB.Id)
	require.NoError(t, err)
	assert.Equal(t, testProxy, proxy)

	// call the load balancer second time, so we can prove it returns the same proxy
	proxy, err = app.balancerService.Next(testLB.Id)
	require.NoError(t, err)
	assert.Equal(t, testProxy, proxy)

	// simulate adding new proxy to pool
	newTestProxy := &domain.Proxy{Id: "p2", Protocol: "http", Host: "127.0.0.1", Port: 8181}
	newTestPool := new(domain.Pool)
	*newTestPool = *testPool

	newTestPool.ProxyIds = []string{"p1", "p2"}

	cp.proxies = []*domain.Proxy{testProxy, newTestProxy}
	cp.pools["pool-1"] = newTestPool

	// handle change event
	changeEvent := event.ChangeEvent{
		ID:           newTestProxy.Id,
		ResourceType: event.ResourceTypeProxy,
		ChangeKind:   event.ChangeKindSaved,
	}

	err = app.HandleEvent(context.Background(), changeEvent)
	require.NoError(t, err)

	proxy, err = app.balancerService.Next(testLB.Id)
	require.NoError(t, err)
	assert.Equal(t, testProxy, proxy)

	proxy, err = app.balancerService.Next(testLB.Id)
	require.NoError(t, err)
	assert.Equal(t, newTestProxy, proxy)
}

func TestApplication_HandleEvent_DeletedRemovesFromCache(t *testing.T) {
	cp := &mockControlPlane{entrypoints: []*domain.Entrypoint{testEP}}

	app := newApp(cp)
	require.NoError(t, app.warmupCache(context.Background()))

	eps, err := app.EntryPoints(context.Background())
	require.NoError(t, err)
	require.Len(t, eps, 1)

	// upstream no longer has the entrypoint
	cp.entrypoints = []*domain.Entrypoint{}

	err = app.HandleEvent(context.Background(), event.ChangeEvent{
		ID:           testEP.Id,
		ResourceType: event.ResourceTypeEntrypoint,
		ChangeKind:   event.ChangeKindDeleted,
	})
	require.NoError(t, err)

	eps, err = app.EntryPoints(context.Background())
	require.NoError(t, err)
	assert.Len(t, eps, 0, "deleted entrypoint must drop out of the cache")
}

func TestApplication_HandleEvent_TopologyEventKeepsBalancerState(t *testing.T) {
	// two proxies make the round-robin cursor position observable
	secondProxy := &domain.Proxy{Id: "p2", Protocol: "http", Host: "127.0.0.1", Port: 8181}
	pool := &domain.Pool{Id: "pool-1", Type: domain.PoolTypeStatic, ProxyIds: []string{"p1", "p2"}}

	cp := &mockControlPlane{
		entrypoints: []*domain.Entrypoint{testEP},
		lbs:         []*domain.LoadBalancer{testLB},
		pools:       map[string]*domain.Pool{"pool-1": pool},
		proxies:     []*domain.Proxy{testProxy, secondProxy},
	}

	app := newApp(cp)
	require.NoError(t, app.Bootstrap(context.Background()))

	// advance the round-robin cursor past p1
	proxy, err := app.balancerService.Next(testLB.Id)
	require.NoError(t, err)
	require.Equal(t, testProxy, proxy)

	err = app.HandleEvent(context.Background(), event.ChangeEvent{
		ID:           testEP.Id,
		ResourceType: event.ResourceTypeEntrypoint,
		ChangeKind:   event.ChangeKindSaved,
	})
	require.NoError(t, err)

	// a rebuild would reset the cursor and serve p1 again
	proxy, err = app.balancerService.Next(testLB.Id)
	require.NoError(t, err)
	assert.Equal(t, secondProxy, proxy, "balancer registry must not be rebuilt on a topology event")
}

func TestApplication_HandleEvent_BalancerEventKeepsTopologyCache(t *testing.T) {
	cp := &mockControlPlane{
		entrypoints: []*domain.Entrypoint{testEP},
		lbs:         []*domain.LoadBalancer{testLB},
		pools:       map[string]*domain.Pool{"pool-1": testPool},
		proxies:     []*domain.Proxy{testProxy},
	}

	app := newApp(cp)
	require.NoError(t, app.Bootstrap(context.Background()))

	// upstream topology changes, but only a proxy event fires
	secondEP := &domain.Entrypoint{Id: "ep-2", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 9090, FlowId: "flow-1"}
	cp.entrypoints = []*domain.Entrypoint{testEP, secondEP}

	err := app.HandleEvent(context.Background(), event.ChangeEvent{
		ID:           testProxy.Id,
		ResourceType: event.ResourceTypeProxy,
		ChangeKind:   event.ChangeKindSaved,
	})
	require.NoError(t, err)

	// topology cache untouched: still only the original entrypoint
	eps, err := app.EntryPoints(context.Background())
	require.NoError(t, err)
	require.Len(t, eps, 1, "topology cache must not be reloaded on a proxy event")
	assert.Equal(t, testEP.Id, eps[0].Id)
}
