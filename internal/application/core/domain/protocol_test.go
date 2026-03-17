package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtocol_IsValid(t *testing.T) {
	for _, tt := range []struct {
		name         string
		protocolType string
		isValid      bool
	}{
		{
			name:         "HTTP Test",
			protocolType: "http",
			isValid:      true,
		},
		{
			name:         "HTTPS Test",
			protocolType: "https",
			isValid:      true,
		},
		{
			name:         "SOCKS5 Test",
			protocolType: "socks5",
			isValid:      true,
		},
		{
			name:         "Unsupported Protocol Test",
			protocolType: "socks4",
			isValid:      false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValid, Protocol(tt.protocolType).IsValid())
		})
	}
}
