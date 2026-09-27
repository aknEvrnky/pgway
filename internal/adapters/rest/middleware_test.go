package rest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aknEvrnky/pgway/internal/adapters/rest"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/aknEvrnky/pgway/internal/schema"
	balancerv1 "github.com/aknEvrnky/pgway/internal/schema/balancer/v1"
	entrypointv1 "github.com/aknEvrnky/pgway/internal/schema/entrypoint/v1"
	flowv1 "github.com/aknEvrnky/pgway/internal/schema/flow/v1"
	poolv1 "github.com/aknEvrnky/pgway/internal/schema/pool/v1"
	proxyv1 "github.com/aknEvrnky/pgway/internal/schema/proxy/v1"
	routerv1 "github.com/aknEvrnky/pgway/internal/schema/router/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuth struct {
	principal *domain.Principal
	err       error
}

func (f *fakeAuth) Authenticate(_ context.Context, token string) (*domain.Principal, error) {
	if f.err != nil {
		return nil, f.err
	}
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}
	return f.principal, nil
}

type fakeCP struct {
	proxy       *domain.Proxy
	proxies     []*domain.Proxy
	pool        *domain.Pool
	pools       []*domain.Pool
	balancer    *domain.LoadBalancer
	balancers   []*domain.LoadBalancer
	router      *domain.Router
	routers     []*domain.Router
	flow        *domain.Flow
	flows       []*domain.Flow
	entrypoint  *domain.Entrypoint
	entrypoints []*domain.Entrypoint
	deleteErr   error
	applyErr    error
}

