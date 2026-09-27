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

type listRoutersEnvelope struct {
	Items      []*domain.Router `json:"items"`
	NextCursor string           `json:"next_cursor"`
	TotalCount int              `json:"total_count"`
}

func TestListRouters_OK(t *testing.T) {
	cp := &fakeCP{
		router: &domain.Router{
			Id:    "edge-router",
			Title: "Edge Host Router",
			Rules: []*domain.RouterRule{
				{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeHost, Value: "youtube.com"}, Target: "am-weighted"},
				{Id: "r2", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "edge-rr"},
			},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/routers", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listRoutersEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "edge-router", got.Items[0].Id)
	assert.Equal(t, 1, got.TotalCount)
}

func TestListRouters_FilterHasCatchAll(t *testing.T) {
	cp := &fakeCP{
		routers: []*domain.Router{
			{
				Id: "with",
				Rules: []*domain.RouterRule{
					{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "edge-rr"},
				},
			},
			{
				Id: "without",
				Rules: []*domain.RouterRule{
					{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeHost, Value: "a.com"}, Target: "edge-rr"},
				},
			},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/routers?has_catch_all=true", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got listRoutersEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got.Items, 1)
	assert.Equal(t, "with", got.Items[0].Id)
}

func TestListRouters_InvalidHasCatchAll(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/routers?has_catch_all=maybe", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListRouters_Pagination(t *testing.T) {
	cp := &fakeCP{
		routers: []*domain.Router{
			{Id: "a", Rules: []*domain.RouterRule{{Id: "r", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "lb"}}},
			{Id: "b", Rules: []*domain.RouterRule{{Id: "r", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "lb"}}},
			{Id: "c", Rules: []*domain.RouterRule{{Id: "r", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "lb"}}},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/routers?page_size=2", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var page1 listRoutersEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page1))
	require.Len(t, page1.Items, 2)
	assert.Equal(t, 3, page1.TotalCount)
	require.NotEmpty(t, page1.NextCursor)

	raw, err := base64.RawURLEncoding.DecodeString(page1.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, "c", string(raw))

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/routers?page_size=2&page_token="+page1.NextCursor, nil)
	req2.Header.Set("Authorization", "Bearer ok")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusOK, rec2.Code)
	var page2 listRoutersEnvelope
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &page2))
	require.Len(t, page2.Items, 1)
	assert.Equal(t, "c", page2.Items[0].Id)
}

func TestApplyRouter_OK(t *testing.T) {
	cp := &fakeCP{
		router: &domain.Router{
			Id: "edge-router",
			Rules: []*domain.RouterRule{
				{Id: "r1", Match: domain.RouterMatch{Type: domain.MatchTypeCatchAll}, Target: "edge-rr"},
			},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"edge-router"},"spec":{"rules":[{"id":"r1","match":{"type":"catch_all"},"target":"edge-rr"}]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got domain.Router
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "edge-router", got.Id)
}

func TestApplyRouter_MissingRefBadRequest(t *testing.T) {
	cp := &fakeCP{
		applyErr: &api.ResourceMissingRefError{
			ResourceType: "router",
			Name:         "edge-router",
			MissingType:  "balancer",
			MissingName:  "missing-lb",
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	body := []byte(`{"metadata":{"name":"edge-router"},"spec":{"rules":[{"id":"r1","match":{"type":"catch_all"},"target":"missing-lb"}]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteRouter_InUseConflict(t *testing.T) {
	cp := &fakeCP{
		router: &domain.Router{Id: "edge-router"},
		deleteErr: &api.ResourceInUseError{
			ResourceType: "router",
			Name:         "edge-router",
			Dependents:   []api.ResourceDependent{{Type: "flow", Name: "main"}},
		},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/routers/edge-router", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "still referenced by flow")
}

func TestDeleteRouter_NotFound(t *testing.T) {
	cp := &fakeCP{deleteErr: domain.ErrNotFound}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/routers/missing", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApplyRouter_InvalidSpec(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	body := []byte(`{"metadata":{"name":"bad"},"spec":{"rules":[]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "rules")
}

func TestApplyRouter_MissingName(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	body := []byte(`{"metadata":{},"spec":{"rules":[{"id":"r1","match":{"type":"catch_all"},"target":"lb"}]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/routers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApplyRouter_InvalidJSON(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/routers", bytes.NewReader([]byte(`{`)))
	req.Header.Set("Authorization", "Bearer ok")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetRouter_OK(t *testing.T) {
	cp := &fakeCP{
		router: &domain.Router{Id: "edge-router"},
	}
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, cp)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/routers/edge-router", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestDeleteRouter_OK(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/routers/edge-router", nil)
	req.Header.Set("Authorization", "Bearer ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}
