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

type Pool struct {
	Id     string            `json:"id"`
	Title  string            `json:"title"`
	Type   PoolType          `json:"type"`
	Labels map[string]string `json:"labels"`

	// Static pool
	ProxyIds []string `json:"proxy_ids,omitempty"`

	// Dynamic pool
	Selector *LabelSelector `json:"selector,omitempty"`

	//Proxies []*Proxy `json:"proxies"`

	hasProxiesResolved bool
	resolvedProxies    []*Proxy
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
		if len(p.ProxyIds) == 0 {
			return fmt.Errorf("static pool %q requires at least one proxy_id", p.Id)
		}
		if p.Selector != nil {
			return fmt.Errorf("static pool %q must not have selector", p.Id)
		}

	case PoolTypeDynamic:
		if p.Selector == nil || len(p.Selector.Allow) == 0 {
			return fmt.Errorf("dynamic pool %q requires selector with at least one allow label", p.Id)
		}
		if len(p.ProxyIds) > 0 {
			return fmt.Errorf("dynamic pool %q must not have proxy_ids", p.Id)
		}
	}

	return nil
}
