package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	controlplanev1 "github.com/aknEvrnky/pgway/gen/pgway/controlplane/v1"
	"github.com/aknEvrnky/pgway/integration/testutil"
	badgerutil "github.com/aknEvrnky/pgway/integration/testutil/badger"
	"github.com/aknEvrnky/pgway/internal/application/controlplane"
)

func newGrpcEnv(t *testing.T) *grpc.ClientConn {
	t.Helper()
	store := badgerutil.NewBadgerStore(t)
	svc := controlplane.NewService(store.Proxies, store.Pools, store.LBs, store.Routers, store.Flows, store.EPs)
	return testutil.NewTestGrpcServer(t, svc)
}

// ---------------------------------------------------------------------------
// Proxy
// ---------------------------------------------------------------------------

func TestGRPC_Proxy(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply→Get", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewProxyServiceClient(conn)
			ctx := context.Background()

			applyResp, err := client.ApplyProxyV1(ctx, &controlplanev1.ApplyProxyV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-proxy"},
				Spec: &controlplanev1.ProxySpecV1{
					Protocol: "http",
					Host:     "127.0.0.1",
					Port:     8080,
				},
			})
			require.NoError(t, err)
			require.NotNil(t, applyResp.Proxy)

			getResp, err := client.GetProxy(ctx, &controlplanev1.GetProxyRequest{Name: "test-proxy"})
			require.NoError(t, err)
			require.NotNil(t, getResp.Proxy)
			assert.Equal(t, applyResp.Proxy.Id, getResp.Proxy.Id)
			assert.Equal(t, "127.0.0.1", getResp.Proxy.Host)
			assert.Equal(t, uint32(8080), getResp.Proxy.Port)
		}},
		{"Apply→List", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewProxyServiceClient(conn)
			ctx := context.Background()

			r1, err := client.ApplyProxyV1(ctx, &controlplanev1.ApplyProxyV1Request{
				Metadata: &controlplanev1.Metadata{Name: "proxy-a"},
				Spec:     &controlplanev1.ProxySpecV1{Protocol: "http", Host: "10.0.0.1", Port: 3128},
			})
			require.NoError(t, err)

			r2, err := client.ApplyProxyV1(ctx, &controlplanev1.ApplyProxyV1Request{
				Metadata: &controlplanev1.Metadata{Name: "proxy-b"},
				Spec:     &controlplanev1.ProxySpecV1{Protocol: "http", Host: "10.0.0.2", Port: 3128},
			})
			require.NoError(t, err)

			listResp, err := client.ListProxies(ctx, &controlplanev1.ListProxiesRequest{})
			require.NoError(t, err)

			ids := make([]string, 0, len(listResp.Proxies))
			for _, p := range listResp.Proxies {
				ids = append(ids, p.Id)
			}
			assert.Contains(t, ids, r1.Proxy.Id)
			assert.Contains(t, ids, r2.Proxy.Id)
		}},
		{"Apply→Delete→Get returns NotFound", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewProxyServiceClient(conn)
			ctx := context.Background()

			_, err := client.ApplyProxyV1(ctx, &controlplanev1.ApplyProxyV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-proxy"},
				Spec:     &controlplanev1.ProxySpecV1{Protocol: "http", Host: "127.0.0.1", Port: 8080},
			})
			require.NoError(t, err)

			_, err = client.DeleteProxy(ctx, &controlplanev1.DeleteProxyRequest{Name: "test-proxy"})
			require.NoError(t, err)

			_, err = client.GetProxy(ctx, &controlplanev1.GetProxyRequest{Name: "test-proxy"})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code())
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Pool
// ---------------------------------------------------------------------------

