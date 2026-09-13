package domain

import "time"

type RegistrationToken struct {
	Timestamps
	Hash      string     `json:"hash"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (t *RegistrationToken) IsExpired(now time.Time) bool {
	return t.ExpiresAt != nil && now.After(*t.ExpiresAt)
}
