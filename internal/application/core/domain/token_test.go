package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToken_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	for _, tt := range []struct {
		name     string
		token    Token
		expected bool
	}{
		{name: "no expiry never expires", token: Token{}, expected: false},
		{name: "future expiry", token: Token{ExpiresAt: &future}, expected: false},
		{name: "past expiry", token: Token{ExpiresAt: &past}, expected: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.token.IsExpired(now))
		})
	}
}

func TestToken_Validate(t *testing.T) {
	tests := []struct {
		name        string
		token       Token
		expectedErr string
	}{
		{
			name: "it is valid if only userID is filled",
			token: Token{
				Hash:    "foo",
				UserId:  "user-1",
				AgentId: "",
			},
			expectedErr: "",
		},
		{
			name: "it is valid if only agentID is filled",
			token: Token{
				Hash:    "foo",
				UserId:  "",
				AgentId: "agent-1",
			},
			expectedErr: "",
		},
		{
			name: "it is invalid if both IDs are filled",
			token: Token{
				Hash:    "foo",
				UserId:  "user-1",
				AgentId: "agent-1",
			},
			expectedErr: "userID and agentID both cannot be filled",
		},
		{
			name: "it is invalid if both IDs are empty",
			token: Token{
				Hash:    "foo",
				UserId:  "",
				AgentId: "",
			},
			expectedErr: "either userID or agentID must be filled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.token.Validate()

			if tt.expectedErr == "" {
				assert.NoError(t, err, "expected the token to be valid")
				return
			}

			assert.ErrorContains(t, err, tt.expectedErr)
		})
	}
}