func TestGRPC_Pool(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply→Get", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewPoolServiceClient(conn)
			ctx := context.Background()

			applyResp, err := client.ApplyPoolV1(ctx, &controlplanev1.ApplyPoolV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-pool"},
				Spec: &controlplanev1.PoolSpecV1{
					Title:    "Test Pool",
					Type:     "static",
					ProxyIds: []string{"p1"},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, applyResp.Pool)

			getResp, err := client.GetPool(ctx, &controlplanev1.GetPoolRequest{Name: "test-pool"})
			require.NoError(t, err)
			require.NotNil(t, getResp.Pool)
			assert.Equal(t, applyResp.Pool.Id, getResp.Pool.Id)
			assert.Equal(t, "Test Pool", getResp.Pool.Title)
		}},
		{"Apply→List", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewPoolServiceClient(conn)
			ctx := context.Background()

			r1, err := client.ApplyPoolV1(ctx, &controlplanev1.ApplyPoolV1Request{
				Metadata: &controlplanev1.Metadata{Name: "pool-a"},
				Spec:     &controlplanev1.PoolSpecV1{Title: "Pool A", Type: "static", ProxyIds: []string{"p1"}},
			})
			require.NoError(t, err)

			r2, err := client.ApplyPoolV1(ctx, &controlplanev1.ApplyPoolV1Request{
				Metadata: &controlplanev1.Metadata{Name: "pool-b"},
				Spec:     &controlplanev1.PoolSpecV1{Title: "Pool B", Type: "static", ProxyIds: []string{"p2"}},
			})
			require.NoError(t, err)

			listResp, err := client.ListPools(ctx, &controlplanev1.ListPoolsRequest{})
			require.NoError(t, err)

			ids := make([]string, 0, len(listResp.Pools))
			for _, p := range listResp.Pools {
				ids = append(ids, p.Id)
			}
			assert.Contains(t, ids, r1.Pool.Id)
			assert.Contains(t, ids, r2.Pool.Id)
		}},
		{"Apply→Delete→Get returns NotFound", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewPoolServiceClient(conn)
			ctx := context.Background()

			_, err := client.ApplyPoolV1(ctx, &controlplanev1.ApplyPoolV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-pool"},
				Spec:     &controlplanev1.PoolSpecV1{Title: "Test Pool", Type: "static", ProxyIds: []string{"p1"}},
			})
			require.NoError(t, err)

			_, err = client.DeletePool(ctx, &controlplanev1.DeletePoolRequest{Name: "test-pool"})
			require.NoError(t, err)

			_, err = client.GetPool(ctx, &controlplanev1.GetPoolRequest{Name: "test-pool"})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code())
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Balancer
// ---------------------------------------------------------------------------

func TestGRPC_Balancer(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply→Get", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewBalancerServiceClient(conn)
			ctx := context.Background()

			applyResp, err := client.ApplyBalancerV1(ctx, &controlplanev1.ApplyBalancerV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-balancer"},
				Spec: &controlplanev1.BalancerSpecV1{
					Title:  "Test Balancer",
					Type:   "round-robin",
					PoolId: "pool-1",
				},
			})
			require.NoError(t, err)
			require.NotNil(t, applyResp.Balancer)

			getResp, err := client.GetBalancer(ctx, &controlplanev1.GetBalancerRequest{Name: "test-balancer"})
			require.NoError(t, err)
			require.NotNil(t, getResp.Balancer)
			assert.Equal(t, applyResp.Balancer.Id, getResp.Balancer.Id)
			assert.Equal(t, "Test Balancer", getResp.Balancer.Title)
		}},
		{"Apply→List", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewBalancerServiceClient(conn)
			ctx := context.Background()

			r1, err := client.ApplyBalancerV1(ctx, &controlplanev1.ApplyBalancerV1Request{
				Metadata: &controlplanev1.Metadata{Name: "balancer-a"},
				Spec:     &controlplanev1.BalancerSpecV1{Title: "Balancer A", Type: "round-robin", PoolId: "pool-1"},
			})
			require.NoError(t, err)

			r2, err := client.ApplyBalancerV1(ctx, &controlplanev1.ApplyBalancerV1Request{
				Metadata: &controlplanev1.Metadata{Name: "balancer-b"},
				Spec:     &controlplanev1.BalancerSpecV1{Title: "Balancer B", Type: "weighted", PoolId: "pool-2"},
			})
			require.NoError(t, err)

			listResp, err := client.ListBalancers(ctx, &controlplanev1.ListBalancersRequest{})
			require.NoError(t, err)

			ids := make([]string, 0, len(listResp.Balancers))
			for _, b := range listResp.Balancers {
				ids = append(ids, b.Id)
			}
			assert.Contains(t, ids, r1.Balancer.Id)
			assert.Contains(t, ids, r2.Balancer.Id)
		}},
		{"Apply→Delete→Get returns NotFound", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewBalancerServiceClient(conn)
			ctx := context.Background()

			_, err := client.ApplyBalancerV1(ctx, &controlplanev1.ApplyBalancerV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-balancer"},
				Spec:     &controlplanev1.BalancerSpecV1{Title: "Test Balancer", Type: "round-robin", PoolId: "pool-1"},
			})
			require.NoError(t, err)

			_, err = client.DeleteBalancer(ctx, &controlplanev1.DeleteBalancerRequest{Name: "test-balancer"})
			require.NoError(t, err)

			_, err = client.GetBalancer(ctx, &controlplanev1.GetBalancerRequest{Name: "test-balancer"})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code())
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

