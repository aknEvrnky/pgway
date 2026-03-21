package domain

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Proxy struct {
	Id       string            `json:"id"`
	Protocol Protocol          `json:"protocol"`
	Host     string            `json:"host"`
	Port     uint16            `json:"port"`
	Auth     *BasicAuth        `json:"auth,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
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

func (p *Proxy) URL() *url.URL {
	u := &url.URL{
		Scheme: string(p.Protocol),
		Host:   p.Addr(),
	}

	if p.HasAuth() {
		u.User = url.UserPassword(p.Auth.User, p.Auth.Pass)
	}

	return u
}

func NewProxyFromURL(str string) (*Proxy, error) {
	// add the schema if it does not exist
	if !strings.Contains(str, "://") {
		str = string(DefaultProtocol) + str
	}

	parsed, err := url.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("url parse: %w", err)
	}

	host, portStr, err := net.SplitHostPort(parsed.Host)

	if err != nil {
		return nil, fmt.Errorf("parsing host:port: %w", err)
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("port parsing: %w", err)
	}

	proxy := &Proxy{
		Protocol: Protocol(parsed.Scheme),
		Host:     host,
		Port:     uint16(port),
	}

	if parsed.User != nil {
		pass, _ := parsed.User.Password()

		auth := &BasicAuth{
			User: parsed.User.Username(),
			Pass: pass,
		}

		proxy.Auth = auth
	}

	return proxy, nil
}
