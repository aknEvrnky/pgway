package net

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Dial_UnsupportedProtocol(t *testing.T) {
	t.Parallel()

	a := NewAdapter(TransportConfig{
		MaxIdleConnsPerHost: 2,
		DialTimeout:         time.Second,
		DNSCacheEnabled:     true,
		DNSCacheTTL:         time.Minute,
	})
	p := &domain.Proxy{Id: "x", Protocol: "ftp", Host: "127.0.0.1", Port: 1}

	_, err := a.Dial(context.Background(), p, "example.com:443")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported proxy protocol")
}

func TestAdapter_DialHTTPProxy_CONNECTOK(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		req, err := http.ReadRequest(bufio.NewReader(c))
		if err != nil {
			return
		}
		assert.Equal(t, http.MethodConnect, req.Method)
		_, _ = fmt.Fprintf(c, "HTTP/1.1 200 Connection Established\r\n\r\n")
	}()

	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	var port uint16
	_, err = fmt.Sscanf(portStr, "%d", &port)
	require.NoError(t, err)

	a := NewAdapter(TransportConfig{
		MaxIdleConnsPerHost: 2,
		DialTimeout:         time.Second,
		DNSCacheEnabled:     true,
		DNSCacheTTL:         time.Minute,
	})
	p := &domain.Proxy{
		Id:       "http-proxy",
		Protocol: domain.ProtocolHTTP,
		Host:     "127.0.0.1",
		Port:     port,
	}

	conn, err := a.Dial(context.Background(), p, "example.com:443")
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

func TestAdapter_DialHTTPProxy_DialError(t *testing.T) {
	t.Parallel()

	a := NewAdapter(TransportConfig{
		MaxIdleConnsPerHost: 2,
		DialTimeout:         50 * time.Millisecond,
		DNSCacheEnabled:     true,
		DNSCacheTTL:         time.Minute,
	})
	p := &domain.Proxy{
		Id:       "down",
		Protocol: domain.ProtocolHTTP,
		Host:     "127.0.0.1",
		Port:     1,
	}

	_, err := a.Dial(context.Background(), p, "example.com:443")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dial proxy")
}

func TestAdapter_DialSOCKS5_DialError(t *testing.T) {
	t.Parallel()

	a := NewAdapter(TransportConfig{
		MaxIdleConnsPerHost: 2,
		DialTimeout:         50 * time.Millisecond,
		DNSCacheEnabled:     true,
		DNSCacheTTL:         time.Minute,
	})
	p := &domain.Proxy{
		Id:       "socks",
		Protocol: domain.ProtocolSOCKS5,
		Host:     "127.0.0.1",
		Port:     1,
	}

	_, err := a.Dial(context.Background(), p, "example.com:80")
	require.Error(t, err)
}
