package balancer

import (
	"fmt"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/balancer/algorithm"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild(t *testing.T) {
	poolWithProxy := func() *domain.Pool {
		p := &domain.Pool{
			Id:      "pool-1",
			Type:    domain.PoolTypeStatic,
			Members: []domain.PoolMember{{ProxyId: "proxy-1", Weight: 1}},
		}
		p.LoadResolvedProxies([]*domain.Proxy{{Id: "proxy-1"}})
		return p
	}()

	dynamicPool := func() *domain.Pool {
		p := &domain.Pool{Id: "pool-dyn", Type: domain.PoolTypeDynamic}
		p.LoadResolvedProxies([]*domain.Proxy{{Id: "proxy-1"}})
		return p
	}()

	for _, tt := range []struct {
		name                 string
		lb                   *domain.LoadBalancer
		p                    *domain.Pool
		expectedBalancerType interface{}
		expectedErr          error
	}{
		{
			name:                 "it builds round robin load balancer",
			p:                    poolWithProxy,
			lb:                   &domain.LoadBalancer{Id: "lb-1", Type: "round-robin", PoolId: "pool-1"},
			expectedBalancerType: &algorithm.RoundRobin{},
			expectedErr:          nil,
		},
		{
			name:                 "it builds weighted load balancer",
			p:                    poolWithProxy,
			lb:                   &domain.LoadBalancer{Id: "lb-1", Type: "weighted", PoolId: "pool-1"},
			expectedBalancerType: &algorithm.Weighted{},
			expectedErr:          nil,
		},
		{
			name:        "weighted rejects dynamic pool",
			p:           dynamicPool,
			lb:          &domain.LoadBalancer{Id: "lb-1", Type: "weighted", PoolId: "pool-dyn"},
			expectedErr: fmt.Errorf(`weighted balancer requires static pool "pool-dyn"`),
		},
		{
			name:                 "it can not build unknown lb type",
			p:                    poolWithProxy,
			lb:                   &domain.LoadBalancer{Id: "lb-1", Type: "unknown", PoolId: "pool-1"},
			expectedBalancerType: &algorithm.RoundRobin{},
			expectedErr:          fmt.Errorf(`unknown balancer type "unknown"`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			balancer, err := Build(tt.lb, tt.p)

			if tt.expectedErr == nil {
				require.NoError(t, err)
				assert.IsType(t, tt.expectedBalancerType, balancer)
			} else {
				require.Nil(t, balancer)
				assert.EqualError(t, err, tt.expectedErr.Error())
			}
		})
	}
}
