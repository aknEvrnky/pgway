package domain

import (
	"fmt"
	"time"
)

// Token is an opaque bearer token record. Hash holds the SHA-256 hex digest
// of the token string; the plain token is never stored.
type Token struct {
	Timestamps
	Hash      string     `json:"hash"`
	UserId    string     `json:"user_id,omitempty"`
	AgentId   string     `json:"agent_id,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (t *Token) IsExpired(now time.Time) bool {
	return t.ExpiresAt != nil && now.After(*t.ExpiresAt)
}

func (t *Token) Validate() error {
	if t.AgentId != "" && t.UserId != "" {
		return fmt.Errorf("userID and agentID both cannot be filled")
	}

	if t.UserId == "" && t.AgentId == "" {
		return fmt.Errorf("either userID or agentID must be filled")
	}

	return nil
}
