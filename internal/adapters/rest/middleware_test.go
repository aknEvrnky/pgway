package rest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	proxy *domain.Proxy
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
func (f *fakeCP) ListProxies(_ context.Context, _ domain.ListParams, _ domain.ProxyFilter) (domain.ListResult[domain.Proxy], error) {
	if f.proxy == nil {
		return domain.ListResult[domain.Proxy]{}, nil
	}
	return domain.ListResult[domain.Proxy]{Items: []*domain.Proxy{f.proxy}}, nil
}
func (f *fakeCP) ApplyProxyV1(_ context.Context, _ schema.Metadata, _ proxyv1.ProxySpecV1) (*domain.Proxy, error) {
	return f.proxy, nil
}
func (f *fakeCP) DeleteProxy(_ context.Context, _ string) error { return nil }

func (f *fakeCP) GetPool(context.Context, string) (*domain.Pool, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) ListPools(context.Context, domain.ListParams, domain.PoolFilter) (domain.ListResult[domain.Pool], error) {
	return domain.ListResult[domain.Pool]{}, nil
}
func (f *fakeCP) ApplyPoolV1(context.Context, schema.Metadata, poolv1.PoolSpecV1) (*domain.Pool, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) DeletePool(context.Context, string) error { return fmt.Errorf("n/a") }

func (f *fakeCP) GetBalancer(context.Context, string) (*domain.LoadBalancer, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) ListBalancers(context.Context, domain.ListParams, domain.BalancerFilter) (domain.ListResult[domain.LoadBalancer], error) {
	return domain.ListResult[domain.LoadBalancer]{}, nil
}
func (f *fakeCP) ApplyBalancerV1(context.Context, schema.Metadata, balancerv1.BalancerSpecV1) (*domain.LoadBalancer, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) DeleteBalancer(context.Context, string) error { return fmt.Errorf("n/a") }

func (f *fakeCP) GetRouter(context.Context, string) (*domain.Router, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) ListRouters(context.Context, domain.ListParams, domain.RouterFilter) (domain.ListResult[domain.Router], error) {
	return domain.ListResult[domain.Router]{}, nil
}
func (f *fakeCP) ApplyRouterV1(context.Context, schema.Metadata, routerv1.RouterSpecV1) (*domain.Router, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) DeleteRouter(context.Context, string) error { return fmt.Errorf("n/a") }

func (f *fakeCP) GetFlow(context.Context, string) (*domain.Flow, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) ListFlows(context.Context, domain.ListParams, domain.FlowFilter) (domain.ListResult[domain.Flow], error) {
	return domain.ListResult[domain.Flow]{}, nil
}
func (f *fakeCP) ApplyFlowV1(context.Context, schema.Metadata, flowv1.FlowSpecV1) (*domain.Flow, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) DeleteFlow(context.Context, string) error { return fmt.Errorf("n/a") }

func (f *fakeCP) GetEntrypoint(context.Context, string) (*domain.Entrypoint, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) ListEntrypoints(context.Context, domain.ListParams, domain.EntrypointFilter) (domain.ListResult[domain.Entrypoint], error) {
	return domain.ListResult[domain.Entrypoint]{}, nil
}
func (f *fakeCP) ApplyEntrypointV1(context.Context, schema.Metadata, entrypointv1.EntrypointSpecV1) (*domain.Entrypoint, error) {
	return nil, fmt.Errorf("n/a")
}
func (f *fakeCP) DeleteEntrypoint(context.Context, string) error { return fmt.Errorf("n/a") }

var _ ports.ControlPlane = (*fakeCP)(nil)
var _ ports.TokenAuthenticator = (*fakeAuth)(nil)

func userPrincipal() *domain.Principal {
	return &domain.Principal{User: &domain.User{Id: "alice", Role: domain.RoleAdmin}}
}

func testAdapter(t *testing.T, cfg config.RestConfig, auth *fakeAuth, authMgr ports.AuthManager, cp ports.ControlPlane) http.Handler {
	t.Helper()
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:0"
	}
	if cfg.CORSAllowOrigins == nil {
		cfg.CORSAllowOrigins = []string{}
	}
	a := rest.NewRestAdapter(cp, auth, authMgr, cfg)
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
