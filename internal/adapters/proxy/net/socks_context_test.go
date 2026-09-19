package net

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/require"
)

// blackholeSOCKS listens and accepts but never completes a SOCKS handshake.
// DialContext must observe cancel/deadline; plain Dial would hang indefinitely.
func blackholeSOCKS(t *testing.T) (addr string) {
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
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1)
				for {
					if _, err := c.Read(buf); err != nil {
						return
					}
				}
			}(c)
		}
	}()

	return ln.Addr().String()
}

func socksProxyAt(t *testing.T, hostPort string) *domain.Proxy {
	t.Helper()
	host, portStr, err := net.SplitHostPort(hostPort)
	require.NoError(t, err)
	port, err := strconv.ParseUint(portStr, 10, 16)
	require.NoError(t, err)
	return &domain.Proxy{
		Id:       "socks-hang",
		Protocol: domain.ProtocolSOCKS5,
		Host:     host,
		Port:     uint16(port),
	}
}

func TestAdapter_DialSOCKS5_ContextCanceled(t *testing.T) {
	t.Parallel()

	addr := blackholeSOCKS(t)
	a := NewAdapter(TransportConfig{DialTimeout: 30 * time.Second})
	p := socksProxyAt(t, addr)
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		_, err := a.Dial(ctx, p, "example.com:80")
		errCh <- err
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		// x/net surfaces cancel as a connection deadline / i/o timeout, not always
		// errors.Is(context.Canceled). What matters: Dial returns promptly.
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Dial did not return after cancel (DialContext not used?)")
	}
}

func TestAdapter_DialSOCKS5_DeadlineExceeded(t *testing.T) {
	t.Parallel()

	addr := blackholeSOCKS(t)
	a := NewAdapter(TransportConfig{DialTimeout: 30 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := a.Dial(ctx, socksProxyAt(t, addr), "example.com:80")
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Less(t, elapsed, 5*time.Second, "Dial ignored context deadline")
}
