package api

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.uber.org/zap"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/balancer"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/dptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- fixtures ---

var (
	testProxy = &domain.Proxy{Id: "p1", Protocol: "http", Host: "127.0.0.1", Port: 8080}
	testPool  = &domain.Pool{Id: "pool-1", Type: domain.PoolTypeStatic, Members: []domain.PoolMember{{ProxyId: "p1", Weight: 1}}}
	testLB    = &domain.LoadBalancer{Id: "lb-1", Type: domain.BalancerTypeRoundRobin, PoolId: "pool-1"}
	testEP    = &domain.Entrypoint{Id: "ep-1", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 8080, FlowId: "flow-1"}
	testFlow  = &domain.Flow{Id: "flow-1", BalancerId: "lb-1"}
)

func newApp(cp *dptest.ControlPlane) *Application {
	return NewApplication(cp, cp, zap.NewNop())
}

// --- tests ---

func TestApplication_Bootstrap(t *testing.T) {
	for _, tt := range []struct {
		name        string
		cp          *dptest.ControlPlane
		expectedErr string
	}{
		{
			name: "successful bootstrap",
			cp: &dptest.ControlPlane{
				Entrypoints: []*domain.Entrypoint{testEP},
				Balancers:   []*domain.LoadBalancer{testLB},
				Pools:       map[string]*domain.Pool{"pool-1": testPool},
				Proxies:     []*domain.Proxy{testProxy},
			},
		},
		{
			name:        "entrypoint repo error stops bootstrap",
			cp:          &dptest.ControlPlane{EntrypointErr: errors.New("db down")},
			expectedErr: "cache bootstrap failed: db down",
		},
		{
			name: "invalid entrypoint fails validation",
			cp: &dptest.ControlPlane{Entrypoints: []*domain.Entrypoint{
				{Id: "ep-bad", Protocol: "ftp", Host: "0.0.0.0", Port: 8080},
			}},
			expectedErr: `entrypoint "ep-bad": invalid protocol: "ftp"`,
		},
		{
			name: "balancer service bootstrap error propagates",
			cp: &dptest.ControlPlane{
				Entrypoints: []*domain.Entrypoint{testEP},
				BalancerErr: errors.New("lb error"),
			},
			expectedErr: "loading balancers: lb error",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := newApp(tt.cp)
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
			app := newApp(&dptest.ControlPlane{Entrypoints: tt.eps})
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
		app := newApp(&dptest.ControlPlane{Entrypoints: []*domain.Entrypoint{testEP}})
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
			app := newApp(&dptest.ControlPlane{Routers: tt.routers})
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
	cpWithLB := &dptest.ControlPlane{
		Balancers: []*domain.LoadBalancer{testLB},
		Pools:     map[string]*domain.Pool{"pool-1": testPool},
		Proxies:   []*domain.Proxy{testProxy},
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
		cp            *dptest.ControlPlane
		entrypointId  string
		expectedProxy *domain.Proxy
		expectedLBId  string
		expectedErr   string
	}{
		{
			name: "flow with direct balancer",
			cp: &dptest.ControlPlane{
				Entrypoints: []*domain.Entrypoint{testEP},
				Flows:       []*domain.Flow{testFlow},
				Balancers:   []*domain.LoadBalancer{testLB},
				Pools:       map[string]*domain.Pool{"pool-1": testPool},
				Proxies:     []*domain.Proxy{testProxy},
			},
			entrypointId:  "ep-1",
			expectedProxy: testProxy,
			expectedLBId:  "lb-1",
		},
		{
			name: "flow with router",
			cp: &dptest.ControlPlane{
				Entrypoints: []*domain.Entrypoint{
					{Id: "ep-2", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 9090, FlowId: "flow-router"},
				},
				Flows: []*domain.Flow{
					{Id: "flow-router", RouterId: "router-1"},
				},
				Routers: []*domain.Router{
					{Id: "router-1", Rules: []*domain.RouterRule{
						{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "lb-1"},
					}},
				},
				Balancers: []*domain.LoadBalancer{testLB},
				Pools:     map[string]*domain.Pool{"pool-1": testPool},
				Proxies:   []*domain.Proxy{testProxy},
			},
			entrypointId:  "ep-2",
			expectedProxy: testProxy,
			expectedLBId:  "lb-1",
		},
		{
			name:          "entrypoint not found",
			cp:            &dptest.ControlPlane{},
			entrypointId:  "missing",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "entrypoint:",
		},
		{
			name: "flow not found",
			cp: &dptest.ControlPlane{
				Entrypoints: []*domain.Entrypoint{testEP},
			},
			entrypointId:  "ep-1",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "flow:",
		},
		{
			name: "flow has neither router nor balancer",
			cp: &dptest.ControlPlane{
				Entrypoints: []*domain.Entrypoint{testEP},
				Flows:       []*domain.Flow{{Id: "flow-1", RouterId: "", BalancerId: ""}},
			},
			entrypointId:  "ep-1",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   `no router or balancer for flow: "flow-1"`,
		},
		{
			name: "router returns no matching rule",
			cp: &dptest.ControlPlane{
				Entrypoints: []*domain.Entrypoint{
					{Id: "ep-3", Protocol: domain.ProtocolHTTP, Host: "0.0.0.0", Port: 9091, FlowId: "flow-r"},
				},
				Flows: []*domain.Flow{
					{Id: "flow-r", RouterId: "router-1"},
				},
				Routers: []*domain.Router{
					{Id: "router-1", Rules: []*domain.RouterRule{
						{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeHost, Value: "only.com"}, Target: "lb-1"},
					}},
				},
				Balancers: []*domain.LoadBalancer{testLB},
				Pools:     map[string]*domain.Pool{"pool-1": testPool},
				Proxies:   []*domain.Proxy{testProxy},
			},
			entrypointId:  "ep-3",
			expectedProxy: nil,
			expectedLBId:  "",
			expectedErr:   "router:",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := newApp(tt.cp)
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
