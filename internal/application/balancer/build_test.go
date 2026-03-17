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
	for _, tt := range []struct {
		name                 string
		lb                   *domain.LoadBalancer
		p                    *domain.Pool
		expectedBalancerType interface{}
		expectedErr          error
	}{
		{
			name: "it builds round robin load balancer",
			p: &domain.Pool{
				Id: "pool-1",
				Proxies: []*domain.Proxy{
					{
						Id: "proxy-1",
					},
				},
			},
			lb: &domain.LoadBalancer{
				Id:     "lb-1",
				Type:   "round-robin",
				PoolId: "pool-1",
			},
			expectedBalancerType: &algorithm.RoundRobin{},
			expectedErr:          nil,
		},
		{
			name: "it can not build unknown lb type",
			p: &domain.Pool{
				Id: "pool-1",
				Proxies: []*domain.Proxy{
					{
						Id: "proxy-1",
					},
				},
			},
			lb: &domain.LoadBalancer{
				Id:     "lb-1",
				Type:   "unknown",
				PoolId: "pool-1",
			},
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
