package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/auth"
	"github.com/aknEvrnky/pgway/internal/ports"
)

const sessionCookieName = "pgway_token"

type loginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	TTLSeconds int64  `json:"ttl_seconds,omitempty"`
	NoExpiry   bool   `json:"no_expiry,omitempty"`
}

type userResponse struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

func (a *Adapter) login(w http.ResponseWriter, r *http.Request) {
	if a.authManager == nil {
		writeError(w, http.StatusServiceUnavailable, "auth not configured")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second
	token, err := a.authManager.Login(r.Context(), req.Username, req.Password, ttl, req.NoExpiry)
	if err != nil {
		writeAuthError(w, err)
		return
	}

	principal, err := a.authenticator.Authenticate(r.Context(), token)
	if err != nil || principal.User == nil {
		writeError(w, http.StatusInternalServerError, "failed to resolve session")
		return
	}

	setSessionCookie(w, r, token, req.TTLSeconds, req.NoExpiry)
	writeJSON(w, http.StatusOK, loginResponse{
		Token: token,
		User: userResponse{
			ID:   principal.User.Id,
			Role: string(principal.User.Role),
		},
	})
}

func (a *Adapter) logout(w http.ResponseWriter, r *http.Request) {
	if a.authManager == nil {
		writeError(w, http.StatusServiceUnavailable, "auth not configured")
		return
	}

	token, ok := ports.TokenFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization")
		return
	}

	if err := a.authManager.Logout(r.Context(), token); err != nil {
		writeAuthError(w, err)
		return
	}

	clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (a *Adapter) me(w http.ResponseWriter, r *http.Request) {
	principal, ok := ports.PrincipalFromContext(r.Context())
	if !ok || principal.User == nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization")
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:   principal.User.Id,
		Role: string(principal.User.Role),
	})
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials),
		errors.Is(err, auth.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, ttlSeconds int64, noExpiry bool) {
	c := &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	}
	if !noExpiry && ttlSeconds > 0 {
		c.MaxAge = int(ttlSeconds)
	}
	http.SetCookie(w, c)
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
	})
}
