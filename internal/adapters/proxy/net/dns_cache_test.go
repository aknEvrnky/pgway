package net

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSCache_HitSkipsLookup(t *testing.T) {
	t.Parallel()

	var calls int
	c := newDNSCache(time.Minute, func(_ context.Context, host string) ([]net.IP, error) {
		calls++
		assert.Equal(t, "proxy.example", host)
		return []net.IP{net.IPv4(1, 2, 3, 4)}, nil
	})

	a1, err := c.LookupIP(context.Background(), "proxy.example")
	require.NoError(t, err)
	a2, err := c.LookupIP(context.Background(), "proxy.example")
	require.NoError(t, err)

	assert.Equal(t, 1, calls)
	assert.Equal(t, a1, a2)
	require.Len(t, a1, 1)
	assert.True(t, a1[0].Equal(net.IPv4(1, 2, 3, 4)))
}

func TestDNSCache_ExpireTriggersLookup(t *testing.T) {
	t.Parallel()

	var calls int
	now := time.Unix(1000, 0)
	c := newDNSCache(time.Second, func(_ context.Context, _ string) ([]net.IP, error) {
		calls++
		return []net.IP{net.IPv4(9, 9, 9, byte(calls))}, nil
	})
	c.now = func() time.Time { return now }

	_, err := c.LookupIP(context.Background(), "proxy.example")
	require.NoError(t, err)
	assert.Equal(t, 1, calls)

	now = now.Add(2 * time.Second)
	addrs, err := c.LookupIP(context.Background(), "proxy.example")
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	require.Len(t, addrs, 1)
	assert.True(t, addrs[0].Equal(net.IPv4(9, 9, 9, 2)))
}

func TestDNSCache_LiteralIPBypassesLookup(t *testing.T) {
	t.Parallel()

	var calls int
	c := newDNSCache(time.Minute, func(_ context.Context, _ string) ([]net.IP, error) {
		calls++
		return nil, errors.New("should not be called")
	})

	addrs, err := c.LookupIP(context.Background(), "127.0.0.1")
	require.NoError(t, err)
	assert.Equal(t, 0, calls)
	require.Len(t, addrs, 1)
	assert.True(t, addrs[0].Equal(net.IPv4(127, 0, 0, 1)))
}

func TestDNSCache_LookupErrorNotCached(t *testing.T) {
	t.Parallel()

	var calls int
	c := newDNSCache(time.Minute, func(_ context.Context, _ string) ([]net.IP, error) {
		calls++
		return nil, errors.New("nxdomain")
	})

	_, err := c.LookupIP(context.Background(), "missing.example")
	require.Error(t, err)
	_, err = c.LookupIP(context.Background(), "missing.example")
	require.Error(t, err)
	assert.Equal(t, 2, calls)
}
