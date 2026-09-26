package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuthManager struct {
	loginToken string
	loginErr   error
	logoutErr  error
	loggedOut  string
}

func (f *fakeAuthManager) InitAdmin(context.Context, string, string, string) (*domain.User, string, error) {
	return nil, "", fmt.Errorf("n/a")
}

func (f *fakeAuthManager) Login(_ context.Context, username, password string, _ time.Duration, _ bool) (string, error) {
	if f.loginErr != nil {
		return "", f.loginErr
	}
	if username == "" || password == "" {
		return "", auth.ErrInvalidCredentials
	}
	return f.loginToken, nil
}

func (f *fakeAuthManager) Logout(_ context.Context, token string) error {
	f.loggedOut = token
	return f.logoutErr
}

func TestLogin_PublicAndSetsCookie(t *testing.T) {
	authn := &fakeAuth{principal: userPrincipal()}
	mgr := &fakeAuthManager{loginToken: "pgw_tok"}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, authn, mgr, &fakeCP{})

	body := []byte(`{"username":"alice","password":"secret-pass"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "pgw_tok", got["token"])
	user, ok := got["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "alice", user["id"])
	assert.Equal(t, "admin", user["role"])

	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "pgway_token", cookies[0].Name)
	assert.Equal(t, "pgw_tok", cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{}, &fakeAuthManager{
		loginErr: auth.ErrInvalidCredentials,
	}, &fakeCP{})

	body := []byte(`{"username":"alice","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestMe_CookieAuth(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, &fakeAuthManager{}, &fakeCP{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "pgway_token", Value: "pgw_tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "alice", got["id"])
	assert.Equal(t, "admin", got["role"])
}

func TestLogout_RevokesAndClearsCookie(t *testing.T) {
	mgr := &fakeAuthManager{}
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: userPrincipal()}, mgr, &fakeCP{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer pgw_tok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "pgw_tok", mgr.loggedOut)
	cleared := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == "pgway_token" && c.MaxAge < 0 {
			cleared = true
		}
	}
	assert.True(t, cleared)
}
