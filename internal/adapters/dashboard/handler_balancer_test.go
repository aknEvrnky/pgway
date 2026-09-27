package dashboard_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/controlplane/api"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listBalancersEnvelope struct {
	Items []struct {
		Id            string `json:"id"`
		Title         string `json:"title"`
		Type          string `json:"type"`
		PoolId        string `json:"pool_id"`
		ResetInterval string `json:"reset_interval"`
	} `json:"items"`
	NextCursor string `json:"next_cursor"`
	TotalCount int    `json:"total_count"`
}

func TestListBalancers_OK(t *testing.T) {
	cp := &fakeCP{
		balancer: &domain.LoadBalancer{
			Id:     "edge-rr",
			Title:  "Edge Round Robin",
			Type:   domain.BalancerTypeRoundRobin,
			PoolId: "am-static-pool",
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balancers", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listBalancersEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "edge-rr", got.Items[0].Id)
	assert.Equal(t, "round-robin", got.Items[0].Type)
	assert.Equal(t, 1, got.TotalCount)
	assert.Empty(t, got.NextCursor)
}

func TestListBalancers_FilterByType(t *testing.T) {
	cp := &fakeCP{
		balancers: []*domain.LoadBalancer{
			{Id: "rr", Type: domain.BalancerTypeRoundRobin, PoolId: "p1"},
			{Id: "w", Type: domain.BalancerTypeWeighted, PoolId: "p1"},
			{Id: "lb", Type: domain.BalancerTypeLeastBytes, PoolId: "p1", ResetInterval: time.Minute},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balancers?type=weighted", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listBalancersEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "w", got.Items[0].Id)
	assert.Equal(t, 1, got.TotalCount)
}

func TestListBalancers_InvalidType(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balancers?type=static", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "round-robin")
}

func TestListBalancers_Pagination(t *testing.T) {
	cp := &fakeCP{
		balancers: []*domain.LoadBalancer{
			{Id: "a", Type: domain.BalancerTypeRoundRobin, PoolId: "p1"},
			{Id: "b", Type: domain.BalancerTypeRoundRobin, PoolId: "p1"},
			{Id: "c", Type: domain.BalancerTypeRoundRobin, PoolId: "p1"},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balancers?page_size=2", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var page1 listBalancersEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page1))
	require.Len(t, page1.Items, 2)
	assert.Equal(t, 3, page1.TotalCount)
	require.NotEmpty(t, page1.NextCursor)

	raw, err := base64.RawURLEncoding.DecodeString(page1.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "c", string(raw))

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/balancers?page_size=2&page_token="+page1.NextCursor, nil)
	req2.Header.Set("Authorization", "Bearer ok")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
	var page2 listBalancersEnvelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &page2))
	require.Len(t, page2.Items, 1)
	assert.Equal(t, "c", page2.Items[0].Id)
	assert.Empty(t, page2.NextCursor)
}

func TestApplyBalancer_RoundRobinOK(t *testing.T) {
	cp := &fakeCP{
		balancer: &domain.LoadBalancer{
			Id:     "edge-rr",
			Type:   domain.BalancerTypeRoundRobin,
			PoolId: "am-static-pool",
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"edge-rr"},"spec":{"type":"round-robin","pool_id":"am-static-pool"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/balancers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "edge-rr", got["id"])
	assert.Equal(t, "round-robin", got["type"])
}

func TestApplyBalancer_LeastBytesResetInterval(t *testing.T) {
	cp := &fakeCP{
		balancer: &domain.LoadBalancer{
			Id:            "eu-least",
			Type:          domain.BalancerTypeLeastBytes,
			PoolId:        "eu-pool",
			ResetInterval: time.Minute,
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"eu-least"},"spec":{"type":"least-bytes","pool_id":"eu-pool","reset_interval":"1m"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/balancers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "1m", got["reset_interval"])
}

func TestApplyBalancer_MissingRefBadRequest(t *testing.T) {
	cp := &fakeCP{
		applyErr: &api.ResourceMissingRefError{
			ResourceType: "balancer",
			Name:         "edge-rr",
			MissingType:  "pool",
			MissingName:  "missing-pool",
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"edge-rr"},"spec":{"type":"round-robin","pool_id":"missing-pool"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/balancers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var got map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Contains(t, got["error"], "not found")
}

func TestApplyBalancer_WeightedRequiresStatic(t *testing.T) {
	cp := &fakeCP{
		applyErr: assertErr("weighted balancer requires static pool \"dyn\" (got \"dynamic\")"),
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"w"},"spec":{"type":"weighted","pool_id":"dyn"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/balancers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "requires static pool")
}

func TestDeleteBalancer_InUseConflict(t *testing.T) {
	cp := &fakeCP{
		balancer: &domain.LoadBalancer{Id: "edge-rr", Type: domain.BalancerTypeRoundRobin, PoolId: "p1"},
		deleteErr: &api.ResourceInUseError{
			ResourceType: "balancer",
			Name:         "edge-rr",
			Dependents:   []api.ResourceDependent{{Type: "flow", Name: "main"}},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/balancers/edge-rr", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "still referenced by flow")
}

func TestDeleteBalancer_NotFound(t *testing.T) {
	cp := &fakeCP{deleteErr: domain.ErrNotFound}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/balancers/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApplyBalancer_InvalidSpec(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	body := []byte(`{"metadata":{"name":"bad"},"spec":{"type":"round-robin"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/balancers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "pool_id")
}

func TestListBalancers_InvalidPageSize(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balancers?page_size=nope", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApplyBalancer_MissingName(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	body := []byte(`{"metadata":{},"spec":{"type":"round-robin","pool_id":"p"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/balancers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteBalancer_OK(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/balancers/edge-rr", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestGetBalancer_OK(t *testing.T) {
	cp := &fakeCP{
		balancer: &domain.LoadBalancer{
			Id:     "edge-rr",
			Type:   domain.BalancerTypeRoundRobin,
			PoolId: "static-1",
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balancers/edge-rr", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "edge-rr", got["id"])
}

func TestGetBalancer_NotFound(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/balancers/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
