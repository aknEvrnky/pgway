package dashboard_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/controlplane/api"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listProxiesEnvelope struct {
	Items      []*domain.Proxy `json:"items"`
	NextCursor string          `json:"next_cursor"`
	TotalCount int             `json:"total_count"`
}

func TestListProxies_OK(t *testing.T) {
	cp := &fakeCP{
		proxy: &domain.Proxy{Id: "p1", Protocol: domain.ProtocolHTTP, Host: "1.1.1.1", Port: 8080},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listProxiesEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "p1", got.Items[0].Id)
	assert.Equal(t, 1, got.TotalCount)
}

func TestListProxies_FilterByProtocol(t *testing.T) {
	cp := &fakeCP{
		proxies: []*domain.Proxy{
			{Id: "http-1", Protocol: domain.ProtocolHTTP, Host: "1.1.1.1", Port: 80},
			{Id: "socks-1", Protocol: domain.ProtocolSOCKS5, Host: "2.2.2.2", Port: 1080},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies?protocol=socks5", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listProxiesEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "socks-1", got.Items[0].Id)
}

func TestListProxies_InvalidProtocol(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies?protocol=ftp", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListProxies_Pagination(t *testing.T) {
	cp := &fakeCP{
		proxies: []*domain.Proxy{
			{Id: "a", Protocol: domain.ProtocolHTTP, Host: "1.1.1.1", Port: 1},
			{Id: "b", Protocol: domain.ProtocolHTTP, Host: "1.1.1.1", Port: 2},
			{Id: "c", Protocol: domain.ProtocolHTTP, Host: "1.1.1.1", Port: 3},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies?page_size=2", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var page1 listProxiesEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page1))
	require.Len(t, page1.Items, 2)
	assert.Equal(t, 3, page1.TotalCount)
	require.NotEmpty(t, page1.NextCursor)

	raw, err := base64.RawURLEncoding.DecodeString(page1.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "c", string(raw))

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/proxies?page_size=2&page_token="+page1.NextCursor, nil)
	req2.Header.Set("Authorization", "Bearer ok")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	var page2 listProxiesEnvelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &page2))
	require.Len(t, page2.Items, 1)
	assert.Equal(t, "c", page2.Items[0].Id)
}

func TestDeleteProxy_InUseConflict(t *testing.T) {
	cp := &fakeCP{
		proxy: &domain.Proxy{Id: "p1", Host: "1.1.1.1", Port: 8080},
		deleteErr: &api.ResourceInUseError{
			ResourceType: "proxy",
			Name:         "p1",
			Dependents:   []api.ResourceDependent{{Type: "pool", Name: "static-1"}},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/proxies/p1", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "still referenced by pool")
}

func TestDeleteProxy_NotFound(t *testing.T) {
	cp := &fakeCP{deleteErr: domain.ErrNotFound}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/proxies/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApplyProxy_MissingRefBadRequest(t *testing.T) {
	cp := &fakeCP{
		applyErr: &api.ResourceMissingRefError{
			ResourceType: "proxy",
			Name:         "p1",
			MissingType:  "pool",
			MissingName:  "x",
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"p1"},"spec":{"protocol":"http","host":"1.1.1.1","port":8080}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/proxies", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Contains(t, got["error"], "not found")
}

func TestListProxies_InvalidPageSize(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies?page_size=abc", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListProxies_InvalidPageToken(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/proxies?page_token=!!", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
