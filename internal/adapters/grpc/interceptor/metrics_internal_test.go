package interceptor

import (
	"fmt"
	"testing"

	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResultFromGRPCError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, metrics.ResultOK},
		{"unauthenticated", status.Error(codes.Unauthenticated, "no token"), metrics.ResultRejected},
		{"permission denied", status.Error(codes.PermissionDenied, "denied"), metrics.ResultRejected},
		{"rate limited", status.Error(codes.ResourceExhausted, "limit"), metrics.ResultRejected},
		{"invalid argument", status.Error(codes.InvalidArgument, "bad"), metrics.ResultRejected},
		{"deadline", status.Error(codes.DeadlineExceeded, "slow"), metrics.ResultTimeout},
		{"internal", status.Error(codes.Internal, "boom"), metrics.ResultError},
		{"wrapped", fmt.Errorf("call: %w", status.Error(codes.Unauthenticated, "x")), metrics.ResultRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resultFromGRPCError(tt.err))
		})
	}
}
