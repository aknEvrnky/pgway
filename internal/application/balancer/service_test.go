package balancer

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockControlPlane implements ports.ControlPlaneReader for balancer tests.
// Only lb/pool/proxy fields are relevant; other methods are no-ops.
type mockControlPlane struct {
	proxies []*domain.Proxy
	pools   map[string]*domain.Pool
	lbs     []*domain.LoadBalancer

	lbErr    error
	poolErr  error
	proxyErr error
}

// --- Balancers ---
func (m *mockControlPlane) ListBalancers(_ context.Context) ([]*domain.LoadBalancer, error) {
	return m.lbs, m.lbErr
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
func (m *mockControlPlane) ListPools(_ context.Context) ([]*domain.Pool, error) {
	result := make([]*domain.Pool, 0, len(m.pools))
	for _, p := range m.pools {
		result = append(result, p)
	}
	return result, m.poolErr
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
func (m *mockControlPlane) ListProxies(_ context.Context) ([]*domain.Proxy, error) {
	return m.proxies, m.proxyErr
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

// --- Stubs (not used by balancer.Service) ---
func (m *mockControlPlane) ListRouters(_ context.Context) ([]*domain.Router, error) { return nil, nil }
func (m *mockControlPlane) GetRouter(_ context.Context, _ string) (*domain.Router, error) {
	return nil, nil
}
func (m *mockControlPlane) ListFlows(_ context.Context) ([]*domain.Flow, error) { return nil, nil }
func (m *mockControlPlane) GetFlow(_ context.Context, _ string) (*domain.Flow, error) {
	return nil, nil
}
func (m *mockControlPlane) ListEntrypoints(_ context.Context) ([]*domain.Entrypoint, error) {
	return nil, nil
}
func (m *mockControlPlane) GetEntrypoint(_ context.Context, _ string) (*domain.Entrypoint, error) {
	return nil, nil
}

// --- fixtures ---

var (
	testProxy = &domain.Proxy{Id: "p1", Protocol: "http", Host: "127.0.0.1", Port: 8080}
	testPool  = &domain.Pool{Id: "pool-1", Type: domain.PoolTypeStatic, ProxyIds: []string{"p1"}}
	testLB    = &domain.LoadBalancer{Id: "lb-1", Type: domain.BalancerTypeRoundRobin, PoolId: "pool-1"}
)

func TestService_Bootstrap(t *testing.T) {
	for _, tt := range []struct {
		name        string
		cp          *mockControlPlane
		expectedErr string
	}{
		{
			name: "successful bootstrap",
			cp: &mockControlPlane{
				lbs:     []*domain.LoadBalancer{testLB},
				pools:   map[string]*domain.Pool{"pool-1": testPool},
				proxies: []*domain.Proxy{testProxy},
			},
		},
		{
			name:        "lbRepo error",
			cp:          &mockControlPlane{lbErr: errors.New("db error")},
			expectedErr: "loading balancers: db error",
		},
		{
			name: "poolRepo error",
			cp: &mockControlPlane{
				lbs:     []*domain.LoadBalancer{testLB},
				poolErr: errors.New("db error"),
			},
			expectedErr: "loading pool: db error",
		},
		{
			name: "pool resolves to empty proxy list",
			cp: &mockControlPlane{
				lbs:   []*domain.LoadBalancer{testLB},
				pools: map[string]*domain.Pool{"pool-1": testPool},
			},
			expectedErr: `balancer "lb-1": no proxy`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.cp)
			err := svc.Bootstrap(context.Background())

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)

			instance, err := svc.Get(testLB.Id)
			require.NoError(t, err)
			assert.NotNil(t, instance)
		})
	}
}

func TestService_Get(t *testing.T) {
	svc := NewService(&mockControlPlane{
		lbs:     []*domain.LoadBalancer{testLB},
		pools:   map[string]*domain.Pool{"pool-1": testPool},
		proxies: []*domain.Proxy{testProxy},
	})
	require.NoError(t, svc.Bootstrap(context.Background()))

	t.Run("existing id", func(t *testing.T) {
		instance, err := svc.Get(testLB.Id)
		require.NoError(t, err)
		assert.NotNil(t, instance)
	})

	t.Run("non-existing id", func(t *testing.T) {
		instance, err := svc.Get("non-existing")
		assert.Nil(t, instance)
		assert.ErrorIs(t, err, ErrBalancerNotFound)
	})
}
