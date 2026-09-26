package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/controlplane"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	var got []*domain.Pool
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "static-1", got[0].Id)
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
		applyErr: &controlplane.ResourceMissingRefError{
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
		deleteErr: &controlplane.ResourceInUseError{
			ResourceType: "pool",
			Name:         "static-1",
			Dependents:   []controlplane.ResourceDependent{{Type: "balancer", Name: "lb-1"}},
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
