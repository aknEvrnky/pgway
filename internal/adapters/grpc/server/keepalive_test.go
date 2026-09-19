package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerKeepaliveParams_Disabled(t *testing.T) {
	t.Parallel()

	for _, interval := range []time.Duration{0, -time.Second} {
		_, _, ok := serverKeepaliveParams(KeepaliveConfig{Interval: interval, Timeout: 20 * time.Second})
		assert.False(t, ok)
		assert.Nil(t, keepaliveServerOptions(KeepaliveConfig{Interval: interval, Timeout: 20 * time.Second}))
	}
}

func TestServerKeepaliveParams_Enabled(t *testing.T) {
	t.Parallel()

	params, policy, ok := serverKeepaliveParams(KeepaliveConfig{
		Interval: time.Minute,
		Timeout:  20 * time.Second,
	})
	require.True(t, ok)
	assert.Equal(t, time.Minute, params.Time)
	assert.Equal(t, 20*time.Second, params.Timeout)
	assert.Equal(t, keepaliveMaxConnectionAge, params.MaxConnectionAge)
	assert.Equal(t, keepaliveEnforcementMinTime, policy.MinTime)

	opts := keepaliveServerOptions(KeepaliveConfig{Interval: time.Minute, Timeout: 20 * time.Second})
	assert.Len(t, opts, 2)
}
