package net

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startTCPStub(t *testing.T) (addr string, port string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	addr = ln.Addr().String()
	_, port, err = net.SplitHostPort(addr)
	require.NoError(t, err)
	return addr, port
}

func TestDialWithDNSCache_HostnameResolvedViaCache(t *testing.T) {
	t.Parallel()

	_, port := startTCPStub(t)
	var lookups int
	cache := newDNSCache(time.Minute, func(_ context.Context, host string) ([]net.IP, error) {
		lookups++
		assert.Equal(t, "proxy.example", host)
		return []net.IP{net.IPv4(127, 0, 0, 1)}, nil
	})
	base := &net.Dialer{Timeout: time.Second}

	conn, err := dialWithDNSCache(context.Background(), base, cache, "tcp", net.JoinHostPort("proxy.example", port))
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	assert.Equal(t, 1, lookups)

	// Second dial hits cache.
	conn, err = dialWithDNSCache(context.Background(), base, cache, "tcp", net.JoinHostPort("proxy.example", port))
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	assert.Equal(t, 1, lookups)
}

func TestDialWithDNSCache_LiteralIPSkipsLookup(t *testing.T) {
	t.Parallel()

	addr, _ := startTCPStub(t)
	var lookups int
	cache := newDNSCache(time.Minute, func(_ context.Context, _ string) ([]net.IP, error) {
		lookups++
		return nil, errors.New("should not lookup")
	})
	base := &net.Dialer{Timeout: time.Second}

	conn, err := dialWithDNSCache(context.Background(), base, cache, "tcp", addr)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	assert.Equal(t, 0, lookups)
}

func TestDialWithDNSCache_InvalidAddressFallsThrough(t *testing.T) {
	t.Parallel()

	cache := newDNSCache(time.Minute, func(_ context.Context, _ string) ([]net.IP, error) {
		t.Fatal("lookup should not run")
		return nil, nil
	})
	base := &net.Dialer{Timeout: 50 * time.Millisecond}

	_, err := dialWithDNSCache(context.Background(), base, cache, "tcp", "not-a-host-port")
	require.Error(t, err)
}

func TestDialWithDNSCache_LookupError(t *testing.T) {
	t.Parallel()

	cache := newDNSCache(time.Minute, func(_ context.Context, _ string) ([]net.IP, error) {
		return nil, errors.New("nxdomain")
	})
	base := &net.Dialer{Timeout: time.Second}

	_, err := dialWithDNSCache(context.Background(), base, cache, "tcp", "proxy.example:8080")
	require.EqualError(t, err, "nxdomain")
}

func TestDialWithDNSCache_EmptyAddrs(t *testing.T) {
	t.Parallel()

	cache := newDNSCache(time.Minute, func(_ context.Context, _ string) ([]net.IP, error) {
		return nil, nil
	})
	base := &net.Dialer{Timeout: time.Second}

	_, err := dialWithDNSCache(context.Background(), base, cache, "tcp", "proxy.example:8080")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no addresses")
}

func TestDialWithDNSCache_TriesNextAddrOnDialFailure(t *testing.T) {
	t.Parallel()

	_, port := startTCPStub(t)
	cache := newDNSCache(time.Minute, func(_ context.Context, _ string) ([]net.IP, error) {
		// TEST-NET-1 fails with short timeout; loopback hits the stub.
		return []net.IP{
			net.IPv4(192, 0, 2, 1),
			net.IPv4(127, 0, 0, 1),
		}, nil
	})
	base := &net.Dialer{Timeout: 200 * time.Millisecond}

	conn, err := dialWithDNSCache(context.Background(), base, cache, "tcp", net.JoinHostPort("proxy.example", port))
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

func TestDialWithDNSCache_AllDialFailures(t *testing.T) {
	t.Parallel()

	cache := newDNSCache(time.Minute, func(_ context.Context, _ string) ([]net.IP, error) {
		return []net.IP{net.IPv4(127, 0, 0, 1)}, nil
	})
	base := &net.Dialer{Timeout: 50 * time.Millisecond}

	_, err := dialWithDNSCache(context.Background(), base, cache, "tcp", "proxy.example:1")
	require.Error(t, err)
}

func TestContextDialer(t *testing.T) {
	t.Parallel()

	var gotNet, gotAddr string
	d := contextDialer(func(_ context.Context, network, address string) (net.Conn, error) {
		gotNet, gotAddr = network, address
		return nil, io.EOF
	})

	_, err := d.Dial("tcp", "host:9")
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, "tcp", gotNet)
	assert.Equal(t, "host:9", gotAddr)

	_, err = d.DialContext(context.Background(), "udp", "other:8")
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, "udp", gotNet)
	assert.Equal(t, "other:8", gotAddr)
}

func TestNewDNSCache_DefaultLookupResolvesLocalhost(t *testing.T) {
	t.Parallel()

	c := newDNSCache(time.Minute, nil)
	addrs, err := c.LookupIP(context.Background(), "localhost")
	require.NoError(t, err)
	require.NotEmpty(t, addrs)
}
