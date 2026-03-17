package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntrypoint_ListenAddr(t *testing.T) {
	ep := Entrypoint{
		Host: "0.0.0.0",
		Port: 443,
	}

	listenAddr := ep.ListenAddr()
	assert.Equal(t, "0.0.0.0:443", listenAddr)
}

func TestEntrypoint_Validate(t *testing.T) {
	for _, tt := range []struct {
		name    string
		ep      Entrypoint
		wantErr string
	}{
		{
			name: "Valid",
			ep: Entrypoint{
				Protocol: "https",
				Host:     "127.0.0.1",
				Port:     443,
			},
		},
		{
			name: "invalid host",
			ep: Entrypoint{
				Protocol: "socks5",
				Host:     "",
				Port:     8080,
			},
			wantErr: "host is required",
		},
		{
			name: "invalid protocol",
			ep: Entrypoint{
				Protocol: "socks4",
				Host:     "0.0.0.0",
				Port:     3030,
			},
			wantErr: `invalid protocol: "socks4"`,
		},
		{
			name: "invalid port",
			ep: Entrypoint{
				Protocol: "socks5",
				Host:     "0.0.0.0",
				Port:     0,
			},
			wantErr: "port is required",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.ep.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, isValid)
			} else {
				assert.EqualError(t, isValid, tt.wantErr)
			}
		})
	}
}
