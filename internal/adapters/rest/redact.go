package rest

import "github.com/aknEvrnky/pgway/internal/application/core/domain"

// redactProxy returns a shallow copy safe for JSON responses (password cleared).
func redactProxy(p *domain.Proxy) *domain.Proxy {
	if p == nil {
		return nil
	}
	out := *p
	if p.Auth != nil {
		auth := *p.Auth
		if auth.Pass != "" {
			auth.Pass = ""
		}
		out.Auth = &auth
	}
	return &out
}

func redactProxies(items []*domain.Proxy) []*domain.Proxy {
	out := make([]*domain.Proxy, len(items))
	for i := range items {
		out[i] = redactProxy(items[i])
	}
	return out
}
