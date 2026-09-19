package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// recordingCP records Apply* routing for dispatcher tests. Unused reader/delete
// methods panic so accidental calls fail loudly.
type recordingCP struct {
	calls   []string
	failKey string
	failErr error
}

func (r *recordingCP) record(key string) error {
	r.calls = append(r.calls, key)
	if r.failKey != "" && r.failKey == key {
		if r.failErr != nil {
			return r.failErr
		}
		return errors.New("forced apply failure")
	}
	return nil
}

func (r *recordingCP) ApplyProxyV1(_ context.Context, meta schema.Metadata, _ proxyv1.ProxySpecV1) (*domain.Proxy, error) {
	if err := r.record("Proxy/v1"); err != nil {
		return nil, err
	}
	return &domain.Proxy{Id: meta.Name}, nil
}
func (r *recordingCP) ApplyPoolV1(_ context.Context, meta schema.Metadata, _ poolv1.PoolSpecV1) (*domain.Pool, error) {
	if err := r.record("Pool/v1"); err != nil {
		return nil, err
	}
	return &domain.Pool{Id: meta.Name}, nil
}
func (r *recordingCP) ApplyBalancerV1(_ context.Context, meta schema.Metadata, _ balancerv1.BalancerSpecV1) (*domain.LoadBalancer, error) {
	if err := r.record("LoadBalancer/v1"); err != nil {
		return nil, err
	}
	return &domain.LoadBalancer{Id: meta.Name}, nil
}
func (r *recordingCP) ApplyRouterV1(_ context.Context, meta schema.Metadata, _ routerv1.RouterSpecV1) (*domain.Router, error) {
	if err := r.record("Router/v1"); err != nil {
		return nil, err
	}
	return &domain.Router{Id: meta.Name}, nil
}
func (r *recordingCP) ApplyFlowV1(_ context.Context, meta schema.Metadata, _ flowv1.FlowSpecV1) (*domain.Flow, error) {
	if err := r.record("Flow/v1"); err != nil {
		return nil, err
	}
	return &domain.Flow{Id: meta.Name}, nil
}
func (r *recordingCP) ApplyEntrypointV1(_ context.Context, meta schema.Metadata, _ entrypointv1.EntrypointSpecV1) (*domain.Entrypoint, error) {
	if err := r.record("Entrypoint/v1"); err != nil {
		return nil, err
	}
	return &domain.Entrypoint{Id: meta.Name}, nil
}

func (r *recordingCP) GetProxy(context.Context, string) (*domain.Proxy, error) {
	panic("unexpected")
}
func (r *recordingCP) ListProxies(context.Context, domain.ListParams, domain.ProxyFilter) (domain.ListResult[domain.Proxy], error) {
	panic("unexpected")
}
func (r *recordingCP) DeleteProxy(context.Context, string) error { panic("unexpected") }
func (r *recordingCP) GetPool(context.Context, string) (*domain.Pool, error) {
	panic("unexpected")
}
func (r *recordingCP) ListPools(context.Context, domain.ListParams, domain.PoolFilter) (domain.ListResult[domain.Pool], error) {
	panic("unexpected")
}
func (r *recordingCP) DeletePool(context.Context, string) error { panic("unexpected") }
func (r *recordingCP) GetBalancer(context.Context, string) (*domain.LoadBalancer, error) {
	panic("unexpected")
}
func (r *recordingCP) ListBalancers(context.Context, domain.ListParams, domain.BalancerFilter) (domain.ListResult[domain.LoadBalancer], error) {
	panic("unexpected")
}
func (r *recordingCP) DeleteBalancer(context.Context, string) error { panic("unexpected") }
func (r *recordingCP) GetRouter(context.Context, string) (*domain.Router, error) {
	panic("unexpected")
}
func (r *recordingCP) ListRouters(context.Context, domain.ListParams, domain.RouterFilter) (domain.ListResult[domain.Router], error) {
	panic("unexpected")
}
func (r *recordingCP) DeleteRouter(context.Context, string) error { panic("unexpected") }
func (r *recordingCP) GetFlow(context.Context, string) (*domain.Flow, error) {
	panic("unexpected")
}
func (r *recordingCP) ListFlows(context.Context, domain.ListParams, domain.FlowFilter) (domain.ListResult[domain.Flow], error) {
	panic("unexpected")
}
func (r *recordingCP) DeleteFlow(context.Context, string) error { panic("unexpected") }
func (r *recordingCP) GetEntrypoint(context.Context, string) (*domain.Entrypoint, error) {
	panic("unexpected")
}
func (r *recordingCP) ListEntrypoints(context.Context, domain.ListParams, domain.EntrypointFilter) (domain.ListResult[domain.Entrypoint], error) {
	panic("unexpected")
}
func (r *recordingCP) DeleteEntrypoint(context.Context, string) error { panic("unexpected") }

