package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestClassifyUpstreamError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		msg    string
	}{
		{
			name:   "nil",
			err:    nil,
			status: http.StatusBadGateway,
			msg:    "bad gateway",
		},
		{
			name:   "context canceled",
			err:    context.Canceled,
			status: http.StatusBadGateway,
			msg:    "bad gateway",
		},
		{
			name:   "context deadline exceeded",
			err:    context.DeadlineExceeded,
			status: http.StatusGatewayTimeout,
			msg:    "gateway timeout",
		},
		{
			name:   "net timeout",
			err:    timeoutError{},
			status: http.StatusGatewayTimeout,
			msg:    "gateway timeout",
		},
		{
			name:   "wrapped net timeout",
			err:    fmt.Errorf("dial: %w", timeoutError{}),
			status: http.StatusGatewayTimeout,
			msg:    "gateway timeout",
		},
		{
			name:   "dns failure",
			err:    &net.DNSError{Err: "no such host", Name: "missing.example"},
			status: http.StatusBadGateway,
			msg:    "dns lookup failed",
		},
		{
			name:   "dns timeout prefers 504",
			err:    &net.DNSError{Err: "i/o timeout", Name: "slow.example", IsTimeout: true},
			status: http.StatusGatewayTimeout,
			msg:    "gateway timeout",
		},
		{
			name:   "connection refused",
			err:    syscall.ECONNREFUSED,
			status: http.StatusBadGateway,
			msg:    "connection refused",
		},
		{
			name:   "wrapped connection refused",
			err:    &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			status: http.StatusBadGateway,
			msg:    "connection refused",
		},
		{
			name:   "generic",
			err:    fmt.Errorf("something else"),
			status: http.StatusBadGateway,
			msg:    "bad gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, msg := classifyUpstreamError(tt.err)
			assert.Equal(t, tt.status, status)
			assert.Equal(t, tt.msg, msg)
		})
	}
}
