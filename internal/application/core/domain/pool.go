package domain

import "fmt"

type PoolType string

const (
	PoolTypeStatic  PoolType = "static"
	PoolTypeDynamic PoolType = "dynamic"
)

func (t PoolType) IsValid() bool {
	return t == PoolTypeStatic || t == PoolTypeDynamic
}

type LabelSelector struct {
	Allow map[string]string `json:"allow,omitempty"`
}

// PoolMember is a static-pool entry with a selection weight.
type PoolMember struct {
	ProxyId string `json:"proxy_id"`
	Weight  int    `json:"weight"`
}

type Pool struct {
	Timestamps
	Id     string            `json:"id"`
	Title  string            `json:"title"`
	Type   PoolType          `json:"type"`
	Labels map[string]string `json:"labels"`

	// Static pool
	Members []PoolMember `json:"members,omitempty"`

	// Dynamic pool
	Selector *LabelSelector `json:"selector,omitempty"`

	hasProxiesResolved bool
	resolvedProxies    []*Proxy
}

// MemberProxyIds returns proxy IDs from static members in order.
func (p *Pool) MemberProxyIds() []string {
	ids := make([]string, 0, len(p.Members))
	for _, m := range p.Members {
		ids = append(ids, m.ProxyId)
	}
	return ids
}

func (p *Pool) LoadResolvedProxies(proxies []*Proxy) {
	p.hasProxiesResolved = true
	p.resolvedProxies = proxies
}

func (p *Pool) HasProxiesResolved() bool {
	return p.hasProxiesResolved
}

func (p *Pool) ResolvedProxies() []*Proxy {
	return p.resolvedProxies
}

func (p *Pool) Validate() error {
	if p.Id == "" {
		return fmt.Errorf("pool id is required")
	}

	if !p.Type.IsValid() {
		return fmt.Errorf("invalid pool type: %q", p.Type)
	}

	switch p.Type {
	case PoolTypeStatic:
		if len(p.Members) == 0 {
			return fmt.Errorf("static pool %q requires at least one member", p.Id)
		}
		if p.Selector != nil {
			return fmt.Errorf("static pool %q must not have selector", p.Id)
		}
		seen := make(map[string]struct{}, len(p.Members))
		for i, m := range p.Members {
			if m.ProxyId == "" {
				return fmt.Errorf("static pool %q member[%d]: proxy_id is required", p.Id, i)
			}
			if m.Weight < 1 {
				return fmt.Errorf("static pool %q member %q: weight must be >= 1", p.Id, m.ProxyId)
			}
			if _, ok := seen[m.ProxyId]; ok {
				return fmt.Errorf("static pool %q has duplicate proxy_id %q", p.Id, m.ProxyId)
			}
			seen[m.ProxyId] = struct{}{}
		}

	case PoolTypeDynamic:
		if p.Selector == nil || len(p.Selector.Allow) == 0 {
			return fmt.Errorf("dynamic pool %q requires selector with at least one allow label", p.Id)
		}
		if len(p.Members) > 0 {
			return fmt.Errorf("dynamic pool %q must not have members", p.Id)
		}
	}

	return nil
}
