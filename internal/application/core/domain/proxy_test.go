package domain

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProxyFromURL(t *testing.T) {
	var testCases = []struct {
		name          string
		proxyUrl      string
		expectedProxy *Proxy
		expectedError string
	}{
		{
			name:     "proxy url can be parsed",
			proxyUrl: "http://127.0.0.1:8080",
			expectedProxy: &Proxy{
				Protocol: "http",
				Host:     "127.0.0.1",
				Port:     8080,
				Auth:     nil,
			},
			expectedError: "",
		},
		{
			name:     "proxy url with auth can be parsed",
			proxyUrl: "https://admin@127.0.0.1:8080",
			expectedProxy: &Proxy{
				Protocol: "https",
				Host:     "127.0.0.1",
				Port:     8080,
				Auth: &BasicAuth{
					User: "admin",
				},
			},
			expectedError: "",
		},
		{
			name:     "proxy url with auth + pass can be parsed",
			proxyUrl: "http://user:pass@127.0.0.1:8080",
			expectedProxy: &Proxy{
				Protocol: "http",
				Host:     "127.0.0.1",
				Port:     8080,
				Auth: &BasicAuth{
					User: "user",
					Pass: "pass",
				},
			},
			expectedError: "",
		},
		{
			name:          "invalid proxy url can not be parsed",
			proxyUrl:      "https://user:pass:127.0.0.1:8080",
			expectedProxy: nil,
			expectedError: "parsing host:port:",
		},
		{
			name:          "invalid port can not be parsed",
			proxyUrl:      "socks5://127.0.0.1:9535635",
			expectedProxy: nil,
			expectedError: "port parsing:",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := NewProxyFromURL(tt.proxyUrl)

			if tt.expectedError != "" {
				assert.ErrorContains(t, err, tt.expectedError)
				assert.Nil(t, proxy)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, proxy)

			assert.Equal(t, tt.expectedProxy.Protocol, proxy.Protocol)
			assert.Equal(t, tt.expectedProxy.Host, proxy.Host)
			assert.Equal(t, tt.expectedProxy.Port, proxy.Port)
			assert.Equal(t, tt.expectedProxy.Auth, proxy.Auth)
		})
	}
}

func TestProxy_Addr(t *testing.T) {
	proxy := Proxy{
		Host: "172.16.0.1",
		Port: 8080,
	}

	assert.Equal(t, "172.16.0.1:8080", proxy.Addr())
}

func TestProxy_HasAuth(t *testing.T) {
	for _, tt := range []struct {
		name            string
		proxy           Proxy
		expectedHasAuth bool
	}{
		{
			name: "has no auth",
			proxy: Proxy{
				Protocol: "http",
				Host:     "172.16.0.1",
				Port:     8080,
				Auth:     nil,
			},
			expectedHasAuth: false,
		},
		{
			name: "has auth with username",
			proxy: Proxy{
				Protocol: "http",
				Host:     "172.16.0.1",
				Port:     8080,
				Auth: &BasicAuth{
					User: "admin",
				},
			},
			expectedHasAuth: true,
		},
		{
			name: "has auth with username and password",
			proxy: Proxy{
				Protocol: "http",
				Host:     "172.16.0.1",
				Port:     8080,
				Auth: &BasicAuth{
					User: "admin",
					Pass: "pass",
				},
			},
			expectedHasAuth: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedHasAuth, tt.proxy.HasAuth())
		})
	}
}

func TestProxy_Validate(t *testing.T) {
	for _, tt := range []struct {
		name        string
		proxy       Proxy
		expectedErr string
	}{
		{
			name: "valid proxy",
			proxy: Proxy{
				Protocol: "http",
				Host:     "127.0.0.1",
				Port:     8080,
			},
			expectedErr: "",
		},
		{
			name: "invalid protocol",
			proxy: Proxy{
				Protocol: "ftp",
				Host:     "127.0.0.1",
				Port:     8080,
			},
			expectedErr: `invalid protocol: "ftp"`,
		},
		{
			name: "invalid host",
			proxy: Proxy{
				Protocol: "https",
				Host:     "",
				Port:     8080,
			},
			expectedErr: "host is required",
		},
		{
			name: "invalid port",
			proxy: Proxy{
				Protocol: "https",
				Host:     "172.25.25.12",
				Port:     0,
			},
			expectedErr: "port is required",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.proxy.Validate()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProxy_URL(t *testing.T) {
	proxy := Proxy{
		Protocol: "socks5",
		Host:     "127.0.0.1",
		Port:     8000,
		Auth: &BasicAuth{
			User: "admin",
			Pass: "pasw1d",
		},
	}

	expectedUrl := &url.URL{
		Scheme: "socks5",
		User:   url.UserPassword("admin", "pasw1d"),
		Host:   "127.0.0.1:8000",
	}

	assert.Equal(t, expectedUrl, proxy.URL())
}
