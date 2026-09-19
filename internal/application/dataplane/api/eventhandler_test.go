package api

import (
	"context"
	"testing"

	"github.com/aknEvrnky/pgway/internal/ports"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/dptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplication_HandleEvent_WarmupCache(t *testing.T) {
	cp := &dptest.ControlPlane{Entrypoints: []*domain.Entrypoint{}}

	app := newApp(cp)
	require.NoError(t, app.warmupCache(context.Background()))
	eps, err := app.EntryPoints(context.Background())
	require.NoError(t, err)
	assert.Len(t, eps, 0)

	cp.Entrypoints = append(cp.Entrypoints, testEP)

	// create a mock event
	mockEvent := ports.ChangeEvent{
		ID:           testEP.Id,
		ResourceType: ports.ResourceTypeEntrypoint,
		ChangeKind:   ports.ChangeKindSaved,
	}

	err = app.HandleEvent(context.Background(), mockEvent)
	require.NoError(t, err)

	eps, err = app.EntryPoints(context.Background())
	require.NoError(t, err)
	assert.Len(t, eps, 1)
	assert.Equal(t, testEP.Id, eps[0].Id)
}

func TestApplication_HandleEvent_BootstrapApplication(t *testing.T) {
	cp := &dptest.ControlPlane{
		Entrypoints: []*domain.Entrypoint{testEP},
		Balancers:   []*domain.LoadBalancer{testLB},
		Pools:       map[string]*domain.Pool{"pool-1": testPool},
		Proxies:     []*domain.Proxy{testProxy},
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

	newTestPool.Members = []domain.PoolMember{
		{ProxyId: "p1", Weight: 1},
		{ProxyId: "p2", Weight: 1},
	}

	cp.Proxies = []*domain.Proxy{testProxy, newTestProxy}
	cp.Pools["pool-1"] = newTestPool

	// handle change event
	changeEvent := ports.ChangeEvent{
		ID:           newTestProxy.Id,
		ResourceType: ports.ResourceTypeProxy,
		ChangeKind:   ports.ChangeKindSaved,
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
	cp := &dptest.ControlPlane{Entrypoints: []*domain.Entrypoint{testEP}}

	app := newApp(cp)
	require.NoError(t, app.warmupCache(context.Background()))

	eps, err := app.EntryPoints(context.Background())
	require.NoError(t, err)
	require.Len(t, eps, 1)

	// upstream no longer has the entrypoint
	cp.Entrypoints = []*domain.Entrypoint{}

	err = app.HandleEvent(context.Background(), ports.ChangeEvent{
		ID:           testEP.Id,
		ResourceType: ports.ResourceTypeEntrypoint,
		ChangeKind:   ports.ChangeKindDeleted,
	})
	require.NoError(t, err)

	eps, err = app.EntryPoints(context.Background())
	require.NoError(t, err)
	assert.Len(t, eps, 0, "deleted entrypoint must drop out of the cache")
}

func TestApplication_HandleEvent_TopologyEventKeepsBalancerState(t *testing.T) {
	// two proxies make the round-robin cursor position observable
	secondProxy := &domain.Proxy{Id: "p2", Protocol: "http", Host: "127.0.0.1", Port: 8181}
	pool := &domain.Pool{Id: "pool-1", Type: domain.PoolTypeStatic, Members: []domain.PoolMember{
		{ProxyId: "p1", Weight: 1},
		{ProxyId: "p2", Weight: 1},
	}}

	cp := &dptest.ControlPlane{
		Entrypoints: []*domain.Entrypoint{testEP},
		Balancers:   []*domain.LoadBalancer{testLB},
		Pools:       map[string]*domain.Pool{"pool-1": pool},
		Proxies:     []*domain.Proxy{testProxy, secondProxy},
	}

	app := newApp(cp)
	require.NoError(t, app.Bootstrap(context.Background()))

	// advance the round-robin cursor past p1
	proxy, err := app.balancerService.Next(testLB.Id)
	require.NoError(t, err)
	require.Equal(t, testProxy, proxy)

	err = app.HandleEvent(context.Background(), ports.ChangeEvent{
		ID:           testEP.Id,
		ResourceType: ports.ResourceTypeEntrypoint,
		ChangeKind:   ports.ChangeKindSaved,
	})
	require.NoError(t, err)

	// a rebuild would reset the cursor and serve p1 again
	proxy, err = app.balancerService.Next(testLB.Id)
	require.NoError(t, err)
	assert.Equal(t, secondProxy, proxy, "balancer registry must not be rebuilt on a topology event")
}

func TestApplication_HandleEvent_BalancerEventKeepsTopologyCache(t *testing.T) {
	cp := &dptest.ControlPlane{
		Entrypoints: []*domain.Entrypoint{testEP},
		Balancers:   []*domain.LoadBalancer{testLB},
		Pools:       map[string]*domain.Pool{"pool-1": testPool},
		Proxies:     []*domain.Proxy{testProxy},
	}

	app := newApp(cp)
	require.NoError(t, app.Bootstrap(context.Background()))

	// upstream topology changes, but only a proxy event fires
	secondEP := &domain.Entrypoint{Id: "ep-2", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 9090, FlowId: "flow-1"}
	cp.Entrypoints = []*domain.Entrypoint{testEP, secondEP}

	err := app.HandleEvent(context.Background(), ports.ChangeEvent{
		ID:           testProxy.Id,
		ResourceType: ports.ResourceTypeProxy,
		ChangeKind:   ports.ChangeKindSaved,
	})
	require.NoError(t, err)

	// topology cache untouched: still only the original entrypoint
	eps, err := app.EntryPoints(context.Background())
	require.NoError(t, err)
	require.Len(t, eps, 1, "topology cache must not be reloaded on a proxy event")
	assert.Equal(t, testEP.Id, eps[0].Id)
}
