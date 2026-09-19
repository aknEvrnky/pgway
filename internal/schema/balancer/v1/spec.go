package v1

import (
	"fmt"
	"time"

	"github.com/aknEvrnky/pgway/internal/application/core/domain"
)

type BalancerSpecV1 struct {
	Title         string `yaml:"title,omitempty" json:"title,omitempty"`
	Type          string `yaml:"type" json:"type"`
	PoolId        string `yaml:"pool_id" json:"pool_id"`
	ResetInterval string `yaml:"reset_interval,omitempty" json:"reset_interval,omitempty"`
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

	if s.ResetInterval != "" {
		if s.Type != "least-bytes" {
			return fmt.Errorf("spec.reset_interval is only valid for type least-bytes")
		}
		d, err := time.ParseDuration(s.ResetInterval)
		if err != nil {
			return fmt.Errorf("spec.reset_interval: %w", err)
		}
		if d <= 0 {
			return fmt.Errorf("spec.reset_interval must be > 0")
		}
	}

	return nil
}

// ResolvedResetInterval returns the effective reset interval
// (default domain.DefaultLeastBytesResetInterval).
func (s BalancerSpecV1) ResolvedResetInterval() time.Duration {
	if s.ResetInterval == "" {
		return domain.DefaultLeastBytesResetInterval
	}
	d, err := time.ParseDuration(s.ResetInterval)
	if err != nil || d <= 0 {
		return domain.DefaultLeastBytesResetInterval
	}
	return d
}
