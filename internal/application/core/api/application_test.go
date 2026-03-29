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

// --- mock ---

// mockControlPlane implements ports.ControlPlaneReader for all api tests.
type mockControlPlane struct {
	entrypoints []*domain.Entrypoint
	flows       []*domain.Flow
	routers     []*domain.Router
	lbs         []*domain.LoadBalancer
	pools       map[string]*domain.Pool
	proxies     []*domain.Proxy

	epErr     error
	flowErr   error
	routerErr error
	lbErr     error
	poolErr   error
	proxyErr  error
}

// --- Entrypoints ---
func (m *mockControlPlane) ListEntrypoints(_ context.Context, _ domain.ListParams) (domain.ListResult[domain.Entrypoint], error) {
	return domain.ListResult[domain.Entrypoint]{Items: m.entrypoints}, m.epErr
}
func (m *mockControlPlane) GetEntrypoint(_ context.Context, name string) (*domain.Entrypoint, error) {
	if m.epErr != nil {
		return nil, m.epErr
	}
	for _, ep := range m.entrypoints {
		if ep.Id == name {
			return ep, nil
		}
	}
	return nil, fmt.Errorf("entrypoint %q not found", name)
}

// --- Flows ---
func (m *mockControlPlane) ListFlows(_ context.Context, _ domain.ListParams) (domain.ListResult[domain.Flow], error) {
	return domain.ListResult[domain.Flow]{Items: m.flows}, m.flowErr
}
func (m *mockControlPlane) GetFlow(_ context.Context, name string) (*domain.Flow, error) {
	if m.flowErr != nil {
		return nil, m.flowErr
	}
	for _, f := range m.flows {
		if f.Id == name {
			return f, nil
		}
	}
	return nil, fmt.Errorf("flow %q not found", name)
}