func (f *fakeCP) GetProxy(_ context.Context, name string) (*domain.Proxy, error) {
	if f.proxy != nil && f.proxy.Id == name {
		cp := *f.proxy
		if f.proxy.Auth != nil {
			auth := *f.proxy.Auth
			cp.Auth = &auth
		}
		return &cp, nil
	}
	return nil, fmt.Errorf("proxy %q not found", name)
}
func (f *fakeCP) ListProxies(_ context.Context, params domain.ListParams, filter domain.ProxyFilter) (domain.ListResult[domain.Proxy], error) {
	all := f.proxies
	if len(all) == 0 && f.proxy != nil {
		all = []*domain.Proxy{f.proxy}
	}

	matched := make([]*domain.Proxy, 0, len(all))
	for _, p := range all {
		if filter.Protocol != "" && string(p.Protocol) != filter.Protocol {
			continue
		}
		if filter.Search != "" {
			q := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(p.Id), q) && !strings.Contains(strings.ToLower(p.Host), q) {
				continue
			}
		}
		matched = append(matched, p)
	}

	result := domain.ListResult[domain.Proxy]{TotalCount: len(matched)}
	start := 0
	if params.Cursor != "" {
		for i, p := range matched {
			if p.Id == params.Cursor {
				start = i
				break
			}
		}
	}
	if params.PageSize <= 0 {
		result.Items = matched[start:]
		return result, nil
	}
	end := start + params.PageSize
	if end > len(matched) {
		end = len(matched)
	}
	result.Items = matched[start:end]
	if end < len(matched) {
		result.NextCursor = matched[end].Id
	}
	return result, nil
}
func (f *fakeCP) ApplyProxyV1(_ context.Context, _ schema.Metadata, _ proxyv1.ProxySpecV1) (*domain.Proxy, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return f.proxy, nil
}
func (f *fakeCP) DeleteProxy(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeCP) GetPool(_ context.Context, name string) (*domain.Pool, error) {
	if f.pool != nil && f.pool.Id == name {
		cp := *f.pool
		return &cp, nil
	}
	return nil, fmt.Errorf("pool %q not found", name)
}
func (f *fakeCP) ListPools(_ context.Context, params domain.ListParams, filter domain.PoolFilter) (domain.ListResult[domain.Pool], error) {
	all := f.pools
	if len(all) == 0 && f.pool != nil {
		all = []*domain.Pool{f.pool}
	}

	matched := make([]*domain.Pool, 0, len(all))
	for _, p := range all {
		if filter.Type != "" && string(p.Type) != filter.Type {
			continue
		}
		if filter.Search != "" {
			q := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(p.Id), q) && !strings.Contains(strings.ToLower(p.Title), q) {
				continue
			}
		}
		matched = append(matched, p)
	}

	result := domain.ListResult[domain.Pool]{TotalCount: len(matched)}
	start := 0
	if params.Cursor != "" {
		for i, p := range matched {
			if p.Id == params.Cursor {
				start = i
				break
			}
		}
	}
	if params.PageSize <= 0 {
		result.Items = matched[start:]
		return result, nil
	}
	end := start + params.PageSize
	if end > len(matched) {
		end = len(matched)
	}
	result.Items = matched[start:end]
	if end < len(matched) {
		result.NextCursor = matched[end].Id
	}
	return result, nil
}
func (f *fakeCP) ApplyPoolV1(_ context.Context, _ schema.Metadata, _ poolv1.PoolSpecV1) (*domain.Pool, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return f.pool, nil
}
func (f *fakeCP) DeletePool(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeCP) GetBalancer(_ context.Context, name string) (*domain.LoadBalancer, error) {
	if f.balancer != nil && f.balancer.Id == name {
		cp := *f.balancer
		return &cp, nil
	}
	return nil, fmt.Errorf("balancer %q not found", name)
}
func (f *fakeCP) ListBalancers(_ context.Context, params domain.ListParams, filter domain.BalancerFilter) (domain.ListResult[domain.LoadBalancer], error) {
	all := f.balancers
	if len(all) == 0 && f.balancer != nil {
		all = []*domain.LoadBalancer{f.balancer}
	}

	matched := make([]*domain.LoadBalancer, 0, len(all))
	for _, lb := range all {
		if filter.Type != "" && string(lb.Type) != filter.Type {
			continue
		}
		if filter.PoolId != "" && lb.PoolId != filter.PoolId {
			continue
		}
		if filter.Search != "" {
			q := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(lb.Id), q) &&
				!strings.Contains(strings.ToLower(lb.Title), q) &&
				!strings.Contains(strings.ToLower(string(lb.Type)), q) &&
				!strings.Contains(strings.ToLower(lb.PoolId), q) {
				continue
			}
		}
		matched = append(matched, lb)
	}

	result := domain.ListResult[domain.LoadBalancer]{TotalCount: len(matched)}
	start := 0
	if params.Cursor != "" {
		for i, lb := range matched {
			if lb.Id == params.Cursor {
				start = i
				break
			}
		}
	}
	if params.PageSize <= 0 {
		result.Items = matched[start:]
		return result, nil
	}
	end := start + params.PageSize
	if end > len(matched) {
		end = len(matched)
	}
	result.Items = matched[start:end]
	if end < len(matched) {
		result.NextCursor = matched[end].Id
	}
	return result, nil
}
func (f *fakeCP) ApplyBalancerV1(_ context.Context, _ schema.Metadata, _ balancerv1.BalancerSpecV1) (*domain.LoadBalancer, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return f.balancer, nil
}
func (f *fakeCP) DeleteBalancer(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeCP) GetRouter(_ context.Context, name string) (*domain.Router, error) {
	if f.router != nil && f.router.Id == name {
		cp := *f.router
		return &cp, nil
	}
	return nil, fmt.Errorf("router %q not found", name)
}
func (f *fakeCP) ListRouters(_ context.Context, params domain.ListParams, filter domain.RouterFilter) (domain.ListResult[domain.Router], error) {
	all := f.routers
	if len(all) == 0 && f.router != nil {
		all = []*domain.Router{f.router}
	}

	matched := make([]*domain.Router, 0, len(all))
	for _, rt := range all {
		if filter.TargetBalancerId != "" {
			found := false
			for _, rule := range rt.Rules {
				if rule != nil && rule.Target == filter.TargetBalancerId {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if filter.HasCatchAll != nil {
			has := false
			for _, rule := range rt.Rules {
				if rule != nil && rule.Match.Type == domain.MatchTypeCatchAll {
					has = true
					break
				}
			}
			if *filter.HasCatchAll != has {
				continue
			}
		}
		if filter.Search != "" {
			q := strings.ToLower(filter.Search)
			hit := strings.Contains(strings.ToLower(rt.Id), q) || strings.Contains(strings.ToLower(rt.Title), q)
			if !hit {
				for _, rule := range rt.Rules {
					if rule == nil {
						continue
					}
					if strings.Contains(strings.ToLower(rule.Target), q) || strings.Contains(strings.ToLower(rule.Id), q) {
						hit = true
						break
					}
				}
			}
			if !hit {
				continue
			}
		}
		matched = append(matched, rt)
	}

	result := domain.ListResult[domain.Router]{TotalCount: len(matched)}
	start := 0
	if params.Cursor != "" {
		for i, rt := range matched {
			if rt.Id == params.Cursor {
				start = i
				break
			}
		}
	}
	if params.PageSize <= 0 {
		result.Items = matched[start:]
		return result, nil
	}
	end := start + params.PageSize
	if end > len(matched) {
		end = len(matched)
	}
	result.Items = matched[start:end]
	if end < len(matched) {
		result.NextCursor = matched[end].Id
	}
	return result, nil
}
func (f *fakeCP) ApplyRouterV1(_ context.Context, _ schema.Metadata, _ routerv1.RouterSpecV1) (*domain.Router, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return f.router, nil
}
func (f *fakeCP) DeleteRouter(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeCP) GetFlow(_ context.Context, name string) (*domain.Flow, error) {
	if f.flow != nil && f.flow.Id == name {
		cp := *f.flow
		return &cp, nil
	}
	return nil, fmt.Errorf("flow %q not found", name)
}
func (f *fakeCP) ListFlows(_ context.Context, params domain.ListParams, filter domain.FlowFilter) (domain.ListResult[domain.Flow], error) {
	all := f.flows
	if len(all) == 0 && f.flow != nil {
		all = []*domain.Flow{f.flow}
	}

	matched := make([]*domain.Flow, 0, len(all))
	for _, fl := range all {
		if filter.RouterId != "" && fl.RouterId != filter.RouterId {
			continue
		}
		if filter.BalancerId != "" && fl.BalancerId != filter.BalancerId {
			continue
		}
		switch filter.Mode {
		case "router":
			if fl.RouterId == "" {
				continue
			}
		case "direct":
			if fl.RouterId != "" || fl.BalancerId == "" {
				continue
			}
		}
		if filter.Search != "" {
			q := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(fl.Id), q) &&
				!strings.Contains(strings.ToLower(fl.RouterId), q) &&
				!strings.Contains(strings.ToLower(fl.BalancerId), q) {
				continue
			}
		}
		matched = append(matched, fl)
	}

	result := domain.ListResult[domain.Flow]{TotalCount: len(matched)}
	start := 0
	if params.Cursor != "" {
		for i, fl := range matched {
			if fl.Id == params.Cursor {
				start = i
				break
			}
		}
	}
	if params.PageSize <= 0 {
		result.Items = matched[start:]
		return result, nil
	}
	end := start + params.PageSize
	if end > len(matched) {
		end = len(matched)
	}
	result.Items = matched[start:end]
	if end < len(matched) {
		result.NextCursor = matched[end].Id
	}
	return result, nil
}
func (f *fakeCP) ApplyFlowV1(context.Context, schema.Metadata, flowv1.FlowSpecV1) (*domain.Flow, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return f.flow, nil
}
func (f *fakeCP) DeleteFlow(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeCP) GetEntrypoint(_ context.Context, name string) (*domain.Entrypoint, error) {
	if f.entrypoint != nil && f.entrypoint.Id == name {
		cp := *f.entrypoint
		return &cp, nil
	}
	return nil, fmt.Errorf("entrypoint %q not found", name)
}
func (f *fakeCP) ListEntrypoints(_ context.Context, params domain.ListParams, filter domain.EntrypointFilter) (domain.ListResult[domain.Entrypoint], error) {
	all := f.entrypoints
	if len(all) == 0 && f.entrypoint != nil {
		all = []*domain.Entrypoint{f.entrypoint}
	}
	matched := make([]*domain.Entrypoint, 0, len(all))
	for _, ep := range all {
		if filter.FlowId != "" && ep.FlowId != filter.FlowId {
			continue
		}
		if filter.Protocol != "" && string(ep.Protocol) != filter.Protocol {
			continue
		}
		if filter.Host != "" && !strings.Contains(strings.ToLower(ep.Host), strings.ToLower(filter.Host)) {
			continue
		}
		if filter.Search != "" {
			q := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(ep.Id), q) && !strings.Contains(strings.ToLower(ep.Title), q) {
				continue
			}
		}
		matched = append(matched, ep)
	}
	result := domain.ListResult[domain.Entrypoint]{TotalCount: len(matched)}
	start := 0
	if params.Cursor != "" {
		for i, ep := range matched {
			if ep.Id == params.Cursor {
				start = i
				break
			}
		}
	}
	if params.PageSize <= 0 {
		result.Items = matched[start:]
		return result, nil
	}
	end := start + params.PageSize
	if end > len(matched) {
		end = len(matched)
	}
	result.Items = matched[start:end]
	if end < len(matched) {
		result.NextCursor = matched[end].Id
	}
	return result, nil
}
func (f *fakeCP) ApplyEntrypointV1(_ context.Context, _ schema.Metadata, _ entrypointv1.EntrypointSpecV1) (*domain.Entrypoint, error) {
	if f.applyErr != nil {
		return nil, f.applyErr
	}
	return f.entrypoint, nil
}
func (f *fakeCP) DeleteEntrypoint(_ context.Context, _ string) error {
	return f.deleteErr
}

var _ ports.ControlPlane = (*fakeCP)(nil)
var _ ports.TokenAuthenticator = (*fakeAuth)(nil)

func userPrincipal() *domain.Principal {
	return &domain.Principal{User: &domain.User{Id: "alice", Role: domain.RoleAdmin}}
}

func testAdapter(t *testing.T, cfg config.RestConfig, auth *fakeAuth, authMgr ports.AuthManager, cp ports.ControlPlane, usersOpt ...ports.UserManager) http.Handler {
	t.Helper()
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:0"
	}
	if cfg.CORSAllowOrigins == nil {
		cfg.CORSAllowOrigins = []string{}
	}
	var users ports.UserManager
	if len(usersOpt) > 0 {
		users = usersOpt[0]
	}
	a := rest.NewRestAdapter(cp, auth, authMgr, users, cfg)
	return a.Handler()
}