func TestGRPC_Router(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply→Get", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewRouterServiceClient(conn)
			ctx := context.Background()

			applyResp, err := client.ApplyRouterV1(ctx, &controlplanev1.ApplyRouterV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-router"},
				Spec: &controlplanev1.RouterSpecV1{
					Title:       "Test Router",
					Description: "integration test router",
					Rules: []*controlplanev1.RuleSpec{
						{Id: "r1", Target: "lb-1", Match: &controlplanev1.MatchSpec{Type: "catch_all"}},
					},
				},
			})
			require.NoError(t, err)
			require.NotNil(t, applyResp.Router)

			getResp, err := client.GetRouter(ctx, &controlplanev1.GetRouterRequest{Name: "test-router"})
			require.NoError(t, err)
			require.NotNil(t, getResp.Router)
			assert.Equal(t, applyResp.Router.Id, getResp.Router.Id)
			assert.Equal(t, "Test Router", getResp.Router.Title)
		}},
		{"Apply→List", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewRouterServiceClient(conn)
			ctx := context.Background()

			catchAllRule := []*controlplanev1.RuleSpec{
				{Id: "r1", Target: "lb-1", Match: &controlplanev1.MatchSpec{Type: "catch_all"}},
			}

			r1, err := client.ApplyRouterV1(ctx, &controlplanev1.ApplyRouterV1Request{
				Metadata: &controlplanev1.Metadata{Name: "router-a"},
				Spec:     &controlplanev1.RouterSpecV1{Title: "Router A", Rules: catchAllRule},
			})
			require.NoError(t, err)

			r2, err := client.ApplyRouterV1(ctx, &controlplanev1.ApplyRouterV1Request{
				Metadata: &controlplanev1.Metadata{Name: "router-b"},
				Spec:     &controlplanev1.RouterSpecV1{Title: "Router B", Rules: catchAllRule},
			})
			require.NoError(t, err)

			listResp, err := client.ListRouters(ctx, &controlplanev1.ListRoutersRequest{})
			require.NoError(t, err)

			ids := make([]string, 0, len(listResp.Routers))
			for _, r := range listResp.Routers {
				ids = append(ids, r.Id)
			}
			assert.Contains(t, ids, r1.Router.Id)
			assert.Contains(t, ids, r2.Router.Id)
		}},
		{"Apply→Delete→Get returns NotFound", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewRouterServiceClient(conn)
			ctx := context.Background()

			_, err := client.ApplyRouterV1(ctx, &controlplanev1.ApplyRouterV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-router"},
				Spec: &controlplanev1.RouterSpecV1{
					Title: "Test Router",
					Rules: []*controlplanev1.RuleSpec{
						{Id: "r1", Target: "lb-1", Match: &controlplanev1.MatchSpec{Type: "catch_all"}},
					},
				},
			})
			require.NoError(t, err)

			_, err = client.DeleteRouter(ctx, &controlplanev1.DeleteRouterRequest{Name: "test-router"})
			require.NoError(t, err)

			_, err = client.GetRouter(ctx, &controlplanev1.GetRouterRequest{Name: "test-router"})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code())
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Flow
// ---------------------------------------------------------------------------

func TestGRPC_Flow(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply→Get", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewFlowServiceClient(conn)
			ctx := context.Background()

			applyResp, err := client.ApplyFlowV1(ctx, &controlplanev1.ApplyFlowV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-flow"},
				Spec: &controlplanev1.FlowSpecV1{
					BalancerId: "lb-1",
				},
			})
			require.NoError(t, err)
			require.NotNil(t, applyResp.Flow)

			getResp, err := client.GetFlow(ctx, &controlplanev1.GetFlowRequest{Name: "test-flow"})
			require.NoError(t, err)
			require.NotNil(t, getResp.Flow)
			assert.Equal(t, applyResp.Flow.Id, getResp.Flow.Id)
			assert.Equal(t, "lb-1", getResp.Flow.BalancerId)
		}},
		{"Apply→List", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewFlowServiceClient(conn)
			ctx := context.Background()

			r1, err := client.ApplyFlowV1(ctx, &controlplanev1.ApplyFlowV1Request{
				Metadata: &controlplanev1.Metadata{Name: "flow-a"},
				Spec:     &controlplanev1.FlowSpecV1{BalancerId: "lb-1"},
			})
			require.NoError(t, err)

			r2, err := client.ApplyFlowV1(ctx, &controlplanev1.ApplyFlowV1Request{
				Metadata: &controlplanev1.Metadata{Name: "flow-b"},
				Spec:     &controlplanev1.FlowSpecV1{BalancerId: "lb-2"},
			})
			require.NoError(t, err)

			listResp, err := client.ListFlows(ctx, &controlplanev1.ListFlowsRequest{})
			require.NoError(t, err)

			ids := make([]string, 0, len(listResp.Flows))
			for _, f := range listResp.Flows {
				ids = append(ids, f.Id)
			}
			assert.Contains(t, ids, r1.Flow.Id)
			assert.Contains(t, ids, r2.Flow.Id)
		}},
		{"Apply→Delete→Get returns NotFound", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewFlowServiceClient(conn)
			ctx := context.Background()

			_, err := client.ApplyFlowV1(ctx, &controlplanev1.ApplyFlowV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-flow"},
				Spec:     &controlplanev1.FlowSpecV1{BalancerId: "lb-1"},
			})
			require.NoError(t, err)

			_, err = client.DeleteFlow(ctx, &controlplanev1.DeleteFlowRequest{Name: "test-flow"})
			require.NoError(t, err)

			_, err = client.GetFlow(ctx, &controlplanev1.GetFlowRequest{Name: "test-flow"})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code())
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Entrypoint
// ---------------------------------------------------------------------------

