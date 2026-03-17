package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBalancerType_IsValid(t *testing.T) {
	for _, tt := range []struct {
		name         string
		balancerType string
		isValid      bool
	}{
		{
			name:         "Round Robin Test",
			balancerType: "round-robin",
			isValid:      true,
		},
		{
			name:         "Weighted Test",
			balancerType: "weighted",
			isValid:      true,
		},
		{
			name:         "Least Bytes Test",
			balancerType: "least-bytes",
			isValid:      true,
		},
		{
			name:         "Invalid Test",
			balancerType: "some-random-string",
			isValid:      false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bType := BalancerType(tt.balancerType)
			assert.Equal(t, tt.isValid, bType.IsValid())
		})
	}
}