func TestAuth_MissingBearer(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{err: fmt.Errorf("bad")}, nil, &fakeCP{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
	req.Header.Set("Authorization", "Bearer bad")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuth_AgentForbidden(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{
		principal: &domain.Principal{Agent: &domain.Agent{Id: "edge-1"}},
	}, nil, &fakeCP{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
	req.Header.Set("Authorization", "Bearer agent-tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestAuth_AllowUser(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{
		proxy: &domain.Proxy{Id: "p1", Host: "1.2.3.4", Port: 8080},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCORS_Allowlist(t *testing.T) {
	h := testAdapter(t, config.RestConfig{
		RateLimitRPS:     0,
		CORSAllowOrigins: []string{"http://localhost:3000"},
	}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/proxies", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "http://localhost:3000", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("denied origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/proxies", nil)
		req.Header.Set("Origin", "https://evil.example")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestRateLimit_PreAuthIPBruteForce(t *testing.T) {
	h := testAdapter(t, config.RestConfig{
		RateLimitRPS:   1,
		RateLimitBurst: 1,
	}, &fakeAuth{err: fmt.Errorf("bad")}, nil, &fakeCP{})

	do := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
		req.Header.Set("Authorization", "Bearer guess")
		req.RemoteAddr = "203.0.113.10:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusUnauthorized, do())
	assert.Equal(t, http.StatusTooManyRequests, do())
}

func TestRateLimit_PostAuthUserBucket(t *testing.T) {
	h := testAdapter(t, config.RestConfig{
		RateLimitRPS:   1,
		RateLimitBurst: 1,
	}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	// Distinct client IPs with fresh IP buckets: the shared per-user bucket
	// must still cap them.
	do := func(remoteAddr string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
		req.Header.Set("Authorization", "Bearer ok")
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, do("203.0.113.1:1000"))
	assert.Equal(t, http.StatusTooManyRequests, do("203.0.113.2:1000"))
}

func TestGetProxy_RedactsPassword(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{
		proxy: &domain.Proxy{
			Id:   "secure",
			Host: "10.0.0.1",
			Port: 3128,
			Auth: &domain.BasicAuth{User: "u", Pass: "s3cret"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies/secure", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var got domain.Proxy
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.NotNil(t, got.Auth)
	assert.Equal(t, "u", got.Auth.User)
	assert.Empty(t, got.Auth.Pass)
}

func TestRecovery_NoStackInBody(t *testing.T) {
	// Panic inside handler after auth via a CP that panics on ListProxies.
	cp := &panicCP{}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "panic")
	assert.Contains(t, rec.Body.String(), "internal server error")
}

// panicCP embeds fakeCP zero value but panics on list.
type panicCP struct{ fakeCP }

func (p *panicCP) ListProxies(context.Context, domain.ListParams, domain.ProxyFilter) (domain.ListResult[domain.Proxy], error) {
	panic("boom")
}