func mustSpec(t *testing.T, v any) []byte {
	t.Helper()
	b, err := yaml.Marshal(v)
	require.NoError(t, err)
	return b
}

func TestDispatcher_Apply_RoutesByKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind    string
		version string
		spec    any
		wantKey string
	}{
		{"Proxy", "v1", proxyv1.ProxySpecV1{URL: "http://127.0.0.1:1"}, "Proxy/v1"},
		{"Pool", "v1", poolv1.PoolSpecV1{Type: "dynamic", Selector: &poolv1.SelectorSpec{Allow: map[string]string{"a": "b"}}}, "Pool/v1"},
		{"LoadBalancer", "v1", balancerv1.BalancerSpecV1{Type: "round-robin", PoolId: "pool-1"}, "LoadBalancer/v1"},
		{"Router", "v1", routerv1.RouterSpecV1{Rules: []routerv1.RuleSpec{{Id: "r1", Target: "lb", Match: routerv1.MatchSpec{Type: "host", Value: "x"}}}}, "Router/v1"},
		{"Flow", "v1", flowv1.FlowSpecV1{BalancerId: "lb-1"}, "Flow/v1"},
		{"Entrypoint", "v1", entrypointv1.EntrypointSpecV1{Protocol: "http", Host: "0.0.0.0", Port: 8080, FlowId: "f1"}, "Entrypoint/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.wantKey, func(t *testing.T) {
			t.Parallel()
			cp := &recordingCP{}
			d := NewDispatcher(cp)
			err := d.Apply(context.Background(), schema.RawResource{
				Kind:     tt.kind,
				Version:  tt.version,
				Metadata: schema.Metadata{Name: "res-1"},
				SpecRaw:  mustSpec(t, tt.spec),
			})
			require.NoError(t, err)
			assert.Equal(t, []string{tt.wantKey}, cp.calls)
		})
	}
}

func TestDispatcher_Apply_UnknownResource(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(&recordingCP{})
	err := d.Apply(context.Background(), schema.RawResource{
		Kind: "Widget", Version: "v1", Metadata: schema.Metadata{Name: "x"}, SpecRaw: []byte("{}"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource")
}

func TestDispatcher_Apply_DecodeError(t *testing.T) {
	t.Parallel()
	d := NewDispatcher(&recordingCP{})
	err := d.Apply(context.Background(), schema.RawResource{
		Kind: "Proxy", Version: "v1", Metadata: schema.Metadata{Name: "p"}, SpecRaw: []byte("url: [\n"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode proxy spec")
}

func TestDispatcher_ApplyAll_WrapsIndexOnError(t *testing.T) {
	t.Parallel()
	cp := &recordingCP{failKey: "Pool/v1"}
	d := NewDispatcher(cp)

	resources := []schema.RawResource{
		{
			Kind: "Proxy", Version: "v1", Metadata: schema.Metadata{Name: "p1"},
			SpecRaw: mustSpec(t, proxyv1.ProxySpecV1{URL: "http://127.0.0.1:1"}),
		},
		{
			Kind: "Pool", Version: "v1", Metadata: schema.Metadata{Name: "pool-1"},
			SpecRaw: mustSpec(t, poolv1.PoolSpecV1{Type: "dynamic", Selector: &poolv1.SelectorSpec{Allow: map[string]string{"k": "v"}}}),
		},
		{
			Kind: "Flow", Version: "v1", Metadata: schema.Metadata{Name: "f1"},
			SpecRaw: mustSpec(t, flowv1.FlowSpecV1{BalancerId: "lb"}),
		},
	}

	err := d.ApplyAll(context.Background(), resources)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource[1] (Pool/v1)")
	assert.Equal(t, []string{"Proxy/v1", "Pool/v1"}, cp.calls)
}

func TestDispatcher_ApplyAll_Success(t *testing.T) {
	t.Parallel()
	cp := &recordingCP{}
	d := NewDispatcher(cp)

	err := d.ApplyAll(context.Background(), []schema.RawResource{
		{
			Kind: "Proxy", Version: "v1", Metadata: schema.Metadata{Name: "p1"},
			SpecRaw: mustSpec(t, proxyv1.ProxySpecV1{URL: "http://127.0.0.1:1"}),
		},
		{
			Kind: "Flow", Version: "v1", Metadata: schema.Metadata{Name: "f1"},
			SpecRaw: mustSpec(t, flowv1.FlowSpecV1{BalancerId: "lb"}),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Proxy/v1", "Flow/v1"}, cp.calls)
}
