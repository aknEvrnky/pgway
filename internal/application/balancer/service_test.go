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

type mockLBRepo struct {
	lbs []*domain.LoadBalancer
	err error
}

func (m *mockLBRepo) GetAll(_ context.Context) ([]*domain.LoadBalancer, error) {
	return m.lbs, m.err
}

func (m *mockLBRepo) Find(_ context.Context, id string) (*domain.LoadBalancer, error) {
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

func (m *mockProxyRepo) Save(_ context.Context, _ *domain.Proxy) error { return m.err }
func (m *mockProxyRepo) Delete(_ context.Context, _ string) error      { return m.err }

var (
	testProxy = &domain.Proxy{Id: "p1", Protocol: "http", Host: "127.0.0.1", Port: 8080}
	testPool  = &domain.Pool{Id: "pool-1", Type: domain.PoolTypeStatic, ProxyIds: []string{"p1"}}
	testLB    = &domain.LoadBalancer{Id: "lb-1", Type: domain.BalancerTypeRoundRobin, PoolId: "pool-1"}
)

func TestService_Bootstrap(t *testing.T) {
	for _, tt := range []struct {
		name        string
		lbRepo      *mockLBRepo
		poolRepo    *mockPoolRepo
		proxyRepo   *mockProxyRepo
		expectedErr string
	}{
		{
			name:      "successful bootstrap",
			lbRepo:    &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:  &mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": testPool}},
			proxyRepo: &mockProxyRepo{proxies: []*domain.Proxy{testProxy}},
		},
		{
			name:        "lbRepo error",
			lbRepo:      &mockLBRepo{err: errors.New("db error")},
			poolRepo:    &mockPoolRepo{},
			proxyRepo:   &mockProxyRepo{},
			expectedErr: "loading balancers: db error",
		},
		{
			name:        "poolRepo error",
			lbRepo:      &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:    &mockPoolRepo{err: errors.New("db error")},
			proxyRepo:   &mockProxyRepo{},
			expectedErr: "loading pool: db error",
		},
		{
			name:        "pool resolves to empty proxy list",
			lbRepo:      &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:    &mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": testPool}},
			proxyRepo:   &mockProxyRepo{},
			expectedErr: `balancer "lb-1": no proxy`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.lbRepo, tt.poolRepo, tt.proxyRepo)
			err := svc.Bootstrap(context.Background())

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)

			// check registry after bootstrap
			instance, err := svc.Get(testLB.Id)
			require.NoError(t, err)
			assert.NotNil(t, instance)
		})
	}
}

func TestService_Get(t *testing.T) {
	svc := NewService(
		&mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
		&mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": testPool}},
		&mockProxyRepo{proxies: []*domain.Proxy{testProxy}},
	)
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
