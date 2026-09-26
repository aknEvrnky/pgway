package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		token  string
		ok     bool
	}{
		{name: "canonical", header: "Bearer tok", token: "tok", ok: true},
		{name: "lowercase scheme", header: "bearer tok", token: "tok", ok: true},
		{name: "uppercase scheme", header: "BEARER tok", token: "tok", ok: true},
		{name: "mixed case scheme", header: "BeArEr tok", token: "tok", ok: true},
		{name: "trailing whitespace trimmed", header: "Bearer tok \t", token: "tok", ok: true},
		{name: "no scheme", header: "tok", ok: false},
		{name: "no space", header: "Bearer", ok: false},
		{name: "empty token", header: "Bearer ", ok: false},
		{name: "whitespace token", header: "Bearer   ", ok: false},
		{name: "wrong scheme", header: "Basic dXNlcjpwYXNz", ok: false},
		{name: "empty header", header: "", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := bearerToken(tt.header)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.token, token)
		})
	}
}

func TestRateLimitKeys(t *testing.T) {
	t.Run("user principal wins", func(t *testing.T) {
		ctx := ports.ContextWithPrincipal(t.Context(), &domain.Principal{
			User: &domain.User{Id: "alice"},
		})
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		req.RemoteAddr = "203.0.113.1:1000"
		assert.Equal(t, "user:alice", userLimitKey(req))
	})

	t.Run("no principal falls back to ip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.1:1000"
		assert.Equal(t, "ip:203.0.113.1", ipLimitKey(req))
		assert.Equal(t, "ip:203.0.113.1", userLimitKey(req))
	})

	t.Run("host without port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "203.0.113.2"
		assert.Equal(t, "ip:203.0.113.2", ipLimitKey(req))
	})
}

func TestLimiterRegistry_AllowEnforcesBurst(t *testing.T) {
	r := newLimiterRegistry(rate.Limit(1), 1)
	assert.True(t, r.allow("k"))
	assert.False(t, r.allow("k"))
	assert.True(t, r.allow("other"))
}

func TestLimiterRegistry_SweepEvictsIdle(t *testing.T) {
	r := newLimiterRegistry(rate.Limit(1), 1)
	require.True(t, r.allow("fresh"))
	require.True(t, r.allow("stale"))

	now := time.Now()
	r.mu.Lock()
	r.entries["stale"].lastSeen = now.Add(-2 * time.Hour)
	r.mu.Unlock()

	r.mu.Lock()
	r.sweep(now)
	r.mu.Unlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	assert.Contains(t, r.entries, "fresh")
	assert.NotContains(t, r.entries, "stale")
}

func TestLimiterRegistry_SweepOnInterval(t *testing.T) {
	r := newLimiterRegistry(rate.Limit(100), 100)
	r.idleTTL = time.Nanosecond
	r.sweepInterval = time.Nanosecond

	require.True(t, r.allow("k"))
	time.Sleep(time.Millisecond)
	require.True(t, r.allow("k2"))

	r.mu.Lock()
	defer r.mu.Unlock()
	assert.NotContains(t, r.entries, "k")
	assert.Contains(t, r.entries, "k2")
}
