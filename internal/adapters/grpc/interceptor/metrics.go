package interceptor

import (
	"context"
	"path"
	"time"

	"github.com/aknEvrnky/pgway/internal/platform/metrics"
	"google.golang.org/grpc"
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
		result := metrics.ResultOK
		if err != nil {
			result = metrics.ResultError
		}
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
