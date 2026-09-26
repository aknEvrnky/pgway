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

type listFlowsEnvelope struct {
	Items      []*domain.Flow `json:"items"`
	NextCursor string         `json:"next_cursor"`
	TotalCount int            `json:"total_count"`
}

func TestListFlows_OK(t *testing.T) {
	cp := &fakeCP{
		flow: &domain.Flow{
			Id:       "edge-flow",
			RouterId: "edge-router",
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flows", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listFlowsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "edge-flow", got.Items[0].Id)
	assert.Equal(t, 1, got.TotalCount)
}

func TestListFlows_FilterMode(t *testing.T) {
	cp := &fakeCP{
		flows: []*domain.Flow{
			{Id: "routed", RouterId: "edge-router"},
			{Id: "direct", BalancerId: "edge-rr"},
			{Id: "both", RouterId: "api-router", BalancerId: "edge-rr"},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flows?mode=direct", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listFlowsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "direct", got.Items[0].Id)

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/flows?mode=router", nil)
	req2.Header.Set("Authorization", "Bearer ok")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
	var got2 listFlowsEnvelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &got2))
	require.Len(t, got2.Items, 2)
}

func TestListFlows_InvalidMode(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flows?mode=weighted", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListFlows_Pagination(t *testing.T) {
	cp := &fakeCP{
		flows: []*domain.Flow{
			{Id: "a", BalancerId: "lb"},
			{Id: "b", BalancerId: "lb"},
			{Id: "c", BalancerId: "lb"},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flows?page_size=2", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var page1 listFlowsEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page1))
	require.Len(t, page1.Items, 2)
	assert.Equal(t, 3, page1.TotalCount)
	require.NotEmpty(t, page1.NextCursor)

	raw, err := base64.RawURLEncoding.DecodeString(page1.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "c", string(raw))

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/flows?page_size=2&page_token="+page1.NextCursor, nil)
	req2.Header.Set("Authorization", "Bearer ok")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
	var page2 listFlowsEnvelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &page2))
	require.Len(t, page2.Items, 1)
	assert.Equal(t, "c", page2.Items[0].Id)
}

func TestApplyFlow_OK(t *testing.T) {
	cp := &fakeCP{
		flow: &domain.Flow{Id: "edge-flow", RouterId: "edge-router"},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"edge-flow"},"spec":{"router_id":"edge-router"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/flows", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got domain.Flow
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "edge-flow", got.Id)
}

func TestGetFlow_OK(t *testing.T) {
	cp := &fakeCP{
		flow: &domain.Flow{Id: "edge-flow", RouterId: "edge-router"},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flows/edge-flow", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got domain.Flow
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "edge-flow", got.Id)
}

func TestGetFlow_NotFound(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/flows/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteFlow_InUseConflict(t *testing.T) {
	cp := &fakeCP{
		flow: &domain.Flow{Id: "edge-flow", RouterId: "edge-router"},
		deleteErr: &controlplane.ResourceInUseError{
			ResourceType: "flow",
			Name:         "edge-flow",
			Dependents:   []controlplane.ResourceDependent{{Type: "entrypoint", Name: "http-in"}},
		},
	}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/flows/edge-flow", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "still referenced by entrypoint")
}

func TestDeleteFlow_NotFound(t *testing.T) {
	cp := &fakeCP{deleteErr: domain.ErrNotFound}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/flows/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
