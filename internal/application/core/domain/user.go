package domain

import "fmt"

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

type User struct {
	Timestamps
	Id           string `json:"id"`
	PasswordHash string `json:"password_hash,omitempty"`
	Role         Role   `json:"role"`
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) Validate() error {
	if u.Id == "" {
		return fmt.Errorf("username is required")
	}

	if !u.Role.IsValid() {
		return fmt.Errorf("invalid role: %q", u.Role)
	}

	if u.PasswordHash == "" {
		return fmt.Errorf("password hash is required")
	}

	return nil
}
