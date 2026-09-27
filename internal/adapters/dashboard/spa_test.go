package dashboard_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/aknEvrnky/pgway/internal/adapters/dashboard"
	"github.com/aknEvrnky/pgway/internal/adapters/dashboard/ui"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSPA_ServesIndexAndFallsBack(t *testing.T) {
	static := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>dash</html>")},
		"app.js":     &fstest.MapFile{Data: []byte("console.log(1)")},
	}

	h := testAdapterWithStatic(t, static)

	t.Run("root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "dash")
	})

	t.Run("asset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "console.log")
	})

	t.Run("client_route_fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/proxies", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "dash")
	})

	t.Run("api_still_requires_auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestSPA_AbsentWithoutStaticFS(t *testing.T) {
	h := testAdapter(t, config.DashboardConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, nil, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUI_StubDisabled(t *testing.T) {
	require.False(t, ui.Enabled())
}

func testAdapterWithStatic(t *testing.T, fsys fs.FS) http.Handler {
	t.Helper()
	a := dashboard.NewAdapter(&fakeCP{}, &fakeAuth{principal: userPrincipal()}, nil, newFakeUserManager(), nil, config.DashboardConfig{RateLimitRPS: 0}, 30*time.Second)
	a.SetStaticFS(fsys)
	return a.Handler()
}
