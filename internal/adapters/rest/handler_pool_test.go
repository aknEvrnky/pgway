package rest_test

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

type listPoolsEnvelope struct {
	Items      []*domain.Pool `json:"items"`
	NextCursor string         `json:"next_cursor"`
	TotalCount int            `json:"total_count"`
}

func TestListPools_OK(t *testing.T) {
	cp := &fakeCP{
		pool: &domain.Pool{
			Id:    "static-1",
			Title: "Static",
			Type:  domain.PoolTypeStatic,
			Members: []domain.PoolMember{
				{ProxyId: "p1", Weight: 1},
			},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pools", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listPoolsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "static-1", got.Items[0].Id)
	assert.Equal(t, 1, got.TotalCount)
	assert.Empty(t, got.NextCursor)
}

func TestListPools_FilterByType(t *testing.T) {
	cp := &fakeCP{
		pools: []*domain.Pool{
			{Id: "s1", Type: domain.PoolTypeStatic, Members: []domain.PoolMember{{ProxyId: "p1", Weight: 1}}},
			{Id: "d1", Type: domain.PoolTypeDynamic, Selector: &domain.LabelSelector{Allow: map[string]string{"k": "v"}}},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pools?type=dynamic", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listPoolsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "d1", got.Items[0].Id)
	assert.Equal(t, 1, got.TotalCount)
}

func TestListPools_InvalidType(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pools?type=weighted", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "static")
}

func TestListPools_Pagination(t *testing.T) {
	cp := &fakeCP{
		pools: []*domain.Pool{
			{Id: "a", Type: domain.PoolTypeStatic, Members: []domain.PoolMember{{ProxyId: "p1", Weight: 1}}},
			{Id: "b", Type: domain.PoolTypeStatic, Members: []domain.PoolMember{{ProxyId: "p1", Weight: 1}}},
			{Id: "c", Type: domain.PoolTypeStatic, Members: []domain.PoolMember{{ProxyId: "p1", Weight: 1}}},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pools?page_size=2", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var page1 listPoolsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page1))
	require.Len(t, page1.Items, 2)
	assert.Equal(t, 3, page1.TotalCount)
	require.NotEmpty(t, page1.NextCursor)

	raw, err := base64.RawURLEncoding.DecodeString(page1.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "c", string(raw))

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/pools?page_size=2&page_token="+page1.NextCursor, nil)
	req2.Header.Set("Authorization", "Bearer ok")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
	var page2 listPoolsEnvelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &page2))
	require.Len(t, page2.Items, 1)
	assert.Equal(t, "c", page2.Items[0].Id)
	assert.Empty(t, page2.NextCursor)
}

func TestApplyPool_StaticOK(t *testing.T) {
	cp := &fakeCP{
		pool: &domain.Pool{
			Id:   "static-1",
			Type: domain.PoolTypeStatic,
			Members: []domain.PoolMember{
				{ProxyId: "p1", Weight: 2},
			},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"static-1"},"spec":{"type":"static","members":[{"proxy_id":"p1","weight":2}]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pools", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got domain.Pool
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "static-1", got.Id)
	assert.Equal(t, domain.PoolTypeStatic, got.Type)
}

func TestApplyPool_MissingRefBadRequest(t *testing.T) {
	cp := &fakeCP{
		applyErr: &api.ResourceMissingRefError{
			ResourceType: "pool",
			Name:         "static-1",
			MissingType:  "proxy",
			MissingName:  "missing-proxy",
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"static-1"},"spec":{"type":"static","members":[{"proxy_id":"missing-proxy"}]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pools", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Contains(t, got["error"], "not found")
}

func TestDeletePool_InUseConflict(t *testing.T) {
	cp := &fakeCP{
		pool: &domain.Pool{Id: "static-1", Type: domain.PoolTypeStatic},
		deleteErr: &api.ResourceInUseError{
			ResourceType: "pool",
			Name:         "static-1",
			Dependents:   []api.ResourceDependent{{Type: "balancer", Name: "lb-1"}},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pools/static-1", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "still referenced by balancer")
}

func TestDeletePool_NotFound(t *testing.T) {
	cp := &fakeCP{deleteErr: domain.ErrNotFound}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pools/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApplyPool_InvalidSpec(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	body := []byte(`{"metadata":{"name":"bad"},"spec":{"type":"static"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pools", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "members")
}

func TestApplyPool_MissingName(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	body := []byte(`{"metadata":{},"spec":{"type":"dynamic","selector":{"allow":{"a":"b"}}}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pools", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetPool_OK(t *testing.T) {
	cp := &fakeCP{
		pool: &domain.Pool{Id: "static-1", Type: domain.PoolTypeStatic},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pools/static-1", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestDeletePool_OK(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pools/static-1", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestListPools_InvalidPageSize(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pools?page_size=0", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
