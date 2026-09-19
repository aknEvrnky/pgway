package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientKeepaliveParams_Disabled(t *testing.T) {
	t.Parallel()

	for _, interval := range []time.Duration{0, -time.Second} {
		_, ok := clientKeepaliveParams(KeepaliveConfig{Interval: interval, Timeout: 20 * time.Second})
		assert.False(t, ok)
		assert.Nil(t, keepaliveDialOptions(KeepaliveConfig{Interval: interval, Timeout: 20 * time.Second}))
	}
}

func TestClientKeepaliveParams_Enabled(t *testing.T) {
	t.Parallel()

	params, ok := clientKeepaliveParams(KeepaliveConfig{
		Interval: time.Minute,
		Timeout:  20 * time.Second,
	})
	require.True(t, ok)
	assert.Equal(t, time.Minute, params.Time)
	assert.Equal(t, 20*time.Second, params.Timeout)
	assert.True(t, params.PermitWithoutStream)

	opts := keepaliveDialOptions(KeepaliveConfig{Interval: time.Minute, Timeout: 20 * time.Second})
	assert.Len(t, opts, 1)
}
