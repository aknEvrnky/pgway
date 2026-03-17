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

func (m *mockLBRepo) GetAll(ctx context.Context) ([]*domain.LoadBalancer, error) {
	return m.lbs, m.err
}

func (m *mockLBRepo) Find(ctx context.Context, id string) (*domain.LoadBalancer, error) {
	for _, lb := range m.lbs {
		if lb.Id == id {
			return lb, nil
		}
	}
	return nil, fmt.Errorf("load balancer %q not found", id)
}

type mockPoolRepo struct {
	pools map[string]*domain.Pool
	err   error
}

func (m *mockPoolRepo) GetAll(ctx context.Context) ([]*domain.Pool, error) {
	result := make([]*domain.Pool, 0, len(m.pools))
	for _, p := range m.pools {
		result = append(result, p)
	}
	return result, m.err
}

func (m *mockPoolRepo) Find(ctx context.Context, id string) (*domain.Pool, error) {
	if m.err != nil {
		return nil, m.err
	}
	pool, ok := m.pools[id]
	if !ok {
		return nil, fmt.Errorf("pool %q not found", id)
	}
	return pool, nil
}

var (
	testProxy = &domain.Proxy{Id: "p1", Protocol: "http", Host: "127.0.0.1", Port: 8080}
	testPool  = &domain.Pool{Id: "pool-1", Proxies: []*domain.Proxy{testProxy}}
	testLB    = &domain.LoadBalancer{Id: "lb-1", Type: domain.BalancerTypeRoundRobin, PoolId: "pool-1"}
)

func TestService_Bootstrap(t *testing.T) {
	for _, tt := range []struct {
		name        string
		lbRepo      *mockLBRepo
		poolRepo    *mockPoolRepo
		expectedErr string
	}{
		{
			name:     "successful bootstrap",
			lbRepo:   &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo: &mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": testPool}},
		},
		{
			name:        "lbRepo error",
			lbRepo:      &mockLBRepo{err: errors.New("db error")},
			poolRepo:    &mockPoolRepo{},
			expectedErr: "loading balancers: db error",
		},
		{
			name:        "poolRepo error",
			lbRepo:      &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:    &mockPoolRepo{err: errors.New("db error")},
			expectedErr: "loading pool: db error",
		},
		{
			name:        "empty pool",
			lbRepo:      &mockLBRepo{lbs: []*domain.LoadBalancer{testLB}},
			poolRepo:    &mockPoolRepo{pools: map[string]*domain.Pool{"pool-1": {}}},
			expectedErr: `balancer "lb-1": no proxy`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.lbRepo, tt.poolRepo)
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
