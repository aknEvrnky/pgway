package domain

import (
	"time"
)

// Token is an opaque bearer token record. Hash holds the SHA-256 hex digest
// of the token string; the plain token is never stored.
type Token struct {
	Timestamps
	Hash      string     `json:"hash"`
	UserId    string     `json:"user_id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (t *Token) IsExpired(now time.Time) bool {
	return t.ExpiresAt != nil && now.After(*t.ExpiresAt)
}
