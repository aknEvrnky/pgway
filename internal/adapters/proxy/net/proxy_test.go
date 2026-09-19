package net

import (
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdapter_TransportPoolParams(t *testing.T) {
	t.Parallel()

	cfg := TransportConfig{
		MaxIdleConns:        1024,
		MaxIdleConnsPerHost: 128,
		IdleConnTimeout:     90 * time.Second,
		DialTimeout:         10 * time.Second,
	}
	a := NewAdapter(cfg)

	p, err := domain.NewProxyFromURL("http://127.0.0.1:8080")
	require.NoError(t, err)
	p.Id = "proxy-1"

	tr := a.transport(p)
	assert.Equal(t, 1024, tr.MaxIdleConns)
	assert.Equal(t, 128, tr.MaxIdleConnsPerHost)
	assert.Equal(t, 90*time.Second, tr.IdleConnTimeout)
	require.NotNil(t, tr.DialContext)

	assert.Equal(t, 10*time.Second, a.dialer.Timeout)
	assert.Equal(t, dialKeepAlive, a.dialer.KeepAlive)

	// Same proxy id reuses the cached transport.
	tr2 := a.transport(p)
	assert.Same(t, tr, tr2)
}

func TestNewAdapter_UnlimitedIdleConns(t *testing.T) {
	t.Parallel()

	a := NewAdapter(TransportConfig{
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: 64,
		IdleConnTimeout:     30 * time.Second,
		DialTimeout:         5 * time.Second,
	})

	p, err := domain.NewProxyFromURL("http://127.0.0.1:8080")
	require.NoError(t, err)
	p.Id = "proxy-2"

	tr := a.transport(p)
	assert.Equal(t, 0, tr.MaxIdleConns)
	assert.Equal(t, 64, tr.MaxIdleConnsPerHost)
}
