package domain

import "fmt"

type Proxy struct {
	Id       string     `json:"id"`
	Protocol Protocol   `json:"protocol"`
	Host     string     `json:"host"`
	Port     uint16     `json:"port"`
	Auth     *BasicAuth `json:"auth,omitempty"`
}

type BasicAuth struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
}

func (p *Proxy) Addr() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

func (p *Proxy) HasAuth() bool {
	return p.Auth != nil && p.Auth.User != ""
}

func (p *Proxy) Validate() error {
	if !p.Protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %q", p.Protocol)
	}

	if p.Host == "" {
		return fmt.Errorf("host is required")
	}

	if p.Port == 0 {
		return fmt.Errorf("port is required")
	}

	return nil
}
