package v1

import (
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBalancerSpecV1_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    BalancerSpecV1
		wantErr string
	}{
		{
			name: "round-robin ok",
			spec: BalancerSpecV1{Type: "round-robin", PoolId: "p1"},
		},
		{
			name: "least-bytes without reset_interval ok",
			spec: BalancerSpecV1{Type: "least-bytes", PoolId: "p1"},
		},
		{
			name: "least-bytes with reset_interval ok",
			spec: BalancerSpecV1{Type: "least-bytes", PoolId: "p1", ResetInterval: "30s"},
		},
		{
			name:    "reset_interval on round-robin rejected",
			spec:    BalancerSpecV1{Type: "round-robin", PoolId: "p1", ResetInterval: "1m"},
			wantErr: "spec.reset_interval is only valid for type least-bytes",
		},
		{
			name:    "invalid reset_interval",
			spec:    BalancerSpecV1{Type: "least-bytes", PoolId: "p1", ResetInterval: "nope"},
			wantErr: `spec.reset_interval: time: invalid duration "nope"`,
		},
		{
			name:    "zero reset_interval",
			spec:    BalancerSpecV1{Type: "least-bytes", PoolId: "p1", ResetInterval: "0s"},
			wantErr: "spec.reset_interval must be > 0",
		},
		{
			name:    "missing type",
			spec:    BalancerSpecV1{PoolId: "p1"},
			wantErr: "spec.type is required",
		},
		{
			name:    "missing pool_id",
			spec:    BalancerSpecV1{Type: "least-bytes"},
			wantErr: "spec.pool_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestBalancerSpecV1_ResolvedResetInterval(t *testing.T) {
	assert.Equal(t, domain.DefaultLeastBytesResetInterval, BalancerSpecV1{}.ResolvedResetInterval())
	assert.Equal(t, 30*time.Second, BalancerSpecV1{ResetInterval: "30s"}.ResolvedResetInterval())
	assert.Equal(t, domain.DefaultLeastBytesResetInterval, BalancerSpecV1{ResetInterval: "nope"}.ResolvedResetInterval())
}
