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

	"github.com/aknEvrnky/pgway/internal/application/controlplane/auth"
	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/aknEvrnky/pgway/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUserManager struct {
	users       map[string]*domain.User
	createErr   error
	listErr     error
	deleteErr   error
	changeErr   error
	resetErr    error
	lastCreate  createCapture
	lastReset   resetCapture
	lastChange  changeCapture
	lastDeleted string
}

type createCapture struct {
	username string
	password string
	role     domain.Role
}

type resetCapture struct {
	username    string
	newPassword string
}

type changeCapture struct {
	username    string
	oldPassword string
	newPassword string
}

func newFakeUserManager(users ...*domain.User) *fakeUserManager {
	m := &fakeUserManager{users: map[string]*domain.User{}}
	for _, u := range users {
		cp := *u
		m.users[u.Id] = &cp
	}
	return m
}

func (f *fakeUserManager) CreateUser(_ context.Context, username, password string, role domain.Role) (*domain.User, string, error) {
	f.lastCreate = createCapture{username: username, password: password, role: role}
	if f.createErr != nil {
		return nil, "", f.createErr
	}
	if _, exists := f.users[username]; exists {
		return nil, "", auth.ErrUserExists
	}
	generated := ""
	if password == "" {
		generated = "temp-generated-pass"
		password = generated
	}
	if role == "" {
		role = domain.RoleMember
	}
	now := time.Now().UTC()
	u := &domain.User{
		Id:           username,
		Role:         role,
		PasswordHash: "hash",
		Timestamps:   domain.Timestamps{CreatedAt: now, UpdatedAt: now},
	}
	f.users[username] = u
	return u, generated, nil
}

func (f *fakeUserManager) ListUsers(_ context.Context, _ domain.ListParams, filter domain.UserFilter) (domain.ListResult[domain.User], error) {
	if f.listErr != nil {
		return domain.ListResult[domain.User]{}, f.listErr
	}
	items := make([]*domain.User, 0, len(f.users))
	for _, u := range f.users {
		if filter.Role != "" && string(u.Role) != filter.Role {
			continue
		}
		if filter.Search != "" && u.Id != filter.Search {
			continue
		}
		cp := *u
		items = append(items, &cp)
	}
	return domain.ListResult[domain.User]{Items: items, TotalCount: len(items)}, nil
}

func (f *fakeUserManager) DeleteUser(_ context.Context, username string) error {
	f.lastDeleted = username
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.users[username]; !ok {
		return fmt.Errorf("user %q not found: %w", username, domain.ErrNotFound)
	}
	delete(f.users, username)
	return nil
}

func (f *fakeUserManager) ChangePassword(_ context.Context, username, oldPassword, newPassword string) error {
	f.lastChange = changeCapture{username: username, oldPassword: oldPassword, newPassword: newPassword}
	if f.changeErr != nil {
		return f.changeErr
	}
	if _, ok := f.users[username]; !ok {
		return fmt.Errorf("user %q not found: %w", username, domain.ErrNotFound)
	}
	return nil
}

func (f *fakeUserManager) ResetPassword(_ context.Context, username, newPassword string) error {
	f.lastReset = resetCapture{username: username, newPassword: newPassword}
	if f.resetErr != nil {
		return f.resetErr
	}
	if _, ok := f.users[username]; !ok {
		return fmt.Errorf("user %q not found: %w", username, domain.ErrNotFound)
	}
	return nil
}

func adminPrincipal() *domain.Principal {
	return &domain.Principal{User: &domain.User{Id: "admin", Role: domain.RoleAdmin}}
}

func memberPrincipal() *domain.Principal {
	return &domain.Principal{User: &domain.User{Id: "bob", Role: domain.RoleMember}}
}

func authHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer tok")
}

