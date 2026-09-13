package v1

import "fmt"

type PoolMemberSpec struct {
	ProxyId string `yaml:"proxy_id" json:"proxy_id"`
	// Weight is optional; nil means default 1. Explicit values must be >= 1.
	Weight *int `yaml:"weight,omitempty" json:"weight,omitempty"`
}

type PoolSpecV1 struct {
	Title    string           `yaml:"title,omitempty" json:"title,omitempty"`
	Type     string           `yaml:"type" json:"type"`
	Members  []PoolMemberSpec `yaml:"members,omitempty" json:"members,omitempty"`
	Selector *SelectorSpec    `yaml:"selector,omitempty" json:"selector,omitempty"`
}

type SelectorSpec struct {
	Allow map[string]string `yaml:"allow,omitempty" json:"allow,omitempty"`
}

func (s PoolSpecV1) Validate() error {
	if s.Type == "" {
		return fmt.Errorf("spec.type is required")
	}

	if s.Type != "static" && s.Type != "dynamic" {
		return fmt.Errorf("spec.type must be \"static\" or \"dynamic\", got %q", s.Type)
	}

	switch s.Type {
	case "static":
		if len(s.Members) == 0 {
			return fmt.Errorf("spec.members is required for static pool")
		}
		if s.Selector != nil {
			return fmt.Errorf("spec.selector must not be set for static pool")
		}
		seen := make(map[string]struct{}, len(s.Members))
		for i, m := range s.Members {
			if m.ProxyId == "" {
				return fmt.Errorf("spec.members[%d].proxy_id is required", i)
			}
			if m.Weight != nil && *m.Weight < 1 {
				return fmt.Errorf("spec.members[%d].weight must be >= 1", i)
			}
			if _, ok := seen[m.ProxyId]; ok {
				return fmt.Errorf("spec.members has duplicate proxy_id %q", m.ProxyId)
			}
			seen[m.ProxyId] = struct{}{}
		}
	case "dynamic":
		if s.Selector == nil || len(s.Selector.Allow) == 0 {
			return fmt.Errorf("spec.selector.allow is required for dynamic pool")
		}
		if len(s.Members) > 0 {
			return fmt.Errorf("spec.members must not be set for dynamic pool")
		}
	}

	return nil
}

// ResolvedWeight returns the effective weight (default 1).
func (m PoolMemberSpec) ResolvedWeight() int {
	if m.Weight == nil {
		return 1
	}
	return *m.Weight
}
