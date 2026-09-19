package balancer

import (
	"context"
	"errors"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/application/dataplane/dptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testProxy = &domain.Proxy{Id: "p1", Protocol: "http", Host: "127.0.0.1", Port: 8080}
	testPool  = &domain.Pool{Id: "pool-1", Type: domain.PoolTypeStatic, Members: []domain.PoolMember{{ProxyId: "p1", Weight: 1}}}
	testLB    = &domain.LoadBalancer{Id: "lb-1", Type: domain.BalancerTypeRoundRobin, PoolId: "pool-1"}
)

func TestService_Bootstrap(t *testing.T) {
	for _, tt := range []struct {
		name        string
		cp          *dptest.ControlPlane
		expectedErr string
	}{
		{
			name: "successful bootstrap",
			cp: &dptest.ControlPlane{
				Balancers: []*domain.LoadBalancer{testLB},
				Pools:     map[string]*domain.Pool{"pool-1": testPool},
				Proxies:   []*domain.Proxy{testProxy},
			},
		},
		{
			name:        "lbRepo error",
			cp:          &dptest.ControlPlane{BalancerErr: errors.New("db error")},
			expectedErr: "loading balancers: db error",
		},
		{
			name: "poolRepo error",
			cp: &dptest.ControlPlane{
				Balancers: []*domain.LoadBalancer{testLB},
				PoolErr:   errors.New("db error"),
			},
			expectedErr: "loading pool: db error",
		},
		{
			name: "pool resolves to empty proxy list",
			cp: &dptest.ControlPlane{
				Balancers: []*domain.LoadBalancer{testLB},
				Pools:     map[string]*domain.Pool{"pool-1": testPool},
			},
			expectedErr: `balancer "lb-1": no proxy`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.cp, tt.cp)
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
	cp := &dptest.ControlPlane{
		Balancers: []*domain.LoadBalancer{testLB},
		Pools:     map[string]*domain.Pool{"pool-1": testPool},
		Proxies:   []*domain.Proxy{testProxy},
	}
	svc := NewService(cp, cp)
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

func TestService_Bootstrap_RemovesDeletedBalancers(t *testing.T) {
	lbGone := &domain.LoadBalancer{Id: "lb-gone", Type: domain.BalancerTypeRoundRobin, PoolId: "pool-1"}
	cp := &dptest.ControlPlane{
		Balancers: []*domain.LoadBalancer{testLB, lbGone},
		Pools:     map[string]*domain.Pool{"pool-1": testPool},
		Proxies:   []*domain.Proxy{testProxy},
	}
	svc := NewService(cp, cp)
	require.NoError(t, svc.Bootstrap(context.Background()))

	_, err := svc.Get("lb-gone")
	require.NoError(t, err)

	cp.Balancers = []*domain.LoadBalancer{testLB}
	require.NoError(t, svc.Bootstrap(context.Background()))

	_, err = svc.Get("lb-gone")
	assert.ErrorIs(t, err, ErrBalancerNotFound)

	_, err = svc.Get(testLB.Id)
	require.NoError(t, err)
}
