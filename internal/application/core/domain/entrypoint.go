package domain

import "fmt"

type Entrypoint struct {
	Id       string   `json:"id"`
	Title    string   `json:"title"`
	Protocol Protocol `json:"protocol"`
	Host     string   `json:"host"`
	Port     uint16   `json:"port"`
	FlowId   string   `json:"flow_id"`
}

func (e *Entrypoint) ListenAddr() string {
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

func (e *Entrypoint) Validate() error {
	if !e.Protocol.IsValid() {
		return fmt.Errorf("invalid protocol: %q", e.Protocol)
	}

	if e.Host == "" {
		return fmt.Errorf("host is required")
	}

	if e.Port == 0 {
		return fmt.Errorf("port is required")
	}

	return nil
}
