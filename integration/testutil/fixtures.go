package testutil

import (
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// NewTestProxy returns a new valid HTTP proxy pointing at 127.0.0.1:8080.
func NewTestProxy() *domain.Proxy {
	return &domain.Proxy{
		Id:       "p1",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     8080,
	}
}

// NewTestPool returns a new static pool that references NewTestProxy.
func NewTestPool() *domain.Pool {
	return &domain.Pool{
		Id:       "pool-1",
		Title:    "test-pool",
		Type:     domain.PoolTypeStatic,
		ProxyIds: []string{"p1"},
	}
}

// NewTestLB returns a new round-robin load balancer that references NewTestPool.
func NewTestLB() *domain.LoadBalancer {
	return &domain.LoadBalancer{
		Id:     "lb-1",
		Title:  "test-lb",
		Type:   domain.BalancerTypeRoundRobin,
		PoolId: "pool-1",
	}
}

// NewTestRouter returns a new router with a single catch-all rule targeting NewTestLB.
func NewTestRouter() *domain.Router {
	return &domain.Router{
		Id:    "router-1",
		Title: "test-router",
		Rules: []*domain.RouterRule{
			{
				Id:     "r1",
				Match:  domain.RouterMatch{Type: domain.MatchTypeCatchAll},
				Target: "lb-1",
			},
		},
	}
}

// NewTestFlow returns a new flow that connects NewTestEntrypoint to NewTestLB.
// RouterId is intentionally empty; set it to NewTestRouter().Id for full routing path tests.
func NewTestFlow() *domain.Flow {
	return &domain.Flow{
		Id:         "flow-1",
		BalancerId: "lb-1",
	}
}

// NewTestEntrypoint returns a new HTTP entry point listening on 0.0.0.0:9090,
// associated with NewTestFlow.
func NewTestEntrypoint() *domain.Entrypoint {
	return &domain.Entrypoint{
		Id:       "ep-1",
		Title:    "test-entrypoint",
		Protocol: domain.ProtocolHTTP,
		Host:     "0.0.0.0",
		Port:     9090,
		FlowId:   "flow-1",
	}
}
