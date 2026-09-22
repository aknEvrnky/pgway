package interceptor

import (
	"context"
	"path"
	"time"

	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnaryMetrics records unary CP RPC count and duration (outermost interceptor).
func UnaryMetrics() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		result := resultFromGRPCError(err)
		metrics.RecordCPRPC(ctx, metrics.CPRPCRecord{
			Method:   shortMethod(info.FullMethod),
			Result:   result,
			Duration: time.Since(start),
		})
		return resp, err
	}
}

func shortMethod(fullMethod string) string {
	// "/pkg.Service/Method" → "Method"
	return path.Base(fullMethod)
}

// resultFromGRPCError maps handler errors to metric result labels: client-side
// rejections (auth, permission, rate limit, bad request) and deadlines are
// distinguished from genuine server errors so they don't pollute the error rate.
func resultFromGRPCError(err error) string {
	if err == nil {
		return metrics.ResultOK
	}
	switch status.Code(err) {
	case codes.Unauthenticated, codes.PermissionDenied, codes.ResourceExhausted, codes.InvalidArgument:
		return metrics.ResultRejected
	case codes.DeadlineExceeded:
		return metrics.ResultTimeout
	default:
		return metrics.ResultError
	}
}
