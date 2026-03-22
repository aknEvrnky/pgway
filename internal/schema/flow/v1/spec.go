package v1

import "fmt"

type FlowSpecV1 struct {
	RouterId   string `yaml:"router_id,omitempty" json:"router_id,omitempty"`
	BalancerId string `yaml:"balancer_id,omitempty" json:"balancer_id,omitempty"`
}

func (s FlowSpecV1) Validate() error {
	if s.RouterId == "" && s.BalancerId == "" {
		return fmt.Errorf("spec must define router_id or balancer_id")
	}
	return nil
}
