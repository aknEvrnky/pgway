package v1

import "fmt"

type EntrypointSpecV1 struct {
	Title    string `yaml:"title,omitempty" json:"title,omitempty"`
	Protocol string `yaml:"protocol" json:"protocol"`
	Host     string `yaml:"host" json:"host"`
	Port     uint16 `yaml:"port" json:"port"`
	FlowId   string `yaml:"flow_id" json:"flow_id"`
}

func (s EntrypointSpecV1) Validate() error {
	if s.Protocol == "" {
		return fmt.Errorf("spec.protocol is required")
	}
	if s.Host == "" {
		return fmt.Errorf("spec.host is required")
	}
	if s.Port == 0 {
		return fmt.Errorf("spec.port is required")
	}
	if s.FlowId == "" {
		return fmt.Errorf("spec.flow_id is required")
	}
	return nil
}