func TestGRPC_Entrypoint(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply→Get", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewEntrypointServiceClient(conn)
			ctx := context.Background()

			applyResp, err := client.ApplyEntrypointV1(ctx, &controlplanev1.ApplyEntrypointV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-entrypoint"},
				Spec: &controlplanev1.EntrypointSpecV1{
					Title:    "Test Entrypoint",
					Protocol: "http",
					Host:     "0.0.0.0",
					Port:     9090,
					FlowId:   "flow-1",
				},
			})
			require.NoError(t, err)
			require.NotNil(t, applyResp.Entrypoint)

			getResp, err := client.GetEntrypoint(ctx, &controlplanev1.GetEntrypointRequest{Name: "test-entrypoint"})
			require.NoError(t, err)
			require.NotNil(t, getResp.Entrypoint)
			assert.Equal(t, applyResp.Entrypoint.Id, getResp.Entrypoint.Id)
			assert.Equal(t, "Test Entrypoint", getResp.Entrypoint.Title)
			assert.Equal(t, uint32(9090), getResp.Entrypoint.Port)
		}},
		{"Apply→List", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewEntrypointServiceClient(conn)
			ctx := context.Background()

			r1, err := client.ApplyEntrypointV1(ctx, &controlplanev1.ApplyEntrypointV1Request{
				Metadata: &controlplanev1.Metadata{Name: "ep-a"},
				Spec:     &controlplanev1.EntrypointSpecV1{Title: "EP A", Protocol: "http", Host: "0.0.0.0", Port: 9091, FlowId: "flow-1"},
			})
			require.NoError(t, err)

			r2, err := client.ApplyEntrypointV1(ctx, &controlplanev1.ApplyEntrypointV1Request{
				Metadata: &controlplanev1.Metadata{Name: "ep-b"},
				Spec:     &controlplanev1.EntrypointSpecV1{Title: "EP B", Protocol: "http", Host: "0.0.0.0", Port: 9092, FlowId: "flow-2"},
			})
			require.NoError(t, err)

			listResp, err := client.ListEntrypoints(ctx, &controlplanev1.ListEntrypointsRequest{})
			require.NoError(t, err)

			ids := make([]string, 0, len(listResp.Entrypoints))
			for _, ep := range listResp.Entrypoints {
				ids = append(ids, ep.Id)
			}
			assert.Contains(t, ids, r1.Entrypoint.Id)
			assert.Contains(t, ids, r2.Entrypoint.Id)
		}},
		{"Apply→Delete→Get returns NotFound", func(t *testing.T) {
			conn := newGrpcEnv(t)
			client := controlplanev1.NewEntrypointServiceClient(conn)
			ctx := context.Background()

			_, err := client.ApplyEntrypointV1(ctx, &controlplanev1.ApplyEntrypointV1Request{
				Metadata: &controlplanev1.Metadata{Name: "test-entrypoint"},
				Spec:     &controlplanev1.EntrypointSpecV1{Title: "Test EP", Protocol: "http", Host: "0.0.0.0", Port: 9090, FlowId: "flow-1"},
			})
			require.NoError(t, err)

			_, err = client.DeleteEntrypoint(ctx, &controlplanev1.DeleteEntrypointRequest{Name: "test-entrypoint"})
			require.NoError(t, err)

			_, err = client.GetEntrypoint(ctx, &controlplanev1.GetEntrypointRequest{Name: "test-entrypoint"})
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.NotFound, st.Code())
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}
