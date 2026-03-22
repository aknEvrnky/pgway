package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/balancer"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockEntryPointRepo struct {
	eps []*domain.Entrypoint
	err error
}

func (m *mockEntryPointRepo) GetAll(_ context.Context) ([]*domain.Entrypoint, error) {
	return m.eps, m.err
}

func (m *mockEntryPointRepo) Find(_ context.Context, id string) (*domain.Entrypoint, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, ep := range m.eps {
		if ep.Id == id {
			return ep, nil
		}
	}
	return nil, fmt.Errorf("entrypoint %q not found", id)
}

func (m *mockEntryPointRepo) Save(_ context.Context, _ *domain.Entrypoint) error { return m.err }
func (m *mockEntryPointRepo) Delete(_ context.Context, _ string) error           { return m.err }

type mockFlowRepo struct {
	flows []*domain.Flow
	err   error
}

func (m *mockFlowRepo) GetAll(_ context.Context) ([]*domain.Flow, error) {
	return m.flows, m.err
}

func (m *mockFlowRepo) Find(_ context.Context, id string) (*domain.Flow, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, f := range m.flows {
		if f.Id == id {
			return f, nil
		}
	}
	return nil, fmt.Errorf("flow %q not found", id)
}

func (m *mockFlowRepo) Save(_ context.Context, _ *domain.Flow) error { return m.err }
func (m *mockFlowRepo) Delete(_ context.Context, _ string) error     { return m.err }

type mockRouterRepo struct {
	routers []*domain.Router
	err     error
}

func (m *mockRouterRepo) GetAll(_ context.Context) ([]*domain.Router, error) {
	return m.routers, m.err
}

