package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRegistrationToken_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	for _, tt := range []struct {
		name     string
		regToken RegistrationToken
		expected bool
	}{
		{name: "no expiry never expires", regToken: RegistrationToken{}, expected: false},
		{name: "future expiry", regToken: RegistrationToken{ExpiresAt: &future}, expected: false},
		{name: "past expiry", regToken: RegistrationToken{ExpiresAt: &past}, expected: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.regToken.IsExpired(now))
		})
	}
}
