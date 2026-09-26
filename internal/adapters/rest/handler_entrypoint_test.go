package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListEntrypoints(t *testing.T) {
	cp := &fakeCP{
		entrypoints: []*domain.Entrypoint{
			{Id: "public-http", Protocol: "http", Host: "0.0.0.0", Port: 8080, FlowId: "edge-flow"},
			{Id: "api-gateway", Protocol: "http", Host: "0.0.0.0", Port: 8443, FlowId: "api-flow"},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entrypoints?page_size=10", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Items      []*domain.Entrypoint `json:"items"`
		TotalCount int                  `json:"total_count"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 2, body.TotalCount)
	require.Len(t, body.Items, 2)
	assert.Equal(t, "public-http", body.Items[0].Id)
}

func TestDeleteEntrypoint_OK(t *testing.T) {
	cp := &fakeCP{
		entrypoint: &domain.Entrypoint{Id: "public-http", Host: "0.0.0.0", Port: 8080, FlowId: "edge-flow"},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entrypoints/public-http", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteEntrypoint_NotFound(t *testing.T) {
	cp := &fakeCP{deleteErr: domain.ErrNotFound}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entrypoints/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
