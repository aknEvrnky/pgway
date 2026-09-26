package rest_test

import (
	"bytes"
	"encoding/base64"
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

type listEntrypointsEnvelope struct {
	Items      []*domain.Entrypoint `json:"items"`
	NextCursor string               `json:"next_cursor"`
	TotalCount int                  `json:"total_count"`
}

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

	var body listEntrypointsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 2, body.TotalCount)
	require.Len(t, body.Items, 2)
	assert.Equal(t, "public-http", body.Items[0].Id)
}

func TestListEntrypoints_FilterFlowId(t *testing.T) {
	cp := &fakeCP{
		entrypoints: []*domain.Entrypoint{
			{Id: "public-http", Protocol: "http", Host: "0.0.0.0", Port: 8080, FlowId: "edge-flow"},
			{Id: "api-gateway", Protocol: "http", Host: "127.0.0.1", Port: 8443, FlowId: "api-flow"},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entrypoints?flow_id=api-flow", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body listEntrypointsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	assert.Equal(t, "api-gateway", body.Items[0].Id)
}

func TestListEntrypoints_Pagination(t *testing.T) {
	cp := &fakeCP{
		entrypoints: []*domain.Entrypoint{
			{Id: "a", Protocol: "http", Host: "0.0.0.0", Port: 1, FlowId: "f"},
			{Id: "b", Protocol: "http", Host: "0.0.0.0", Port: 2, FlowId: "f"},
			{Id: "c", Protocol: "http", Host: "0.0.0.0", Port: 3, FlowId: "f"},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entrypoints?page_size=2", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var page1 listEntrypointsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page1))
	require.Len(t, page1.Items, 2)
	require.NotEmpty(t, page1.NextCursor)
	raw, err := base64.RawURLEncoding.DecodeString(page1.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "c", string(raw))

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/entrypoints?page_size=2&page_token="+page1.NextCursor, nil)
	req2.Header.Set("Authorization", "Bearer ok")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusOK, rec2.Code)
	var page2 listEntrypointsEnvelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &page2))
	require.Len(t, page2.Items, 1)
	assert.Equal(t, "c", page2.Items[0].Id)
}

func TestListEntrypoints_InvalidPageSize(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entrypoints?page_size=0", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListEntrypoints_InvalidPageToken(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entrypoints?page_token=!!", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetEntrypoint_OK(t *testing.T) {
	cp := &fakeCP{
		entrypoint: &domain.Entrypoint{Id: "public-http", Protocol: "http", Host: "0.0.0.0", Port: 8080, FlowId: "edge-flow"},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entrypoints/public-http", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got domain.Entrypoint
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "public-http", got.Id)
}

func TestGetEntrypoint_NotFound(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entrypoints/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApplyEntrypoint_OK(t *testing.T) {
	cp := &fakeCP{
		entrypoint: &domain.Entrypoint{Id: "public-http", Protocol: "http", Host: "0.0.0.0", Port: 8080, FlowId: "edge-flow"},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"public-http"},"spec":{"protocol":"http","host":"0.0.0.0","port":8080,"flow_id":"edge-flow"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entrypoints", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got domain.Entrypoint
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "public-http", got.Id)
}

func TestApplyEntrypoint_MissingName(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	body := []byte(`{"metadata":{},"spec":{"protocol":"http","host":"0.0.0.0","port":8080,"flow_id":"edge-flow"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entrypoints", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApplyEntrypoint_InvalidSpec(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	body := []byte(`{"metadata":{"name":"ep"},"spec":{"protocol":"http","host":"0.0.0.0","port":0,"flow_id":"edge-flow"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entrypoints", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApplyEntrypoint_InvalidJSON(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/entrypoints", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApplyEntrypoint_MissingRef(t *testing.T) {
	cp := &fakeCP{
		applyErr: &controlplane.ResourceMissingRefError{
			ResourceType: "entrypoint",
			Name:         "public-http",
			MissingType:  "flow",
			MissingName:  "missing-flow",
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"public-http"},"spec":{"protocol":"http","host":"0.0.0.0","port":8080,"flow_id":"missing-flow"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entrypoints", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
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
