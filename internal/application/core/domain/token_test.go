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
