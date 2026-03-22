package v1

import "fmt"

type BalancerSpecV1 struct {
	Title  string `yaml:"title,omitempty" json:"title,omitempty"`
	Type   string `yaml:"type" json:"type"`
	PoolId string `yaml:"pool_id" json:"pool_id"`
}

func (s BalancerSpecV1) Validate() error {
	if s.Type == "" {
		return fmt.Errorf("spec.type is required")
	}

	validTypes := map[string]bool{
		"round-robin": true,
		"weighted":    true,
		"least-bytes": true,
	}
	if !validTypes[s.Type] {
		return fmt.Errorf("spec.type must be one of round-robin, weighted, least-bytes; got %q", s.Type)
	}

	if s.PoolId == "" {
		return fmt.Errorf("spec.pool_id is required")
	}

	return nil
}
