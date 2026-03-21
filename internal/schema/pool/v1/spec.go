package v1

import "fmt"

type PoolSpecV1 struct {
	Title    string        `yaml:"title,omitempty" json:"title,omitempty"`
	Type     string        `yaml:"type" json:"type"`
	ProxyIds []string      `yaml:"proxy_ids,omitempty" json:"proxy_ids,omitempty"`
	Selector *SelectorSpec `yaml:"selector,omitempty" json:"selector,omitempty"`
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
		if len(s.ProxyIds) == 0 {
			return fmt.Errorf("spec.proxy_ids is required for static pool")
		}
		if s.Selector != nil {
			return fmt.Errorf("spec.selector must not be set for static pool")
		}
	case "dynamic":
		if s.Selector == nil || len(s.Selector.Allow) == 0 {
			return fmt.Errorf("spec.selector.allow is required for dynamic pool")
		}
		if len(s.ProxyIds) > 0 {
			return fmt.Errorf("spec.proxy_ids must not be set for dynamic pool")
		}
	}

	return nil
}
