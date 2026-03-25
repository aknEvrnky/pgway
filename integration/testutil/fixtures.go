package testutil

import (
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// TestProxy is a valid HTTP proxy pointing at 127.0.0.1:8080.
var TestProxy = &domain.Proxy{
	Id:       "p1",
	Protocol: "http",
	Host:     "127.0.0.1",
	Port:     8080,
}

// TestPool is a static pool that references TestProxy.
var TestPool = &domain.Pool{
	Id:       "pool-1",
	Type:     domain.PoolTypeStatic,
	ProxyIds: []string{"p1"},
}

// TestLB is a round-robin load balancer that references TestPool.
var TestLB = &domain.LoadBalancer{
	Id:     "lb-1",
	Type:   domain.BalancerTypeRoundRobin,
	PoolId: "pool-1",
}

// TestRouter is a router with a single catch-all rule targeting TestLB.
var TestRouter = &domain.Router{
	Id: "router-1",
	Rules: []*domain.RouterRule{
		{
			Id:     "r1",
			Match:  domain.RouterMatch{Type: domain.MatchTypeCatchAll},
			Target: "lb-1",
		},
	},
}

// TestFlow is a flow that connects TestEntrypoint to TestLB.
var TestFlow = &domain.Flow{
	Id:         "flow-1",
	BalancerId: "lb-1",
}

// TestEntrypoint is an HTTP entry point listening on 0.0.0.0:9090,
// associated with TestFlow.
var TestEntrypoint = &domain.Entrypoint{
	Id:       "ep-1",
	Protocol: domain.ProtocolHTTP,
	Host:     "0.0.0.0",
	Port:     9090,
	FlowId:   "flow-1",
}
