package net

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
	"golang.org/x/net/proxy"
)

// dialKeepAlive is the TCP keepalive period for upstream dials (not configurable).
const dialKeepAlive = 30 * time.Second

// TransportConfig configures per-proxy http.Transport pool and dialer settings.
type TransportConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	DialTimeout         time.Duration
	// DNSCacheEnabled turns on fixed-TTL hostname caching for upstream dials.
	DNSCacheEnabled bool
	// DNSCacheTTL is how long successful lookups are reused when DNSCacheEnabled.
	DNSCacheTTL time.Duration
}

// Adapter is a secondary adapter that opens connections to upstream proxies.
// Each proxy gets its own http.Transport, isolating connection pools.
type Adapter struct {
	transports  sync.Map
	cfg         TransportConfig
	dialer      *net.Dialer
	dialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

func NewAdapter(cfg TransportConfig) *Adapter {
	base := &net.Dialer{
		Timeout:   cfg.DialTimeout,
		KeepAlive: dialKeepAlive,
	}

	a := &Adapter{
		cfg:    cfg,
		dialer: base,
	}

	if cfg.DNSCacheEnabled {
		cache := newDNSCache(cfg.DNSCacheTTL, nil)
		a.dialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialWithDNSCache(ctx, base, cache, network, address)
		}
	} else {
		a.dialContext = base.DialContext
	}

	return a
}

// dialWithDNSCache resolves hostnames via cache then dials; literal IPs dial directly.
func dialWithDNSCache(ctx context.Context, base *net.Dialer, cache *dnsCache, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return base.DialContext(ctx, network, address)
	}
	if net.ParseIP(host) != nil {
		return base.DialContext(ctx, network, address)
	}

	ips, err := cache.LookupIP(ctx, host)
	if err != nil {
		return nil, err
	}

	var firstErr error
	for _, ip := range ips {
		conn, dialErr := base.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		if firstErr == nil {
			firstErr = dialErr
		}
	}
	if firstErr == nil {
		return nil, fmt.Errorf("dns cache: no addresses for %s", host)
	}
	return nil, firstErr
}

// contextDialer adapts a DialContext func to proxy.Dialer / proxy.ContextDialer.
type contextDialer func(ctx context.Context, network, address string) (net.Conn, error)

func (d contextDialer) Dial(network, address string) (net.Conn, error) {
	return d(context.Background(), network, address)
}

func (d contextDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d(ctx, network, address)
}

// transport returns the http.Transport for the given proxy, creating one if it doesn't exist.
func (a *Adapter) transport(p *domain.Proxy) *http.Transport {
	t := &http.Transport{
		Proxy:               http.ProxyURL(p.URL()),
		MaxIdleConns:        a.cfg.MaxIdleConns,
		MaxIdleConnsPerHost: a.cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     a.cfg.IdleConnTimeout,
		DialContext:         a.dialContext,
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
	conn, err := a.dialContext(ctx, "tcp", p.Addr())

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
		_ = conn.Close()
		return nil, fmt.Errorf("write CONNECT request: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("read CONNECT response: %w", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
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

	dialer, err := proxy.SOCKS5("tcp", p.Addr(), auth, contextDialer(a.dialContext))
	if err != nil {
		return nil, fmt.Errorf("create socks5 dialer: %w", err)
	}

	if cd, ok := dialer.(proxy.ContextDialer); ok {
		return cd.DialContext(ctx, "tcp", target)
	}
	return dialer.Dial("tcp", target)
}
