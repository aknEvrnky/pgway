package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/controlplane/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/ports"
)

type listUsersResponse struct {
	Items      []userDetailResponse `json:"items"`
	NextCursor string               `json:"next_cursor,omitempty"`
	TotalCount int                  `json:"total_count"`
}

type userDetailResponse struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role,omitempty"`
}

type createUserResponse struct {
	User              userDetailResponse `json:"user"`
	GeneratedPassword string             `json:"generated_password,omitempty"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password,omitempty"`
	NewPassword string `json:"new_password,omitempty"`
}

func (a *Adapter) listUsers(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if a.users == nil {
		writeError(w, http.StatusServiceUnavailable, "users not configured")
		return
	}

	q := r.URL.Query()
	role := q.Get("role")
	if role != "" && !domain.Role(role).IsValid() {
		writeError(w, http.StatusBadRequest, `role must be "admin" or "member"`)
		return
	}

	pageSize, err := parsePageSize(q.Get("page_size"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cursor, err := decodePageToken(q.Get("page_token"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid page_token")
		return
	}

	result, err := a.users.ListUsers(r.Context(), domain.ListParams{
		PageSize: pageSize,
		Cursor:   cursor,
	}, domain.UserFilter{
		Search: q.Get("search"),
		Role:   role,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]userDetailResponse, 0, len(result.Items))
	for _, u := range result.Items {
		if u != nil {
			items = append(items, toUserDetail(u))
		}
	}

	writeJSON(w, http.StatusOK, listUsersResponse{
		Items:      items,
		NextCursor: encodePageToken(result.NextCursor),
		TotalCount: result.TotalCount,
	})
}

func (a *Adapter) createUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if a.users == nil {
		writeError(w, http.StatusServiceUnavailable, "users not configured")
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Username) == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}
	if req.Role != "" && !domain.Role(req.Role).IsValid() {
		writeError(w, http.StatusBadRequest, `role must be "admin" or "member"`)
		return
	}

	user, generated, err := a.users.CreateUser(r.Context(), req.Username, req.Password, domain.Role(req.Role))
	if err != nil {
		writeUserError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, createUserResponse{
		User:              toUserDetail(user),
		GeneratedPassword: generated,
	})
}

func (a *Adapter) deleteUser(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	if a.users == nil {
		writeError(w, http.StatusServiceUnavailable, "users not configured")
		return
	}

	username := r.PathValue("username")
	if username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}

	if err := a.users.DeleteUser(r.Context(), username); err != nil {
		writeUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *Adapter) changeUserPassword(w http.ResponseWriter, r *http.Request) {
	if a.users == nil {
		writeError(w, http.StatusServiceUnavailable, "users not configured")
		return
	}

	principal, ok := ports.PrincipalFromContext(r.Context())
	if !ok || principal.User == nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization")
		return
	}
	actor := principal.User

	username := r.PathValue("username")
	if username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "new_password is required")
		return
	}

	// Admin reset: new_password only (Users admin UI). Works for any target,
	// including the admin's own account — gRPC ChangePassword requires
	// old_password for self, but that path is the old+new branch below.
	if req.OldPassword == "" {
		if !actor.IsAdmin() {
			if username != actor.Id {
				writeError(w, http.StatusForbidden, "admin role required")
				return
			}
			writeError(w, http.StatusBadRequest, "old_password is required")
			return
		}

		if err := a.users.ResetPassword(r.Context(), username, req.NewPassword); err != nil {
			writeUserError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Self-change with current password verification.
	if username != actor.Id {
		writeError(w, http.StatusForbidden, "old_password can only be used for your own account")
		return
	}

	if err := a.users.ChangePassword(r.Context(), username, req.OldPassword, req.NewPassword); err != nil {
		writeUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// requireAdmin writes 403 when the caller is authenticated but not an admin.
// Auth middleware already rejects missing/invalid tokens with 401.
func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	principal, ok := ports.PrincipalFromContext(r.Context())
	if !ok || principal.User == nil || !principal.User.IsAdmin() {
		writeError(w, http.StatusForbidden, "admin role required")
		return false
	}
	return true
}

func toUserDetail(u *domain.User) userDetailResponse {
	out := userDetailResponse{
		ID:   u.Id,
		Role: string(u.Role),
	}
	if !u.CreatedAt.IsZero() {
		out.CreatedAt = u.CreatedAt.UTC().Format(time.RFC3339)
	}
	if !u.UpdatedAt.IsZero() {
		out.UpdatedAt = u.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return out
}

func writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrLastAdmin):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, auth.ErrUserExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, auth.ErrWeakPassword):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrNotFound) || strings.Contains(err.Error(), "not found"):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