// --- Routers ---
func (m *mockControlPlane) ListRouters(_ context.Context, _ domain.ListParams) (domain.ListResult[domain.Router], error) {
	return domain.ListResult[domain.Router]{Items: m.routers}, m.routerErr
}
func (m *mockControlPlane) GetRouter(_ context.Context, name string) (*domain.Router, error) {
	if m.routerErr != nil {
		return nil, m.routerErr
	}
	for _, r := range m.routers {
		if r.Id == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("router %q not found", name)
}

// --- Balancers ---
func (m *mockControlPlane) ListBalancers(_ context.Context, _ domain.ListParams) (domain.ListResult[domain.LoadBalancer], error) {
	return domain.ListResult[domain.LoadBalancer]{Items: m.lbs}, m.lbErr
}
func (m *mockControlPlane) GetBalancer(_ context.Context, name string) (*domain.LoadBalancer, error) {
	if m.lbErr != nil {
		return nil, m.lbErr
	}
	for _, lb := range m.lbs {
		if lb.Id == name {
			return lb, nil
		}
	}
	return nil, fmt.Errorf("load balancer %q not found", name)
}

// --- Pools ---
func (m *mockControlPlane) ListPools(_ context.Context, _ domain.ListParams) (domain.ListResult[domain.Pool], error) {
	result := make([]*domain.Pool, 0, len(m.pools))
	for _, p := range m.pools {
		result = append(result, p)
	}
	return domain.ListResult[domain.Pool]{Items: result}, m.poolErr
}
func (m *mockControlPlane) GetPool(_ context.Context, name string) (*domain.Pool, error) {
	if m.poolErr != nil {
		return nil, m.poolErr
	}
	p, ok := m.pools[name]
	if !ok {
		return nil, fmt.Errorf("pool %q not found", name)
	}
	return p, nil
}

// --- Proxies ---
func (m *mockControlPlane) ListProxies(_ context.Context, _ domain.ListParams) (domain.ListResult[domain.Proxy], error) {
	return domain.ListResult[domain.Proxy]{Items: m.proxies}, m.proxyErr
}
func (m *mockControlPlane) GetProxy(_ context.Context, name string) (*domain.Proxy, error) {
	if m.proxyErr != nil {
		return nil, m.proxyErr
	}
	for _, p := range m.proxies {
		if p.Id == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("proxy %q not found", name)
}
func (m *mockControlPlane) GetProxiesByIds(_ context.Context, ids []string) ([]*domain.Proxy, error) {
	if m.proxyErr != nil {
		return nil, m.proxyErr
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
func (m *mockControlPlane) FindProxiesByLabels(_ context.Context, labels map[string]string) ([]*domain.Proxy, error) {
	if m.proxyErr != nil {
		return nil, m.proxyErr
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

// --- fixtures ---

var (
	testProxy  = &domain.Proxy{Id: "p1", Protocol: "http", Host: "127.0.0.1", Port: 8080}
	testPool   = &domain.Pool{Id: "pool-1", Type: domain.PoolTypeStatic, ProxyIds: []string{"p1"}}
	testLB     = &domain.LoadBalancer{Id: "lb-1", Type: domain.BalancerTypeRoundRobin, PoolId: "pool-1"}
	testEP     = &domain.Entrypoint{Id: "ep-1", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 8080, FlowId: "flow-1"}
	testFlow   = &domain.Flow{Id: "flow-1", BalancerId: "lb-1"}
)

func newApp(cp *mockControlPlane) *Application {
	return NewApplication(cp)
}

// --- tests ---

func TestApplication_Bootstrap(t *testing.T) {
	for _, tt := range []struct {
		name        string
		cp          *mockControlPlane
		expectedErr string
	}{
		{
			name: "successful bootstrap",
			cp: &mockControlPlane{
				entrypoints: []*domain.Entrypoint{testEP},
				lbs:         []*domain.LoadBalancer{testLB},
				pools:       map[string]*domain.Pool{"pool-1": testPool},
				proxies:     []*domain.Proxy{testProxy},
			},
		},
		{
			name:        "entrypoint repo error stops bootstrap",
			cp:          &mockControlPlane{epErr: errors.New("db down")},
			expectedErr: "cache bootstrap failed: db down",
		},
		{
			name: "invalid entrypoint fails validation",
			cp: &mockControlPlane{entrypoints: []*domain.Entrypoint{
				{Id: "ep-bad", Protocol: "ftp", Host: "0.0.0.0", Port: 8080},
			}},
			expectedErr: `entrypoint "ep-bad": invalid protocol: "ftp"`,
		},
		{
			name: "balancer service bootstrap error propagates",
			cp: &mockControlPlane{
				entrypoints: []*domain.Entrypoint{testEP},
				lbErr:       errors.New("lb error"),
			},
			expectedErr: "loading balancers: lb error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(tt.cp)
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
			app := NewApplication(&mockControlPlane{entrypoints: tt.eps})
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
		app := newApp(&mockControlPlane{entrypoints: []*domain.Entrypoint{testEP}})
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
		routers        []*domain.Router
		host           string
		expectedTarget string
		expectedErr    string
	}{
		{
			name:           "routes to specific target",
			routers:        []*domain.Router{routerWithRules},
			host:           "youtube.com",
			expectedTarget: "lb-video",
		},
		{
			name:           "falls back to catch_all",
			routers:        []*domain.Router{routerWithRules},
			host:           "other.com",
			expectedTarget: "lb-default",
		},
		{
			name:        "router not found",
			routers:     []*domain.Router{},
			expectedErr: `finding router "router-1":`,
		},
		{
			name: "no matching rule",
			routers: []*domain.Router{
				{Id: "router-1", Rules: []*domain.RouterRule{
					{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeHost, Value: "only.com"}, Target: "lb-x"},
				}},
			},
			host:        "other.com",
			expectedErr: `router "router-1": no matching rule found`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(&mockControlPlane{routers: tt.routers})
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
	cpWithLB := &mockControlPlane{
		lbs:     []*domain.LoadBalancer{testLB},
		pools:   map[string]*domain.Pool{"pool-1": testPool},
		proxies: []*domain.Proxy{testProxy},
	}

	t.Run("successful release", func(t *testing.T) {
		app := newApp(cpWithLB)
		require.NoError(t, app.Bootstrap(context.Background()))

		err := app.Release(context.Background(), "lb-1", domain.BalancerResult{ProxyId: "p1", Bytes: 1024})
		assert.NoError(t, err)
	})

	t.Run("balancer not found", func(t *testing.T) {
		app := newApp(cpWithLB)
		require.NoError(t, app.Bootstrap(context.Background()))

		err := app.Release(context.Background(), "non-existing", domain.BalancerResult{})
		assert.Error(t, err)
	})

	t.Run("release without bootstrap", func(t *testing.T) {
		app := newApp(cpWithLB)
		// Bootstrap çağrılmadı — registry boş

		err := app.Release(context.Background(), "lb-1", domain.BalancerResult{})
		assert.ErrorIs(t, err, balancer.ErrBalancerNotFound)
	})
}

func TestApplication_ExecuteFlow(t *testing.T) {
	for _, tt := range []struct {
		name          string
		cp            *mockControlPlane
		entrypointId  string
		expectedProxy *domain.Proxy
		expectedLBId  string
		expectedErr   string
	}{
		{
			name: "flow with direct balancer",
			cp: &mockControlPlane{
				entrypoints: []*domain.Entrypoint{testEP},
				flows:       []*domain.Flow{testFlow},
				lbs:         []*domain.LoadBalancer{testLB},
				pools:       map[string]*domain.Pool{"pool-1": testPool},
				proxies:     []*domain.Proxy{testProxy},
			},
			entrypointId:  "ep-1",
			expectedProxy: testProxy,
			expectedLBId:  "lb-1",
		},
		{
			name: "flow with router",
			cp: &mockControlPlane{
				entrypoints: []*domain.Entrypoint{
					{Id: "ep-2", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 9090, FlowId: "flow-router"},
				},
				flows: []*domain.Flow{
					{Id: "flow-router", RouterId: "router-1"},
				},
				routers: []*domain.Router{
					{Id: "router-1", Rules: []*domain.RouterRule{
						{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "lb-1"},
					}},
				},
				lbs:     []*domain.LoadBalancer{testLB},
				pools:   map[string]*domain.Pool{"pool-1": testPool},
				proxies: []*domain.Proxy{testProxy},
			},
			entrypointId:  "ep-2",
			expectedProxy: testProxy,
			expectedLBId:  "lb-1",
		},
		{
			name:          "entrypoint not found",
			cp:            &mockControlPlane{},
			entrypointId:  "missing",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "entrypoint:",
		},
		{
			name: "flow not found",
			cp: &mockControlPlane{
				entrypoints: []*domain.Entrypoint{testEP},
			},
			entrypointId:  "ep-1",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "flow:",
		},
		{
			name: "flow has neither router nor balancer",
			cp: &mockControlPlane{
				entrypoints: []*domain.Entrypoint{testEP},
				flows:       []*domain.Flow{{Id: "flow-1", RouterId: "", BalancerId: ""}},
			},
			entrypointId:  "ep-1",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   `no router or balancer for flow: "flow-1"`,
		},
		{
			name: "router returns no matching rule",
			cp: &mockControlPlane{
				entrypoints: []*domain.Entrypoint{
					{Id: "ep-3", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 9091, FlowId: "flow-r"},
				},
				flows: []*domain.Flow{
					{Id: "flow-r", RouterId: "router-1"},
				},
				routers: []*domain.Router{
					{Id: "router-1", Rules: []*domain.RouterRule{
						{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeHost, Value: "only.com"}, Target: "lb-1"},
					}},
				},
				lbs:     []*domain.LoadBalancer{testLB},
				pools:   map[string]*domain.Pool{"pool-1": testPool},
				proxies: []*domain.Proxy{testProxy},
			},
			entrypointId:  "ep-3",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "router:",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApplication(tt.cp)
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
