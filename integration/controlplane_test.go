package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/ports"

	"github.com/aknEvrnky/pgway/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
)

// ---------------------------------------------------------------------------
// Proxy
// ---------------------------------------------------------------------------

func TestControlPlane_Proxy(t *testing.T) {
	t.Parallel()

	proxySpec := proxyv1.ProxySpecV1{
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     8080,
	}

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply creates new with ULID when name empty", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: ""}

			result, err := svc.ApplyProxyV1(ctx, meta, proxySpec)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Id, "ULID should be assigned")

			require.Len(t, spyPub.Events, 1)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeProxy,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[0])
		}},
		{"Apply creates new — timestamps set", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-proxy"}

			before := time.Now()
			result, err := svc.ApplyProxyV1(ctx, meta, proxySpec)
			require.NoError(t, err)
			assert.True(t, !result.CreatedAt.Before(before), "CreatedAt should be >= before")
			assert.True(t, !result.UpdatedAt.Before(before), "UpdatedAt should be >= before")
		}},
		{"Apply updates existing — CreatedAt preserved, UpdatedAt advances", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-proxy"}

			result, err := svc.ApplyProxyV1(ctx, meta, proxySpec)
			require.NoError(t, err)

			time.Sleep(time.Millisecond) // ensure UpdatedAt advances; 1ms sufficient on Linux/macOS

			result2, err := svc.ApplyProxyV1(ctx, meta, proxySpec)
			require.NoError(t, err)
			assert.True(t, result.CreatedAt.Equal(result2.CreatedAt), "CreatedAt must be preserved on update")
			assert.True(t, result2.UpdatedAt.After(result.UpdatedAt), "UpdatedAt must advance on update")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeProxy,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[0])
		}},
		{"GetProxy returns persisted proxy", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-proxy"}

			_, err := svc.ApplyProxyV1(ctx, meta, proxySpec)
			require.NoError(t, err)

			got, err := svc.GetProxy(ctx, "test-proxy")
			require.NoError(t, err)
			assert.Equal(t, "test-proxy", got.Id)
			assert.Equal(t, "127.0.0.1", got.Host)
			assert.Equal(t, uint16(8080), got.Port)
		}},
		{"ListProxies returns all applied proxies", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "proxy-a"}, proxySpec)
			require.NoError(t, err)
			_, err = svc.ApplyProxyV1(ctx, schema.Metadata{Name: "proxy-b"}, proxySpec)
			require.NoError(t, err)

			result, err := svc.ListProxies(ctx, domain.ListParams{}, domain.ProxyFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, p := range result.Items {
				ids[p.Id] = true
			}
			assert.True(t, ids["proxy-a"])
			assert.True(t, ids["proxy-b"])
		}},
		{"DeleteProxy removes proxy", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-proxy"}

			_, err := svc.ApplyProxyV1(ctx, meta, proxySpec)
			require.NoError(t, err)

			require.NoError(t, svc.DeleteProxy(ctx, "test-proxy"))

			_, err = svc.GetProxy(ctx, "test-proxy")
			assert.ErrorContains(t, err, "not found", "get after delete should return error")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           "test-proxy",
				ResourceType: ports.ResourceTypeProxy,
				ChangeKind:   ports.ChangeKindDeleted,
			}, spyPub.Events[1])
		}},
		{"Delete non-existent proxy returns error", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			err := svc.DeleteProxy(ctx, "ghost-proxy")
			assert.ErrorContains(t, err, "not found")
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

func TestControlPlane_Pool(t *testing.T) {
	t.Parallel()

	poolSpec := poolv1.PoolSpecV1{
		Title:   "test-pool",
		Type:    "static",
		Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
	}

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply creates new with ULID when name empty", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			result, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: ""}, poolSpec)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Id)

			require.Len(t, spyPub.Events, 1)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypePool,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[0])
		}},
		{"Apply creates new — timestamps set", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			before := time.Now()
			result, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "test-pool"}, poolSpec)
			require.NoError(t, err)
			assert.True(t, !result.CreatedAt.Before(before), "CreatedAt should be >= before")
			assert.True(t, !result.UpdatedAt.Before(before), "UpdatedAt should be >= before")
		}},
		{"Apply updates existing — CreatedAt preserved, UpdatedAt advances", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-pool"}

			result, err := svc.ApplyPoolV1(ctx, meta, poolSpec)
			require.NoError(t, err)

			time.Sleep(time.Millisecond) // ensure UpdatedAt advances; 1ms sufficient on Linux/macOS

			result2, err := svc.ApplyPoolV1(ctx, meta, poolSpec)
			require.NoError(t, err)
			assert.True(t, result.CreatedAt.Equal(result2.CreatedAt), "CreatedAt must be preserved on update")
			assert.True(t, result2.UpdatedAt.After(result.UpdatedAt), "UpdatedAt should advance after update")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypePool,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[0])
		}},
		{"GetPool returns persisted pool", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "test-pool"}, poolSpec)
			require.NoError(t, err)

			got, err := svc.GetPool(ctx, "test-pool")
			require.NoError(t, err)
			assert.Equal(t, "test-pool", got.Id)
			assert.Equal(t, "test-pool", got.Title)
		}},
		{"ListPools returns all applied pools", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-a"}, poolSpec)
			require.NoError(t, err)
			_, err = svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-b"}, poolSpec)
			require.NoError(t, err)

			result, err := svc.ListPools(ctx, domain.ListParams{}, domain.PoolFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, p := range result.Items {
				ids[p.Id] = true
			}
			assert.True(t, ids["pool-a"])
			assert.True(t, ids["pool-b"])
		}},
		{"DeletePool removes pool", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "test-pool"}, poolSpec)
			require.NoError(t, err)

			require.NoError(t, svc.DeletePool(ctx, "test-pool"))

			_, err = svc.GetPool(ctx, "test-pool")
			assert.ErrorContains(t, err, "not found", "get after delete should return error")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           "test-pool",
				ResourceType: ports.ResourceTypePool,
				ChangeKind:   ports.ChangeKindDeleted,
			}, spyPub.Events[1])
		}},
		{"Delete non-existent pool returns error", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			err := svc.DeletePool(ctx, "ghost-pool")
			assert.ErrorContains(t, err, "not found")
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

func TestControlPlane_Balancer(t *testing.T) {
	t.Parallel()

	lbSpec := balancerv1.BalancerSpecV1{
		Title:  "test-lb",
		Type:   "round-robin",
		PoolId: "pool-1",
	}

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply creates new with ULID when name empty", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			result, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: ""}, lbSpec)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Id)

			require.Len(t, spyPub.Events, 1)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeBalancer,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[0])
		}},
		{"Apply creates new — timestamps set", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			before := time.Now()
			result, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "test-lb"}, lbSpec)
			require.NoError(t, err)
			assert.True(t, !result.CreatedAt.Before(before), "CreatedAt should be >= before")
			assert.True(t, !result.UpdatedAt.Before(before), "UpdatedAt should be >= before")
		}},
		{"Apply updates existing — CreatedAt preserved, UpdatedAt advances", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-lb"}

			result, err := svc.ApplyBalancerV1(ctx, meta, lbSpec)
			require.NoError(t, err)

			time.Sleep(time.Millisecond) // ensure UpdatedAt advances; 1ms sufficient on Linux/macOS

			result2, err := svc.ApplyBalancerV1(ctx, meta, lbSpec)
			require.NoError(t, err)
			assert.True(t, result.CreatedAt.Equal(result2.CreatedAt), "CreatedAt must be preserved on update")
			assert.True(t, result2.UpdatedAt.After(result.UpdatedAt), "UpdatedAt should advance after update")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeBalancer,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[1])
		}},
		{"GetBalancer returns persisted balancer", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "test-lb"}, lbSpec)
			require.NoError(t, err)

			got, err := svc.GetBalancer(ctx, "test-lb")
			require.NoError(t, err)
			assert.Equal(t, "test-lb", got.Id)
			assert.Equal(t, "pool-1", got.PoolId)
		}},
		{"ListBalancers returns all applied balancers", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-a"}, lbSpec)
			require.NoError(t, err)
			_, err = svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-b"}, lbSpec)
			require.NoError(t, err)

			result, err := svc.ListBalancers(ctx, domain.ListParams{}, domain.BalancerFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, lb := range result.Items {
				ids[lb.Id] = true
			}
			assert.True(t, ids["lb-a"])
			assert.True(t, ids["lb-b"])
		}},
		{"DeleteBalancer removes balancer", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "test-lb"}, lbSpec)
			require.NoError(t, err)

			require.NoError(t, svc.DeleteBalancer(ctx, "test-lb"))

			_, err = svc.GetBalancer(ctx, "test-lb")
			assert.ErrorContains(t, err, "not found", "get after delete should return error")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           "test-lb",
				ResourceType: ports.ResourceTypeBalancer,
				ChangeKind:   ports.ChangeKindDeleted,
			}, spyPub.Events[1])
		}},
		{"Delete non-existent balancer returns error", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			err := svc.DeleteBalancer(ctx, "ghost-lb")
			assert.ErrorContains(t, err, "not found")
		}},
		{"weighted apply requires static pool", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "dyn-pool"}, poolv1.PoolSpecV1{
				Title:    "dyn",
				Type:     "dynamic",
				Selector: &poolv1.SelectorSpec{Allow: map[string]string{"env": "prod"}},
			})
			require.NoError(t, err)

			_, err = svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "w-lb"}, balancerv1.BalancerSpecV1{
				Title:  "weighted",
				Type:   "weighted",
				PoolId: "dyn-pool",
			})
			require.ErrorContains(t, err, "requires static pool")
		}},
		{"weighted apply succeeds for static pool", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "static-pool"}, poolv1.PoolSpecV1{
				Title:   "static",
				Type:    "static",
				Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
			})
			require.NoError(t, err)

			lb, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "w-lb"}, balancerv1.BalancerSpecV1{
				Title:  "weighted",
				Type:   "weighted",
				PoolId: "static-pool",
			})
			require.NoError(t, err)
			assert.Equal(t, domain.BalancerTypeWeighted, lb.Type)
		}},
		{"least-bytes apply stores reset_interval", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-lb"}, poolv1.PoolSpecV1{
				Title:   "static",
				Type:    "static",
				Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
			})
			require.NoError(t, err)

			lb, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-bytes"}, balancerv1.BalancerSpecV1{
				Title:         "least bytes",
				Type:          "least-bytes",
				PoolId:        "pool-lb",
				ResetInterval: "30s",
			})
			require.NoError(t, err)
			assert.Equal(t, domain.BalancerTypeLeastBytes, lb.Type)
			assert.Equal(t, 30*time.Second, lb.ResetInterval)
		}},
		{"least-bytes apply defaults reset_interval to 1m", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "pool-lb2"}, poolv1.PoolSpecV1{
				Title:   "static",
				Type:    "static",
				Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
			})
			require.NoError(t, err)

			lb, err := svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "lb-bytes-def"}, balancerv1.BalancerSpecV1{
				Title:  "least bytes",
				Type:   "least-bytes",
				PoolId: "pool-lb2",
			})
			require.NoError(t, err)
			assert.Equal(t, time.Minute, lb.ResetInterval)
		}},
		{"pool cannot become dynamic while weighted balancer references it", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyPoolV1(ctx, schema.Metadata{Name: "static-pool"}, poolv1.PoolSpecV1{
				Title:   "static",
				Type:    "static",
				Members: []poolv1.PoolMemberSpec{{ProxyId: "p1"}},
			})
			require.NoError(t, err)
			_, err = svc.ApplyBalancerV1(ctx, schema.Metadata{Name: "w-lb"}, balancerv1.BalancerSpecV1{
				Title:  "weighted",
				Type:   "weighted",
				PoolId: "static-pool",
			})
			require.NoError(t, err)

			_, err = svc.ApplyPoolV1(ctx, schema.Metadata{Name: "static-pool"}, poolv1.PoolSpecV1{
				Title:    "now-dyn",
				Type:     "dynamic",
				Selector: &poolv1.SelectorSpec{Allow: map[string]string{"env": "prod"}},
			})
			require.ErrorContains(t, err, "referenced by weighted balancer")
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

func TestControlPlane_Router(t *testing.T) {
	t.Parallel()

	routerSpec := routerv1.RouterSpecV1{
		Title:       "test-router",
		Description: "integration test router",
		Rules: []routerv1.RuleSpec{
			{
				Id:     "r1",
				Match:  routerv1.MatchSpec{Type: "catch_all"},
				Target: "lb-1",
			},
		},
	}

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply creates new with ULID when name empty", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			result, err := svc.ApplyRouterV1(ctx, schema.Metadata{Name: ""}, routerSpec)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Id)

			require.Len(t, spyPub.Events, 1)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeRouter,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[0])
		}},
		{"Apply creates new — timestamps set", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			before := time.Now()
			result, err := svc.ApplyRouterV1(ctx, schema.Metadata{Name: "test-router"}, routerSpec)
			require.NoError(t, err)
			assert.True(t, !result.CreatedAt.Before(before), "CreatedAt should be >= before")
			assert.True(t, !result.UpdatedAt.Before(before), "UpdatedAt should be >= before")
		}},
		{"Apply updates existing — CreatedAt preserved, UpdatedAt advances", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-router"}

			result, err := svc.ApplyRouterV1(ctx, meta, routerSpec)
			require.NoError(t, err)

			time.Sleep(time.Millisecond) // ensure UpdatedAt advances; 1ms sufficient on Linux/macOS

			result2, err := svc.ApplyRouterV1(ctx, meta, routerSpec)
			require.NoError(t, err)
			assert.True(t, result.CreatedAt.Equal(result2.CreatedAt), "CreatedAt must be preserved on update")
			assert.True(t, result2.UpdatedAt.After(result.UpdatedAt), "UpdatedAt should advance after update")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeRouter,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[1])
		}},
		{"GetRouter returns persisted router", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyRouterV1(ctx, schema.Metadata{Name: "test-router"}, routerSpec)
			require.NoError(t, err)

			got, err := svc.GetRouter(ctx, "test-router")
			require.NoError(t, err)
			assert.Equal(t, "test-router", got.Id)
			assert.Equal(t, "test-router", got.Title)
			assert.Len(t, got.Rules, 1)
		}},
		{"ListRouters returns all applied routers", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyRouterV1(ctx, schema.Metadata{Name: "router-a"}, routerSpec)
			require.NoError(t, err)
			_, err = svc.ApplyRouterV1(ctx, schema.Metadata{Name: "router-b"}, routerSpec)
			require.NoError(t, err)

			result, err := svc.ListRouters(ctx, domain.ListParams{}, domain.RouterFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, r := range result.Items {
				ids[r.Id] = true
			}
			assert.True(t, ids["router-a"])
			assert.True(t, ids["router-b"])
		}},
		{"DeleteRouter removes router", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyRouterV1(ctx, schema.Metadata{Name: "test-router"}, routerSpec)
			require.NoError(t, err)

			require.NoError(t, svc.DeleteRouter(ctx, "test-router"))

			_, err = svc.GetRouter(ctx, "test-router")
			assert.ErrorContains(t, err, "not found", "get after delete should return error")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           "test-router",
				ResourceType: ports.ResourceTypeRouter,
				ChangeKind:   ports.ChangeKindDeleted,
			}, spyPub.Events[1])
		}},
		{"Delete non-existent router returns error", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			err := svc.DeleteRouter(ctx, "ghost-router")
			assert.ErrorContains(t, err, "not found")
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

func TestControlPlane_Flow(t *testing.T) {
	t.Parallel()

	flowSpec := flowv1.FlowSpecV1{
		BalancerId: "lb-1",
	}

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply creates new with ULID when name empty", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			result, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: ""}, flowSpec)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Id)

			require.Len(t, spyPub.Events, 1)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeFlow,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[0])
		}},
		{"Apply creates new — timestamps set", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			before := time.Now()
			result, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "test-flow"}, flowSpec)
			require.NoError(t, err)
			assert.True(t, !result.CreatedAt.Before(before), "CreatedAt should be >= before")
			assert.True(t, !result.UpdatedAt.Before(before), "UpdatedAt should be >= before")
		}},
		{"Apply updates existing — CreatedAt preserved, UpdatedAt advances", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-flow"}

			result, err := svc.ApplyFlowV1(ctx, meta, flowSpec)
			require.NoError(t, err)

			time.Sleep(time.Millisecond) // ensure UpdatedAt advances; 1ms sufficient on Linux/macOS

			result2, err := svc.ApplyFlowV1(ctx, meta, flowSpec)
			require.NoError(t, err)
			assert.True(t, result.CreatedAt.Equal(result2.CreatedAt), "CreatedAt must be preserved on update")
			assert.True(t, result2.UpdatedAt.After(result.UpdatedAt), "UpdatedAt should advance after update")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeFlow,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[1])
		}},
		{"GetFlow returns persisted flow", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "test-flow"}, flowSpec)
			require.NoError(t, err)

			got, err := svc.GetFlow(ctx, "test-flow")
			require.NoError(t, err)
			assert.Equal(t, "test-flow", got.Id)
			assert.Equal(t, "lb-1", got.BalancerId)
		}},
		{"ListFlows returns all applied flows", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-a"}, flowSpec)
			require.NoError(t, err)
			_, err = svc.ApplyFlowV1(ctx, schema.Metadata{Name: "flow-b"}, flowSpec)
			require.NoError(t, err)

			result, err := svc.ListFlows(ctx, domain.ListParams{}, domain.FlowFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, f := range result.Items {
				ids[f.Id] = true
			}
			assert.True(t, ids["flow-a"])
			assert.True(t, ids["flow-b"])
		}},
		{"DeleteFlow removes flow", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyFlowV1(ctx, schema.Metadata{Name: "test-flow"}, flowSpec)
			require.NoError(t, err)

			require.NoError(t, svc.DeleteFlow(ctx, "test-flow"))

			_, err = svc.GetFlow(ctx, "test-flow")
			assert.ErrorContains(t, err, "not found", "get after delete should return error")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           "test-flow",
				ResourceType: ports.ResourceTypeFlow,
				ChangeKind:   ports.ChangeKindDeleted,
			}, spyPub.Events[1])
		}},
		{"Delete non-existent flow returns error", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			err := svc.DeleteFlow(ctx, "ghost-flow")
			assert.ErrorContains(t, err, "not found")
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

func TestControlPlane_Entrypoint(t *testing.T) {
	t.Parallel()

	epSpec := entrypointv1.EntrypointSpecV1{
		Title:    "test-ep",
		Protocol: "http",
		Host:     "0.0.0.0",
		Port:     9090,
		FlowId:   "flow-1",
	}

	for _, tt := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"Apply creates new with ULID when name empty", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			result, err := svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: ""}, epSpec)
			require.NoError(t, err)
			assert.NotEmpty(t, result.Id)

			require.Len(t, spyPub.Events, 1)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeEntrypoint,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[0])
		}},
		{"Apply creates new — timestamps set", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			before := time.Now()
			result, err := svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "test-ep"}, epSpec)
			require.NoError(t, err)
			assert.True(t, !result.CreatedAt.Before(before), "CreatedAt should be >= before")
			assert.True(t, !result.UpdatedAt.Before(before), "UpdatedAt should be >= before")
		}},
		{"Apply updates existing — CreatedAt preserved, UpdatedAt advances", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()
			meta := schema.Metadata{Name: "test-ep"}

			result, err := svc.ApplyEntrypointV1(ctx, meta, epSpec)
			require.NoError(t, err)

			time.Sleep(time.Millisecond) // ensure UpdatedAt advances; 1ms sufficient on Linux/macOS

			result2, err := svc.ApplyEntrypointV1(ctx, meta, epSpec)
			require.NoError(t, err)
			assert.True(t, result.CreatedAt.Equal(result2.CreatedAt), "CreatedAt must be preserved on update")
			assert.True(t, result2.UpdatedAt.After(result.UpdatedAt), "UpdatedAt should advance after update")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           result.Id,
				ResourceType: ports.ResourceTypeEntrypoint,
				ChangeKind:   ports.ChangeKindSaved,
			}, spyPub.Events[1])
		}},
		{"GetEntrypoint returns persisted entrypoint", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "test-ep"}, epSpec)
			require.NoError(t, err)

			got, err := svc.GetEntrypoint(ctx, "test-ep")
			require.NoError(t, err)
			assert.Equal(t, "test-ep", got.Id)
			assert.Equal(t, "0.0.0.0", got.Host)
			assert.Equal(t, uint16(9090), got.Port)
			assert.Equal(t, "flow-1", got.FlowId)
		}},
		{"ListEntrypoints returns all applied entrypoints", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "ep-a"}, epSpec)
			require.NoError(t, err)
			_, err = svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "ep-b"}, epSpec)
			require.NoError(t, err)

			result, err := svc.ListEntrypoints(ctx, domain.ListParams{}, domain.EntrypointFilter{})
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			ids := map[string]bool{}
			for _, ep := range result.Items {
				ids[ep.Id] = true
			}
			assert.True(t, ids["ep-a"])
			assert.True(t, ids["ep-b"])
		}},
		{"DeleteEntrypoint removes entrypoint", func(t *testing.T) {
			svc, spyPub := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			_, err := svc.ApplyEntrypointV1(ctx, schema.Metadata{Name: "test-ep"}, epSpec)
			require.NoError(t, err)

			require.NoError(t, svc.DeleteEntrypoint(ctx, "test-ep"))

			_, err = svc.GetEntrypoint(ctx, "test-ep")
			assert.ErrorContains(t, err, "not found", "get after delete should return error")

			require.Len(t, spyPub.Events, 2)
			assert.Equal(t, ports.ChangeEvent{
				ID:           "test-ep",
				ResourceType: ports.ResourceTypeEntrypoint,
				ChangeKind:   ports.ChangeKindDeleted,
			}, spyPub.Events[1])
		}},
		{"Delete non-existent entrypoint returns error", func(t *testing.T) {
			svc, _ := testutil.NewSvcWithPublisher(t)
			ctx := context.Background()

			err := svc.DeleteEntrypoint(ctx, "ghost-ep")
			assert.ErrorContains(t, err, "not found")
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

// ---------------------------------------------------------------------------
// Event publishing
// ---------------------------------------------------------------------------

func TestControlPlane_PublisherFailureDoesNotBreakWrites(t *testing.T) {
	t.Parallel()

	svc := testutil.NewSvc(t, testutil.FailingPublisher{})
	ctx := context.Background()

	proxySpec := proxyv1.ProxySpecV1{
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     8080,
	}

	_, err := svc.ApplyProxyV1(ctx, schema.Metadata{Name: "test-proxy"}, proxySpec)
	require.NoError(t, err, "apply must succeed even when event publish fails")

	require.NoError(t, svc.DeleteProxy(ctx, "test-proxy"),
		"delete must succeed even when event publish fails")
}
