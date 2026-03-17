package net

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"golang.org/x/net/proxy"
)

// Adapter is a secondary adapter that opens connections to upstream proxies.
// Each proxy gets its own http.Transport, isolating connection pools and DNS caches.
type Adapter struct {
	transports sync.Map
}

func NewAdapter() *Adapter {
	return &Adapter{}
}

// transport returns the http.Transport for the given proxy, creating one if it doesn't exist.
func (a *Adapter) transport(p *domain.Proxy) *http.Transport {
	t := &http.Transport{
		Proxy: http.ProxyURL(p.URL()),
	}

	actual, _ := a.transports.LoadOrStore(p.Id, t)

	return actual.(*http.Transport)
}

// RoundTrip forwards an HTTP request through the upstream proxy.
func (a *Adapter) RoundTrip(ctx context.Context, p *domain.Proxy, r *http.Request) (*http.Response, error) {
	return a.transport(p).RoundTrip(r.WithContext(ctx))
}

// Dial opens a TCP connection through the upstream proxy for CONNECT tunneling.
func (a *Adapter) Dial(ctx context.Context, p *domain.Proxy, target string) (net.Conn, error) {
	switch p.Protocol {
	case domain.ProtocolHTTP, domain.ProtocolHTTPS:
		return a.dialHTTPProxy(ctx, p, target)
	case domain.ProtocolSOCKS5:
		return a.dialSOCKS5(ctx, p, target)
	default:
		return nil, fmt.Errorf("unsupported proxy protocol: %s", p.Protocol)
	}
}

// dialHTTPProxy opens a CONNECT tunnel through an HTTP proxy.
func (a *Adapter) dialHTTPProxy(ctx context.Context, p *domain.Proxy, target string) (net.Conn, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", p.Addr())

	if err != nil {
		return nil, fmt.Errorf("dial proxy %s: %w", p.Addr(), err)
	}

	req := &http.Request{
		Method: http.MethodConnect,
		URL:    p.URL(),
		Header: make(http.Header),
		Host:   target,
	}

	req.Header.Set("Host", target)
	req.Header.Set("Proxy-Connection", "keep-alive")

	if p.HasAuth() {
		credentials := base64.StdEncoding.EncodeToString(
			[]byte(p.Auth.User + ":" + p.Auth.Pass),
		)
		req.Header.Set("Proxy-Authorization", "Basic "+credentials)
	}

	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("write CONNECT request: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("read CONNECT response: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return nil, fmt.Errorf("proxy CONNECT failed: %s", resp.Status)
	}

	return conn, nil
}

// dialSOCKS5 opens a connection through a SOCKS5 proxy.
func (a *Adapter) dialSOCKS5(ctx context.Context, p *domain.Proxy, target string) (net.Conn, error) {
	var auth *proxy.Auth
	if p.HasAuth() {
		auth = &proxy.Auth{
			User:     p.Auth.User,
			Password: p.Auth.Pass,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", p.Addr(), auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("create socks5 dialer: %w", err)
	}

	return dialer.Dial("tcp", target)
}
