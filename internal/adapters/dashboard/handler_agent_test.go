package dashboard_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAgentManager struct {
	agents []domain.Agent
	err    error
}

func (f *fakeAgentManager) CreateRegistrationToken(context.Context, time.Duration) (string, error) {
	return "", nil
}
func (f *fakeAgentManager) Register(context.Context, string, domain.Agent) (*domain.Agent, string, error) {
	return nil, "", nil
}
func (f *fakeAgentManager) Heartbeat(context.Context) (time.Time, error) { return time.Time{}, nil }
func (f *fakeAgentManager) Deregister(context.Context) error             { return nil }
func (f *fakeAgentManager) DeleteAgent(context.Context, string) error    { return nil }

func (f *fakeAgentManager) ListAgents(_ context.Context, _ domain.ListParams, filter domain.AgentFilter) (domain.ListResult[domain.Agent], error) {
	if f.err != nil {
		return domain.ListResult[domain.Agent]{}, f.err
	}
	items := make([]*domain.Agent, 0, len(f.agents))
	for i := range f.agents {
		ag := f.agents[i]
		if filter.Search != "" && ag.Id != filter.Search && ag.Hostname != filter.Search {
			continue
		}
		items = append(items, &ag)
	}
	return domain.ListResult[domain.Agent]{Items: items, TotalCount: len(items)}, nil
}

func TestListAgents_OK(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	hb := now.Add(-5 * time.Second)
	stale := now.Add(-2 * time.Minute)

	agents := &fakeAgentManager{agents: []domain.Agent{
		{Id: "edge-1", Hostname: "host-a", Version: "0.2.0", LastHeartbeat: &hb, Timestamps: domain.Timestamps{CreatedAt: now, UpdatedAt: now}},
		{Id: "edge-2", Hostname: "host-b", LastHeartbeat: &stale, Timestamps: domain.Timestamps{CreatedAt: now, UpdatedAt: now}},
		{Id: "edge-3", Hostname: "host-c", Timestamps: domain.Timestamps{CreatedAt: now, UpdatedAt: now}},
	}}

	h := testAdapterWithAgents(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, agents, 30*time.Second)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Items []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
		TotalCount int `json:"total_count"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, 3, got.TotalCount)
	require.Len(t, got.Items, 3)
	assert.Equal(t, "active", got.Items[0].Status)
	assert.Equal(t, "disconnected", got.Items[1].Status)
	assert.Equal(t, "passive", got.Items[2].Status)
}

func TestListAgents_Empty(t *testing.T) {
	h := testAdapterWithAgents(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, &fakeAgentManager{}, 30*time.Second)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got struct {
		Items      []any `json:"items"`
		TotalCount int   `json:"total_count"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, 0, got.TotalCount)
	assert.Empty(t, got.Items)
}

func TestListAgents_Unauthorized(t *testing.T) {
	h := testAdapterWithAgents(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, &fakeAgentManager{}, 30*time.Second)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestListAgents_MemberOK(t *testing.T) {
	h := testAdapterWithAgents(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: memberPrincipal()}, &fakeAgentManager{agents: []domain.Agent{
		{Id: "a1", Timestamps: domain.Timestamps{CreatedAt: time.Now(), UpdatedAt: time.Now()}},
	}}, 30*time.Second)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
