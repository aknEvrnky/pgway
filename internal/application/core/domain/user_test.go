package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_Validate(t *testing.T) {
	for _, tt := range []struct {
		name        string
		user        User
		expectedErr string
	}{
		{
			name: "valid admin",
			user: User{Id: "alice", PasswordHash: "$2a$10$hash", Role: RoleAdmin},
		},
		{
			name: "valid member",
			user: User{Id: "bob", PasswordHash: "$2a$10$hash", Role: RoleMember},
		},
		{
			name:        "missing username",
			user:        User{PasswordHash: "$2a$10$hash", Role: RoleAdmin},
			expectedErr: "username is required",
		},
		{
			name:        "invalid role",
			user:        User{Id: "alice", PasswordHash: "$2a$10$hash", Role: "superuser"},
			expectedErr: `invalid role: "superuser"`,
		},
		{
			name:        "missing password hash",
			user:        User{Id: "alice", Role: RoleAdmin},
			expectedErr: "password hash is required",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	assert.True(t, RoleAdmin.IsValid())
	assert.True(t, RoleMember.IsValid())
	assert.False(t, Role("root").IsValid())
	assert.False(t, Role("").IsValid())
}

func TestUser_IsAdmin(t *testing.T) {
	assert.True(t, (&User{Role: RoleAdmin}).IsAdmin())
	assert.False(t, (&User{Role: RoleMember}).IsAdmin())
}
