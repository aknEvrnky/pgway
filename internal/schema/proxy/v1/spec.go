package v1

import (
	"fmt"
)

type ProxySpecV1 struct {
	URL string `yaml:"url,omitempty" json:"url,omitempty"`

	Protocol string    `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	Host     string    `yaml:"host,omitempty" json:"host,omitempty"`
	Port     uint16    `yaml:"port,omitempty" json:"port,omitempty"`
	Auth     *AuthSpec `yaml:"auth,omitempty" json:"auth,omitempty"`
}

type AuthSpec struct {
	User string `yaml:"user" json:"user"`
	Pass string `yaml:"pass" json:"pass"`
}

func (s ProxySpecV1) Validate() error {
	if s.URL != "" {
		return nil
	}

	if s.Protocol == "" {
		return fmt.Errorf("spec.protocol is required when url is not provided")
	}
	if s.Host == "" {
		return fmt.Errorf("spec.host is required when url is not provided")
	}
	if s.Port == 0 {
		return fmt.Errorf("spec.port is required when url is not provided")
	}

	return nil
}
