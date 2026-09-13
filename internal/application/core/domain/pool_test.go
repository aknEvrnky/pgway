package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolValidate(t *testing.T) {
	tests := []struct {
		name    string
		pool    *Pool
		wantErr string
	}{
		{
			name: "static ok",
			pool: &Pool{
				Id:   "pool-1",
				Type: PoolTypeStatic,
				Members: []PoolMember{
					{ProxyId: "p1", Weight: 1},
					{ProxyId: "p2", Weight: 3},
				},
			},
		},
		{
			name: "static requires members",
			pool: &Pool{Id: "pool-1", Type: PoolTypeStatic},
			wantErr: `static pool "pool-1" requires at least one member`,
		},
		{
			name: "static rejects weight < 1",
			pool: &Pool{
				Id:      "pool-1",
				Type:    PoolTypeStatic,
				Members: []PoolMember{{ProxyId: "p1", Weight: 0}},
			},
			wantErr: `static pool "pool-1" member "p1": weight must be >= 1`,
		},
		{
			name: "static rejects duplicate proxy_id",
			pool: &Pool{
				Id:   "pool-1",
				Type: PoolTypeStatic,
				Members: []PoolMember{
					{ProxyId: "p1", Weight: 1},
					{ProxyId: "p1", Weight: 2},
				},
			},
			wantErr: `static pool "pool-1" has duplicate proxy_id "p1"`,
		},
		{
			name: "dynamic ok",
			pool: &Pool{
				Id:       "pool-1",
				Type:     PoolTypeDynamic,
				Selector: &LabelSelector{Allow: map[string]string{"env": "prod"}},
			},
		},
		{
			name: "dynamic rejects members",
			pool: &Pool{
				Id:       "pool-1",
				Type:     PoolTypeDynamic,
				Selector: &LabelSelector{Allow: map[string]string{"env": "prod"}},
				Members:  []PoolMember{{ProxyId: "p1", Weight: 1}},
			},
			wantErr: `dynamic pool "pool-1" must not have members`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pool.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestPoolMemberProxyIds(t *testing.T) {
	p := &Pool{Members: []PoolMember{{ProxyId: "a", Weight: 1}, {ProxyId: "b", Weight: 2}}}
	assert.Equal(t, []string{"a", "b"}, p.MemberProxyIds())
}