func (m *mockRouterRepo) Find(_ context.Context, id string) (*domain.Router, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, r := range m.routers {
		if r.Id == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("router %q not found", id)
}

func (m *mockRouterRepo) Save(_ context.Context, _ *domain.Router) error { return m.err }
func (m *mockRouterRepo) Delete(_ context.Context, _ string) error       { return m.err }

type mockLBRepo struct {
	lbs []*domain.LoadBalancer
	err error
}

func (m *mockLBRepo) GetAll(_ context.Context) ([]*domain.LoadBalancer, error) {
	return m.lbs, m.err
}

func (m *mockLBRepo) Find(_ context.Context, id string) (*domain.LoadBalancer, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, lb := range m.lbs {
		if lb.Id == id {
			return lb, nil
		}
	}
	return nil, fmt.Errorf("load balancer %q not found", id)
}

func (m *mockLBRepo) Save(_ context.Context, _ *domain.LoadBalancer) error { return m.err }
func (m *mockLBRepo) Delete(_ context.Context, _ string) error             { return m.err }

type mockPoolRepo struct {
	pools map[string]*domain.Pool
	err   error
}

func (m *mockPoolRepo) GetAll(_ context.Context) ([]*domain.Pool, error) {
	result := make([]*domain.Pool, 0, len(m.pools))
	for _, p := range m.pools {
		result = append(result, p)
	}
	return result, m.err
}

func (m *mockPoolRepo) Find(_ context.Context, id string) (*domain.Pool, error) {
	if m.err != nil {
		return nil, m.err
	}
	pool, ok := m.pools[id]
	if !ok {
		return nil, fmt.Errorf("pool %q not found", id)
	}
	return pool, nil
}

func (m *mockPoolRepo) Save(_ context.Context, _ *domain.Pool) error { return m.err }
func (m *mockPoolRepo) Delete(_ context.Context, _ string) error     { return m.err }

type mockProxyRepo struct {
	proxies []*domain.Proxy
	err     error
}

func (m *mockProxyRepo) GetAll(_ context.Context) ([]*domain.Proxy, error) {
	return m.proxies, m.err
}

func (m *mockProxyRepo) Find(_ context.Context, id string) (*domain.Proxy, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, p := range m.proxies {
		if p.Id == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("proxy %q not found", id)
}

func (m *mockProxyRepo) GetByIds(_ context.Context, ids []string) ([]*domain.Proxy, error) {
	if m.err != nil {
		return nil, m.err
	}
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	var result []*domain.Proxy
	for _, p := range m.proxies {
		if _, ok := idSet[p.Id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockProxyRepo) FindByLabels(_ context.Context, labels map[string]string) ([]*domain.Proxy, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*domain.Proxy
outer:
	for _, p := range m.proxies {
		for k, v := range labels {
			if p.Labels[k] != v {
				continue outer
			}
		}
		result = append(result, p)
	}
	return result, nil
}

func (m *mockProxyRepo) Save(_ context.Context, proxy *domain.Proxy) error {
	if m.err != nil {
		return m.err
	}
	for i, p := range m.proxies {
		if p.Id == proxy.Id {
			m.proxies[i] = proxy
			return nil
		}
	}
	m.proxies = append(m.proxies, proxy)
	return nil
}

func (m *mockProxyRepo) Delete(_ context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	for i, p := range m.proxies {
		if p.Id == id {
			m.proxies = append(m.proxies[:i], m.proxies[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("proxy %q not found", id)
}

// --- fixtures ---

var (
	testProxy  = &domain.Proxy{Id: "p1", Protocol: "http", Host: "127.0.0.1", Port: 8080}
	testPool   = &domain.Pool{Id: "pool-1", Type: domain.PoolTypeStatic, ProxyIds: []string{"p1"}}
	testLB     = &domain.LoadBalancer{Id: "lb-1", Type: domain.BalancerTypeRoundRobin, PoolId: "pool-1"}
	testEP     = &domain.Entrypoint{Id: "ep-1", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 8080, FlowId: "flow-1"}
	testFlow   = &domain.Flow{Id: "flow-1", BalancerId: "lb-1"}
	testRouter = &domain.Router{
		Id: "router-1",
		Rules: []*domain.RouterRule{
			{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "lb-1"},
		},
	}
)

func newApp(
	eps []*domain.Entrypoint,
	flows []*domain.Flow,
	routers []*domain.Router,
	lbs []*domain.LoadBalancer,
	pools map[string]*domain.Pool,
) *Application {
	return NewApplication(
		&mockEntryPointRepo{eps: eps},
		&mockFlowRepo{flows: flows},
		&mockRouterRepo{routers: routers},
		&mockLBRepo{lbs: lbs},
		&mockPoolRepo{pools: pools},
		&mockProxyRepo{proxies: []*domain.Proxy{testProxy}},
	)
}

// --- tests ---

func TestApplication_Bootstrap(t *testing.T) {
	for _, tt := range []struct {
		name        string
		epRepo      *mockEntryPointRepo
		lbRepo      *mockLBRepo
		poolRepo    *mockPoolRepo
		proxyRepo   *mockProxyRepo
		expectedErr string
	}{
		{
			name:      "successful bootstrap",
			epRepo:    &mockEntryPointRepo{eps: []*domain.Entrypoint{testEP}},
			lbRepo:    &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:  &mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": testPool}},
			proxyRepo: &mockProxyRepo{proxies: []*domain.Proxy{testProxy}},
		},
		{
			name:        "entrypoint repo error stops bootstrap",
			epRepo:      &mockEntryPointRepo{err: errors.New("db down")},
			lbRepo:      &mockLBRepo{},
			poolRepo:    &mockPoolRepo{},
			proxyRepo:   &mockProxyRepo{},
			expectedErr: "cache bootstrap failed: db down",
		},
		{
			name: "invalid entrypoint fails validation",
			epRepo: &mockEntryPointRepo{eps: []*domain.Entrypoint{
				{Id: "ep-bad", Protocol: "ftp", Host: "0.0.0.0", Port: 8080},
			}},
			lbRepo:      &mockLBRepo{},
			poolRepo:    &mockPoolRepo{},
			proxyRepo:   &mockProxyRepo{},
			expectedErr: `entrypoint "ep-bad": invalid protocol: "ftp"`,
		},
		{
			name:        "balancer service bootstrap error propagates",
			epRepo:      &mockEntryPointRepo{eps: []*domain.Entrypoint{testEP}},
			lbRepo:      &mockLBRepo{err: errors.New("lb error")},
			poolRepo:    &mockPoolRepo{},
			proxyRepo:   &mockProxyRepo{},
			expectedErr: "loading balancers: lb error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(tt.epRepo, &mockFlowRepo{}, &mockRouterRepo{}, tt.lbRepo, tt.poolRepo, tt.proxyRepo)
			err := app.Bootstrap(context.Background())

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplication_ValidateAll(t *testing.T) {
	for _, tt := range []struct {
		name        string
		eps         []*domain.Entrypoint
		expectedErr string
	}{
		{
			name: "all valid entrypoints",
			eps:  []*domain.Entrypoint{testEP},
		},
		{
			name: "entrypoint missing host",
			eps: []*domain.Entrypoint{
				{Id: "ep-x", Protocol: domain.ProtocolHTTP, Host: "", Port: 9000},
			},
			expectedErr: `entrypoint "ep-x": host is required`,
		},
		{
			name: "entrypoint missing port",
			eps: []*domain.Entrypoint{
				{Id: "ep-x", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 0},
			},
			expectedErr: `entrypoint "ep-x": port is required`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(
				&mockEntryPointRepo{eps: tt.eps},
				&mockFlowRepo{}, &mockRouterRepo{}, &mockLBRepo{}, &mockPoolRepo{}, &mockProxyRepo{},
			)
			require.NoError(t, app.warmupCache(context.Background()))
			err := app.validateAll(context.Background())

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplication_LoadEntryPoints(t *testing.T) {
	t.Run("returns all entrypoints", func(t *testing.T) {
		app := newApp([]*domain.Entrypoint{testEP}, nil, nil, nil, nil)
		require.NoError(t, app.warmupCache(context.Background()))
		eps, err := app.EntryPoints(context.Background())
		require.NoError(t, err)
		assert.Len(t, eps, 1)
		assert.Equal(t, testEP.Id, eps[0].Id)
	})
}

func TestApplication_RouteRequest(t *testing.T) {
	routerWithRules := &domain.Router{
		Id: "router-1",
		Rules: []*domain.RouterRule{
			{
				Id:     "r1",
				Match:  domain.RouterMatch{Type: domain.MatchTypeHost, Value: "youtube.com"},
				Target: "lb-video",
			},
			{
				Id:     "fallback",
				Match:  domain.RouterMatch{Type: domain.MatchTypeCatchAll},
				Target: "lb-default",
			},
		},
	}

	for _, tt := range []struct {
		name           string
		routerRepo     *mockRouterRepo
		host           string
		expectedTarget string
		expectedErr    string
	}{
		{
			name:           "routes to specific target",
			routerRepo:     &mockRouterRepo{routers: []*domain.Router{routerWithRules}},
			host:           "youtube.com",
			expectedTarget: "lb-video",
		},
		{
			name:           "falls back to catch_all",
			routerRepo:     &mockRouterRepo{routers: []*domain.Router{routerWithRules}},
			host:           "other.com",
			expectedTarget: "lb-default",
		},
		{
			name:        "router not found",
			routerRepo:  &mockRouterRepo{routers: []*domain.Router{}},
			expectedErr: `finding router "router-1":`,
		},
		{
			name: "no matching rule",
			routerRepo: &mockRouterRepo{routers: []*domain.Router{
				{Id: "router-1", Rules: []*domain.RouterRule{
					{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeHost, Value: "only.com"}, Target: "lb-x"},
				}},
			}},
			host:        "other.com",
			expectedErr: `router "router-1": no matching rule found`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(
				&mockEntryPointRepo{}, &mockFlowRepo{}, tt.routerRepo, &mockLBRepo{}, &mockPoolRepo{}, &mockProxyRepo{},
			)
			require.NoError(t, app.warmupCache(context.Background()))

			req, _ := http.NewRequest("GET", "http://"+tt.host+"/", nil)
			target, err := app.RouteRequest(context.Background(), "router-1", req)

			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedTarget, target)
			}
		})
	}
}

func TestApplication_Release(t *testing.T) {
	t.Run("successful release", func(t *testing.T) {
		app := newApp(nil, nil, nil, []*domain.LoadBalancer{testLB}, map[string]*domain.Pool{"pool-1": testPool})
		require.NoError(t, app.Bootstrap(context.Background()))

		err := app.Release(context.Background(), "lb-1", domain.BalancerResult{ProxyId: "p1", Bytes: 1024})
		assert.NoError(t, err)
	})

	t.Run("balancer not found", func(t *testing.T) {
		app := newApp(nil, nil, nil, []*domain.LoadBalancer{testLB}, map[string]*domain.Pool{"pool-1": testPool})
		require.NoError(t, app.Bootstrap(context.Background()))

		err := app.Release(context.Background(), "non-existing", domain.BalancerResult{})
		assert.Error(t, err)
	})

	t.Run("release without bootstrap", func(t *testing.T) {
		app := newApp(nil, nil, nil, []*domain.LoadBalancer{testLB}, map[string]*domain.Pool{"pool-1": testPool})
		// Bootstrap çağrılmadı — registry boş

		err := app.Release(context.Background(), "lb-1", domain.BalancerResult{})
		assert.ErrorIs(t, err, balancer.ErrBalancerNotFound)
	})
}

func TestApplication_ExecuteFlow(t *testing.T) {
	for _, tt := range []struct {
		name          string
		epRepo        *mockEntryPointRepo
		flowRepo      *mockFlowRepo
		routerRepo    *mockRouterRepo
		lbRepo        *mockLBRepo
		poolRepo      *mockPoolRepo
		proxyRepo     *mockProxyRepo
		entrypointId  string
		expectedProxy *domain.Proxy
		expectedLBId  string
		expectedErr   string
	}{
		{
			name:          "flow with direct balancer",
			epRepo:        &mockEntryPointRepo{eps: []*domain.Entrypoint{testEP}},
			flowRepo:      &mockFlowRepo{flows: []*domain.Flow{testFlow}},
			routerRepo:    &mockRouterRepo{},
			lbRepo:        &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:      &mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": testPool}},
			proxyRepo:     &mockProxyRepo{proxies: []*domain.Proxy{testProxy}},
			entrypointId:  "ep-1",
			expectedProxy: testProxy,
			expectedLBId:  "lb-1",
		},
		{
			name:   "flow with router",
			epRepo: &mockEntryPointRepo{eps: []*domain.Entrypoint{{Id: "ep-2", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 9090, FlowId: "flow-router"}}},
			flowRepo: &mockFlowRepo{flows: []*domain.Flow{
				{Id: "flow-router", RouterId: "router-1"},
			}},
			routerRepo: &mockRouterRepo{routers: []*domain.Router{
				{Id: "router-1", Rules: []*domain.RouterRule{
					{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "lb-1"},
				}},
			}},
			lbRepo:        &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:      &mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": testPool}},
			proxyRepo:     &mockProxyRepo{proxies: []*domain.Proxy{testProxy}},
			entrypointId:  "ep-2",
			expectedProxy: testProxy,
			expectedLBId:  "lb-1",
		},
		{
			name:          "entrypoint not found",
			epRepo:        &mockEntryPointRepo{eps: []*domain.Entrypoint{}},
			flowRepo:      &mockFlowRepo{},
			routerRepo:    &mockRouterRepo{},
			lbRepo:        &mockLBRepo{},
			poolRepo:      &mockPoolRepo{},
			proxyRepo:     &mockProxyRepo{},
			entrypointId:  "missing",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "entrypoint:",
		},
		{
			name:          "flow not found",
			epRepo:        &mockEntryPointRepo{eps: []*domain.Entrypoint{testEP}},
			flowRepo:      &mockFlowRepo{flows: []*domain.Flow{}},
			routerRepo:    &mockRouterRepo{},
			lbRepo:        &mockLBRepo{},
			poolRepo:      &mockPoolRepo{},
			proxyRepo:     &mockProxyRepo{},
			entrypointId:  "ep-1",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "flow:",
		},
		{
			name:   "flow has neither router nor balancer",
			epRepo: &mockEntryPointRepo{eps: []*domain.Entrypoint{testEP}},
			flowRepo: &mockFlowRepo{flows: []*domain.Flow{
				{Id: "flow-1", RouterId: "", BalancerId: ""},
			}},
			routerRepo:    &mockRouterRepo{},
			lbRepo:        &mockLBRepo{},
			poolRepo:      &mockPoolRepo{},
			proxyRepo:     &mockProxyRepo{},
			entrypointId:  "ep-1",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   `no router or balancer for flow: "flow-1"`,
		},
		{
			name:   "router returns no matching rule",
			epRepo: &mockEntryPointRepo{eps: []*domain.Entrypoint{{Id: "ep-3", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 9091, FlowId: "flow-r"}}},
			flowRepo: &mockFlowRepo{flows: []*domain.Flow{
				{Id: "flow-r", RouterId: "router-1"},
			}},
			routerRepo: &mockRouterRepo{routers: []*domain.Router{
				{Id: "router-1", Rules: []*domain.RouterRule{
					{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeHost, Value: "only.com"}, Target: "lb-1"},
				}},
			}},
			lbRepo:        &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:      &mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": testPool}},
			proxyRepo:     &mockProxyRepo{proxies: []*domain.Proxy{testProxy}},
			entrypointId:  "ep-3",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "router:",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(tt.epRepo, tt.flowRepo, tt.routerRepo, tt.lbRepo, tt.poolRepo, tt.proxyRepo)
			require.NoError(t, app.Bootstrap(context.Background()))

			req, _ := http.NewRequest("GET", "http://example.com/", nil)
			proxy, lbId, err := app.ExecuteFlow(context.Background(), tt.entrypointId, req)

			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				assert.Nil(t, proxy)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedProxy, proxy)
				assert.Equal(t, tt.expectedLBId, lbId)
			}
		})
	}
}
