package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrincipal_Kind(t *testing.T) {
	tests := []struct {
		name         string
		user         *User
		agent        *Agent
		expectedKind PrincipalKind
	}{
		{
			name: "the kind is user if user is not nil",
			user: &User{
				Id: "user-1",
			},
			agent:        nil,
			expectedKind: PrincipalKindUser,
		},
		{
			name: "the kind is agent if user is nil",
			user: nil,
			agent: &Agent{
				Id: "agent-perry",
			},
			expectedKind: PrincipalKindAgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			principal := Principal{
				User:  tt.user,
				Agent: tt.agent,
			}

			assert.Equal(t, tt.expectedKind, principal.Kind())
		})
	}
}
