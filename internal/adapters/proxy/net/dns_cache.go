package net

import (
	"context"
	"net"
	"sync"
	"time"
)

type cacheEntry struct {
	addrs     []net.IP
	expiresAt time.Time
}

// dnsCache is a fixed-TTL hostname → address cache for upstream dials.
type dnsCache struct {
	ttl    time.Duration
	lookup func(ctx context.Context, host string) ([]net.IP, error)
	now    func() time.Time
	cache  sync.Map
}

func newDNSCache(ttl time.Duration, lookup func(ctx context.Context, host string) ([]net.IP, error)) *dnsCache {
	if lookup == nil {
		lookup = func(ctx context.Context, host string) ([]net.IP, error) {
			addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			out := make([]net.IP, 0, len(addrs))
			for _, a := range addrs {
				out = append(out, a.IP)
			}
			return out, nil
		}
	}
	return &dnsCache{
		ttl:    ttl,
		lookup: lookup,
		now:    time.Now,
	}
}

// LookupIP returns cached addresses when fresh, otherwise looks up and stores.
// Literal IP hosts are returned without consulting the cache or lookup.
func (c *dnsCache) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}

	now := c.now()
	if v, ok := c.cache.Load(host); ok {
		e := v.(cacheEntry)
		if now.Before(e.expiresAt) {
			return e.addrs, nil
		}
	}

	addrs, err := c.lookup(ctx, host)
	if err != nil {
		return nil, err
	}

	stored := append([]net.IP(nil), addrs...)
	c.cache.Store(host, cacheEntry{
		addrs:     stored,
		expiresAt: now.Add(c.ttl),
	})
	return stored, nil
}
