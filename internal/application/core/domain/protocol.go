package domain

type Protocol string

const (
	ProtocolHTTP   Protocol = "http"
	ProtocolHTTPS  Protocol = "https"
	ProtocolSOCKS5 Protocol = "socks5"
)

const DefaultProtocol = ProtocolHTTP

func (p Protocol) IsValid() bool {
	switch p {
	case ProtocolHTTP, ProtocolHTTPS, ProtocolSOCKS5:
		return true
	}
	return false
}
