package dptest

import (
	"context"
	"fmt"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

// ControlPlane is a shared in-memory fake for dataplane tests.
type ControlPlane struct {
	Entrypoints []*domain.Entrypoint
	Flows       []*domain.Flow
	Routers     []*domain.Router
	Balancers   []*domain.LoadBalancer
	Pools       map[string]*domain.Pool
	Proxies     []*domain.Proxy

	EntrypointErr error
	FlowErr       error
	RouterErr     error
	BalancerErr   error
	PoolErr       error
	ProxyErr      error
}

// --- Entrypoints ---
func (m *ControlPlane) ListEntrypoints(_ context.Context, _ domain.ListParams, _ domain.EntrypointFilter) (domain.ListResult[domain.Entrypoint], error) {
	return domain.ListResult[domain.Entrypoint]{Items: m.Entrypoints}, m.EntrypointErr
}
func (m *ControlPlane) GetEntrypoint(_ context.Context, name string) (*domain.Entrypoint, error) {
	if m.EntrypointErr != nil {
		return nil, m.EntrypointErr
	}
	for _, ep := range m.Entrypoints {
		if ep.Id == name {
			return ep, nil
		}
	}
	return nil, fmt.Errorf("entrypoint %q not found", name)
}

// --- Flows ---
func (m *ControlPlane) ListFlows(_ context.Context, _ domain.ListParams, _ domain.FlowFilter) (domain.ListResult[domain.Flow], error) {
	return domain.ListResult[domain.Flow]{Items: m.Flows}, m.FlowErr
}
func (m *ControlPlane) GetFlow(_ context.Context, name string) (*domain.Flow, error) {
	if m.FlowErr != nil {
		return nil, m.FlowErr
	}
	for _, f := range m.Flows {
		if f.Id == name {
			return f, nil
		}
	}
	return nil, fmt.Errorf("flow %q not found", name)
}

// --- Routers ---
func (m *ControlPlane) ListRouters(_ context.Context, _ domain.ListParams, _ domain.RouterFilter) (domain.ListResult[domain.Router], error) {
	return domain.ListResult[domain.Router]{Items: m.Routers}, m.RouterErr
}
func (m *ControlPlane) GetRouter(_ context.Context, name string) (*domain.Router, error) {
	if m.RouterErr != nil {
		return nil, m.RouterErr
	}
	for _, r := range m.Routers {
		if r.Id == name {
			return r, nil
		}
	}
	return nil, fmt.Errorf("router %q not found", name)
}

// --- Balancers ---
func (m *ControlPlane) ListBalancers(_ context.Context, _ domain.ListParams, _ domain.BalancerFilter) (domain.ListResult[domain.LoadBalancer], error) {
	return domain.ListResult[domain.LoadBalancer]{Items: m.Balancers}, m.BalancerErr
}
func (m *ControlPlane) GetBalancer(_ context.Context, name string) (*domain.LoadBalancer, error) {
	if m.BalancerErr != nil {
		return nil, m.BalancerErr
	}
	for _, lb := range m.Balancers {
		if lb.Id == name {
			return lb, nil
		}
	}
	return nil, fmt.Errorf("load balancer %q not found", name)
}

// --- Pools ---
func (m *ControlPlane) ListPools(_ context.Context, _ domain.ListParams, _ domain.PoolFilter) (domain.ListResult[domain.Pool], error) {
	result := make([]*domain.Pool, 0, len(m.Pools))
	for _, p := range m.Pools {
		result = append(result, p)
	}
	return domain.ListResult[domain.Pool]{Items: result}, m.PoolErr
}
func (m *ControlPlane) GetPool(_ context.Context, name string) (*domain.Pool, error) {
	if m.PoolErr != nil {
		return nil, m.PoolErr
	}
	p, ok := m.Pools[name]
	if !ok {
		return nil, fmt.Errorf("pool %q not found", name)
	}
	return p, nil
}

// --- Proxies ---
func (m *ControlPlane) ListProxies(_ context.Context, _ domain.ListParams, _ domain.ProxyFilter) (domain.ListResult[domain.Proxy], error) {
	return domain.ListResult[domain.Proxy]{Items: m.Proxies}, m.ProxyErr
}
func (m *ControlPlane) GetProxy(_ context.Context, name string) (*domain.Proxy, error) {
	if m.ProxyErr != nil {
		return nil, m.ProxyErr
	}
	for _, p := range m.Proxies {
		if p.Id == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("proxy %q not found", name)
}
func (m *ControlPlane) GetProxiesByIds(_ context.Context, ids []string) ([]*domain.Proxy, error) {
	if m.ProxyErr != nil {
		return nil, m.ProxyErr
	}
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	var result []*domain.Proxy
	for _, p := range m.Proxies {
		if _, ok := idSet[p.Id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}
func (m *ControlPlane) FindProxiesByLabels(_ context.Context, labels map[string]string) ([]*domain.Proxy, error) {
	if m.ProxyErr != nil {
		return nil, m.ProxyErr
	}
	var result []*domain.Proxy
outer:
	for _, p := range m.Proxies {
		for k, v := range labels {
			if p.Labels[k] != v {
				continue outer
			}
		}
		result = append(result, p)
	}
	return result, nil
}

// --- fixtures ---