func TestListUsers_AdminOK(t *testing.T) {
	users := newFakeUserManager(
		&domain.User{Id: "admin", Role: domain.RoleAdmin},
		&domain.User{Id: "bob", Role: domain.RoleMember},
	)
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, float64(2), got["total_count"])
	items, ok := got["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestListUsers_MemberForbidden(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: memberPrincipal()}, nil, &fakeCP{}, newFakeUserManager())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestListUsers_InvalidRole(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, newFakeUserManager())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?role=super", nil)
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateUser_AdminOK(t *testing.T) {
	users := newFakeUserManager()
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	body := []byte(`{"username":"carol","password":"password123","role":"member"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var got createUserAPIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "carol", got.User.ID)
	assert.Equal(t, "member", got.User.Role)
	assert.Empty(t, got.GeneratedPassword)
	assert.Equal(t, "carol", users.lastCreate.username)
}

func TestCreateUser_GeneratedPassword(t *testing.T) {
	users := newFakeUserManager()
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	body := []byte(`{"username":"dave","role":"admin"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var got createUserAPIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "dave", got.User.ID)
	assert.Equal(t, "admin", got.User.Role)
	assert.NotEmpty(t, got.GeneratedPassword)
}

func TestCreateUser_MemberForbidden(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: memberPrincipal()}, nil, &fakeCP{}, newFakeUserManager())

	body := []byte(`{"username":"eve","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCreateUser_Conflict(t *testing.T) {
	users := newFakeUserManager(&domain.User{Id: "carol", Role: domain.RoleMember})
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	body := []byte(`{"username":"carol","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestCreateUser_MissingUsername(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, newFakeUserManager())

	body := []byte(`{"password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDeleteUser_AdminOK(t *testing.T) {
	users := newFakeUserManager(&domain.User{Id: "bob", Role: domain.RoleMember})
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/bob", nil)
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "bob", users.lastDeleted)
}

func TestDeleteUser_LastAdminConflict(t *testing.T) {
	users := newFakeUserManager(&domain.User{Id: "admin", Role: domain.RoleAdmin})
	users.deleteErr = auth.ErrLastAdmin
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/admin", nil)
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestDeleteUser_MemberForbidden(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: memberPrincipal()}, nil, &fakeCP{}, newFakeUserManager())

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/admin", nil)
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDeleteUser_NotFound(t *testing.T) {
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, newFakeUserManager())

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/ghost", nil)
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestChangePassword_SelfOK(t *testing.T) {
	users := newFakeUserManager(&domain.User{Id: "bob", Role: domain.RoleMember})
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: memberPrincipal()}, nil, &fakeCP{}, users)

	body := []byte(`{"old_password":"oldpass12","new_password":"newpass12"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/bob/password", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "bob", users.lastChange.username)
	assert.Equal(t, "oldpass12", users.lastChange.oldPassword)
	assert.Equal(t, "newpass12", users.lastChange.newPassword)
}

func TestChangePassword_AdminResetOther(t *testing.T) {
	users := newFakeUserManager(&domain.User{Id: "bob", Role: domain.RoleMember})
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	body := []byte(`{"new_password":"resetpass1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/bob/password", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "bob", users.lastReset.username)
	assert.Equal(t, "resetpass1", users.lastReset.newPassword)
}

func TestChangePassword_AdminResetSelf(t *testing.T) {
	users := newFakeUserManager(&domain.User{Id: "admin", Role: domain.RoleAdmin})
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	body := []byte(`{"new_password":"resetpass1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/admin/password", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "admin", users.lastReset.username)
	assert.Equal(t, "resetpass1", users.lastReset.newPassword)
	assert.Empty(t, users.lastChange.username)
}

func TestChangePassword_AdminResetRequiresPassword(t *testing.T) {
	users := newFakeUserManager(&domain.User{Id: "bob", Role: domain.RoleMember})
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: adminPrincipal()}, nil, &fakeCP{}, users)

	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/bob/password", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestChangePassword_MemberCannotResetOther(t *testing.T) {
	users := newFakeUserManager(&domain.User{Id: "admin", Role: domain.RoleAdmin})
	h := testAdapter(t, config.RestConfig{RateLimitRPS: 0}, &fakeAuth{principal: memberPrincipal()}, nil, &fakeCP{}, users)

	body := []byte(`{"new_password":"hackedpassword"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/admin/password", bytes.NewReader(body))
	authHeader(req)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

type createUserAPIResponse struct {
	User struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	} `json:"user"`
	GeneratedPassword string `json:"generated_password"`
}
