package v1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoolSpecV1Validate(t *testing.T) {
	w0 := 0
	w3 := 3

	tests := []struct {
		name    string
		spec    PoolSpecV1
		wantErr string
	}{
		{
			name: "static ok with default weight",
			spec: PoolSpecV1{
				Type:    "static",
				Members: []PoolMemberSpec{{ProxyId: "p1"}, {ProxyId: "p2", Weight: &w3}},
			},
		},
		{
			name: "static rejects weight 0",
			spec: PoolSpecV1{
				Type:    "static",
				Members: []PoolMemberSpec{{ProxyId: "p1", Weight: &w0}},
			},
			wantErr: "spec.members[0].weight must be >= 1",
		},
		{
			name:    "static requires members",
			spec:    PoolSpecV1{Type: "static"},
			wantErr: "spec.members is required for static pool",
		},
		{
			name: "dynamic rejects members",
			spec: PoolSpecV1{
				Type:     "dynamic",
				Selector: &SelectorSpec{Allow: map[string]string{"a": "b"}},
				Members:  []PoolMemberSpec{{ProxyId: "p1"}},
			},
			wantErr: "spec.members must not be set for dynamic pool",
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

func TestPoolMemberSpecResolvedWeight(t *testing.T) {
	w := 5
	require.Equal(t, 1, PoolMemberSpec{ProxyId: "p1"}.ResolvedWeight())
	require.Equal(t, 5, PoolMemberSpec{ProxyId: "p1", Weight: &w}.ResolvedWeight())
}
